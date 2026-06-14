package worker

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseReferenceID is the wire boundary between payment-service's
// reference_id format and meal-service's routing decision (single vs batch).
// Locking the *exact* prefix semantics — a regression here means a batch
// payment confirms a single reservation (or vice versa), and the affected
// rows can't be reverse-engineered from the event later.
func TestParseReferenceID_SinglePrefix(t *testing.T) {
	id := uuid.New()
	got, isBatch, err := parseReferenceID("res_" + id.String())

	require.NoError(t, err)
	assert.Equal(t, id, got)
	assert.False(t, isBatch, "res_ prefix must route to single reservation path")
}

func TestParseReferenceID_BatchPrefix(t *testing.T) {
	id := uuid.New()
	got, isBatch, err := parseReferenceID("bat_" + id.String())

	require.NoError(t, err)
	assert.Equal(t, id, got)
	assert.True(t, isBatch, "bat_ prefix must route to batch reservation path")
}

func TestParseReferenceID_UnknownPrefix(t *testing.T) {
	// No prefix recognized — uuid.Parse fails on the literal "foo_<uuid>"
	// and the consumer returns the error rather than coercing to single.
	_, _, err := parseReferenceID("foo_22222222-2222-2222-2222-222222222222")
	require.Error(t, err)
}

func TestParseReferenceID_NoPrefix(t *testing.T) {
	// Bare UUID parses fine and defaults to single (isBatch=false).
	// This is the canonical "no prefix" tolerance — payment-service mock
	// historically sent bare UUIDs in early dev. Pinned to keep that
	// backwards-compatible path working.
	id := uuid.New()
	got, isBatch, err := parseReferenceID(id.String())

	require.NoError(t, err)
	assert.Equal(t, id, got)
	assert.False(t, isBatch)
}

func TestParseReferenceID_InvalidUUID(t *testing.T) {
	_, _, err := parseReferenceID("res_not-a-uuid")
	require.Error(t, err, "uuid.Parse must reject malformed UUIDs even with valid prefix")
}

func TestParseReferenceID_EmptyString(t *testing.T) {
	_, isBatch, err := parseReferenceID("")
	require.Error(t, err)
	assert.False(t, isBatch, "empty input must not route to batch path")
}

func TestParseReferenceID_BatchPrefixOnlyDetectedAtStart(t *testing.T) {
	// "res_bat_<uuid>" must NOT be misidentified as batch — only a leading
	// "bat_" counts. Strip "res_" first → "bat_<uuid>" → that *would* be
	// batch on a fresh input, but isBatch is computed on the ORIGINAL input
	// before any trimming. Pin that ordering.
	id := uuid.New()
	got, isBatch, err := parseReferenceID("res_bat_" + id.String())

	require.NoError(t, err)
	assert.False(t, isBatch, "isBatch reflects the leading prefix of the original string, not what's left after trimming")
	// After stripping "res_" then "bat_" the bare uuid is what remains.
	assert.Equal(t, id, got)
}
