package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nonsoemeka-backend/internal/auth"
	"nonsoemeka-backend/internal/config"
	"nonsoemeka-backend/internal/handlers"
	"nonsoemeka-backend/internal/repository"
	"nonsoemeka-backend/internal/services"
)

// connectTestDB opens a connection to the test database.
// If TEST_DSN is not set the test is skipped.
func connectTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set — skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err, "failed to open test DB connection")
	t.Cleanup(func() { pool.Close() })
	return pool
}

// testConfig returns a minimal config suitable for tests.
func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret: "test-access-secret-min-32-characters!",
			RefreshSecret: "test-refresh-secret-min-32-chars!!",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    7 * 24 * time.Hour,
		},
	}
}

// ---------------------------------------------------------------------------
// Pure unit tests (no DB required)
// ---------------------------------------------------------------------------

// TestAuth_Login_SuccessAndRefresh exercises the Login→Refresh→Logout flow
// against the live test database.
func TestAuth_Login_SuccessAndRefresh(t *testing.T) {
	pool := connectTestDB(t)
	cfg := testConfig()

	userRepo := repository.NewUserRepository()
	authSvc := services.NewAuthService(pool, userRepo, cfg)

	// The seeded admin credentials must exist in the test DB.
	user, accessToken, rawRefresh, err := authSvc.Login(context.Background(), "admin", "AdminPass123!")
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, rawRefresh)
	assert.Equal(t, "ADMIN", string(user.Role))
	assert.NotEmpty(t, user.Email)

	// Verify email is in the JWT claims.
	claims, err := auth.ParseAccessToken(accessToken, cfg.JWT.AccessSecret)
	require.NoError(t, err)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, "admin", claims.Username)

	// Refresh should rotate tokens.
	newAccess, newRefresh, err := authSvc.Refresh(context.Background(), rawRefresh)
	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	assert.NotEqual(t, rawRefresh, newRefresh, "refresh token should be rotated")

	// Old refresh token must be invalidated (reuse detection).
	_, _, err = authSvc.Refresh(context.Background(), rawRefresh)
	assert.Error(t, err, "revoked refresh token must not be accepted")

	// Logout.
	err = authSvc.Logout(context.Background(), newRefresh)
	assert.NoError(t, err)
}

// TestAuth_Login_WrongPassword ensures invalid credentials are rejected.
func TestAuth_Login_WrongPassword(t *testing.T) {
	pool := connectTestDB(t)
	cfg := testConfig()

	userRepo := repository.NewUserRepository()
	authSvc := services.NewAuthService(pool, userRepo, cfg)

	_, _, _, err := authSvc.Login(context.Background(), "admin", "wrong-password")
	assert.Error(t, err)
}

// TestAuth_Login_UnknownUser ensures unknown usernames return an error.
func TestAuth_Login_UnknownUser(t *testing.T) {
	pool := connectTestDB(t)
	cfg := testConfig()

	userRepo := repository.NewUserRepository()
	authSvc := services.NewAuthService(pool, userRepo, cfg)

	_, _, _, err := authSvc.Login(context.Background(), "no_such_user_xyz", "any-pass")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// HTTP handler integration tests
// ---------------------------------------------------------------------------

// TestAuthHandler_Login exercises the full Login HTTP handler.
func TestAuthHandler_Login(t *testing.T) {
	pool := connectTestDB(t)
	cfg := testConfig()

	userRepo := repository.NewUserRepository()
	authSvc := services.NewAuthService(pool, userRepo, cfg)
	h := handlers.NewAuthHandler(authSvc, cfg)

	router := chi.NewRouter()
	router.Post("/api/v1/auth/login", h.Login)

	// Successful login.
	body := `{"username":"admin","password":"AdminPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")
	assert.Contains(t, w.Header().Get("Set-Cookie"), "refresh_token")

	// Wrong password.
	body = `{"username":"admin","password":"wrong"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthHandler_Refresh exercises the Refresh HTTP handler.
func TestAuthHandler_Refresh(t *testing.T) {
	pool := connectTestDB(t)
	cfg := testConfig()

	userRepo := repository.NewUserRepository()
	authSvc := services.NewAuthService(pool, userRepo, cfg)
	h := handlers.NewAuthHandler(authSvc, cfg)

	router := chi.NewRouter()
	router.Post("/api/v1/auth/login", h.Login)
	router.Post("/api/v1/auth/refresh", h.Refresh)

	// First, login to get a refresh cookie.
	body := `{"username":"admin","password":"AdminPass123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Extract the Set-Cookie header for the refresh token.
	setCookie := w.Header().Get("Set-Cookie")
	require.NotEmpty(t, setCookie)
	// Parse cookie value from: refresh_token=<value>; ...
	var refreshCookieVal string
	for _, part := range strings.Split(setCookie, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "refresh_token=") {
			refreshCookieVal = strings.TrimPrefix(part, "refresh_token=")
			break
		}
	}
	require.NotEmpty(t, refreshCookieVal, "refresh_token cookie must be present after login")

	// Call Refresh with the cookie.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshCookieVal})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "access_token")

	// Calling Refresh without a cookie must fail.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// Stub test — demonstrates pattern for testing GenerateAccessToken email claim
// without a database (pure unit).
// ---------------------------------------------------------------------------

func TestAuth_EmailInJWTClaims(t *testing.T) {
	userID := uuid.New()
	secret := "unit-test-secret-key-min-32-chars!"

	tokenStr, err := auth.GenerateAccessToken(
		userID, "staff_user", "staff@nonsoemeka.com", "STAFF", secret, 5*time.Minute,
	)
	require.NoError(t, err)

	claims, err := auth.ParseAccessToken(tokenStr, secret)
	require.NoError(t, err)
	assert.Equal(t, "staff@nonsoemeka.com", claims.Email)
	assert.Equal(t, "staff_user", claims.Username)
	assert.Equal(t, "STAFF", claims.Role)
}
