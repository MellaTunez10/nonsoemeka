package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nonsoemeka-backend/internal/auth"
	"nonsoemeka-backend/internal/config"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/handlers"
	"nonsoemeka-backend/internal/middleware"
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

// testConfig returns a minimal config suitable for handler tests.
func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-min-32-characters!",
			RefreshSecret: "test-refresh-secret-min-32-chars!!",
		},
	}
}

// injectClaimsMiddleware is a test helper that bypasses JWT validation and
// injects pre-built claims directly into the request context.
func injectClaimsMiddleware(claims *auth.Claims) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.CtxKeyUser, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// buildCheckoutHandler wires up the CheckoutHandler with real repositories.
func buildCheckoutHandler(pool *pgxpool.Pool) *handlers.CheckoutHandler {
	return handlers.NewCheckoutHandler(services.NewCheckoutService(
		pool,
		repository.NewSaleRepository(),
		repository.NewBatchRepository(),
		repository.NewProductRepository(),
		repository.NewInventoryMovementRepository(),
		repository.NewSettingsRepository(),
		repository.NewUserRepository(),
	))
}

// loginAndGetUser performs a real login and returns the admin user and their ID.
func loginAndGetUser(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, string) {
	t.Helper()
	cfg := testConfig()
	authSvc := services.NewAuthService(pool, repository.NewUserRepository(), cfg)
	user, _, _, err := authSvc.Login(context.Background(), "admin", "AdminPass123!")
	require.NoError(t, err, "login must succeed for checkout integration test")
	return user.ID, user.Email
}

// ---------------------------------------------------------------------------
// Checkout handler integration tests
// ---------------------------------------------------------------------------

// TestCheckoutHandler_NoItems verifies that a checkout with an empty item list
// returns HTTP 400.
func TestCheckoutHandler_NoItems(t *testing.T) {
	pool := connectTestDB(t)
	adminID, adminEmail := loginAndGetUser(t, pool)
	h := buildCheckoutHandler(pool)

	fakeClaims := &auth.Claims{
		UserID:   adminID,
		Username: "admin",
		Email:    adminEmail,
		Role:     "ADMIN",
	}

	router := chi.NewRouter()
	router.Use(injectClaimsMiddleware(fakeClaims))
	router.Post("/api/v1/checkout", h.ProcessCheckout)

	body := `{"idempotency_key":"test-key-noitems","items":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCheckoutHandler_UnknownProduct verifies that a checkout referencing a
// non-existent product returns an appropriate error (400 or 404).
func TestCheckoutHandler_UnknownProduct(t *testing.T) {
	pool := connectTestDB(t)
	adminID, adminEmail := loginAndGetUser(t, pool)
	h := buildCheckoutHandler(pool)

	fakeClaims := &auth.Claims{
		UserID:   adminID,
		Username: "admin",
		Email:    adminEmail,
		Role:     "ADMIN",
	}

	router := chi.NewRouter()
	router.Use(injectClaimsMiddleware(fakeClaims))
	router.Post("/api/v1/checkout", h.ProcessCheckout)

	nonExistentID := uuid.New()
	body := fmt.Sprintf(
		`{"idempotency_key":"test-key-unknown","items":[{"product_id":"%s","quantity":1}]}`,
		nonExistentID,
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound,
		"expected 400 or 404, got %d: %s", w.Code, w.Body.String())
}

// TestCheckoutHandler_MissingAuth verifies that the handler rejects requests
// without a valid JWT when using the real AuthMiddleware.
func TestCheckoutHandler_MissingAuth(t *testing.T) {
	pool := connectTestDB(t)
	h := buildCheckoutHandler(pool)
	cfg := testConfig()

	router := chi.NewRouter()
	router.Use(middleware.AuthMiddleware(cfg.JWT.AccessSecret))
	router.Post("/api/v1/checkout", h.ProcessCheckout)

	body := fmt.Sprintf(
		`{"idempotency_key":"test-key-noauth","items":[{"product_id":"%s","quantity":1}]}`,
		uuid.New(),
	)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestCheckoutHandler_Idempotency verifies that sending the same idempotency_key
// twice returns the same cached receipt without double-charging.
// Requires at least one product with available stock in the test DB.
func TestCheckoutHandler_Idempotency(t *testing.T) {
	pool := connectTestDB(t)
	adminID, adminEmail := loginAndGetUser(t, pool)

	// Find the first active product with non-zero stock.
	var productID uuid.UUID
	err := pool.QueryRow(context.Background(), `
		SELECT p.id FROM products p
		JOIN batches b ON b.product_id = p.id
		WHERE p.is_active = true
		  AND b.quantity_remaining > 1
		  AND b.expiry_date > CURRENT_DATE
		LIMIT 1
	`).Scan(&productID)
	if err != nil {
		t.Skip("no seeded product with stock in test DB — skipping idempotency test")
	}

	h := buildCheckoutHandler(pool)
	fakeClaims := &auth.Claims{
		UserID:   adminID,
		Username: "admin",
		Email:    adminEmail,
		Role:     "ADMIN",
	}

	router := chi.NewRouter()
	router.Use(injectClaimsMiddleware(fakeClaims))
	router.Post("/api/v1/checkout", h.ProcessCheckout)

	key := uuid.New().String()
	body := fmt.Sprintf(
		`{"idempotency_key":"%s","items":[{"product_id":"%s","quantity":1}]}`,
		key, productID,
	)

	// First request.
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusCreated, w1.Code, "first checkout must succeed: %s", w1.Body.String())

	var r1 dto.ReceiptResponse
	require.NoError(t, json.NewDecoder(w1.Body).Decode(&r1))

	// Second request with the same idempotency_key must return the same sale.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusCreated, w2.Code, "idempotent replay must also return 201: %s", w2.Body.String())

	var r2 dto.ReceiptResponse
	require.NoError(t, json.NewDecoder(w2.Body).Decode(&r2))
	assert.Equal(t, r1.ID, r2.ID, "idempotent requests must return the same sale ID")
}
