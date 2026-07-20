package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"nonsoemeka-backend/internal/audit"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/repository"
)

type SettingsService interface {
	GetSettings(ctx context.Context) (dto.SettingsResponse, error)
	UpdateSettings(ctx context.Context, actorID uuid.UUID, req dto.UpdateSettingsRequest) (dto.SettingsResponse, error)
}

type settingsService struct {
	pool         *pgxpool.Pool
	settingsRepo repository.SettingsRepository
	auditRepo    repository.AuditRepository
}

func NewSettingsService(pool *pgxpool.Pool, settingsRepo repository.SettingsRepository, auditRepo repository.AuditRepository) SettingsService {
	return &settingsService{
		pool:         pool,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
	}
}

func (s *settingsService) GetSettings(ctx context.Context) (dto.SettingsResponse, error) {
	all, err := s.settingsRepo.GetAll(ctx, s.pool)
	if err != nil {
		return dto.SettingsResponse{}, err
	}

	res := dto.SettingsResponse{
		DefaultMarkupPercentage: "25.00",
		ExpiryAlertDays:         30,
		LowStockThreshold:       10,
		PharmacyName:            "Nonsoemeka Pharmacy",
		ReceiptFooter:           "Thank you for trusting Nonsoemeka Pharmacy! Wish you good health.",
	}

	for _, setting := range all {
		switch setting.Key {
		case "default_markup_percentage":
			var val string
			if json.Unmarshal(setting.Value, &val) == nil {
				res.DefaultMarkupPercentage = val
			}
		case "expiry_alert_days":
			var val int
			if json.Unmarshal(setting.Value, &val) == nil {
				res.ExpiryAlertDays = val
			}
		case "low_stock_threshold":
			var val int
			if json.Unmarshal(setting.Value, &val) == nil {
				res.LowStockThreshold = val
			}
		case "pharmacy_name":
			var val string
			if json.Unmarshal(setting.Value, &val) == nil {
				res.PharmacyName = val
			}
		case "receipt_footer":
			var val string
			if json.Unmarshal(setting.Value, &val) == nil {
				res.ReceiptFooter = val
			}
		}
	}

	return res, nil
}

func (s *settingsService) UpdateSettings(ctx context.Context, actorID uuid.UUID, req dto.UpdateSettingsRequest) (dto.SettingsResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.SettingsResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	currentSettings, err := s.GetSettings(ctx)
	if err != nil {
		return dto.SettingsResponse{}, err
	}

	changes := make(map[string]interface{})

	if req.DefaultMarkupPercentage != nil {
		bytes, _ := json.Marshal(*req.DefaultMarkupPercentage)
		if err := s.settingsRepo.Set(ctx, tx, "default_markup_percentage", bytes, &actorID); err != nil {
			return dto.SettingsResponse{}, err
		}
		changes["default_markup_percentage"] = map[string]string{"before": currentSettings.DefaultMarkupPercentage, "after": *req.DefaultMarkupPercentage}
	}

	if req.ExpiryAlertDays != nil {
		bytes, _ := json.Marshal(*req.ExpiryAlertDays)
		if err := s.settingsRepo.Set(ctx, tx, "expiry_alert_days", bytes, &actorID); err != nil {
			return dto.SettingsResponse{}, err
		}
		changes["expiry_alert_days"] = map[string]interface{}{"before": currentSettings.ExpiryAlertDays, "after": *req.ExpiryAlertDays}
	}

	if req.LowStockThreshold != nil {
		bytes, _ := json.Marshal(*req.LowStockThreshold)
		if err := s.settingsRepo.Set(ctx, tx, "low_stock_threshold", bytes, &actorID); err != nil {
			return dto.SettingsResponse{}, err
		}
		changes["low_stock_threshold"] = map[string]interface{}{"before": currentSettings.LowStockThreshold, "after": *req.LowStockThreshold}
	}

	if req.PharmacyName != nil {
		bytes, _ := json.Marshal(*req.PharmacyName)
		if err := s.settingsRepo.Set(ctx, tx, "pharmacy_name", bytes, &actorID); err != nil {
			return dto.SettingsResponse{}, err
		}
		changes["pharmacy_name"] = map[string]string{"before": currentSettings.PharmacyName, "after": *req.PharmacyName}
	}

	if req.ReceiptFooter != nil {
		bytes, _ := json.Marshal(*req.ReceiptFooter)
		if err := s.settingsRepo.Set(ctx, tx, "receipt_footer", bytes, &actorID); err != nil {
			return dto.SettingsResponse{}, err
		}
		changes["receipt_footer"] = map[string]string{"before": currentSettings.ReceiptFooter, "after": *req.ReceiptFooter}
	}

	if len(changes) > 0 {
		if err := audit.LogAction(ctx, s.auditRepo, tx, actorID, "SETTINGS_UPDATED", "settings", nil, changes); err != nil {
			return dto.SettingsResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.SettingsResponse{}, fmt.Errorf("failed to commit settings transaction: %w", err)
	}

	return s.GetSettings(ctx)
}
