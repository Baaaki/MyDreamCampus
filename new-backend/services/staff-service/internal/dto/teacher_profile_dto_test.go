package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEducation_RoundTrip(t *testing.T) {
	e := Education{ID: "e-1", Degree: "PhD", Institution: "MIT", Department: "CS", Year: 2020}
	data, err := json.Marshal(e)
	require.NoError(t, err)

	var got Education
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, e, got)
}

func TestArticle_OmitsEmptyOptionals(t *testing.T) {
	a := Article{ID: "1", Title: "X", Journal: "J", Year: 2024, Authors: "A"}
	data, err := json.Marshal(a)
	require.NoError(t, err)
	str := string(data)
	for _, omitted := range []string{"doi", "journalType", "issuePageYear", "language"} {
		assert.NotContains(t, str, omitted)
	}
}

func TestProject_OptionalEndYear(t *testing.T) {
	p := Project{ID: "p-1", Title: "X", Role: "PI", Funder: "TUBITAK", StartYear: 2024, Status: "ongoing"}
	data, err := json.Marshal(p)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "endYear")

	end := 2025
	p.EndYear = &end
	data, err = json.Marshal(p)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"endYear":2025`)
}

func TestUpdateTeacherProfileRequest_AllOptional(t *testing.T) {
	// Empty body must parse — patch semantics
	var got UpdateTeacherProfileRequest
	require.NoError(t, json.Unmarshal([]byte(`{}`), &got))
	assert.Nil(t, got.AcademicTitle)
	assert.Nil(t, got.Education)
}

func TestUpdateTeacherProfileRequest_ParsesNestedArrays(t *testing.T) {
	body := []byte(`{
		"academic_title":"Prof.",
		"faculty":"Engineering",
		"education":[{"id":"e1","degree":"PhD","institution":"X","department":"CS","year":2020}]
	}`)
	var got UpdateTeacherProfileRequest
	require.NoError(t, json.Unmarshal(body, &got))
	require.NotNil(t, got.AcademicTitle)
	assert.Equal(t, "Prof.", *got.AcademicTitle)
	require.NotNil(t, got.Education)
	require.Len(t, *got.Education, 1)
	assert.Equal(t, "PhD", (*got.Education)[0].Degree)
}

func TestTeacherProfileResponse_DefaultsToEmptyArrays(t *testing.T) {
	// Important for frontend: nil slices serialize as null,
	// empty []T{} serializes as []. Make sure consumers can handle both.
	r := TeacherProfileResponse{
		ID: "1", StaffID: "s1", FirstName: "A", LastName: "B", Email: "x@y.tr",
		Education: []Education{}, Articles: []Article{},
		Bulletins: []Bulletin{}, Projects: []Project{},
		Awards: []Award{}, Scholarships: []Scholarship{},
		AdminAssignments: []AdminAssignment{},
	}
	data, err := json.Marshal(r)
	require.NoError(t, err)
	str := string(data)
	assert.Contains(t, str, `"education":[]`)
	assert.Contains(t, str, `"articles":[]`)
	assert.Contains(t, str, `"projects":[]`)
}
