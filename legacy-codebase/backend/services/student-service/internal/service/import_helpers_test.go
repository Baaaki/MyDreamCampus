package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectedCSVHeader mirrors the header literal used by the production code
// (import_service.go:56). Keeping it here so a header-shape change in
// production trips this test loudly.
var expectedCSVHeader = []string{
	"student_number", "first_name", "last_name", "email",
	"faculty", "department", "enrollment_year", "class_level",
}

func TestSanitizeCSVCell(t *testing.T) {
	t.Run("trims leading and trailing whitespace", func(t *testing.T) {
		assert.Equal(t, "alice", sanitizeCSVCell("  alice  "))
		assert.Equal(t, "bob", sanitizeCSVCell("\tbob\n"))
	})

	t.Run("normal cells pass through unchanged", func(t *testing.T) {
		assert.Equal(t, "Computer Science", sanitizeCSVCell("Computer Science"))
		assert.Equal(t, "20210001", sanitizeCSVCell("20210001"))
	})

	t.Run("empty cell stays empty", func(t *testing.T) {
		assert.Equal(t, "", sanitizeCSVCell(""))
		assert.Equal(t, "", sanitizeCSVCell("   "))
	})

	t.Run("formula prefixes are neutralized with leading apostrophe", func(t *testing.T) {
		// Excel and Google Sheets evaluate cells starting with these characters
		// as formulas — escaping with `'` keeps them as literal text.
		dangerous := map[string]string{
			"=cmd|' /C calc'!A0": "'=cmd|' /C calc'!A0",
			"+SUM(A1:A2)":        "'+SUM(A1:A2)",
			"-1+1":               "'-1+1",
			"@SUM(1+1)":          "'@SUM(1+1)",
		}
		for in, want := range dangerous {
			assert.Equal(t, want, sanitizeCSVCell(in), "input=%q", in)
		}
	})

	t.Run("tab and CR prefixes are neutralized after trim was tried", func(t *testing.T) {
		// strings.TrimSpace removes leading tabs/CR before the formula check
		// runs, so a pure "\tfoo" becomes "foo" — not "'\tfoo". This test
		// pins that observable behavior so a future refactor that reorders
		// trim/sanitize has to update it consciously.
		assert.Equal(t, "foo", sanitizeCSVCell("\tfoo"))
		assert.Equal(t, "foo", sanitizeCSVCell("\rfoo"))
	})

	t.Run("trim does not undo formula injection inside the cell", func(t *testing.T) {
		// Whitespace then "=" should still be flagged: the production order is
		// trim -> first-char check, so "  =danger" trims to "=danger" which
		// then gets the apostrophe.
		assert.Equal(t, "'=danger", sanitizeCSVCell("  =danger"))
	})
}

func TestValidateHeader(t *testing.T) {
	t.Run("accepts an exact match", func(t *testing.T) {
		got := append([]string(nil), expectedCSVHeader...)
		assert.True(t, validateHeader(got, expectedCSVHeader))
	})

	t.Run("tolerates surrounding whitespace in incoming header", func(t *testing.T) {
		got := []string{
			"student_number ", " first_name", "\tlast_name", "email",
			"faculty", "department", "enrollment_year", "class_level\n",
		}
		assert.True(t, validateHeader(got, expectedCSVHeader))
	})

	t.Run("rejects different length", func(t *testing.T) {
		short := expectedCSVHeader[:7]
		assert.False(t, validateHeader(short, expectedCSVHeader))

		long := append(append([]string{}, expectedCSVHeader...), "extra")
		assert.False(t, validateHeader(long, expectedCSVHeader))
	})

	t.Run("rejects mismatched column name", func(t *testing.T) {
		got := append([]string(nil), expectedCSVHeader...)
		got[3] = "e_mail" // typo
		assert.False(t, validateHeader(got, expectedCSVHeader))
	})

	t.Run("rejects different column ordering", func(t *testing.T) {
		got := append([]string(nil), expectedCSVHeader...)
		got[0], got[1] = got[1], got[0]
		assert.False(t, validateHeader(got, expectedCSVHeader))
	})

	t.Run("is case sensitive", func(t *testing.T) {
		got := append([]string(nil), expectedCSVHeader...)
		got[0] = "Student_Number"
		assert.False(t, validateHeader(got, expectedCSVHeader))
	})

	t.Run("rejects empty input against non-empty expected", func(t *testing.T) {
		assert.False(t, validateHeader(nil, expectedCSVHeader))
		assert.False(t, validateHeader([]string{}, expectedCSVHeader))
	})
}

func TestParseStudentRecord(t *testing.T) {
	validRecord := func() []string {
		return []string{
			"20210001", "Ahmet", "Yılmaz", "ahmet@univ.edu",
			"Engineering", "Computer Science", "2021", "2",
		}
	}

	t.Run("parses a well-formed row", func(t *testing.T) {
		params, err := parseStudentRecord(validRecord(), 2)
		require.NoError(t, err)

		assert.Equal(t, "20210001", params.StudentNumber)
		assert.Equal(t, "Ahmet", params.FirstName)
		assert.Equal(t, "Yılmaz", params.LastName)
		assert.Equal(t, "ahmet@univ.edu", params.Email)
		assert.Equal(t, "Engineering", params.Faculty)
		assert.Equal(t, "Computer Science", params.Department)
		assert.Equal(t, int32(2021), params.EnrollmentYear)
		assert.Equal(t, int16(2), params.ClassLevel)

		// Advisor is set to the zero UUID — actual assignment happens later
		// in the flow once an advisor is matched. The point here is that the
		// row parser does not invent one.
		assert.Equal(t, [16]byte{}, params.AdvisorID.Bytes)
	})

	t.Run("rejects rows with the wrong column count", func(t *testing.T) {
		short := validRecord()[:7]
		_, err := parseStudentRecord(short, 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Line 5")
		assert.Contains(t, err.Error(), "expected 8 columns")
	})

	t.Run("rejects non-numeric enrollment_year", func(t *testing.T) {
		row := validRecord()
		row[6] = "twenty-one"
		_, err := parseStudentRecord(row, 7)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Line 7")
		assert.Contains(t, err.Error(), "enrollment_year")
	})

	t.Run("rejects non-numeric class_level", func(t *testing.T) {
		row := validRecord()
		row[7] = "freshman"
		_, err := parseStudentRecord(row, 9)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Line 9")
		assert.Contains(t, err.Error(), "class_level")
	})

	t.Run("rejects class_level out of range [1,6]", func(t *testing.T) {
		// Numeric out-of-range values that sanitize() leaves untouched (no
		// formula-prefix) reach the explicit range check.
		for _, level := range []string{"0", "7", "100"} {
			row := validRecord()
			row[7] = level
			_, err := parseStudentRecord(row, 3)
			require.Error(t, err, "level=%s", level)
			assert.Contains(t, err.Error(), "class_level must be between 1 and 6", "level=%s", level)
		}
	})

	t.Run("negative class_level is rejected by the parse step (formula sanitization runs first)", func(t *testing.T) {
		// "-1" is intercepted by sanitizeCSVCell before strconv ever sees it
		// because '-' is treated as an Excel-formula leading character. The
		// resulting "'-1" then fails ParseInt — so the user sees an "invalid
		// class_level" error rather than the range message. This trip-up is
		// load-bearing: it documents that sanitization runs before validation.
		row := validRecord()
		row[7] = "-1"
		_, err := parseStudentRecord(row, 3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid class_level")
	})

	t.Run("accepts the full valid class_level range", func(t *testing.T) {
		for _, level := range []string{"1", "2", "3", "4", "5", "6"} {
			row := validRecord()
			row[7] = level
			_, err := parseStudentRecord(row, 1)
			assert.NoError(t, err, "level=%s", level)
		}
	})

	t.Run("sanitizes formula-injection in any field", func(t *testing.T) {
		row := validRecord()
		row[1] = "=HYPERLINK(\"evil\")" // first_name with formula
		params, err := parseStudentRecord(row, 4)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(params.FirstName, "'="),
			"first_name should be neutralized; got %q", params.FirstName)
	})

	t.Run("trims surrounding whitespace via cell sanitization", func(t *testing.T) {
		row := validRecord()
		row[0] = "  20210001  "
		row[3] = "  ahmet@univ.edu\n"
		params, err := parseStudentRecord(row, 2)
		require.NoError(t, err)
		assert.Equal(t, "20210001", params.StudentNumber)
		assert.Equal(t, "ahmet@univ.edu", params.Email)
	})
}
