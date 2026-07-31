package sync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nonsoemeka-backend/internal/database"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
	"nonsoemeka-backend/internal/sync"
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

	migrationsDir := filepath.Join("..", "..", "migrations")
	err = database.RunMigrations(context.Background(), pool, migrationsDir)
	require.NoError(t, err, "failed to run database migrations on test DB")

	return pool
}

func TestProcessPush_MovementSavepoints(t *testing.T) {
	pool := connectTestDB(t)
	ctx := context.Background()

	// 1. Initialize repositories and service
	productRepo := repository.NewProductRepository()
	batchRepo := repository.NewBatchRepository()
	settingsRepo := repository.NewSettingsRepository()
	saleRepo := repository.NewSaleRepository()
	movementRepo := repository.NewInventoryMovementRepository()
	auditRepo := repository.NewAuditRepository()
	userRepo := repository.NewUserRepository()

	syncService := sync.NewSyncService(
		pool,
		productRepo,
		batchRepo,
		settingsRepo,
		saleRepo,
		movementRepo,
		auditRepo,
		userRepo,
	)

	// 2. Seed prerequisites directly using pgx so we can rely on them
	userID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, username, password_hash, role, email) VALUES ($1, $2, 'hash', 'ADMIN', $3) ON CONFLICT DO NOTHING`, userID, "admin_"+uuid.New().String(), "email_"+uuid.New().String()+"@test.com")
	require.NoError(t, err)

	productID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO products (id, name, sku) VALUES ($1, 'Test Product', $2)`, productID, "SKU-"+uuid.New().String())
	require.NoError(t, err)

	batchID := uuid.New()
	// Set quantity received to 100, remaining to 10. Cost is 10.0, markup is 50.0 (resulting selling_price = 15.0)
	_, err = pool.Exec(ctx, `INSERT INTO batches (id, product_id, batch_number, quantity_received, quantity_remaining, expiry_date, cost_price, markup_percentage) VALUES ($1, $2, $3, 100, 10, NOW() + INTERVAL '1 year', 10.0, 50.0)`, batchID, productID, "BATCH-"+uuid.New().String())
	require.NoError(t, err)

	// 3. Prepare Push Payload with two movements
	validMovementID := uuid.New()
	invalidMovementID := uuid.New()
	syncProductID := uuid.New()
	syncBatchID := uuid.New()
	syncSettingKey := "test_sync_setting_" + uuid.New().String()
	syncUserID := uuid.New()
	missingSyncUserID := uuid.New()

	_, err = pool.Exec(ctx, `INSERT INTO users (id, username, password_hash, role, email) VALUES ($1, $2, 'hash', 'STAFF', $3) ON CONFLICT DO NOTHING`, syncUserID, "staff_"+uuid.New().String(), "email_staff_"+uuid.New().String()+"@test.com")
	require.NoError(t, err)

	payload := sync.PushPayload{
		Products: []sync.SyncProduct{
			{
				ID:        syncProductID,
				Name:      "Test Sync Product",
				SKU:       "SYNC-SKU-" + uuid.New().String(),
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		Batches: []sync.SyncBatch{
			{
				ID:                syncBatchID,
				ProductID:         syncProductID,
				BatchNumber:       "SYNC-BATCH",
				QuantityReceived:  100,
				QuantityRemaining: 100,
				ExpiryDate:        time.Now().Add(time.Hour * 24 * 365),
				CostPrice:         decimal.NewFromFloat(10.0),
				MarkupPercentage:  decimal.NewFromFloat(50.0),
				ReceivedAt:        time.Now(),
			},
		},
		Settings: []models.Setting{
			{
				Key:       syncSettingKey,
				Value:     []byte(`{"test":true}`),
				UpdatedBy: &userID,
				UpdatedAt: time.Now(),
			},
		},
		UserStates: []sync.UserSecurityState{
			{
				ID:          syncUserID,
				IsActive:    true,
				LockedUntil: nil,
				UpdatedAt:   time.Now(),
			},
			{
				ID:          missingSyncUserID,
				IsActive:    false,
				LockedUntil: nil,
				UpdatedAt:   time.Now(),
			},
		},
		Movements: []sync.SyncMovement{
			{
				ID:            validMovementID,
				BatchID:       batchID,
				MovementType:  "DISPENSED",
				QuantityDelta: -5, // valid, 10 - 5 = 5
				ReferenceID:   nil,
				Reason:        func() *string { s := "Sold"; return &s }(),
				CreatedBy:     userID,
				CreatedAt:     time.Now(),
			},
			{
				ID:            invalidMovementID,
				BatchID:       batchID,
				MovementType:  "DISPENSED",
				QuantityDelta: -20, // invalid, 5 - 20 = -15 (< 0)
				ReferenceID:   nil,
				Reason:        func() *string { s := "Sold way too much"; return &s }(),
				CreatedBy:     userID,
				CreatedAt:     time.Now(),
			},
		},
	}

	// 4. Run ProcessPush
	resp, err := syncService.ProcessPush(ctx, payload)
	require.NoError(t, err, "ProcessPush should not return a fatal error for a data bounds violation")

	// 5. Assert the response segregates the successful and failed movements
	assert.Contains(t, resp.ProcessedIDs["movements"], validMovementID, "Valid movement should be processed")
	assert.NotContains(t, resp.ProcessedIDs["movements"], invalidMovementID, "Invalid movement should NOT be processed")

	assert.Contains(t, resp.FailedIDs["movements"], invalidMovementID, "Invalid movement should be in FailedIDs")
	assert.NotEmpty(t, resp.FailureReasons[invalidMovementID], "Failure reason should be provided for invalid movement")

	assert.Contains(t, resp.ProcessedIDs["products"], syncProductID)
	assert.Contains(t, resp.ProcessedIDs["batches"], syncBatchID)
	assert.Contains(t, resp.ProcessedKeys, syncSettingKey)
	assert.Contains(t, resp.ProcessedIDs["users"], syncUserID)

	assert.Contains(t, resp.FailedIDs["users"], missingSyncUserID, "Missing user should be in FailedIDs")
	assert.NotEmpty(t, resp.FailureReasons[missingSyncUserID], "Failure reason should be provided for missing user")

	// 6. Assert DB state - valid movement is present and batch quantity is updated
	var remaining int
	err = pool.QueryRow(ctx, `SELECT quantity_remaining FROM batches WHERE id = $1`, batchID).Scan(&remaining)
	require.NoError(t, err)
	assert.Equal(t, 5, remaining, "Batch quantity should be deducted by the valid movement only")

	var validCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM inventory_movements WHERE id = $1`, validMovementID).Scan(&validCount)
	require.NoError(t, err)
	assert.Equal(t, 1, validCount, "Valid movement should be persisted in the database")

	// 7. Assert DB state - invalid movement is marked FAILED in the DB
	var syncStatus, failureReason string
	err = pool.QueryRow(ctx, `SELECT sync_status, sync_failure_reason FROM inventory_movements WHERE id = $1`, invalidMovementID).Scan(&syncStatus, &failureReason)
	require.NoError(t, err, "Invalid movement should be persisted so we can flag it as FAILED")

	assert.Equal(t, "FAILED", syncStatus, "Invalid movement should be marked FAILED in the db")
	assert.Contains(t, failureReason, "invalid quantity", "Database failure reason should mention invalid quantity")

	// 8. Call ProcessPush AGAIN with the exact same payload (simulating Local node retry)
	resp2, err2 := syncService.ProcessPush(ctx, payload)
	require.NoError(t, err2, "ProcessPush should handle retries cleanly")

	// 9. Assert the second response ALSO segregates the successful and failed movements correctly
	assert.Contains(t, resp2.ProcessedIDs["movements"], validMovementID, "Valid movement should STILL be returned as processed on retry")
	assert.NotContains(t, resp2.ProcessedIDs["movements"], invalidMovementID, "Invalid movement should NOT be processed on retry")

	assert.Contains(t, resp2.FailedIDs["movements"], invalidMovementID, "Invalid movement should STILL be in FailedIDs on retry")
	assert.NotEmpty(t, resp2.FailureReasons[invalidMovementID], "Failure reason should be provided for invalid movement on retry")

	assert.Contains(t, resp2.ProcessedIDs["products"], syncProductID, "Product should be returned as processed on retry")
	assert.Contains(t, resp2.ProcessedIDs["batches"], syncBatchID, "Batch should be returned as processed on retry")
	assert.Contains(t, resp2.ProcessedKeys, syncSettingKey, "Setting should be returned as processed on retry")
	assert.Contains(t, resp2.ProcessedIDs["users"], syncUserID, "User should be returned as processed on retry")

	// 10. Assert DB state did NOT change further
	var remainingAfterRetry int
	err = pool.QueryRow(ctx, `SELECT quantity_remaining FROM batches WHERE id = $1`, batchID).Scan(&remainingAfterRetry)
	require.NoError(t, err)
	assert.Equal(t, 5, remainingAfterRetry, "Batch quantity should NOT be deducted again on retry")
}
