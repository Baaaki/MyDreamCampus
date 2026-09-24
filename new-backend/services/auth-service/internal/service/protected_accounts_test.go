package service

import (
	"context"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type protectedUserRepoMock struct {
	userStore
	user db.User
	err  error
}

func (m *protectedUserRepoMock) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	if m.err != nil {
		return db.User{}, m.err
	}
	return m.user, nil
}

type protectedSessionRepoMock struct {
	sessionStore
	sessions []db.Session
	err      error
}

func (m *protectedSessionRepoMock) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]db.Session, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sessions, nil
}

func TestChangePassword_DemoAccount_Forbidden(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	userID := uuid.New()
	s := &AuthService{
		authRepo: &protectedUserRepoMock{
			user: db.User{
				ID:     utils.UUIDToPgtype(userID),
				IsDemo: true,
			},
		},
	}

	_, _, err := s.ChangePassword(context.Background(), userID, dto.ChangePasswordRequest{
		OldPassword: "OldPassword123!",
		NewPassword: "NewPassword123!",
	})
	assert.ErrorIs(t, err, serviceErrors.ErrDemoAccountActionForbidden)
}

func TestLogoutAll_DemoAccount_Forbidden(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	userID := uuid.New()
	s := &AuthService{
		authRepo: &protectedUserRepoMock{
			user: db.User{
				ID:     utils.UUIDToPgtype(userID),
				IsDemo: true,
			},
		},
	}

	err := s.LogoutAll(context.Background(), userID, "access-token")
	assert.ErrorIs(t, err, serviceErrors.ErrDemoAccountActionForbidden)
}

func TestDeleteSession_DemoAccount_CannotDeleteOtherSessions(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	userID := uuid.New()
	otherSessionID := uuid.New()
	currentJTI := "current-jti"

	s := &AuthService{
		authRepo: &protectedUserRepoMock{
			user: db.User{
				ID:     utils.UUIDToPgtype(userID),
				IsDemo: true,
			},
		},
		sessionRepo: &protectedSessionRepoMock{
			sessions: []db.Session{
				{
					ID:              utils.UUIDToPgtype(otherSessionID),
					RefreshTokenJti: "other-jti",
				},
			},
		},
	}

	err := s.DeleteSession(context.Background(), otherSessionID, userID, currentJTI)
	assert.ErrorIs(t, err, serviceErrors.ErrDemoAccountActionForbidden)
}

func TestDeleteSession_CurrentSession_ReturnsErrCannotTerminateSession(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	userID := uuid.New()
	currentSessionID := uuid.New()
	currentJTI := "current-jti"

	s := &AuthService{
		authRepo: &protectedUserRepoMock{
			user: db.User{
				ID:     utils.UUIDToPgtype(userID),
				IsDemo: true,
			},
		},
		sessionRepo: &protectedSessionRepoMock{
			sessions: []db.Session{
				{
					ID:              utils.UUIDToPgtype(currentSessionID),
					RefreshTokenJti: currentJTI,
				},
			},
		},
	}

	err := s.DeleteSession(context.Background(), currentSessionID, userID, currentJTI)
	assert.ErrorIs(t, err, serviceErrors.ErrCannotTerminateSession)
}

func TestRequestPasswordReset_ProtectedAccount_SilentlyIgnored(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{
		Admin: config.AdminConfig{
			Email: "admin@university.edu.tr",
		},
		Demo: config.DemoConfig{
			AdminEmail:   "demo.admin@mydreamcampus.com",
			TeacherEmail: "ahmet.yilmaz@uni.edu.tr",
			StudentEmail: "zeynep.sahin@uni.edu.tr",
		},
		ProtectedAccountEmails: []string{
			"admin@university.edu.tr",
			"demo.admin@mydreamcampus.com",
			"ahmet.yilmaz@uni.edu.tr",
			"zeynep.sahin@uni.edu.tr",
		},
	}

	s := &AuthService{
		config: cfg,
	}

	protectedEmails := []string{
		"admin@university.edu.tr",
		"ADMIN@university.edu.tr",
		"demo.admin@mydreamcampus.com",
		"ahmet.yilmaz@uni.edu.tr",
		"zeynep.sahin@uni.edu.tr",
	}

	for _, email := range protectedEmails {
		err := s.RequestPasswordReset(context.Background(), email)
		assert.NoError(t, err, "must silently return nil for protected email %s", email)
	}
}

func TestGetUserSessions_DemoAccount_MasksIPAndShortensDevice(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	userID := uuid.New()
	currentSessionID := uuid.New()
	otherSessionID := uuid.New()
	currentJTI := "current-jti"

	currIP := "85.105.12.34"
	currDev := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	otherIP := "192.168.1.100"
	otherDev := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

	s := &AuthService{
		authRepo: &protectedUserRepoMock{
			user: db.User{
				ID:     utils.UUIDToPgtype(userID),
				IsDemo: true,
			},
		},
		sessionRepo: &protectedSessionRepoMock{
			sessions: []db.Session{
				{
					ID:              utils.UUIDToPgtype(currentSessionID),
					RefreshTokenJti: currentJTI,
					IpAddress:       &currIP,
					DeviceInfo:      &currDev,
					CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
					LastUsedAt:      pgtype.Timestamp{Time: time.Now(), Valid: true},
					ExpiresAt:       pgtype.Timestamp{Time: time.Now().Add(time.Hour), Valid: true},
				},
				{
					ID:              utils.UUIDToPgtype(otherSessionID),
					RefreshTokenJti: "other-jti",
					IpAddress:       &otherIP,
					DeviceInfo:      &otherDev,
					CreatedAt:       pgtype.Timestamp{Time: time.Now(), Valid: true},
					LastUsedAt:      pgtype.Timestamp{Time: time.Now(), Valid: true},
					ExpiresAt:       pgtype.Timestamp{Time: time.Now().Add(time.Hour), Valid: true},
				},
			},
		},
	}

	resp, err := s.GetUserSessions(context.Background(), userID, currentJTI)
	require.NoError(t, err)
	require.Len(t, resp.Sessions, 2)

	// Find current and other
	var currSession, otherSession dto.SessionResponse
	for _, sess := range resp.Sessions {
		if sess.IsCurrent {
			currSession = sess
		} else {
			otherSession = sess
		}
	}

	// Current session unmasked
	assert.Equal(t, currIP, *currSession.IPAddress)
	assert.Equal(t, currDev, *currSession.DeviceInfo)

	// Other session masked & shortened
	assert.Equal(t, "192.168.x.x", *otherSession.IPAddress)
	assert.Equal(t, string([]rune(otherDev)[:24])+"...", *otherSession.DeviceInfo)
}

func TestMaskIP_And_ShortenDeviceInfo(t *testing.T) {
	assert.Equal(t, "192.168.x.x", maskIP("192.168.1.1"))
	assert.Equal(t, "85.105.x.x", maskIP("85.105.20.30"))
	assert.Equal(t, "2001:db8::x:x", maskIP("2001:db8:85a3::8a2e"))
	assert.Equal(t, "", maskIP(""))

	assert.Equal(t, "short device", shortenDeviceInfo("short device"))
	assert.Equal(t, "123456789012345678901234...", shortenDeviceInfo("123456789012345678901234567890"))
}
