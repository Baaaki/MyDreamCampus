package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringToPgText(t *testing.T) {
	t.Run("empty string is invalid", func(t *testing.T) {
		got := StringToPgText("")
		assert.False(t, got.Valid)
		assert.Empty(t, got.String)
	})
	t.Run("non-empty is valid", func(t *testing.T) {
		got := StringToPgText("hello")
		assert.True(t, got.Valid)
		assert.Equal(t, "hello", got.String)
	})
}

func TestPointerStringToPgText(t *testing.T) {
	tests := []struct {
		name    string
		input   *string
		wantOk  bool
		wantStr string
	}{
		{"nil pointer", nil, false, ""},
		{"empty pointer", strPtr(""), false, ""},
		{"valid pointer", strPtr("data"), true, "data"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PointerStringToPgText(tt.input)
			assert.Equal(t, tt.wantOk, got.Valid)
			assert.Equal(t, tt.wantStr, got.String)
		})
	}
}

func TestPgTextRoundTrip(t *testing.T) {
	original := "campus@example.tr"
	pg := StringToPgText(original)
	assert.Equal(t, original, PgTextToString(pg))
	assert.Equal(t, original, PgtypeTextToString(pg))

	invalid := pgtype.Text{Valid: false}
	assert.Equal(t, "", PgTextToString(invalid))
	assert.Nil(t, PgtypeTextToStringPtr(invalid))
	assert.Nil(t, PgTextToStringPtr(invalid))
}

func TestUUIDPgtypeRoundTrip(t *testing.T) {
	id := uuid.New()
	pg := UUIDToPgtype(id)
	require.True(t, pg.Valid)
	assert.Equal(t, id, PgtypeToUUID(pg))
	assert.Equal(t, id.String(), PgtypeToUUIDString(pg))

	// Aliases must behave identically
	assert.Equal(t, pg, UUIDToPgUUID(id))
	assert.Equal(t, id, PgUUIDToUUID(pg))
}

func TestPointerUUIDToPgtype(t *testing.T) {
	id := uuid.New()
	pg := PointerUUIDToPgtype(&id)
	assert.True(t, pg.Valid)
	assert.Equal(t, id, uuid.UUID(pg.Bytes))

	pg = PointerUUIDToPgtype(nil)
	assert.False(t, pg.Valid)
}

func TestUUIDToPgtypeNullable(t *testing.T) {
	id := uuid.New()
	assert.True(t, UUIDToPgtypeNullable(id).Valid)
	assert.False(t, UUIDToPgtypeNullable(uuid.Nil).Valid, "nil UUID must map to invalid")
}

func TestUUIDArrayToPgtype(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	out := UUIDArrayToPgtype(ids)
	require.Len(t, out, 3)
	for i, id := range ids {
		assert.True(t, out[i].Valid)
		assert.Equal(t, id, uuid.UUID(out[i].Bytes))
	}
	assert.Empty(t, UUIDArrayToPgtype(nil), "nil input returns empty slice")
}

func TestPointerInt16ToPgInt2(t *testing.T) {
	v := int16(42)
	got := PointerInt16ToPgInt2(&v)
	assert.True(t, got.Valid)
	assert.Equal(t, int16(42), got.Int16)

	got = PointerInt16ToPgInt2(nil)
	assert.False(t, got.Valid)
}

func TestStringPointerToString(t *testing.T) {
	assert.Equal(t, "hi", StringPointerToString(strPtr("hi")))
	assert.Equal(t, "", StringPointerToString(nil))
	assert.Equal(t, "hi", StringPointerValue(strPtr("hi")))
	assert.Equal(t, "", StringPointerValue(nil))
}

func TestStringToPointer(t *testing.T) {
	assert.Nil(t, StringToPointer(""))
	p := StringToPointer("data")
	require.NotNil(t, p)
	assert.Equal(t, "data", *p)
}

func TestBoolHelpers(t *testing.T) {
	bp := BoolPtr(true)
	require.NotNil(t, bp)
	assert.True(t, *bp)

	assert.True(t, DerefBool(BoolPtr(true), false))
	assert.False(t, DerefBool(nil, false))
	assert.True(t, DerefBool(nil, true), "default value must be returned on nil")

	pg := BoolToPgBool(true)
	assert.True(t, pg.Valid)
	assert.True(t, PgBoolToBool(pg))
	assert.False(t, PgBoolToBool(pgtype.Bool{Valid: false}))
}

func TestInt32Helpers(t *testing.T) {
	v := Int32Ptr(7)
	require.NotNil(t, v)
	assert.Equal(t, int32(7), *v)
	assert.Equal(t, int32(7), DerefInt32(v, 0))
	assert.Equal(t, int32(99), DerefInt32(nil, 99))
}

func TestInt16Helpers(t *testing.T) {
	pg := Int16ToPgInt2(5)
	assert.True(t, pg.Valid)
	assert.Equal(t, int16(5), pg.Int16)
	assert.Equal(t, pg, Int16ToPgtypeNullable(5))

	p := PgInt2ToInt16Ptr(pg)
	require.NotNil(t, p)
	assert.Equal(t, int16(5), *p)
	assert.Nil(t, PgInt2ToInt16Ptr(pgtype.Int2{Valid: false}))

	assert.Equal(t, int16(11), Int16PointerValue(int16Ptr(11)))
	assert.Equal(t, int16(0), Int16PointerValue(nil))
}

func TestNumericConversion(t *testing.T) {
	pg := Float64ToPgNumeric(123.45)
	assert.True(t, pg.Valid)
	got, err := PgNumericToFloat64(pg)
	require.NoError(t, err)
	assert.InDelta(t, 123.45, got, 0.001)

	got, err = PgNumericToFloat64(pgtype.Numeric{Valid: false})
	require.NoError(t, err)
	assert.Equal(t, float64(0), got)
}

func TestTimestampRoundTrip(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 30, 0, 0, time.UTC)

	ts := TimeToPgTimestamp(now)
	assert.True(t, ts.Valid)
	assert.Equal(t, now, PgTimestampToTime(ts))
	assert.Equal(t, time.Time{}, PgTimestampToTime(pgtype.Timestamp{Valid: false}))

	tz := TimeToPgTimestamptz(now)
	assert.True(t, tz.Valid)
	assert.Equal(t, now, PgTimestamptzToTime(tz))
	assert.Equal(t, time.Time{}, PgTimestamptzToTime(pgtype.Timestamptz{Valid: false}))

	p := PgTimestamptzToTimePtr(tz)
	require.NotNil(t, p)
	assert.Equal(t, now, *p)
	assert.Nil(t, PgTimestamptzToTimePtr(pgtype.Timestamptz{Valid: false}))
}

func TestStringToPgtypeNullableAlias(t *testing.T) {
	// alias: must equal direct call
	a := StringToPgText("x")
	b := StringToPgtypeNullable("x")
	assert.Equal(t, a, b)
}

func strPtr(s string) *string { return &s }
func int16Ptr(i int16) *int16 { return &i }
