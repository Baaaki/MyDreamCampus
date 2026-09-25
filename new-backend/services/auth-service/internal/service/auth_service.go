package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	"github.com/baaaki/mydreamcampus/auth/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/events"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/redis"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// authCache is the slice of Redis the auth service uses — an interface so
// the lockout and revocation paths can be tested without a Redis server.
type authCache interface {
	BlacklistAccessToken(ctx context.Context, jti string, remainingTTL time.Duration) error
	BlacklistAllUserTokens(ctx context.Context, userID string, tokenVersion int) error
	LoginFailureCount(ctx context.Context, key string) (int64, error)
	RecordLoginFailure(ctx context.Context, key string, window time.Duration) (int64, error)
	ClearLoginFailures(ctx context.Context, key string) error
	StoreResetToken(ctx context.Context, token, email string, expiry time.Duration) error
	ConsumeResetToken(ctx context.Context, token string) (string, error)
}

var _ authCache = (*redis.ClientWrapper)(nil)

// userStore and sessionStore are the slices of the repositories the auth
// service uses — interfaces so the session paths can be tested without a
// database.
type userStore interface {
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.CreateUserRow, error)
	CreateOutboxEvent(ctx context.Context, params db.CreateOutboxEventParams) (db.CreateOutboxEventRow, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, forcePasswordChange bool) error
	IncrementTokenVersion(ctx context.Context, userID uuid.UUID) (int32, error)
	AdminExists(ctx context.Context) (bool, error)
	SetSuperAdmin(ctx context.Context, email string) error
	GetActiveDemoUsers(ctx context.Context) ([]db.GetActiveDemoUsersRow, error)
	EnsureDemoUserFlags(ctx context.Context, email string) error
}

type sessionStore interface {
	CreateSession(ctx context.Context, params db.CreateSessionParams) (db.Session, error)
	GetSessionByJTI(ctx context.Context, jti string) (db.Session, error)
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]db.Session, error)
	RotateSession(ctx context.Context, oldJTI string, params db.CreateSessionParams) (db.Session, error)
	DeleteSession(ctx context.Context, jti string) error
	DeleteSessionByID(ctx context.Context, sessionID, userID uuid.UUID) error
	DeleteAllUserSessions(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredSessions(ctx context.Context) error
}

var (
	_ userStore    = (*repository.AuthRepository)(nil)
	_ sessionStore = (*repository.SessionRepository)(nil)
)

type AuthService struct {
	authRepo    userStore
	sessionRepo sessionStore
	eventRepo   *repository.EventRepository
	redisClient authCache
	config      *config.Config
}

// signingMethods pins parsing to the one algorithm tokens are minted with,
// so a token signed some other way is refused rather than trusted.
var signingMethods = []string{jwt.SigningMethodHS256.Alg()}

// maxLoginFailures is how many wrong passwords one address may try against
// one account before it is locked out for ACCOUNT_LOCK_DURATION_MINUTES.
const maxLoginFailures = 5

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

	// The lockout is keyed on the account AND the client address. Keyed on
	// the account alone, anyone who knows an e-mail — the admin's included —
	// could lock its owner out with five bad guesses.
	lockKey := loginFailureKey(req.Email, ipAddress)
	if s.loginLockedOut(ctx, lockKey, serviceLogger) {
		// Same work, same answer as a wrong password: a distinct lockout
		// response would reveal that the address exists and was targeted.
		utils.VerifyDummyPassword(req.Password)
		serviceLogger.Warn("login attempt while locked out", zap.String("ip", ipAddress))
		return dto.LoginResponse{}, "", serviceErrors.ErrAccountLocked
	}

	// Get user by email
	user, err := s.authRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// Check if user not found
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			// Timing-safe: run a real Argon2id verification against a
			// precomputed dummy hash so response time matches the
			// password-mismatch branch and email enumeration is closed.
			utils.VerifyDummyPassword(req.Password)
			// Counted like a wrong password, so unknown addresses lock out
			// exactly like real ones.
			s.recordLoginFailure(ctx, lockKey, serviceLogger)
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
		s.recordLoginFailure(ctx, lockKey, serviceLogger)
		serviceLogger.Warn("login attempt for deactivated account")
		return dto.LoginResponse{}, "", serviceErrors.ErrInvalidCredentials
	}

	// Verify password
	if !utils.VerifyPassword(user.PasswordHash, req.Password) {
		s.recordLoginFailure(ctx, lockKey, serviceLogger)
		serviceLogger.Warn("invalid password")
		return dto.LoginResponse{}, "", serviceErrors.ErrInvalidCredentials
	}

	if err := s.redisClient.ClearLoginFailures(ctx, lockKey); err != nil {
		serviceLogger.Warn("failed to clear login failures", zap.Error(err))
	}

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
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
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
			ID:           utils.PgtypeToUUID(user.ID).String(),
			Email:        user.Email,
			Role:         user.Role,
			Department:   user.Department,
			IsSuperadmin: user.IsSuperadmin,
			IsDemo:       user.IsDemo,
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

// Logout blacklists the caller's access token and ends the session the
// refresh token belongs to. refreshToken may be empty — a client that lost
// it still expects logout to end the access token it is holding.
func (s *AuthService) Logout(ctx context.Context, refreshToken string, accessToken string, authenticatedUserID string) error {
	if accessToken != "" {
		if err := s.blacklistAccessToken(ctx, accessToken); err != nil {
			logger.Error("failed to blacklist access token", zap.Error(err))
		}
	}

	if refreshToken == "" {
		logger.Warn("logout without refresh token; session left to expire",
			zap.String("user_id", authenticatedUserID),
		)
		return nil
	}

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

	jti, _ := claims["jti"].(string)
	if jti == "" {
		return serviceErrors.ErrInvalidToken
	}
	if err := s.sessionRepo.DeleteSession(ctx, jti); err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrSessionNotFoundRepo) {
			// Logout must succeed even if the session is already gone.
			logger.Info("logout attempted for already deleted session", zap.String("jti", jti))
		} else {
			logger.Error("database error deleting session", zap.Error(err), zap.String("jti", jti))
		}
	}

	logger.Info("user logged out", zap.String("refresh_jti", jti))
	return nil
}

// blacklistAccessToken adds an access token to the Redis blacklist
func (s *AuthService) blacklistAccessToken(ctx context.Context, tokenString string) error {
	// Parse the access token to get JTI and expiry
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(s.config.JWT.Secret), nil
	}, jwt.WithValidMethods(signingMethods))
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

// loginFailureKey scopes the failure counter to one account from one
// address. The e-mail is hashed so Redis never holds it in clear text.
func loginFailureKey(email, ip string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "login_fail:" + hex.EncodeToString(sum[:]) + ":" + ip
}

// loginLockedOut fails open: when Redis cannot answer, the fail-closed login
// rate limit still bounds how fast passwords can be tried.
func (s *AuthService) loginLockedOut(ctx context.Context, key string, log *zap.Logger) bool {
	failures, err := s.redisClient.LoginFailureCount(ctx, key)
	if err != nil {
		log.Error("login failure count unavailable", zap.Error(err))
		return false
	}
	return failures >= maxLoginFailures
}

func (s *AuthService) recordLoginFailure(ctx context.Context, key string, log *zap.Logger) {
	failures, err := s.redisClient.RecordLoginFailure(ctx, key, s.lockDuration())
	if err != nil {
		log.Error("failed to record login failure", zap.Error(err))
		return
	}
	if failures == maxLoginFailures {
		log.Warn("login locked out after repeated failures",
			zap.Int64("failures", failures),
			zap.Duration("lock_duration", s.lockDuration()),
		)
	}
}

func (s *AuthService) lockDuration() time.Duration {
	minutes := s.config.Timeout.AccountLockDurationMinutes
	if minutes <= 0 {
		minutes = 30
	}
	return time.Duration(minutes) * time.Minute
}

// revokeOutstandingAccessTokens raises the minimum token version JWTAuth
// accepts for the user. Access tokens are never looked up, so this — not the
// token_version column — is what makes an already-issued one fail. Failure
// is logged, not returned: the DB side (version bump, sessions) has already
// committed and still revokes every refresh token.
func (s *AuthService) revokeOutstandingAccessTokens(ctx context.Context, userID string, newVersion int) {
	if err := s.redisClient.BlacklistAllUserTokens(ctx, userID, newVersion); err != nil {
		logger.Error("failed to revoke outstanding access tokens",
			zap.Error(err),
			zap.String("user_id", userID),
		)
	}
}

// LogoutAll invalidates all sessions for a user and blacklists all tokens
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID, accessToken string) error {
	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			logger.Warn("user not found for logout all",
				zap.Error(err),
			)
			return serviceErrors.ErrUserNotFound
		}
		if sharedErrors.Is(err, sharedErrors.ErrQueryFailed) {
			return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	if user.IsDemo {
		return serviceErrors.ErrDemoAccountActionForbidden
	}

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

	userIDStr := userID.String()
	s.revokeOutstandingAccessTokens(ctx, userIDStr, int(newVersion))

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

	// Comma-ok throughout: a bare assertion on a missing claim panics, and
	// Recovery turns that into a 500 for what is just a bad token.
	userIDClaim, _ := claims["user_id"].(string)
	userID, err := uuid.Parse(userIDClaim)
	if err != nil {
		return dto.RefreshResponse{}, "", serviceErrors.ErrInvalidToken
	}

	jti, _ := claims["jti"].(string)
	if jti == "" {
		return dto.RefreshResponse{}, "", serviceErrors.ErrInvalidToken
	}
	versionClaim, _ := claims["token_version"].(float64)
	tokenVersion := int32(versionClaim)

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

	// Replace the old session atomically. A lost race does not revoke the
	// user's other sessions: two tabs refreshing at once is normal, and
	// that would log the user out everywhere.
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
	_, err = s.sessionRepo.RotateSession(ctx, jti, db.CreateSessionParams{
		UserID:          user.ID,
		RefreshTokenJti: newJTI,
		DeviceInfo:      session.DeviceInfo,
		IpAddress:       session.IpAddress,
		ExpiresAt:       pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrSessionNotFoundRepo) {
			logger.Warn("refresh token already rotated",
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
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			return dto.ChangePasswordResponse{}, "", serviceErrors.ErrUserNotFound
		}
		return dto.ChangePasswordResponse{}, "", sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	if user.IsDemo {
		return dto.ChangePasswordResponse{}, "", serviceErrors.ErrDemoAccountActionForbidden
	}

	// Verify old password
	if !utils.VerifyPassword(user.PasswordHash, req.OldPassword) {
		logger.Warn("invalid old password during password change",
			zap.String("user_id", userID.String()),
		)
		return dto.ChangePasswordResponse{}, "", serviceErrors.ErrInvalidOldPassword
	}

	if err := utils.ValidatePasswordPolicy(req.NewPassword); err != nil {
		return dto.ChangePasswordResponse{}, "", serviceErrors.ErrWeakPassword
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

	// Deleting the sessions below kills every refresh token, but the access
	// tokens already handed out stay valid until they expire. Whoever took
	// the old password may be holding one, so they are cut off here too.
	s.revokeOutstandingAccessTokens(ctx, userID.String(), int(utils.DerefInt32(user.TokenVersion, 0)))

	// Every session goes, the caller's included; the caller gets a fresh one
	// below together with tokens carrying the new version.
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
	expiresAt := time.Now().Add(time.Duration(s.config.JWT.RefreshTokenExpiry) * time.Hour)
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
		Message:     "Şifreniz değiştirildi",
		AccessToken: accessToken,
		ExpiresIn:   s.config.JWT.AccessTokenExpiry * 60,
	}, refreshToken, nil
}

// RequestPasswordReset initiates a password reset flow
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	serviceLogger := logger.WithContextAndFields(ctx,
		zap.String("service", "AuthService"),
		zap.String("method", "RequestPasswordReset"),
	)

	// Silently ignore password reset for protected accounts (super admin and demo accounts)
	if s.config != nil && s.config.IsProtectedEmail(email) {
		serviceLogger.Info("password reset requested for protected account; silently ignoring", zap.String("email", email))
		return nil
	}

	// Get user
	user, err := s.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			// Do not leak user existence
			serviceLogger.Info("password reset requested for non-existent user")
			return nil
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	if !utils.DerefBool(user.IsActive, false) {
		serviceLogger.Info("password reset requested for deactivated account")
		return nil
	}

	// Generate reset token and store in Redis
	resetToken := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)

	if err := s.redisClient.StoreResetToken(ctx, resetToken, email, 1*time.Hour); err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, fmt.Errorf("failed to store reset token: %w", err))
	}

	// Insert into outbox_events
	payloadBytes, _ := json.Marshal(buildUserPasswordResetRequestedPayload(
		utils.PgtypeToUUID(user.ID).String(),
		email,
		resetToken,
		expiresAt,
	))

	_, err = s.authRepo.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		CorrelationID: utils.CorrelationIDFromContext(ctx),
		EventType:     events.EventTypeUserPasswordResetRequested,
		RoutingKey:    events.RoutingKeyUserPasswordResetRequested,
		Payload:       payloadBytes,
	})
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, fmt.Errorf("failed to create outbox event: %w", err))
	}

	serviceLogger.Info("password reset requested", zap.String("user_id", utils.PgtypeToUUID(user.ID).String()))
	return nil
}

// ResetPassword completes the flow RequestPasswordReset starts: it spends
// the e-mailed token, sets the new password and ends every session, since
// whoever needed a reset may not be the only one holding the old password.
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Policy first: a rejected password must not burn the single-use token.
	if err := utils.ValidatePasswordPolicy(newPassword); err != nil {
		return serviceErrors.ErrWeakPassword
	}

	email, err := s.redisClient.ConsumeResetToken(ctx, token)
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, fmt.Errorf("failed to consume reset token: %w", err))
	}
	if email == "" {
		return serviceErrors.ErrInvalidResetToken
	}

	user, err := s.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			// The account went away between request and reset.
			return serviceErrors.ErrInvalidResetToken
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	if !utils.DerefBool(user.IsActive, false) {
		return serviceErrors.ErrInvalidResetToken
	}

	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	userID := utils.PgtypeToUUID(user.ID)
	// UpdatePassword bumps token_version, which retires every refresh token.
	if err := s.authRepo.UpdatePassword(ctx, userID, hash, false); err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	updated, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	s.revokeOutstandingAccessTokens(ctx, userID.String(), int(utils.DerefInt32(updated.TokenVersion, 0)))
	if err := s.sessionRepo.DeleteAllUserSessions(ctx, userID); err != nil {
		logger.Error("failed to delete sessions after password reset", zap.Error(err))
	}

	logger.Info("password reset completed", zap.String("user_id", userID.String()))
	return nil
}

// GetUserSessions returns all active sessions for a user
func (s *AuthService) GetUserSessions(ctx context.Context, userID uuid.UUID, currentJTI string) (dto.SessionsResponse, error) {
	sessions, err := s.sessionRepo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return dto.SessionsResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	user, err := s.authRepo.GetUserByID(ctx, userID)
	isDemo := err == nil && user.IsDemo

	sessionResponses := make([]dto.SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		isCurrent := session.RefreshTokenJti == currentJTI
		ipAddress := session.IpAddress
		deviceInfo := session.DeviceInfo

		if isDemo && !isCurrent {
			if ipAddress != nil {
				masked := maskIP(*ipAddress)
				ipAddress = &masked
			}
			if deviceInfo != nil {
				shortened := shortenDeviceInfo(*deviceInfo)
				deviceInfo = &shortened
			}
		}

		sessionResponses = append(sessionResponses, dto.SessionResponse{
			ID:         utils.PgtypeToUUID(session.ID).String(),
			DeviceInfo: deviceInfo,
			IPAddress:  ipAddress,
			CreatedAt:  session.CreatedAt.Time,
			LastUsedAt: session.LastUsedAt.Time,
			ExpiresAt:  session.ExpiresAt.Time,
			IsCurrent:  isCurrent,
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
				return serviceErrors.ErrCannotTerminateSession
			}
		}
	}

	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		if sharedErrors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
			return serviceErrors.ErrUserNotFound
		}
		return sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	if user.IsDemo {
		return serviceErrors.ErrDemoAccountActionForbidden
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

func maskIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if strings.Contains(ip, ".") {
		parts := strings.Split(ip, ".")
		if len(parts) >= 2 {
			return parts[0] + "." + parts[1] + ".x.x"
		}
		return "x.x.x.x"
	}
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		if len(parts) >= 2 {
			return parts[0] + ":" + parts[1] + "::x:x"
		}
		return "x::x"
	}
	return "x.x.x.x"
}

func shortenDeviceInfo(device string) string {
	runes := []rune(device)
	if len(runes) > 24 {
		return string(runes[:24]) + "..."
	}
	return device
}

// SeedAdmin creates the initial admin user
func (s *AuthService) SeedAdmin(ctx context.Context) error {
	// Check if admin already exists
	exists, err := s.authRepo.AdminExists(ctx)
	if err != nil {
		return err
	}

	if exists {
		if err := s.authRepo.SetSuperAdmin(ctx, s.config.Admin.Email); err != nil {
			logger.Warn("failed to set superadmin on existing admin user", zap.Error(err))
		}
		logger.Info("admin user already exists, ensured superadmin")
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
		IsSuperadmin:        utils.BoolPtr(true),
		IsDemo:              utils.BoolPtr(false),
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	if err := s.authRepo.SetSuperAdmin(ctx, s.config.Admin.Email); err != nil {
		logger.Warn("failed to set superadmin on newly seeded admin user", zap.Error(err))
	}

	logger.Info("admin user seeded successfully",
		zap.String("email", s.config.Admin.Email),
	)

	return nil
}

// SeedDemoAdmin creates the initial demo admin user if demo mode is enabled
func (s *AuthService) SeedDemoAdmin(ctx context.Context) error {
	if !s.config.Demo.Enabled || s.config.Demo.AdminEmail == "" {
		return nil
	}

	// Check if user already exists
	_, err := s.authRepo.GetUserByEmail(ctx, s.config.Demo.AdminEmail)
	if err == nil {
		if err := s.authRepo.EnsureDemoUserFlags(ctx, s.config.Demo.AdminEmail); err != nil {
			logger.Warn("failed to ensure demo flags on existing demo admin", zap.Error(err))
		}
		logger.Info("demo admin user already exists", zap.String("email", s.config.Demo.AdminEmail))
		return nil
	}
	if !errors.Is(err, serviceErrors.ErrUserNotFoundRepo) {
		return fmt.Errorf("check demo admin user: %w", err)
	}

	// Hash password (password = email)
	passwordHash, err := utils.HashPassword(s.config.Demo.AdminEmail)
	if err != nil {
		return fmt.Errorf("failed to hash demo admin password: %w", err)
	}

	demoAdminID := uuid.New()
	_, err = s.authRepo.CreateUser(ctx, db.CreateUserParams{
		ID:                  utils.UUIDToPgtype(demoAdminID),
		Email:               s.config.Demo.AdminEmail,
		PasswordHash:        passwordHash,
		Role:                "admin",
		Department:          nil,
		IsActive:            utils.BoolPtr(true),
		TokenVersion:        utils.Int32Ptr(1),
		ForcePasswordChange: utils.BoolPtr(false),
		IsSuperadmin:        utils.BoolPtr(false),
		IsDemo:              utils.BoolPtr(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create demo admin user: %w", err)
	}

	logger.Info("demo admin user seeded successfully",
		zap.String("email", s.config.Demo.AdminEmail),
		zap.String("id", demoAdminID.String()),
	)
	return nil
}

// EnsureDemoAccountFlags marks the demo teacher and student at startup. The
// seed marks them too, but it runs only on an empty system: a stack seeded
// before the flag existed would never list them on the login page. Addresses
// with no account yet match no row; on a fresh stack the seed marks them.
func (s *AuthService) EnsureDemoAccountFlags(ctx context.Context) error {
	if !s.config.Demo.Enabled {
		return nil
	}
	for _, email := range []string{s.config.Demo.TeacherEmail, s.config.Demo.StudentEmail} {
		if email == "" {
			continue
		}
		if err := s.authRepo.EnsureDemoUserFlags(ctx, email); err != nil {
			return fmt.Errorf("mark demo account %s: %w", email, err)
		}
	}
	return nil
}

// GetDemoAccounts returns active demo accounts
func (s *AuthService) GetDemoAccounts(ctx context.Context) ([]dto.DemoAccountResponse, error) {
	users, err := s.authRepo.GetActiveDemoUsers(ctx)
	if err != nil {
		return nil, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	result := make([]dto.DemoAccountResponse, 0, len(users))
	for _, u := range users {
		var label string
		switch u.Role {
		case "admin":
			label = "Demo Yönetici"
		case "teacher":
			label = "Demo Öğretmen"
		case "student":
			label = "Demo Öğrenci"
		default:
			label = "Demo Kullanıcı"
		}

		result = append(result, dto.DemoAccountResponse{
			Role:     u.Role,
			Label:    label,
			Email:    u.Email,
			Password: u.Email,
		})
	}

	return result, nil
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
				// processed_events is pruned by the shared
				// eventbus.RetentionWorker, so every service gets the same
				// window from one place instead of auth alone.
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
	// Tokens, sessions and reset links run on the real clock: JWT
	// validation and sessions.sql compare against real time, and a
	// simulated issue time would log everyone out when the time machine
	// moves.
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.config.JWT.AccessTokenExpiry) * time.Minute)
	jti := uuid.New().String() // Unique token ID for blacklist tracking

	claims := jwt.MapClaims{
		"user_id":               utils.PgtypeToUUID(user.ID).String(),
		"role":                  user.Role,
		"department":            utils.StringPointerToString(user.Department),
		"token_version":         utils.DerefInt32(user.TokenVersion, 0),
		"jti":                   jti,
		"force_password_change": utils.DerefBool(user.ForcePasswordChange, false),
		"token_type":            string(utils.AccessToken),
		"exp":                   expiresAt.Unix(),
		"iat":                   now.Unix(),
	}
	if user.IsSuperadmin {
		claims["super_admin"] = true
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
		"token_type":    string(utils.RefreshToken),
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
	}, jwt.WithValidMethods(signingMethods))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if err := requireRefreshType(claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// requireRefreshType rejects an access token handed to a refresh-token path.
// Both share the signing key, so the signature alone cannot tell them apart;
// without this an access token could rotate itself into a fresh session.
func requireRefreshType(claims jwt.MapClaims) error {
	if tokenType, _ := claims["token_type"].(string); tokenType != string(utils.RefreshToken) {
		return utils.ErrWrongTokenType
	}
	return nil
}

// parseRefreshTokenWithoutValidation parses token with signature validation but without expiry check
func (s *AuthService) parseRefreshTokenWithoutValidation(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.config.JWT.Secret), nil
	}, jwt.WithoutClaimsValidation(), jwt.WithValidMethods(signingMethods))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}
	if err := requireRefreshType(claims); err != nil {
		return nil, err
	}
	return claims, nil
}
