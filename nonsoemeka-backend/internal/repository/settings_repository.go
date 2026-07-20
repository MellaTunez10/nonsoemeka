package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/models"
)

type SettingsRepository interface {
	Get(ctx context.Context, db DBTX, key string) (models.Setting, error)
	Set(ctx context.Context, db DBTX, key string, value json.RawMessage, updatedBy *uuid.UUID) error
	GetAll(ctx context.Context, db DBTX) ([]models.Setting, error)
}

type postgresSettingsRepository struct{}

func NewSettingsRepository() SettingsRepository {
	return &postgresSettingsRepository{}
}

func (r *postgresSettingsRepository) Get(ctx context.Context, db DBTX, key string) (models.Setting, error) {
	query := `SELECT key, value, updated_by, updated_at FROM settings WHERE key = $1`
	var s models.Setting
	err := db.QueryRow(ctx, query, key).Scan(&s.Key, &s.Value, &s.UpdatedBy, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Setting{}, apperrors.ErrNotFound
		}
		return models.Setting{}, fmt.Errorf("failed to get setting %s: %w", key, err)
	}
	return s, nil
}

func (r *postgresSettingsRepository) Set(ctx context.Context, db DBTX, key string, value json.RawMessage, updatedBy *uuid.UUID) error {
	query := `
		INSERT INTO settings (key, value, updated_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()
	`
	_, err := db.Exec(ctx, query, key, value, updatedBy)
	if err != nil {
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}
	return nil
}

func (r *postgresSettingsRepository) GetAll(ctx context.Context, db DBTX) ([]models.Setting, error) {
	query := `SELECT key, value, updated_by, updated_at FROM settings`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}
	defer rows.Close()

	var list []models.Setting
	for rows.Next() {
		var s models.Setting
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedBy, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}
