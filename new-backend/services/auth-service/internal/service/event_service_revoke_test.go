package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// userRowTx stands in for one auth.users row inside a transaction. It
// applies the WHERE clauses of the queries the event handlers run, so the
// order of those queries matters the way it does against Postgres.
type userRowTx struct {
	pgx.Tx
	email        string
	tokenVersion int32
	exists       bool
	committed    bool
}

func (t *userRowTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "name: UpdateUser ") {
		if email, ok := args[0].(*string); ok && email != nil {
			t.email = *email
		}
	}
	return pgconn.CommandTag{}, nil
}

func (t *userRowTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	switch {
	case !t.exists:
	case strings.Contains(sql, "name: CheckEmailVersionSync "):
		if args[1].(string) != t.email {
			t.tokenVersion++
			return versionRow{version: t.tokenVersion}
		}
	case strings.Contains(sql, "name: DeactivateUser "):
		t.tokenVersion++
		return versionRow{version: t.tokenVersion}
	case strings.Contains(sql, "name: GetUserByID "):
		return userScanRow{email: t.email}
	}
	return versionRow{err: pgx.ErrNoRows}
}

func (t *userRowTx) Commit(context.Context) error   { t.committed = true; return nil }
func (t *userRowTx) Rollback(context.Context) error { return nil }

type versionRow struct {
	version int32
	err     error
}

func (r versionRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	v := r.version
	*dest[0].(**int32) = &v
	return nil
}

type userScanRow struct {
	email string
	err   error
}

func (r userScanRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) > 1 {
		if ptr, ok := dest[1].(*string); ok {
			*ptr = r.email
		}
	}
	return nil
}

type singleTx struct{ tx *userRowTx }

func (b singleTx) Begin(context.Context) (pgx.Tx, error) { return b.tx, nil }

type neverProcessed struct{}

func (neverProcessed) IsEventProcessed(context.Context, string) (bool, error) { return false, nil }

type recordingRevoker struct {
	calls map[string]int
	err   error
}

func (r *recordingRevoker) BlacklistAllUserTokens(_ context.Context, userID string, version int) error {
	r.calls[userID] = version
	return r.err
}

func newRevokeTestService(t *testing.T, tx *userRowTx) (*EventService, *recordingRevoker) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	revoker := &recordingRevoker{calls: map[string]int{}}
	return &EventService{
		eventRepo: neverProcessed{},
		pool:      singleTx{tx: tx},
		cache:     revoker,
	}, revoker
}

func TestHandleUserDeactivated_RevokesWithNewTokenVersion(t *testing.T) {
	tx := &userRowTx{email: "a@uni.edu.tr", tokenVersion: 3, exists: true}
	svc, revoker := newRevokeTestService(t, tx)
	userID := uuid.NewString()

	err := svc.HandleUserDeactivated(context.Background(), dto.UserDeactivatedEvent{
		BaseEvent: dto.BaseEvent{EventID: "ev-1", EventType: "student.deactivated"},
		Data:      dto.UserDeactivatedData{ID: userID},
	})

	require.NoError(t, err)
	assert.True(t, tx.committed)
	assert.Equal(t, map[string]int{userID: 4}, revoker.calls)
}

func TestHandleUserDeactivated_UnknownUser_CompletesWithoutRevoke(t *testing.T) {
	svc, revoker := newRevokeTestService(t, &userRowTx{})

	err := svc.HandleUserDeactivated(context.Background(), dto.UserDeactivatedEvent{
		BaseEvent: dto.BaseEvent{EventID: "ev-2", EventType: "staff.deactivated"},
		Data:      dto.UserDeactivatedData{ID: uuid.NewString()},
	})

	require.NoError(t, err)
	assert.Empty(t, revoker.calls)
}

func TestHandleUserDeactivated_RedisDown_StillSucceeds(t *testing.T) {
	tx := &userRowTx{tokenVersion: 1, exists: true}
	svc, revoker := newRevokeTestService(t, tx)
	revoker.err = errors.New("redis down")

	err := svc.HandleUserDeactivated(context.Background(), dto.UserDeactivatedEvent{
		BaseEvent: dto.BaseEvent{EventID: "ev-3", EventType: "student.deactivated"},
		Data:      dto.UserDeactivatedData{ID: uuid.NewString()},
	})

	require.NoError(t, err, "the DB side is committed; a Redis failure must not fail the event")
	assert.True(t, tx.committed)
}

func TestHandleUserUpdated_EmailChanged_RevokesWithNewTokenVersion(t *testing.T) {
	tx := &userRowTx{email: "old@uni.edu.tr", tokenVersion: 2, exists: true}
	svc, revoker := newRevokeTestService(t, tx)
	userID := uuid.NewString()

	err := svc.HandleUserUpdated(context.Background(), dto.UserUpdatedEvent{
		BaseEvent: dto.BaseEvent{EventID: "ev-4", EventType: "staff.updated"},
		Data: dto.UserUpdatedData{
			ID:            userID,
			ChangedFields: map[string]string{"email": "new@uni.edu.tr"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "new@uni.edu.tr", tx.email)
	assert.Equal(t, map[string]int{userID: 3}, revoker.calls)
}

func TestHandleUserUpdated_DepartmentOnly_DoesNotRevoke(t *testing.T) {
	tx := &userRowTx{email: "a@uni.edu.tr", tokenVersion: 2, exists: true}
	svc, revoker := newRevokeTestService(t, tx)

	err := svc.HandleUserUpdated(context.Background(), dto.UserUpdatedEvent{
		BaseEvent: dto.BaseEvent{EventID: "ev-5", EventType: "staff.updated"},
		Data: dto.UserUpdatedData{
			ID:            uuid.NewString(),
			ChangedFields: map[string]string{"department": "Fizik"},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, revoker.calls)
}

func TestHandleUserDeactivated_ProtectedAccount_Ignored(t *testing.T) {
	tx := &userRowTx{email: "admin@university.edu.tr", tokenVersion: 2, exists: true}
	svc, revoker := newRevokeTestService(t, tx)
	svc.config = &config.Config{
		Admin: config.AdminConfig{
			Email: "admin@university.edu.tr",
		},
	}

	err := svc.HandleUserDeactivated(context.Background(), dto.UserDeactivatedEvent{
		BaseEvent: dto.BaseEvent{EventID: "ev-protected", EventType: "staff.deactivated"},
		Data:      dto.UserDeactivatedData{ID: uuid.NewString()},
	})

	require.NoError(t, err)
	assert.True(t, tx.committed, "event should be committed as processed so it's not redelivered")
	assert.Equal(t, int32(2), tx.tokenVersion, "user must NOT be deactivated; token version unchanged")
	assert.Empty(t, revoker.calls, "no access token revocation for protected user")
}
