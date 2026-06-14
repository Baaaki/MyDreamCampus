package utils

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// StringToPgText converts string to pgtype.Text
func StringToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// PointerStringToPgText converts *string to pgtype.Text
func PointerStringToPgText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// PgTextToString converts pgtype.Text to string
func PgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// UUIDToPgtype converts uuid.UUID to pgtype.UUID
func UUIDToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// PgtypeToUUID converts pgtype.UUID to uuid.UUID
func PgtypeToUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

// PgtypeToUUIDString converts pgtype.UUID to string
func PgtypeToUUIDString(id pgtype.UUID) string {
	return uuid.UUID(id.Bytes).String()
}

// PointerUUIDToPgtype converts *uuid.UUID to pgtype.UUID
func PointerUUIDToPgtype(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{
		Bytes: *id,
		Valid: true,
	}
}

// PointerInt16ToPgInt2 converts *int16 to pgtype.Int2
func PointerInt16ToPgInt2(i *int16) pgtype.Int2 {
	if i == nil {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{Int16: *i, Valid: true}
}

// StringPointerToString converts *string to string
func StringPointerToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// StringToPointer converts string to *string
func StringToPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// PgtypeTextToStringPtr converts pgtype.Text to *string
func PgtypeTextToStringPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// BoolPtr converts bool to *bool
func BoolPtr(b bool) *bool {
	return &b
}

// DerefBool dereferences *bool with default value
func DerefBool(b *bool, defaultValue bool) bool {
	if b == nil {
		return defaultValue
	}
	return *b
}

// DerefInt32 dereferences *int32 with default value
func DerefInt32(i *int32, defaultValue int32) int32 {
	if i == nil {
		return defaultValue
	}
	return *i
}


// Int32Ptr converts int32 to *int32
func Int32Ptr(i int32) *int32 {
	return &i
}

// PgtypeTextToString converts pgtype.Text to string (alias for compatibility)
func PgtypeTextToString(t pgtype.Text) string {
	return PgTextToString(t)
}

// StringPointerValue returns string value from pointer, empty string if nil
func StringPointerValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Int16PointerValue returns int16 value from pointer, zero if nil
func Int16PointerValue(i *int16) int16 {
	if i == nil {
		return 0
	}
	return *i
}

// PgTextToStringPtr converts pgtype.Text to *string (alias for compatibility)
func PgTextToStringPtr(t pgtype.Text) *string {
	return PgtypeTextToStringPtr(t)
}

// StringToPgtypeNullable converts string to pgtype.Text (nullable)
func StringToPgtypeNullable(s string) pgtype.Text {
	return StringToPgText(s)
}

// Int16ToPgtypeNullable converts int16 to pgtype.Int2
func Int16ToPgtypeNullable(i int16) pgtype.Int2 {
	return pgtype.Int2{Int16: i, Valid: true}
}

// UUIDToPgtypeNullable converts uuid.UUID to pgtype.UUID (nullable)
func UUIDToPgtypeNullable(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// UUIDArrayToPgtype converts []uuid.UUID to []pgtype.UUID
func UUIDArrayToPgtype(ids []uuid.UUID) []pgtype.UUID {
	result := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		result[i] = UUIDToPgtype(id)
	}
	return result
}

// BoolToPgBool converts bool to pgtype.Bool
func BoolToPgBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

// PgBoolToBool converts pgtype.Bool to bool
func PgBoolToBool(b pgtype.Bool) bool {
	return b.Valid && b.Bool
}

// Int16ToPgInt2 converts int16 to pgtype.Int2
func Int16ToPgInt2(i int16) pgtype.Int2 {
	return pgtype.Int2{Int16: i, Valid: true}
}

// PgInt2ToInt16Ptr converts pgtype.Int2 to *int16
func PgInt2ToInt16Ptr(i pgtype.Int2) *int16 {
	if !i.Valid {
		return nil
	}
	return &i.Int16
}

// Float64ToPgNumeric converts float64 to pgtype.Numeric
func Float64ToPgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	// pgtype.Numeric.Scan accepts string format
	n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

// PgNumericToFloat64 converts pgtype.Numeric to float64
func PgNumericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, nil
	}
	// Use Float64Value method which returns (float64, error)
	f, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	return f.Float64, nil
}

// TimeToPgTimestamp converts time.Time to pgtype.Timestamp
func TimeToPgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t, Valid: true}
}

// PgTimestampToTime converts pgtype.Timestamp to time.Time
func PgTimestampToTime(t pgtype.Timestamp) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// TimeToPgTimestamptz converts time.Time to pgtype.Timestamptz
func TimeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// PgTimestamptzToTime converts pgtype.Timestamptz to time.Time
func PgTimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// PgTimestamptzToTimePtr converts pgtype.Timestamptz to *time.Time
func PgTimestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// TimeToPgDate converts time.Time to pgtype.Date
func TimeToPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

// PgDateToTime converts pgtype.Date to time.Time
func PgDateToTime(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}

// Aliases for consistency
func UUIDToPgUUID(id uuid.UUID) pgtype.UUID {
	return UUIDToPgtype(id)
}

func PgUUIDToUUID(id pgtype.UUID) uuid.UUID {
	return PgtypeToUUID(id)
}
