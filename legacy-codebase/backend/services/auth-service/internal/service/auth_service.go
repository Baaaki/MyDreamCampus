package service

import (
	"context"
	"fmt"
	"time"

	"github.com/baaaki/mydreamcampus/auth-service/config"
	"github.com/baaaki/mydreamcampus/auth-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/clock"
	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/auth-service/internal/errors"
	"github.com/baaaki/mydreamcampus/auth-service/internal/repository"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/errors"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/redis"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
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
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, deviceInfo, ipAddress string) (dto.LoginResponse, string, error) {
	// Create child logger with service context
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "AuthService"),
		zap.String("method", "Login"),
		zap.String("email", req.Email),
	)

	// Rate limit check (implement with Redis later)
	// TODO: Email-based rate limiting
	// TODO: IP-based rate limiting

	// Get user by email
	user, err := s.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Check if user not found
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			// Timing-safe: run a real Argon2id verification against a
			// precomputed dummy hash so response time matches the
			// password-mismatch branch and email enumeration is closed.
			utils.VerifyDummyPassword(req.Password)
			serviceLogger.Warn("login attempt for non-existent user")
			return dto.LoginResponse{}, "", serviceErrors.ErrInvalidCredentials
		}
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.LoginResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.LoginResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check if account is active. Treat deactivated accounts identically
	// to "user not found" externally to avoid leaking which emails exist
	// in the system; only the audit trail records the real reason.
	if !utils.DerefBool(user.IsActive, false) {
		utils.VerifyDummyPassword(req.Password)
		serviceLogger.Warn("login attempt for deactivated account")
		return dto.LoginResponse{}, "", serviceErrors.ErrInvalidCredentials
	}

	// Check if account is locked
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(clock.Now()) {
		serviceLogger.Warn("login attempt for locked account",
			zap.Time("locked_until", user.LockedUntil.Time),
		)
		return dto.LoginResponse{}, "", serviceErrors.ErrAccountLocked
	}

	// Verify password
	if !utils.VerifyPassword(user.PasswordHash, req.Password) {
		// Increment failed attempts
		_ = s.authRepo.IncrementFailedLoginAttempts(ctx, utils.PgtypeToUUID(user.ID))

		// Lock account if too many failures (5+ attempts)
		if utils.DerefInt32(user.FailedLoginAttempts, 0)+1 >= 5 {
			lockUntil := clock.Now().Add(30 * time.Minute)
			_ = s.authRepo.LockAccount(ctx, db.LockAccountParams{
				ID: user.ID,
				LockedUntil: pgtype.Timestamp{
					Time:  lockUntil,
					Valid: true,
				},
			})
			serviceLogger.Warn("account locked due to too many failed attempts",
				zap.Int("failed_attempts", 5),
			)
		}

		serviceLogger.Warn("invalid password")
		return dto.LoginResponse{}, "", serviceErrors.ErrInvalidCredentials
	}

	// Reset failed login attempts on successful login
	_ = s.authRepo.ResetFailedLoginAttempts(ctx, utils.PgtypeToUUID(user.ID))

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		// Wrap and return, handler will log
		return dto.LoginResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Generate refresh token
	refreshToken, jti, err := s.generateRefreshToken(user)
	if err != nil {
		// Wrap and return, handler will log
		return dto.LoginResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Create session
	expiresAt := clock.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	refreshTokenTTL := time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour
	deviceInfoPtr := utils.StringToPointer(deviceInfo)
	ipAddressPtr := utils.StringToPointer(ipAddress)
	_, err = s.sessionRepo.CreateSession(ctx, db.CreateSessionParams{
		UserID:          user.ID,
		RefreshTokenJti: jti,
		DeviceInfo:      deviceInfoPtr,
		IpAddress:       ipAddressPtr,
		ExpiresAt:       pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.LoginResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.LoginResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Store refresh token in Redis
	userIDStr := utils.PgtypeToUUID(user.ID).String()
	if err := s.redisClient.StoreRefreshToken(ctx, jti, userIDStr, refreshTokenTTL); err != nil {
		logger.Error("failed to store refresh token in Redis",
			zap.Error(err),
			zap.String("user_id", userIDStr),
		)
		// Continue even if Redis fails - DB session is the source of truth
	}

	serviceLogger.Info("login successful in database",
		zap.String("user_id", utils.PgtypeToUUID(user.ID).String()),
		zap.String("role", user.Role),
	)

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

	return response, refreshToken, nil
}

// Logout invalidates the current session and blacklists the access token
func (s *AuthService) Logout(ctx context.Context, refreshToken string, accessToken string, authenticatedUserID string) error {
	// Parse refresh token without validation (even expired tokens should be processable)
	claims, err := s.parseRefreshTokenWithoutValidation(refreshToken)
	if err != nil {
		return serviceErrors.ErrInvalidToken
	}

	// Verify refresh token ownership — prevent cross-user session deletion
	tokenUserID, ok := claims["user_id"].(string)
	if !ok || tokenUserID != authenticatedUserID {
		logger.Warn("logout attempt with mismatched refresh token owner",
			zap.String("authenticated_user", authenticatedUserID),
			zap.String("token_user", tokenUserID),
		)
		return serviceErrors.ErrInvalidToken
	}

	// Delete session from DB
	jti := claims["jti"].(string)
	err = s.sessionRepo.DeleteSession(ctx, jti)
	if err != nil {
		// Check if session not found
		if sharedErrors.Is(err, serviceErrors.ErrSessionNotFoundRepo) {
			// Don't return error, logout should succeed even if session not found
			logger.Info("logout attempted for already deleted session",
				zap.String("jti", jti),
			)
		} else if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			logger.Error("database error deleting session",
				zap.Error(err),
				zap.String("jti", jti),
			)
			// Don't return error, logout should succeed
		}
	}

	// Delete refresh token from Redis
	if err := s.redisClient.DeleteRefreshToken(ctx, jti); err != nil {
		logger.Error("failed to delete refresh token from Redis",
			zap.Error(err),
			zap.String("jti", jti),
		)
	}

	// Blacklist the access token if provided
	if accessToken != "" {
		if err := s.blacklistAccessToken(ctx, accessToken); err != nil {
			logger.Error("failed to blacklist access token",
				zap.Error(err),
			)
			// Continue even if blacklisting fails
		}
	}

	logger.Info("user logged out",
		zap.String("refresh_jti", jti),
	)

	return nil
}

// blacklistAccessToken adds an access token to the Redis blacklist
func (s *AuthService) blacklistAccessToken(ctx context.Context, tokenString string) error {
	// Parse the access token to get JTI and expiry
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(s.config.JWT.Secret), nil
	})
	if err != nil {
		return err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return fmt.Errorf("missing jti in token")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("missing exp in token")
	}

	// Calculate remaining TTL
	expiresAt := time.Unix(int64(exp), 0)
	remainingTTL := time.Until(expiresAt)

	// Add to blacklist
	return s.redisClient.BlacklistAccessToken(ctx, jti, remainingTTL)
}

// LogoutAll invalidates all sessions for a user and blacklists all tokens
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID, accessToken string) error {
	// Increment token version (invalidates all tokens)
	newVersion, err := s.authRepo.IncrementTokenVersion(ctx, userID)
	if err != nil {
		// Check if user not found
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			logger.Warn("user not found for logout all",
				zap.Error(err),
			)
			return serviceErrors.ErrUserNotFound
		}
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Update Redis - set minimum token version (any token with lower version is invalid)
	userIDStr := userID.String()
	if err := s.redisClient.SetTokenVersion(ctx, userIDStr, int(newVersion)); err != nil {
		logger.Error("failed to update token version in Redis",
			zap.Error(err),
		)
	}

	// Also store min token version for quick blacklist check
	if err := s.redisClient.BlacklistAllUserTokens(ctx, userIDStr, int(newVersion)); err != nil {
		logger.Error("failed to blacklist all user tokens in Redis",
			zap.Error(err),
		)
	}

	// Delete all refresh tokens from Redis for this user
	if err := s.redisClient.DeleteAllUserRefreshTokens(ctx, userIDStr); err != nil {
		logger.Error("failed to delete all refresh tokens from Redis",
			zap.Error(err),
		)
	}

	// Blacklist the current access token
	if accessToken != "" {
		if err := s.blacklistAccessToken(ctx, accessToken); err != nil {
			logger.Error("failed to blacklist current access token",
				zap.Error(err),
			)
		}
	}

	// Delete all sessions from DB
	err = s.sessionRepo.DeleteAllUserSessions(ctx, userID)
	if err != nil {
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	logger.Info("user logged out from all devices",
		zap.String("user_id", userIDStr),
		zap.Int32("new_token_version", newVersion),
	)

	return nil
}

// RefreshAccessToken generates a new access token using refresh token
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (dto.RefreshResponse, string, error) {
	// Parse and validate refresh token
	claims, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return dto.RefreshResponse{}, "", serviceErrors.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims["user_id"].(string))
	if err != nil {
		return dto.RefreshResponse{}, "", serviceErrors.ErrInvalidToken
	}

	jti := claims["jti"].(string)
	tokenVersion := int32(claims["token_version"].(float64))

	// Check if session exists
	session, err := s.sessionRepo.GetSessionByJTI(ctx, jti)
	if err != nil {
		// Check if session not found
		if sharedErrors.Is(err, serviceErrors.ErrSessionNotFoundRepo) {
			logger.Warn("refresh attempt with invalid session",
				zap.String("jti", jti),
			)
			return dto.RefreshResponse{}, "", serviceErrors.ErrSessionNotFound
		}
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Get user
	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		// Check if user not found
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			return dto.RefreshResponse{}, "", serviceErrors.ErrUserNotFound
		}
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Check token version
	if tokenVersion != utils.DerefInt32(user.TokenVersion, 0) {
		logger.Warn("refresh attempt with revoked token",
			zap.String("user_id", userID.String()),
			zap.Int32("token_version", tokenVersion),
			zap.Int32("current_version", utils.DerefInt32(user.TokenVersion, 0)),
		)
		return dto.RefreshResponse{}, "", serviceErrors.ErrTokenVersionMismatch
	}

	// Generate new tokens (Token Rotation)
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	newRefreshToken, newJTI, err := s.generateRefreshToken(user)
	if err != nil {
		return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Delete old session
	_ = s.sessionRepo.DeleteSession(ctx, jti)

	// Create new session
	expiresAt := clock.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	_, err = s.sessionRepo.CreateSession(ctx, db.CreateSessionParams{
		UserID:          user.ID,
		RefreshTokenJti: newJTI,
		DeviceInfo:      session.DeviceInfo,
		IpAddress:       session.IpAddress,
		ExpiresAt:       pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		// Check for query failures - wrap and return, handler will log
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		// Unexpected error - wrap and return, handler will log
		return dto.RefreshResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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

	if err := utils.ValidatePasswordPolicy(req.NewPassword); err != nil {
		return dto.ChangePasswordResponse{}, "", err
	}

	// Hash new password
	newPasswordHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		// Wrap and return, handler will log
		return dto.ChangePasswordResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Update password (this also increments token_version)
	err = s.authRepo.UpdatePassword(ctx, userID, newPasswordHash, false)
	if err != nil {
		// Wrap and return, handler will log
		return dto.ChangePasswordResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Get updated user with new token version
	user, err = s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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
		return dto.ChangePasswordResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	refreshToken, jti, err := s.generateRefreshToken(user)
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	// Create new session
	expiresAt := clock.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	_, err = s.sessionRepo.CreateSession(ctx, db.CreateSessionParams{
		UserID:          user.ID,
		RefreshTokenJti: jti,
		ExpiresAt:       pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return dto.ChangePasswordResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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
		return dto.SessionsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
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
	now := clock.Now()
	expiresAt := now.Add(time.Duration(s.config.JWT.AccessTokenExpiry) * time.Minute)
	jti := uuid.New().String() // Unique token ID for blacklist tracking

	claims := jwt.MapClaims{
		"user_id":               utils.PgtypeToUUID(user.ID).String(),
		"role":                  user.Role,
		"department":            utils.StringPointerToString(user.Department),
		"token_version":         utils.DerefInt32(user.TokenVersion, 0),
		"jti":                   jti,
		"force_password_change": utils.DerefBool(user.ForcePasswordChange, false),
		"exp":                   expiresAt.Unix(),
		"iat":                   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// generateRefreshToken creates a JWT refresh token
func (s *AuthService) generateRefreshToken(user db.User) (string, string, error) {
	now := clock.Now()
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
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
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

// parseRefreshTokenWithoutValidation parses token with signature validation but without expiry check
func (s *AuthService) parseRefreshTokenWithoutValidation(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.config.JWT.Secret), nil
	}, jwt.WithoutClaimsValidation())

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
