package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/baaaki/mydreamcampus/auth-service/config"
	"github.com/baaaki/mydreamcampus/auth-service/internal/dto"
	"github.com/baaaki/mydreamcampus/auth-service/internal/handler"
	"github.com/baaaki/mydreamcampus/auth-service/internal/repository"
	"github.com/baaaki/mydreamcampus/auth-service/internal/service"
	"github.com/baaaki/mydreamcampus/shared/database"
	"github.com/baaaki/mydreamcampus/shared/logger"
	sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
	"github.com/baaaki/mydreamcampus/shared/redis"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite manages integration tests
type IntegrationTestSuite struct {
	suite.Suite
	router          *gin.Engine
	pool            *pgxpool.Pool
	redisClient     *redis.ClientWrapper
	authRepo        *repository.AuthRepository
	sessionRepo     *repository.SessionRepository
	eventRepo       *repository.EventRepository
	authService     *service.AuthService
	authHandler     *handler.AuthHandler
	cfg             *config.Config
	testUserID      uuid.UUID
	testUserEmail   string
	testUserPwd     string
	testAccessToken string
}

// SetupSuite runs once before all tests
func (suite *IntegrationTestSuite) SetupSuite() {
	// Set test environment variables
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("JWT_SECRET", "test-secret-key-minimum-32-characters-long-for-security")
	os.Setenv("PORT", "8082")

	// Use test database
	testDBURL := os.Getenv("TEST_DB_URL")
	if testDBURL == "" {
		testDBURL = "postgresql://postgres:postgres@localhost:5432/mydreamcampus_auth_test?sslmode=disable"
	}
	os.Setenv("DB_URL", testDBURL)

	os.Setenv("REDIS_ADDR", "localhost:6379")
	os.Setenv("REDIS_DB", "1") // Use DB 1 for tests
	os.Setenv("RABBITMQ_URL", "amqp://rabbitmq:rabbitmq@localhost:5672/")

	// Load config
	cfg, err := config.Load()
	suite.Require().NoError(err, "Failed to load config")
	suite.cfg = cfg

	// Set JWT_SECRET as environment variable
	os.Setenv("JWT_SECRET", cfg.JWT.Secret)

	// Initialize logger
	err = logger.Init("test")
	suite.Require().NoError(err, "Failed to initialize logger")

	// Initialize database
	pool, err := database.NewPostgresPool(cfg.Database.URL)
	suite.Require().NoError(err, "Failed to connect to database")
	suite.pool = pool

	// Initialize Redis
	redisClient, err := redis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	suite.Require().NoError(err, "Failed to connect to Redis")
	suite.redisClient = redisClient

	// Initialize repositories
	suite.authRepo = repository.NewAuthRepository(pool)
	suite.sessionRepo = repository.NewSessionRepository(pool)
	suite.eventRepo = repository.NewEventRepository(pool)

	// Initialize service
	suite.authService = service.NewAuthService(
		suite.authRepo,
		suite.sessionRepo,
		suite.eventRepo,
		redisClient,
		cfg,
	)

	// Initialize handler
	suite.authHandler = handler.NewAuthHandler(suite.authService, cfg)

	// Setup router
	gin.SetMode(gin.TestMode)
	suite.router = suite.setupRouter()

	// Create test user
	suite.createTestUser()
}

// setupRouter creates test router
func (suite *IntegrationTestSuite) setupRouter() *gin.Engine {
	router := gin.New()
	router.Use(sharedMiddleware.Recovery())
	router.Use(sharedMiddleware.CORS())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	api := router.Group("/api/v1/auth")
	{
		api.POST("/login", suite.authHandler.Login)
		api.POST("/refresh", suite.authHandler.RefreshToken)

		protected := api.Group("")
		protected.Use(sharedMiddleware.JWTAuth())
		{
			protected.POST("/logout", suite.authHandler.Logout)
			protected.POST("/logout-all", suite.authHandler.LogoutAll)
			protected.POST("/change-password", suite.authHandler.ChangePassword)
			protected.GET("/sessions", suite.authHandler.GetSessions)
			protected.DELETE("/sessions/:id", suite.authHandler.DeleteSession)
		}
	}

	return router
}

// createTestUser creates a test user in database
func (suite *IntegrationTestSuite) createTestUser() {
	suite.testUserID = uuid.New()
	suite.testUserEmail = fmt.Sprintf("test-%s@university.edu.tr", suite.testUserID.String()[:8])
	suite.testUserPwd = "TestPassword123!"

	hashedPwd, err := utils.HashPassword(suite.testUserPwd)
	suite.Require().NoError(err)

	ctx := context.Background()

	isActive := true
	tokenVersion := int32(1)
	forcePasswordChange := false

	_, err = suite.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, is_active, token_version, force_password_change)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, suite.testUserID, suite.testUserEmail, hashedPwd, "student", &isActive, &tokenVersion, &forcePasswordChange)

	suite.Require().NoError(err, "Failed to create test user")
}

// TearDownSuite runs once after all tests
func (suite *IntegrationTestSuite) TearDownSuite() {
	// Clean up test user
	if suite.pool != nil {
		ctx := context.Background()
		_, _ = suite.pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'test-%'")
		_, _ = suite.pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", pgtype.UUID{Bytes: suite.testUserID, Valid: true})
		suite.pool.Close()
	}

	if suite.redisClient != nil {
		suite.redisClient.Close()
	}

	logger.Sync()
}

// TestHealthEndpoint tests health check
func (suite *IntegrationTestSuite) TestHealthEndpoint() {
	req := httptest.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response map[string]string
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
}

// TestLogin tests user login
func (suite *IntegrationTestSuite) TestLogin() {
	loginReq := dto.LoginRequest{
		Email:    suite.testUserEmail,
		Password: suite.testUserPwd,
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response dto.LoginResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.AccessToken)
	assert.Equal(suite.T(), 900, response.ExpiresIn)
	assert.Equal(suite.T(), suite.testUserEmail, response.User.Email)

	// Save access token for other tests
	suite.testAccessToken = response.AccessToken

	// Check refresh token cookie
	cookies := resp.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "refresh_token" {
			refreshCookie = cookie
			break
		}
	}
	assert.NotNil(suite.T(), refreshCookie)
	assert.True(suite.T(), refreshCookie.HttpOnly)
}

// TestLoginInvalidCredentials tests login with wrong password
func (suite *IntegrationTestSuite) TestLoginInvalidCredentials() {
	loginReq := dto.LoginRequest{
		Email:    suite.testUserEmail,
		Password: "WrongPassword123!",
	}

	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
}

// TestGetSessions tests retrieving user sessions
func (suite *IntegrationTestSuite) TestGetSessions() {
	// First login to create session
	suite.TestLogin()

	req := httptest.NewRequest("GET", "/api/v1/auth/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+suite.testAccessToken)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusOK, resp.Code)

	var response dto.SessionsResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), response.Sessions)
}

// TestProtectedEndpointWithoutAuth tests accessing protected endpoint without token
func (suite *IntegrationTestSuite) TestProtectedEndpointWithoutAuth() {
	req := httptest.NewRequest("GET", "/api/v1/auth/sessions", nil)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
}

// TestProtectedEndpointWithInvalidToken tests with invalid token
func (suite *IntegrationTestSuite) TestProtectedEndpointWithInvalidToken() {
	req := httptest.NewRequest("GET", "/api/v1/auth/sessions", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
}

// TestChangePasswordValidation tests change password input validation
func (suite *IntegrationTestSuite) TestChangePasswordValidation() {
	changeReq := dto.ChangePasswordRequest{
		OldPassword: "short",
		NewPassword: "NewPassword123!",
	}

	body, _ := json.Marshal(changeReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.testAccessToken)
	resp := httptest.NewRecorder()

	suite.router.ServeHTTP(resp, req)

	assert.Equal(suite.T(), http.StatusBadRequest, resp.Code)
}

// TestRateLimiting tests that multiple failed login attempts are handled
func (suite *IntegrationTestSuite) TestMultipleFailedLogins() {
	loginReq := dto.LoginRequest{
		Email:    suite.testUserEmail,
		Password: "WrongPassword",
	}

	for range 3 {
		body, _ := json.Marshal(loginReq)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		suite.router.ServeHTTP(resp, req)

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.Code)
	}
}

// TestConcurrentLogins tests multiple concurrent logins
func (suite *IntegrationTestSuite) TestConcurrentLogins() {
	done := make(chan bool, 5)

	for range 5 {
		go func() {
			loginReq := dto.LoginRequest{
				Email:    suite.testUserEmail,
				Password: suite.testUserPwd,
			}

			body, _ := json.Marshal(loginReq)
			req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			suite.router.ServeHTTP(resp, req)

			assert.Equal(suite.T(), http.StatusOK, resp.Code)
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 5 {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			suite.T().Fatal("Timeout waiting for concurrent logins")
		}
	}
}

// TestIntegrationTestSuite runs the test suite
func TestIntegrationTestSuite(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=true to run.")
	}

	suite.Run(t, new(IntegrationTestSuite))
}
