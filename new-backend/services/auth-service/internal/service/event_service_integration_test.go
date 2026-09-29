//go:build integration

package service

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	"github.com/baaaki/mydreamcampus/auth/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/testutil/pgtest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopRevoker struct{}

func (noopRevoker) BlacklistAllUserTokens(context.Context, string, int) error { return nil }

func newEventDB(t *testing.T) (*EventService, *pgxpool.Pool) {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	pool := pgtest.New(t, "auth", "../sql/migrations")
	svc := NewEventService(repository.NewAuthRepository(pool), repository.NewEventRepository(pool), pool, noopRevoker{}, nil)
	return svc, pool
}

func studentCreated(userID uuid.UUID, email string) dto.StudentCreatedEvent {
	return dto.StudentCreatedEvent{
		BaseEvent: dto.BaseEvent{EventID: uuid.NewString(), EventType: "student.created"},
		Data:      dto.StudentCreatedData{ID: userID.String(), Email: email, FirstName: "Zeynep", LastName: "Şahin"},
	}
}

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, pool.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}

func TestHandleStudentCreated_Redelivery_SkippedWithoutASecondWelcome(t *testing.T) {
	svc, pool := newEventDB(t)
	ctx := context.Background()
	id := uuid.New()
	require.NoError(t, svc.HandleStudentCreated(ctx, studentCreated(id, "zeynep@uni.edu.tr")))

	again := studentCreated(id, "zeynep@uni.edu.tr")
	require.NoError(t, svc.HandleStudentCreated(ctx, again))

	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.users"))
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.outbox_events"), "one welcome e-mail")
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.processed_events WHERE event_id = $1", again.EventID))
}

// The email belongs to another user id. Retrying cannot fix that, so the
// event must be logged and marked processed, not bounced through the retry
// line into the dead-letter queue. The unique violation aborts the
// transaction the handler goes on to use, which only Postgres shows.
func TestHandleStudentCreated_EmailOwnedByAnotherUser_MarkedProcessed(t *testing.T) {
	svc, pool := newEventDB(t)
	ctx := context.Background()
	owner := uuid.New()
	require.NoError(t, svc.HandleStudentCreated(ctx, studentCreated(owner, "zeynep@uni.edu.tr")))

	clash := studentCreated(uuid.New(), "zeynep@uni.edu.tr")
	err := svc.HandleStudentCreated(ctx, clash)

	require.NoError(t, err)
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.users"))
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.users WHERE id = $1", owner), "the owner keeps the email")
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.outbox_events"), "no welcome for a user that was not created")
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.processed_events WHERE event_id = $1", clash.EventID))
}

func TestHandleStaffCreated_EmailOwnedByAnotherUser_MarkedProcessed(t *testing.T) {
	svc, pool := newEventDB(t)
	ctx := context.Background()
	require.NoError(t, svc.HandleStudentCreated(ctx, studentCreated(uuid.New(), "ahmet@uni.edu.tr")))

	clash := dto.StaffCreatedEvent{
		BaseEvent: dto.BaseEvent{EventID: uuid.NewString(), EventType: "staff.created"},
		Data:      dto.StaffCreatedData{ID: uuid.NewString(), Email: "ahmet@uni.edu.tr", Role: "teacher", FirstName: "Ahmet", LastName: "Yılmaz"},
	}
	err := svc.HandleStaffCreated(ctx, clash)

	require.NoError(t, err)
	assert.Equal(t, 0, count(t, pool, "SELECT count(*) FROM auth.users WHERE role = 'teacher'"))
	assert.Equal(t, 1, count(t, pool, "SELECT count(*) FROM auth.processed_events WHERE event_id = $1", clash.EventID))
}
