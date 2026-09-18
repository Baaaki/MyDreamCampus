package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/baaaki/mydreamcampus/auth/internal/dto"
	authErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/config"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuthService returns canned results; only the fields a test sets matter.
type fakeAuthService struct {
	loginErr      error
	refreshErr    error
	changeErr     error
	deleteErr     error
	logoutRefresh string
	logoutAccess  string
}

func (f *fakeAuthService) Login(context.Context, dto.LoginRequest, string, string) (dto.LoginResponse, string, error) {
	if f.loginErr != nil {
		return dto.LoginResponse{}, "", f.loginErr
	}
	return dto.LoginResponse{AccessToken: "access-1", User: dto.UserResponse{ID: "u1", Role: "student"}}, "refresh-1", nil
}

func (f *fakeAuthService) Logout(_ context.Context, refreshToken, accessToken, _ string) error {
	f.logoutRefresh, f.logoutAccess = refreshToken, accessToken
	return nil
}

func (f *fakeAuthService) LogoutAll(context.Context, uuid.UUID, string) error { return nil }

func (f *fakeAuthService) RefreshAccessToken(context.Context, string) (dto.RefreshResponse, string, error) {
	if f.refreshErr != nil {
		return dto.RefreshResponse{}, "", f.refreshErr
	}
	return dto.RefreshResponse{AccessToken: "access-2"}, "refresh-2", nil
}

func (f *fakeAuthService) ChangePassword(context.Context, uuid.UUID, dto.ChangePasswordRequest) (dto.ChangePasswordResponse, string, error) {
	if f.changeErr != nil {
		return dto.ChangePasswordResponse{}, "", f.changeErr
	}
	return dto.ChangePasswordResponse{AccessToken: "access-3"}, "refresh-3", nil
}

func (f *fakeAuthService) RequestPasswordReset(context.Context, string) error { return nil }

func (f *fakeAuthService) ResetPassword(context.Context, string, string) error { return nil }

func (f *fakeAuthService) GetUserSessions(context.Context, uuid.UUID, string) (dto.SessionsResponse, error) {
	return dto.SessionsResponse{}, nil
}

func (f *fakeAuthService) DeleteSession(context.Context, uuid.UUID, uuid.UUID, string) error {
	return f.deleteErr
}

const flowUserID = "3f2a1b0c-9d8e-4f7a-8b6c-5d4e3f2a1b0c"

func newFlowRouter(t *testing.T, svc *fakeAuthService) *gin.Engine {
	t.Helper()
	require.NoError(t, logger.Init("test"))
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	cfg.JWT.AccessTokenExpiry = 15
	cfg.JWT.RefreshTokenExpiry = 24
	h := NewAuthHandler(svc, cfg)

	r := gin.New()
	authed := func(c *gin.Context) { c.Set("user_id", flowUserID); c.Set("jti", "jti-1"); c.Next() }
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/refresh", h.RefreshToken)
	r.POST("/api/auth/logout", authed, h.Logout)
	r.POST("/api/auth/change-password", authed, h.ChangePassword)
	r.DELETE("/api/auth/sessions/:id", authed, h.DeleteSession)
	return r
}

func do(r *gin.Engine, method, path, body string, headers map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func setCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, ck := range w.Result().Cookies() {
		if ck.Name == name && ck.MaxAge >= 0 {
			return ck
		}
	}
	return nil
}

func bodyField(t *testing.T, w *httptest.ResponseRecorder, field string) (any, bool) {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	v, ok := m[field]
	return v, ok
}

const loginBody = `{"email":"ogrenci@uni.edu.tr","password":"Sifre1234"}`

func TestLogin_Browser_RefreshTokenOnlyInScopedCookie(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{}), "POST", "/api/auth/login", loginBody, nil)

	require.Equal(t, http.StatusOK, w.Code)
	_, inBody := bodyField(t, w, "refresh_token")
	assert.False(t, inBody, "a browser must never get the refresh token in a readable body")
	ck := setCookie(w, refreshCookie)
	require.NotNil(t, ck)
	assert.Equal(t, "/api/auth", ck.Path)
	assert.True(t, ck.HttpOnly)
}

func TestLogin_Mobile_RefreshTokenInBodyWithoutCookies(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{}), "POST", "/api/auth/login", loginBody,
		map[string]string{ClientTypeHeader: "mobile"})

	require.Equal(t, http.StatusOK, w.Code)
	v, _ := bodyField(t, w, "refresh_token")
	assert.Equal(t, "refresh-1", v)
	assert.Nil(t, setCookie(w, refreshCookie))
}

func TestLogin_LockedOut_AnswersLikeWrongPassword(t *testing.T) {
	locked := do(newFlowRouter(t, &fakeAuthService{loginErr: authErrors.ErrAccountLocked}), "POST", "/api/auth/login", loginBody, nil)
	wrong := do(newFlowRouter(t, &fakeAuthService{loginErr: authErrors.ErrInvalidCredentials}), "POST", "/api/auth/login", loginBody, nil)

	assert.Equal(t, http.StatusUnauthorized, locked.Code)
	assert.Equal(t, wrong.Code, locked.Code)
	assert.Equal(t, wrong.Body.String(), locked.Body.String())
}

func TestRefresh_InvalidToken_Returns401AndClearsCookies(t *testing.T) {
	for _, err := range []error{
		authErrors.ErrInvalidToken, authErrors.ErrSessionNotFound,
		authErrors.ErrUserNotFound, authErrors.ErrTokenVersionMismatch,
	} {
		w := do(newFlowRouter(t, &fakeAuthService{refreshErr: err}), "POST", "/api/auth/refresh", "", nil,
			&http.Cookie{Name: refreshCookie, Value: "stale"})

		assert.Equal(t, http.StatusUnauthorized, w.Code, "error %v", err)
		assert.Contains(t, strings.Join(w.Header().Values("Set-Cookie"), ";"), "refresh_token=;")
	}
}

func TestRefresh_ServerFailure_StaysA500(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{refreshErr: sharedErrors.ErrInternal}), "POST", "/api/auth/refresh", "", nil,
		&http.Cookie{Name: refreshCookie, Value: "ok"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRefresh_CookieToken_NewTokenNotInBody(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{}), "POST", "/api/auth/refresh", "", nil,
		&http.Cookie{Name: refreshCookie, Value: "cookie-token"})

	require.Equal(t, http.StatusOK, w.Code)
	_, inBody := bodyField(t, w, "refresh_token")
	assert.False(t, inBody, "a token that came as an HttpOnly cookie must not come back readable")
	assert.NotNil(t, setCookie(w, refreshCookie))
}

func TestRefresh_BodyToken_NewTokenInBody(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{}), "POST", "/api/auth/refresh", `{"refresh_token":"body-token"}`, nil)

	require.Equal(t, http.StatusOK, w.Code)
	v, _ := bodyField(t, w, "refresh_token")
	assert.Equal(t, "refresh-2", v)
}

func TestLogout_BodyRefreshToken_IsUsed(t *testing.T) {
	svc := &fakeAuthService{}
	w := do(newFlowRouter(t, svc), "POST", "/api/auth/logout", `{"refresh_token":"mobile-refresh"}`,
		map[string]string{"Authorization": "Bearer mobile-access"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "mobile-refresh", svc.logoutRefresh)
	assert.Equal(t, "mobile-access", svc.logoutAccess)
}

func TestLogout_NoRefreshToken_StillEndsAccessToken(t *testing.T) {
	svc := &fakeAuthService{}
	w := do(newFlowRouter(t, svc), "POST", "/api/auth/logout", "", map[string]string{"Authorization": "Bearer lone-access"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "lone-access", svc.logoutAccess)
}

func TestChangePassword_WrongOldPassword_Returns400(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{changeErr: authErrors.ErrInvalidOldPassword}), "POST", "/api/auth/change-password",
		`{"old_password":"Eskisifre1","new_password":"Yenisifre1"}`, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	code, _ := bodyField(t, w, "error")
	assert.Equal(t, "INVALID_OLD_PASSWORD", code)
}

func TestChangePassword_UnexpectedError_DoesNotLeakDetails(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{changeErr: sharedErrors.Wrap(sharedErrors.ErrInternal, assert.AnError)}),
		"POST", "/api/auth/change-password", `{"old_password":"Eskisifre1","new_password":"Yenisifre1"}`, nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), assert.AnError.Error())
}

func TestDeleteSession_CurrentSession_Returns400WithCode(t *testing.T) {
	w := do(newFlowRouter(t, &fakeAuthService{deleteErr: authErrors.ErrCannotTerminateSession}), "DELETE",
		"/api/auth/sessions/"+uuid.NewString(), "", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	code, _ := bodyField(t, w, "error")
	assert.Equal(t, "CANNOT_TERMINATE_CURRENT_SESSION", code)
}
