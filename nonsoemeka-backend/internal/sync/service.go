package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
)

type SyncService interface {
	ProcessPush(ctx context.Context, payload PushPayload) (PushResponse, error)
	GetUsersForPull(ctx context.Context, since string) (PullUsersResponse, error)
	ApplyPulledUsers(ctx context.Context, users []SyncUser) error
	RegisterSeedAdmin(ctx context.Context, payload SeedAdminPayload) error
}

type syncService struct {
	pool         *pgxpool.Pool
	productRepo  repository.ProductRepository
	batchRepo    repository.BatchRepository
	settingsRepo repository.SettingsRepository
	saleRepo     repository.SaleRepository
	movementRepo repository.InventoryMovementRepository
	auditRepo    repository.AuditRepository
	userRepo     repository.UserRepository
}

func NewSyncService(
	pool *pgxpool.Pool,
	productRepo repository.ProductRepository,
	batchRepo repository.BatchRepository,
	settingsRepo repository.SettingsRepository,
	saleRepo repository.SaleRepository,
	movementRepo repository.InventoryMovementRepository,
	auditRepo repository.AuditRepository,
	userRepo repository.UserRepository,
) SyncService {
	return &syncService{
		pool:         pool,
		productRepo:  productRepo,
		batchRepo:    batchRepo,
		settingsRepo: settingsRepo,
		saleRepo:     saleRepo,
		movementRepo: movementRepo,
		auditRepo:    auditRepo,
		userRepo:     userRepo,
	}
}

func (s *syncService) ProcessPush(ctx context.Context, payload PushPayload) (PushResponse, error) {
	resp := PushResponse{
		ProcessedIDs: map[string][]uuid.UUID{
			"products": {},
			"batches":  {},
			"sales":    {},
			"audit":    {},
		},
		FailedIDs: map[string][]uuid.UUID{
			"movements": {},
		},
		FailureReasons: map[uuid.UUID]string{},
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "SET LOCAL statement_timeout = '30s'")
	if err != nil {
		return resp, fmt.Errorf("failed to set statement timeout: %w", err)
	}

	// 2. Products
	for _, p := range payload.Products {
		prod := models.Product{
			ID:          p.ID,
			Name:        p.Name,
			SKU:         p.SKU,
			Description: p.Description,
			IsActive:    p.IsActive,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
		_, err := s.productRepo.CreateIdempotent(ctx, tx, prod)
		if err != nil {
			return resp, fmt.Errorf("failed to sync product %s: %w", p.ID, err)
		}
		resp.ProcessedIDs["products"] = append(resp.ProcessedIDs["products"], p.ID)
	}

	// 3. Batches
	for _, b := range payload.Batches {
		batch := models.Batch{
			ID:                b.ID,
			ProductID:         b.ProductID,
			BatchNumber:       b.BatchNumber,
			QuantityReceived:  b.QuantityReceived,
			QuantityRemaining: b.QuantityRemaining,
			ExpiryDate:        b.ExpiryDate,
			CostPrice:         b.CostPrice,
			MarkupPercentage:  b.MarkupPercentage,
			ReceivedAt:        b.ReceivedAt,
		}
		_, err := s.batchRepo.CreateIdempotent(ctx, tx, batch)
		if err != nil {
			return resp, fmt.Errorf("failed to sync batch %s: %w", b.ID, err)
		}
		resp.ProcessedIDs["batches"] = append(resp.ProcessedIDs["batches"], b.ID)
	}

	// 4. Settings
	for _, setting := range payload.Settings {
		sModel := models.Setting{
			Key:       setting.Key,
			Value:     setting.Value,
			UpdatedBy: setting.UpdatedBy,
			UpdatedAt: setting.UpdatedAt,
		}
		err := s.settingsRepo.Upsert(ctx, tx, sModel)
		if err != nil {
			return resp, fmt.Errorf("failed to sync setting %s: %w", setting.Key, err)
		}
		resp.ProcessedKeys = append(resp.ProcessedKeys, setting.Key)
	}

	// 5. Sales
	for _, salePush := range payload.Sales {
		sale := models.Sale{
			ID:             salePush.Sale.ID,
			StaffID:        salePush.Sale.StaffID,
			TotalAmount:    salePush.Sale.TotalAmount,
			IdempotencyKey: salePush.Sale.IdempotencyKey,
			CreatedAt:      salePush.Sale.CreatedAt,
		}
		var items []models.SaleItem
		for _, item := range salePush.Items {
			items = append(items, models.SaleItem{
				ID:        item.ID,
				SaleID:    item.SaleID,
				ProductID: item.ProductID,
				BatchID:   item.BatchID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
			})
		}
		_, err := s.saleRepo.CreateIdempotent(ctx, tx, sale, items)
		if err != nil {
			return resp, fmt.Errorf("failed to sync sale %s: %w", salePush.Sale.ID, err)
		}
		resp.ProcessedIDs["sales"] = append(resp.ProcessedIDs["sales"], salePush.Sale.ID)
	}

	// 6. Movements
	for _, m := range payload.Movements {
		sp, err := tx.Begin(ctx)
		if err != nil {
			return resp, fmt.Errorf("failed to begin savepoint: %w", err)
		}

		movement := models.InventoryMovement{
			ID:            m.ID,
			BatchID:       m.BatchID,
			MovementType:  m.MovementType,
			QuantityDelta: m.QuantityDelta,
			ReferenceID:   m.ReferenceID,
			Reason:        m.Reason,
			CreatedBy:     m.CreatedBy,
			CreatedAt:     m.CreatedAt,
		}

		inserted, err := s.movementRepo.CreateIdempotent(ctx, sp, movement)
		if err != nil {
			sp.Rollback(ctx)
			return resp, fmt.Errorf("failed to create movement %s: %w", m.ID, err)
		}

		if !inserted {
			sp.Commit(ctx)
			status, reason, err := s.movementRepo.GetSyncStatus(ctx, tx, m.ID)
			if err != nil {
				return resp, fmt.Errorf("failed to get sync status for existing movement %s: %w", m.ID, err)
			}
			if status == "FAILED" {
				resp.FailedIDs["movements"] = append(resp.FailedIDs["movements"], m.ID)
				resp.FailureReasons[m.ID] = reason
			} else {
				resp.ProcessedIDs["movements"] = append(resp.ProcessedIDs["movements"], m.ID)
			}
			continue
		}

		batch, err := s.batchRepo.LockByID(ctx, sp, m.BatchID)
		if err != nil {
			sp.Rollback(ctx)
			return resp, fmt.Errorf("failed to lock batch %s for movement %s: %w", m.BatchID, m.ID, err)
		}

		newRemaining := batch.QuantityRemaining + m.QuantityDelta
		if newRemaining < 0 || newRemaining > batch.QuantityReceived {
			sp.Commit(ctx)
			reason := fmt.Sprintf("invalid quantity: remaining %d, delta %d, received %d", batch.QuantityRemaining, m.QuantityDelta, batch.QuantityReceived)
			if markErr := s.movementRepo.MarkSyncFailed(ctx, tx, m.ID, reason); markErr != nil {
				return resp, fmt.Errorf("failed to mark movement %s sync failed: %w", m.ID, markErr)
			}
			resp.FailedIDs["movements"] = append(resp.FailedIDs["movements"], m.ID)
			resp.FailureReasons[m.ID] = reason
			continue
		}

		batch.QuantityRemaining = newRemaining
		if err := s.batchRepo.Update(ctx, sp, batch); err != nil {
			sp.Rollback(ctx)
			return resp, fmt.Errorf("failed to update batch %s for movement %s: %w", batch.ID, m.ID, err)
		}

		sp.Commit(ctx)
		resp.ProcessedIDs["movements"] = append(resp.ProcessedIDs["movements"], m.ID)
	}

	// 7. AuditLogs
	for _, a := range payload.AuditLogs {
		logEntry := models.AuditLog{
			ID:          a.ID,
			ActorID:     a.ActorID,
			Action:      a.Action,
			TargetTable: a.TargetTable,
			TargetID:    a.TargetID,
			Metadata:    a.Metadata,
			CreatedAt:   a.CreatedAt,
		}
		_, err := s.auditRepo.CreateIdempotent(ctx, tx, logEntry)
		if err != nil {
			return resp, fmt.Errorf("failed to sync audit log %s: %w", a.ID, err)
		}
		resp.ProcessedIDs["audit"] = append(resp.ProcessedIDs["audit"], a.ID)
	}

	// 8. UserStates
	for _, us := range payload.UserStates {
		err := s.userRepo.ApplyMostRestrictiveSecurityState(ctx, tx, us.ID, us.IsActive, us.LockedUntil)
		if err != nil {
			if errors.Is(err, apperrors.ErrNotFound) {
				resp.FailedIDs["users"] = append(resp.FailedIDs["users"], us.ID)
				resp.FailureReasons[us.ID] = "user not found on cloud"
				continue
			}
			return resp, fmt.Errorf("failed to sync user state %s: %w", us.ID, err)
		}
		resp.ProcessedIDs["users"] = append(resp.ProcessedIDs["users"], us.ID)
	}

	if err := tx.Commit(ctx); err != nil {
		return resp, fmt.Errorf("failed to commit sync transaction: %w", err)
	}

	return resp, nil
}

func (s *syncService) GetUsersForPull(ctx context.Context, since string) (PullUsersResponse, error) {
	var sinceTime time.Time
	if since != "" {
		t, err := time.Parse(time.RFC3339, since)
		if err == nil {
			sinceTime = t
		}
	}

	users, srvTime, err := s.userRepo.ListUpdatedSince(ctx, s.pool, sinceTime)
	if err != nil {
		return PullUsersResponse{}, fmt.Errorf("failed to list users: %w", err)
	}

	var syncUsers []SyncUser
	for _, u := range users {
		syncUsers = append(syncUsers, SyncUser{
			ID:           u.ID,
			Username:     u.Username,
			Email:        u.Email,
			PasswordHash: u.PasswordHash,
			Role:         models.UserRole(u.Role),
			IsActive:     u.IsActive,
			LockedUntil:  u.LockedUntil,
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
		})
	}

	return PullUsersResponse{
		Users:      syncUsers,
		NextCursor: srvTime.Format(time.RFC3339Nano),
	}, nil
}

func (s *syncService) ApplyPulledUsers(ctx context.Context, users []SyncUser) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, u := range users {
		user := models.User{
			ID:           u.ID,
			Username:     u.Username,
			Email:        u.Email,
			PasswordHash: u.PasswordHash,
			Role:         u.Role,
			IsActive:     u.IsActive,
			LockedUntil:  u.LockedUntil,
			CreatedAt:    u.CreatedAt,
			UpdatedAt:    u.UpdatedAt,
		}
		if err := s.userRepo.UpsertFromCloud(ctx, tx, user); err != nil {
			return fmt.Errorf("failed to upsert user %s: %w", u.ID, err)
		}
		if err := s.userRepo.ApplyMostRestrictiveSecurityState(ctx, tx, user.ID, user.IsActive, user.LockedUntil); err != nil {
			return fmt.Errorf("failed to apply security state for user %s: %w", u.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit pulled users: %w", err)
	}

	return nil
}

func (s *syncService) RegisterSeedAdmin(ctx context.Context, payload SeedAdminPayload) error {
	inserted, err := s.userRepo.CreateIfNotExists(ctx, s.pool, payload.ID, payload.Username, payload.Email, payload.PasswordHash, payload.Role)
	if err != nil {
		return fmt.Errorf("failed to register seed admin: %w", err)
	}

	if inserted {
		slog.Info("Seed admin registered", "id", payload.ID, "username", payload.Username)
	} else {
		slog.Info("Seed admin already exists", "id", payload.ID, "username", payload.Username)
	}

	return nil
}
