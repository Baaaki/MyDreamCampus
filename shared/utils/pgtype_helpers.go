package utils

import (
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
