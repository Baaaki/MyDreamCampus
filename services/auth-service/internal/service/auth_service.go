package service

import (
	"context"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/auth-service/config"
	"github.com/baaaki/mydreamcampus/auth-service/internal/db"
	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	"github.com/baaaki/mydreamcampus/auth-service/internal/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/redis"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
)

type AuthService struct {
	authRepo    *repository.AuthRepository
	sessionRepo *repository.SessionRepository
	eventRepo   *repository.EventRepository
	redisClient *redis.ClientWrapper
	config      *config.Config
}

func NewAuthService(
	authRepo *repository.AuthRepository,
	sessionRepo *repository.SessionRepository,
	eventRepo *repository.EventRepository,
	redisClient *redis.ClientWrapper,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		authRepo:    authRepo,
		sessionRepo: sessionRepo,
		eventRepo:   eventRepo,
		redisClient: redisClient,
		config:      cfg,
	}
}

// Login authenticates a user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, deviceInfo, ipAddress string) (dto.LoginResponse, error) {
	// Rate limit check (implement with Redis later)
	// TODO: Email-based rate limiting
	// TODO: IP-based rate limiting

	// Get user by email
	user, err := s.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		logger.Warn("login attempt for non-existent user",
			zap.String("email", req.Email),
		)
		return dto.LoginResponse{}, sharedErrors.ErrUnauthorized
	}

	// Check if account is active
	if !utils.DerefBool(user.IsActive, false) {
		logger.Warn("login attempt for deactivated account",
			zap.String("email", req.Email),
		)
		return dto.LoginResponse{}, fmt.Errorf("account deactivated")
	}

	// Check if account is locked
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now()) {
		logger.Warn("login attempt for locked account",
			zap.String("email", req.Email),
			zap.Time("locked_until", user.LockedUntil.Time),
		)
		return dto.LoginResponse{}, fmt.Errorf("account locked until %s", user.LockedUntil.Time.Format(time.RFC3339))
	}

	// Verify password
	if !utils.VerifyPassword(user.PasswordHash, req.Password) {
		// Increment failed attempts
		_ = s.authRepo.IncrementFailedLoginAttempts(ctx, utils.PgtypeToUUID(user.ID))

		// Lock account if too many failures (5+ attempts)
		if utils.DerefInt32(user.FailedLoginAttempts, 0)+1 >= 5 {
			lockUntil := time.Now().Add(30 * time.Minute)
			_ = s.authRepo.LockAccount(ctx, db.LockAccountParams{
				ID: user.ID,
				LockedUntil: pgtype.Timestamp{
					Time:  lockUntil,
					Valid: true,
				},
			})
			logger.Warn("account locked due to too many failed attempts",
				zap.String("email", req.Email),
			)
		}

		logger.Warn("invalid password",
			zap.String("email", req.Email),
		)
		return dto.LoginResponse{}, sharedErrors.ErrUnauthorized
	}

	// Reset failed login attempts on successful login
	_ = s.authRepo.ResetFailedLoginAttempts(ctx, utils.PgtypeToUUID(user.ID))

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		logger.Error("failed to generate access token",
			zap.Error(err),
		)
		return dto.LoginResponse{}, sharedErrors.ErrInternalServer
	}

	// Generate refresh token (stored in cookie, not returned in response)
	_, jti, err := s.generateRefreshToken(user)
	if err != nil {
		logger.Error("failed to generate refresh token",
			zap.Error(err),
		)
		return dto.LoginResponse{}, sharedErrors.ErrInternalServer
	}

	// Create session
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	deviceInfoPtr := utils.StringToPointer(deviceInfo)
	ipAddressPtr := utils.StringToPointer(ipAddress)
	_, err = s.sessionRepo.CreateSession(ctx, db.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenJti:  jti,
		DeviceInfo:       deviceInfoPtr,
		IpAddress:        ipAddressPtr,
		ExpiresAt:        pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		logger.Error("failed to create session",
			zap.Error(err),
		)
		return dto.LoginResponse{}, sharedErrors.ErrInternalServer
	}

	// Build response
	response := dto.LoginResponse{
		AccessToken:         accessToken,
		ExpiresIn:           s.config.JWT.AccessTokenExpiry * 60, // convert to seconds
		ForcePasswordChange: utils.DerefBool(user.ForcePasswordChange, false),
		User: dto.UserResponse{
			ID:         utils.PgtypeToUUID(user.ID).String(),
			Email:      user.Email,
			Role:       user.Role,
			Department: user.Department,
		},
	}

	if utils.DerefBool(user.ForcePasswordChange, false) {
		response.Message = "İlk girişinizde şifrenizi değiştirmeniz gerekmektedir."
	}

	logger.Info("user logged in successfully",
		zap.String("email", req.Email),
		zap.String("role", user.Role),
	)

	return response, nil
}

// Logout invalidates the current session
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	// Parse token without validation (even expired tokens should be processable)
	claims, err := s.parseRefreshTokenWithoutValidation(refreshToken)
	if err != nil {
		return sharedErrors.ErrUnauthorized
	}

	// Delete session
	jti := claims["jti"].(string)
	err = s.sessionRepo.DeleteSession(ctx, jti)
	if err != nil {
		logger.Error("failed to delete session",
			zap.Error(err),
			zap.String("jti", jti),
		)
		// Don't return error, logout should succeed even if session not found
	}

	logger.Info("user logged out",
		zap.String("jti", jti),
	)

	return nil
}

// LogoutAll invalidates all sessions for a user
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	// Increment token version (invalidates all tokens)
	newVersion, err := s.authRepo.IncrementTokenVersion(ctx, userID)
	if err != nil {
		logger.Error("failed to increment token version",
			zap.Error(err),
		)
		return sharedErrors.ErrInternalServer
	}

	// Update Redis cache
	err = s.redisClient.SetTokenVersion(ctx, userID.String(), int(newVersion))
	if err != nil {
		logger.Error("failed to update token version in Redis",
			zap.Error(err),
		)
		// Continue even if Redis fails
	}

	// Delete all sessions
	err = s.sessionRepo.DeleteAllUserSessions(ctx, userID)
	if err != nil {
		logger.Error("failed to delete all sessions",
			zap.Error(err),
		)
		return sharedErrors.ErrInternalServer
	}

	logger.Info("user logged out from all devices",
		zap.String("user_id", userID.String()),
	)

	return nil
}

// RefreshAccessToken generates a new access token using refresh token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (dto.RefreshResponse, string, error) {
	// Parse and validate refresh token
	claims, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.ErrUnauthorized
	}

	userID, err := uuid.Parse(claims["user_id"].(string))
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.ErrUnauthorized
	}

	jti := claims["jti"].(string)
	tokenVersion := int32(claims["token_version"].(float64))

	// Check if session exists
	session, err := s.sessionRepo.GetSessionByJTI(ctx, jti)
	if err != nil {
		logger.Warn("refresh attempt with invalid session",
			zap.String("jti", jti),
		)
		return dto.RefreshResponse{}, "", sharedErrors.ErrUnauthorized
	}

	// Get user
	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.ErrUnauthorized
	}

	// Check token version
	if tokenVersion != utils.DerefInt32(user.TokenVersion, 0) {
		logger.Warn("refresh attempt with revoked token",
			zap.String("user_id", userID.String()),
			zap.Int32("token_version", tokenVersion),
			zap.Int32("current_version", utils.DerefInt32(user.TokenVersion, 0)),
		)
		return dto.RefreshResponse{}, "", sharedErrors.ErrUnauthorized
	}

	// Generate new tokens (Token Rotation)
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.ErrInternalServer
	}

	newRefreshToken, newJTI, err := s.generateRefreshToken(user)
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.ErrInternalServer
	}

	// Delete old session
	_ = s.sessionRepo.DeleteSession(ctx, jti)

	// Create new session
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	_, err = s.sessionRepo.CreateSession(ctx, db.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenJti:  newJTI,
		DeviceInfo:       session.DeviceInfo,
		IpAddress:        session.IpAddress,
		ExpiresAt:        pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.ErrInternalServer
	}

	logger.Info("access token refreshed",
		zap.String("user_id", userID.String()),
	)

	return dto.RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   s.config.JWT.AccessTokenExpiry * 60,
	}, newRefreshToken, nil
}

// ChangePassword changes user password
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) (dto.ChangePasswordResponse, string, error) {
	// Get user
	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrUnauthorized
	}

	// Verify old password
	if !utils.VerifyPassword(user.PasswordHash, req.OldPassword) {
		logger.Warn("invalid old password during password change",
			zap.String("user_id", userID.String()),
		)
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrUnauthorized
	}

	// Validate new password (TODO: implement password policy)
	if len(req.NewPassword) < 8 {
		return dto.ChangePasswordResponse{}, "", fmt.Errorf("password must be at least 8 characters")
	}

	// Hash new password
	newPasswordHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		logger.Error("failed to hash password",
			zap.Error(err),
		)
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrInternalServer
	}

	// Update password (this also increments token_version)
	err = s.authRepo.UpdatePassword(ctx, userID, newPasswordHash, false)
	if err != nil {
		logger.Error("failed to update password",
			zap.Error(err),
		)
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrInternalServer
	}

	// Get updated user with new token version
	user, err = s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrInternalServer
	}

	// Update Redis cache
	err = s.redisClient.SetTokenVersion(ctx, userID.String(), int(utils.DerefInt32(user.TokenVersion, 0)))
	if err != nil {
		logger.Error("failed to update token version in Redis",
			zap.Error(err),
		)
	}

	// Delete all sessions except current one
	err = s.sessionRepo.DeleteAllUserSessions(ctx, userID)
	if err != nil {
		logger.Error("failed to delete sessions",
			zap.Error(err),
		)
	}

	// Generate new tokens for current session
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrInternalServer
	}

	refreshToken, jti, err := s.generateRefreshToken(user)
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrInternalServer
	}

	// Create new session
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	_, err = s.sessionRepo.CreateSession(ctx, db.CreateSessionParams{
		UserID:          user.ID,
		RefreshTokenJti: jti,
		ExpiresAt:       pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.ErrInternalServer
	}

	logger.Info("password changed successfully",
		zap.String("user_id", userID.String()),
	)

	return dto.ChangePasswordResponse{
		Message:     "Password changed successfully",
		AccessToken: accessToken,
		ExpiresIn:   s.config.JWT.AccessTokenExpiry * 60,
	}, refreshToken, nil
}

// GetUserSessions returns all active sessions for a user
func (s *AuthService) GetUserSessions(ctx context.Context, userID uuid.UUID, currentJTI string) (dto.SessionsResponse, error) {
	sessions, err := s.sessionRepo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return dto.SessionsResponse{}, sharedErrors.ErrInternalServer
	}

	var sessionResponses []dto.SessionResponse
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, dto.SessionResponse{
			ID:         utils.PgtypeToUUID(session.ID).String(),
			DeviceInfo: session.DeviceInfo,
			IPAddress:  session.IpAddress,
			CreatedAt:  session.CreatedAt.Time,
			LastUsedAt: session.LastUsedAt.Time,
			IsCurrent:  session.RefreshTokenJti == currentJTI,
		})
	}

	return dto.SessionsResponse{Sessions: sessionResponses}, nil
}

// DeleteSession deletes a specific session
func (s *AuthService) DeleteSession(ctx context.Context, sessionID, userID uuid.UUID, currentJTI string) error {
	// Get session to check if it's current
	sessions, err := s.sessionRepo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return sharedErrors.ErrInternalServer
	}

	for _, session := range sessions {
		if utils.PgtypeToUUID(session.ID) == sessionID {
			if session.RefreshTokenJti == currentJTI {
				return fmt.Errorf("cannot terminate current session, use logout instead")
			}
		}
	}

	err = s.sessionRepo.DeleteSessionByID(ctx, sessionID, userID)
	if err != nil {
		return sharedErrors.ErrInternalServer
	}

	logger.Info("session deleted",
		zap.String("session_id", sessionID.String()),
		zap.String("user_id", userID.String()),
	)

	return nil
}

// SeedAdmin creates the initial admin user
func (s *AuthService) SeedAdmin(ctx context.Context) error {
	// Check if admin already exists
	exists, err := s.authRepo.AdminExists(ctx)
	if err != nil {
		return err
	}

	if exists {
		logger.Info("admin user already exists, skipping seed")
		return nil
	}

	// Hash default password
	passwordHash, err := utils.HashPassword(s.config.Admin.InitialPassword)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Fixed admin UUID (same as in Staff Service)
	adminID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Create admin user
	_, err = s.authRepo.CreateUser(ctx, db.CreateUserParams{
		ID:                  utils.UUIDToPgtype(adminID),
		Email:               s.config.Admin.Email,
		PasswordHash:        passwordHash,
		Role:                "admin",
		Department:          nil,
		IsActive:            utils.BoolPtr(true),
		TokenVersion:        utils.Int32Ptr(1),
		ForcePasswordChange: utils.BoolPtr(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	logger.Info("admin user seeded successfully",
		zap.String("email", s.config.Admin.Email),
	)

	return nil
}

// StartCleanupScheduler starts background cleanup tasks
func (s *AuthService) StartCleanupScheduler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				// Cleanup expired sessions
				err := s.sessionRepo.CleanupExpiredSessions(ctx)
				if err != nil {
					logger.Error("failed to cleanup expired sessions",
						zap.Error(err),
					)
				} else {
					logger.Info("expired sessions cleaned up")
				}

				// Cleanup old processed events (30 days)
				err = s.eventRepo.CleanupOldProcessedEvents(ctx, "30 days")
				if err != nil {
					logger.Error("failed to cleanup old processed events",
						zap.Error(err),
					)
				} else {
					logger.Info("old processed events cleaned up")
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	logger.Info("cleanup scheduler started")
}

// generateAccessToken creates a JWT access token
func (s *AuthService) generateAccessToken(user db.User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.config.JWT.AccessTokenExpiry) * time.Minute)

	claims := jwt.MapClaims{
		"user_id":       utils.PgtypeToUUID(user.ID).String(),
		"role":          user.Role,
		"department":    utils.StringPointerToString(user.Department),
		"token_version": utils.DerefInt32(user.TokenVersion, 0),
		"exp":           expiresAt.Unix(),
		"iat":           now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// generateRefreshToken creates a JWT refresh token
func (s *AuthService) generateRefreshToken(user db.User) (string, string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	jti := uuid.New().String()

	claims := jwt.MapClaims{
		"user_id":       utils.PgtypeToUUID(user.ID).String(),
		"jti":           jti,
		"token_version": user.TokenVersion,
		"exp":           expiresAt.Unix(),
		"iat":           now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return "", "", err
	}

	return tokenString, jti, nil
}

// parseRefreshToken parses and validates a refresh token
func (s *AuthService) parseRefreshToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.config.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// parseRefreshTokenWithoutValidation parses token without expiry validation
func (s *AuthService) parseRefreshTokenWithoutValidation(tokenString string) (jwt.MapClaims, error) {
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWT.Secret), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// hashPasswordArgon2 hashes a password using Argon2id (alternative to utils.HashPassword)
func hashPasswordArgon2(password string) string {
	salt := []byte("mydreamcampus-salt-2025") // In production, use random salt per user
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%x", hash)
}
