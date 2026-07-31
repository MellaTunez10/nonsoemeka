package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"nonsoemeka-backend/internal/auth"
	"nonsoemeka-backend/internal/config"
	"nonsoemeka-backend/internal/handlers"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/sync"
)

// Fake repo to return a successful empty list instead of panicking
type mockSyncTrackingRepo struct {
	sync.SyncTrackingRepository // Embed to satisfy interface for methods we don't call
}

func (m *mockSyncTrackingRepo) ListFailedMovements(ctx context.Context, db sync.DBTX, offset, limit int) ([]models.InventoryMovement, int, error) {
	return []models.InventoryMovement{}, 0, nil
}

func TestSyncAdminFailedItemsAuth(t *testing.T) {
	// 1. Setup mock config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:            8080,
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    5 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
		RateLimit: config.RateLimitConfig{
			GlobalPerMinute: 1000,
			LoginPerMinute:  10,
		},
		JWT: config.JWTConfig{
			AccessSecret:  "test-secret",
			AccessTTL:     1 * time.Hour,
			RefreshSecret: "refresh-secret",
			RefreshTTL:    24 * time.Hour,
		},
		Sync: config.SyncConfig{
			ServerMode: "CLOUD",
			NodeKey:    "test-sync-key",
			CloudURL:   "http://localhost:8080",
			Interval:   30 * time.Second,
		},
	}

	mockRepo := &mockSyncTrackingRepo{}

	// Create a real handler with our fake repo (pool and service can be nil as ListFailedMovements doesn't use them in our mock)
	syncHandler := sync.NewSyncHandler(nil, nil, mockRepo)

	r := setupRouter(
		cfg,
		(*pgxpool.Pool)(nil),
		&handlers.AuthHandler{},
		&handlers.InventoryHandler{},
		&handlers.CheckoutHandler{},
		&handlers.FinancialsHandler{},
		&handlers.ReportsHandler{},
		&handlers.StaffManagementHandler{},
		&handlers.SettingsHandler{},
		syncHandler,
	)

	t.Run("Rejects request with only sync node key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sync/failed", nil)
		// Add the sync key but no JWT
		req.Header.Set("X-Sync-Node-Key", "test-sync-key")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Rejects request with Staff JWT (Requires Admin)", func(t *testing.T) {
		token, _ := auth.GenerateAccessToken(uuid.New(), "user123", "user@example.com", "STAFF", cfg.JWT.AccessSecret, cfg.JWT.AccessTTL)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sync/failed", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("Succeeds with Admin JWT (Requires NO sync node key)", func(t *testing.T) {
		token, _ := auth.GenerateAccessToken(uuid.New(), "admin123", "admin@example.com", "ADMIN", cfg.JWT.AccessSecret, cfg.JWT.AccessTTL)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sync/failed", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Now we assert exactly on 200 OK without panicking
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
