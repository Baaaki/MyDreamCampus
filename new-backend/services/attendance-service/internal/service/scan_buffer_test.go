package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlushSessionBuffer_CorruptRecord_DroppedWithTheRest(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	_, err := w.service(nil).ScanQR(context.Background(), s[0], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err)
	w.buffer[sessionID.String()][s[1].String()] = "{not json"

	err = FlushSessionBuffer(context.Background(), attRecordView{w}, attRedisView{w}, sessionID.String())

	require.NoError(t, err)
	assert.Len(t, w.records, 1)
	assert.Empty(t, w.buffer[sessionID.String()], "a record that never parses must not be replayed forever")
}

func TestFlushSessionBuffer_InsertFails_KeepsParsedScans(t *testing.T) {
	w, courseID, s := newAttWorld(t)
	sessionID := openTheory(t, w, courseID, 1)
	_, err := w.service(nil).ScanQR(context.Background(), s[0], scanRequest(w, sessionID, time.Now()))
	require.NoError(t, err)
	w.buffer[sessionID.String()][s[1].String()] = "{not json"
	w.failBatchInsert = true

	err = FlushSessionBuffer(context.Background(), attRecordView{w}, attRedisView{w}, sessionID.String())

	require.Error(t, err)
	assert.Empty(t, w.records)
	assert.Contains(t, w.buffer[sessionID.String()], s[0].String(), "the scan waits for the next flush")
	assert.NotContains(t, w.buffer[sessionID.String()], s[1].String())
}
