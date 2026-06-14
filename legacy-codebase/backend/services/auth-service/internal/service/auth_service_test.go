package service

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/auth-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthRepository mocks auth repository
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(db.User), args.Error(1)
}

func (m *MockAuthRepository) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.User), args.Error(1)
}

func (m *MockAuthRepository) UpdateUserPassword(ctx context.Context, id pgtype.UUID, passwordHash string) error {
	args := m.Called(ctx, id, passwordHash)
	return args.Error(0)
}

func (m *MockAuthRepository) IncrementTokenVersion(ctx context.Context, id pgtype.UUID) (int32, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int32), args.Error(1)
}

// MockSessionRepository mocks session repository
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) CreateSession(ctx context.Context, userID pgtype.UUID, jti string, deviceInfo, ipAddress *string, expiresAt time.Time) (db.Session, error) {
	args := m.Called(ctx, userID, jti, deviceInfo, ipAddress, expiresAt)
	return args.Get(0).(db.Session), args.Error(1)
}

func (m *MockSessionRepository) GetSessionByJTI(ctx context.Context, jti string) (db.Session, error) {
	args := m.Called(ctx, jti)
	return args.Get(0).(db.Session), args.Error(1)
}

func (m *MockSessionRepository) DeleteSession(ctx context.Context, sessionID, userID pgtype.UUID) error {
	args := m.Called(ctx, sessionID, userID)
	return args.Error(0)
}

func (m *MockSessionRepository) DeleteAllUserSessions(ctx context.Context, userID pgtype.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionRepository) GetUserSessions(ctx context.Context, userID pgtype.UUID) ([]db.Session, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]db.Session), args.Error(1)
}

func (m *MockSessionRepository) UpdateSessionLastUsed(ctx context.Context, jti string) error {
	args := m.Called(ctx, jti)
	return args.Error(0)
}

// MockEventRepository mocks event repository
type MockEventRepository struct {
	mock.Mock
}

func (m *MockEventRepository) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	args := m.Called(ctx, eventID)
	return args.Bool(0), args.Error(1)
}

func (m *MockEventRepository) MarkEventProcessed(ctx context.Context, eventID string) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

// MockRedisClient mocks Redis client
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) SetTokenVersion(ctx context.Context, userID string, version int) error {
	args := m.Called(ctx, userID, version)
	return args.Error(0)
}

func (m *MockRedisClient) GetTokenVersion(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockRedisClient) Close() error {
	return nil
}

// Helper to create pgtype.UUID from uuid.UUID
func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// TestGetUserSessions tests retrieving user sessions
func TestGetUserSessions(t *testing.T) {
	// Setup mocks
	sessionRepo := new(MockSessionRepository)

	ctx := context.Background()
	userID := uuid.New()
	pgUserID := pgUUID(userID)
	currentJTI := "current-jti-123"

	// Mock data
	now := time.Now()
	sessions := []db.Session{
		{
			ID:              pgUUID(uuid.New()),
			UserID:          pgUserID,
			RefreshTokenJti: currentJTI,
			DeviceInfo:      stringPtr("Chrome/Linux"),
			IpAddress:       stringPtr("192.168.1.1"),
			CreatedAt:       pgtype.Timestamp{Time: now, Valid: true},
			LastUsedAt:      pgtype.Timestamp{Time: now, Valid: true},
			ExpiresAt:       pgtype.Timestamp{Time: now.Add(24 * time.Hour), Valid: true},
		},
		{
			ID:              pgUUID(uuid.New()),
			UserID:          pgUserID,
			RefreshTokenJti: "other-jti-456",
			DeviceInfo:      stringPtr("Firefox/Windows"),
			IpAddress:       stringPtr("192.168.1.2"),
			CreatedAt:       pgtype.Timestamp{Time: now.Add(-2 * time.Hour), Valid: true},
			LastUsedAt:      pgtype.Timestamp{Time: now.Add(-1 * time.Hour), Valid: true},
			ExpiresAt:       pgtype.Timestamp{Time: now.Add(22 * time.Hour), Valid: true},
		},
	}

	// Setup expectation
	sessionRepo.On("GetUserSessions", ctx, pgUserID).Return(sessions, nil)

	// Execute
	result, err := sessionRepo.GetUserSessions(ctx, pgUserID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, currentJTI, result[0].RefreshTokenJti)
	assert.Equal(t, "other-jti-456", result[1].RefreshTokenJti)

	sessionRepo.AssertExpectations(t)
}

// TestDeleteSession tests deleting a specific session
func TestDeleteSession(t *testing.T) {
	// Setup
	sessionRepo := new(MockSessionRepository)
	ctx := context.Background()

	userID := uuid.New()
	sessionID := uuid.New()
	pgUserID := pgUUID(userID)
	pgSessionID := pgUUID(sessionID)

	// Setup expectation
	sessionRepo.On("DeleteSession", ctx, pgSessionID, pgUserID).Return(nil)

	// Execute
	err := sessionRepo.DeleteSession(ctx, pgSessionID, pgUserID)

	// Assert
	assert.NoError(t, err)
	sessionRepo.AssertExpectations(t)
}

// TestDeleteAllUserSessions tests deleting all user sessions
func TestDeleteAllUserSessions(t *testing.T) {
	// Setup
	sessionRepo := new(MockSessionRepository)
	authRepo := new(MockAuthRepository)

	ctx := context.Background()
	userID := uuid.New()
	pgUserID := pgUUID(userID)

	// Setup expectations
	sessionRepo.On("DeleteAllUserSessions", ctx, pgUserID).Return(nil)
	authRepo.On("IncrementTokenVersion", ctx, pgUserID).Return(int32(2), nil)

	// Execute
	err := sessionRepo.DeleteAllUserSessions(ctx, pgUserID)

	// Assert
	assert.NoError(t, err)
	sessionRepo.AssertExpectations(t)
}

// TestPasswordHashing tests password utility functions
func TestPasswordHashing(t *testing.T) {
	password := "TestPassword123!"

	// Hash password
	hash, err := utils.HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify correct password
	valid := utils.VerifyPassword(hash, password)
	assert.True(t, valid)

	// Verify wrong password
	invalid := utils.VerifyPassword(hash, "WrongPassword123!")
	assert.False(t, invalid)
}

// TestEventIdempotency tests event processing idempotency
func TestEventIdempotency(t *testing.T) {
	eventRepo := new(MockEventRepository)
	ctx := context.Background()
	eventID := "event-123"

	t.Run("New event not processed", func(t *testing.T) {
		eventRepo.On("IsEventProcessed", ctx, eventID).Return(false, nil).Once()

		processed, err := eventRepo.IsEventProcessed(ctx, eventID)

		assert.NoError(t, err)
		assert.False(t, processed)
	})

	t.Run("Mark event as processed", func(t *testing.T) {
		eventRepo.On("MarkEventProcessed", ctx, eventID).Return(nil).Once()

		err := eventRepo.MarkEventProcessed(ctx, eventID)

		assert.NoError(t, err)
	})

	t.Run("Already processed event", func(t *testing.T) {
		eventRepo.On("IsEventProcessed", ctx, eventID).Return(true, nil).Once()

		processed, err := eventRepo.IsEventProcessed(ctx, eventID)

		assert.NoError(t, err)
		assert.True(t, processed)
	})

	eventRepo.AssertExpectations(t)
}

// TestRedisTokenVersionCache tests Redis token version caching
func TestRedisTokenVersionCache(t *testing.T) {
	redisClient := new(MockRedisClient)
	ctx := context.Background()
	userID := uuid.New().String()
	version := 5

	t.Run("Set token version", func(t *testing.T) {
		redisClient.On("SetTokenVersion", ctx, userID, version).Return(nil).Once()

		err := redisClient.SetTokenVersion(ctx, userID, version)

		assert.NoError(t, err)
	})

	t.Run("Get token version", func(t *testing.T) {
		redisClient.On("GetTokenVersion", ctx, userID).Return(version, nil).Once()

		cachedVersion, err := redisClient.GetTokenVersion(ctx, userID)

		assert.NoError(t, err)
		assert.Equal(t, version, cachedVersion)
	})

	redisClient.AssertExpectations(t)
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
