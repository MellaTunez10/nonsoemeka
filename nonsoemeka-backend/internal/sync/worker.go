package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
)

type SyncWorker struct {
	pool         *pgxpool.Pool
	trackingRepo SyncTrackingRepository
	syncService  SyncService
	settingsRepo repository.SettingsRepository
	userRepo     repository.UserRepository
	cloudURL     string
	nodeKey      string
	interval     time.Duration
	client       *http.Client
}

func NewSyncWorker(
	pool *pgxpool.Pool,
	trackingRepo SyncTrackingRepository,
	syncService SyncService,
	settingsRepo repository.SettingsRepository,
	userRepo repository.UserRepository,
	cloudURL string,
	nodeKey string,
	interval time.Duration,
) *SyncWorker {
	return &SyncWorker{
		pool:         pool,
		trackingRepo: trackingRepo,
		syncService:  syncService,
		settingsRepo: settingsRepo,
		userRepo:     userRepo,
		cloudURL:     cloudURL,
		nodeKey:      nodeKey,
		interval:     interval,
		client:       &http.Client{Timeout: 60 * time.Second},
	}
}

func (w *SyncWorker) Start(ctx context.Context) {
	slog.Info("sync worker starting", "mode", "LOCAL", "cloud_url", w.cloudURL, "interval", w.interval)

	// One-time seed admin registration on first successful connection
	w.tryRegisterSeedAdmin(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("sync worker shutting down")
			return
		case <-ticker.C:
			w.runSyncCycle(ctx)
		}
	}
}

func (w *SyncWorker) runSyncCycle(ctx context.Context) {
	// Check if Cloud is reachable
	if !w.isCloudHealthy(ctx) {
		slog.Debug("cloud node unreachable, skipping sync cycle")
		return
	}

	// Push phase
	w.pushToCloud(ctx)

	// Pull phase
	w.pullUsersFromCloud(ctx)
}

func (w *SyncWorker) isCloudHealthy(ctx context.Context) bool {
	ctxReq, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxReq, http.MethodGet, w.cloudURL+"/healthz", nil)
	if err != nil {
		return false
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (w *SyncWorker) pushToCloud(ctx context.Context) {
	limit := 500

	products, _ := w.trackingRepo.FetchPendingProducts(ctx, w.pool, limit)
	batches, _ := w.trackingRepo.FetchPendingBatches(ctx, w.pool, limit)
	settings, _ := w.trackingRepo.FetchPendingSettings(ctx, w.pool, limit)
	sales, _ := w.trackingRepo.FetchPendingSales(ctx, w.pool, limit)
	movements, _ := w.trackingRepo.FetchPendingMovements(ctx, w.pool, limit)
	auditLogs, _ := w.trackingRepo.FetchPendingAuditLogs(ctx, w.pool, limit)
	userStates, _ := w.trackingRepo.FetchPendingUserStates(ctx, w.pool, limit)

	if len(products) == 0 && len(batches) == 0 && len(settings) == 0 && len(sales) == 0 && len(movements) == 0 && len(auditLogs) == 0 && len(userStates) == 0 {
		return
	}

	payload := PushPayload{
		Products:   products,
		Batches:    batches,
		Settings:   settings,
		Sales:      sales,
		Movements:  movements,
		AuditLogs:  auditLogs,
		UserStates: userStates,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal push payload", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cloudURL+"/api/v1/sync/push", bytes.NewReader(data))
	if err != nil {
		slog.Error("failed to create push request", "error", err)
		return
	}
	req.Header.Set("X-Sync-Node-Key", w.nodeKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		slog.Error("failed to push to cloud", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("non-200 response from cloud push", "status", resp.StatusCode)
		return
	}

	var pushResp PushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushResp); err != nil {
		slog.Error("failed to decode push response", "error", err)
		return
	}

	processedCount := 0
	failedCount := 0

	for table, ids := range pushResp.ProcessedIDs {
		if err := w.trackingRepo.MarkSynced(ctx, w.pool, table, ids); err != nil {
			slog.Error("failed to mark synced", "table", table, "error", err)
		}
		processedCount += len(ids)
	}

	if len(pushResp.ProcessedKeys) > 0 {
		if err := w.trackingRepo.MarkSettingsSynced(ctx, w.pool, pushResp.ProcessedKeys); err != nil {
			slog.Error("failed to mark settings synced", "error", err)
		}
		processedCount += len(pushResp.ProcessedKeys)
	}

	for table, ids := range pushResp.FailedIDs {
		if err := w.trackingRepo.MarkFailed(ctx, w.pool, table, ids, "rejected by cloud"); err != nil {
			slog.Error("failed to mark failed", "table", table, "error", err)
		}
		failedCount += len(ids)
	}

	slog.Info("sync push completed", "processed", processedCount, "failed", failedCount)
}

func (w *SyncWorker) pullUsersFromCloud(ctx context.Context) {
	setting, err := w.settingsRepo.Get(ctx, w.pool, "sync_pull_cursor")
	cursor := ""
	if err == nil {
		var c string
		if err := json.Unmarshal(setting.Value, &c); err == nil {
			cursor = c
		}
	} else if err != apperrors.ErrNotFound {
		slog.Error("failed to get pull cursor", "error", err)
		return
	}

	reqURL := w.cloudURL + "/api/v1/sync/pull-users"
	if cursor != "" {
		reqURL += "?since=" + cursor
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		slog.Error("failed to create pull request", "error", err)
		return
	}
	req.Header.Set("X-Sync-Node-Key", w.nodeKey)

	resp, err := w.client.Do(req)
	if err != nil {
		slog.Error("failed to pull users from cloud", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("non-200 response from cloud pull", "status", resp.StatusCode)
		return
	}

	var pullResp PullUsersResponse
	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		slog.Error("failed to decode pull response", "error", err)
		return
	}

	if len(pullResp.Users) == 0 {
		return
	}

	if err := w.syncService.ApplyPulledUsers(ctx, pullResp.Users); err != nil {
		slog.Error("failed to apply pulled users", "error", err)
		return
	}

	cursorJSON, _ := json.Marshal(pullResp.NextCursor)
	if err := w.settingsRepo.Set(ctx, w.pool, "sync_pull_cursor", cursorJSON, nil); err != nil {
		slog.Error("failed to save pull cursor", "error", err)
	}

	slog.Info("sync pull completed", "users_pulled", len(pullResp.Users))
}

func (w *SyncWorker) tryRegisterSeedAdmin(ctx context.Context) {
	_, err := w.settingsRepo.Get(ctx, w.pool, "local_seed_admin_registered")
	if err == nil {
		return
	} else if err != apperrors.ErrNotFound {
		slog.Error("failed to check local_seed_admin_registered flag", "error", err)
		return
	}

	if !w.isCloudHealthy(ctx) {
		slog.Debug("cloud node unreachable, skipping seed admin registration")
		return
	}

	var admin models.User
	err = w.pool.QueryRow(ctx, `SELECT id, username, email, password_hash, role FROM users WHERE role = 'ADMIN' LIMIT 1`).Scan(&admin.ID, &admin.Username, &admin.Email, &admin.PasswordHash, &admin.Role)
	if err != nil {
		slog.Error("failed to query local seed admin", "error", err)
		return
	}

	payload := SeedAdminPayload{
		ID:           admin.ID,
		Username:     admin.Username,
		Email:        admin.Email,
		PasswordHash: admin.PasswordHash,
		Role:         admin.Role,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("failed to marshal seed admin payload", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cloudURL+"/api/v1/sync/register-seed-admin", bytes.NewReader(data))
	if err != nil {
		slog.Error("failed to create seed admin request", "error", err)
		return
	}
	req.Header.Set("X-Sync-Node-Key", w.nodeKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		slog.Error("failed to register seed admin", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		trueJSON, _ := json.Marshal(true)
		if err := w.settingsRepo.Set(ctx, w.pool, "local_seed_admin_registered", trueJSON, nil); err != nil {
			slog.Error("failed to set local_seed_admin_registered flag", "error", err)
		} else {
			slog.Info("seed admin registered on cloud successfully")
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("failed to register seed admin, non-200 response", "status", resp.StatusCode, "body", string(body))
	}
}

// collectIDs is a small utility to extract UUIDs from sync payload slices.
func collectIDs[T any](items []T, getID func(T) uuid.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, len(items))
	for i, item := range items {
		ids[i] = getID(item)
	}
	return ids
}
