package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"nonsoemeka-backend/internal/models"
)

type InventoryMovementRepository interface {
	Create(ctx context.Context, db DBTX, m models.InventoryMovement) error
	ListByBatch(ctx context.Context, db DBTX, batchID uuid.UUID, page, pageSize int) ([]models.InventoryMovement, int, error)
	List(ctx context.Context, db DBTX, movementType *string, page, pageSize int) ([]models.InventoryMovement, int, error)
	CreateIdempotent(ctx context.Context, db DBTX, m models.InventoryMovement) (bool, error)
	MarkSyncFailed(ctx context.Context, db DBTX, id uuid.UUID, reason string) error
	GetSyncStatus(ctx context.Context, db DBTX, id uuid.UUID) (string, string, error)
}

type postgresInventoryMovementRepository struct{}

func NewInventoryMovementRepository() InventoryMovementRepository {
	return &postgresInventoryMovementRepository{}
}

func (r *postgresInventoryMovementRepository) Create(ctx context.Context, db DBTX, m models.InventoryMovement) error {
	query := `
		INSERT INTO inventory_movements (batch_id, movement_type, quantity_delta, reference_id, reason, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(ctx, query, m.BatchID, m.MovementType, m.QuantityDelta, m.ReferenceID, m.Reason, m.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to create inventory movement: %w", err)
	}
	return nil
}

func (r *postgresInventoryMovementRepository) CreateIdempotent(ctx context.Context, db DBTX, m models.InventoryMovement) (bool, error) {
	query := `
		INSERT INTO inventory_movements (id, batch_id, movement_type, quantity_delta, reference_id, reason, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
		RETURNING id
	`
	var id uuid.UUID
	err := db.QueryRow(ctx, query, m.ID, m.BatchID, m.MovementType, m.QuantityDelta, m.ReferenceID, m.Reason, m.CreatedBy, m.CreatedAt).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to idempotently create inventory movement: %w", err)
	}
	return true, nil
}

func (r *postgresInventoryMovementRepository) MarkSyncFailed(ctx context.Context, db DBTX, id uuid.UUID, reason string) error {
	query := `UPDATE inventory_movements SET sync_status = 'FAILED', sync_failure_reason = $2 WHERE id = $1`
	_, err := db.Exec(ctx, query, id, reason)
	if err != nil {
		return fmt.Errorf("failed to mark inventory movement sync failed: %w", err)
	}
	return nil
}

func (r *postgresInventoryMovementRepository) GetSyncStatus(ctx context.Context, db DBTX, id uuid.UUID) (string, string, error) {
	query := `SELECT sync_status, COALESCE(sync_failure_reason, '') FROM inventory_movements WHERE id = $1`
	var status, reason string
	err := db.QueryRow(ctx, query, id).Scan(&status, &reason)
	if err != nil {
		return "", "", fmt.Errorf("failed to get sync status for movement: %w", err)
	}
	return status, reason, nil
}

func (r *postgresInventoryMovementRepository) ListByBatch(ctx context.Context, db DBTX, batchID uuid.UUID, page, pageSize int) ([]models.InventoryMovement, int, error) {
	offset := (page - 1) * pageSize
	countQuery := `SELECT COUNT(*) FROM inventory_movements WHERE batch_id = $1`
	var total int
	if err := db.QueryRow(ctx, countQuery, batchID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count inventory movements for batch: %w", err)
	}

	query := `
		SELECT im.id, im.batch_id, b.batch_number, b.product_id, p.name as product_name,
		       im.movement_type, im.quantity_delta, im.reference_id, im.reason,
		       COALESCE(im.created_by, '00000000-0000-0000-0000-000000000000'::uuid) as created_by,
		       COALESCE(u.username, 'Deleted User') as created_by_name, im.created_at
		FROM inventory_movements im
		JOIN batches b ON im.batch_id = b.id
		JOIN products p ON b.product_id = p.id
		LEFT JOIN users u ON im.created_by = u.id
		WHERE im.batch_id = $1
		ORDER BY im.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.Query(ctx, query, batchID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list inventory movements by batch: %w", err)
	}
	defer rows.Close()

	var movements []models.InventoryMovement
	for rows.Next() {
		var m models.InventoryMovement
		if err := rows.Scan(
			&m.ID, &m.BatchID, &m.BatchNumber, &m.ProductID, &m.ProductName,
			&m.MovementType, &m.QuantityDelta, &m.ReferenceID, &m.Reason,
			&m.CreatedBy, &m.CreatedByName, &m.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan movement row: %w", err)
		}
		movements = append(movements, m)
	}

	return movements, total, nil
}

func (r *postgresInventoryMovementRepository) List(ctx context.Context, db DBTX, movementType *string, page, pageSize int) ([]models.InventoryMovement, int, error) {
	offset := (page - 1) * pageSize
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if movementType != nil && *movementType != "" {
		whereClause += fmt.Sprintf(" AND im.movement_type = $%d", argIdx)
		args = append(args, *movementType)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM inventory_movements im %s`, whereClause)
	var total int
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count inventory movements: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT im.id, im.batch_id, b.batch_number, b.product_id, p.name as product_name,
		       im.movement_type, im.quantity_delta, im.reference_id, im.reason,
		       COALESCE(im.created_by, '00000000-0000-0000-0000-000000000000'::uuid) as created_by,
		       COALESCE(u.username, 'Deleted User') as created_by_name, im.created_at
		FROM inventory_movements im
		JOIN batches b ON im.batch_id = b.id
		JOIN products p ON b.product_id = p.id
		LEFT JOIN users u ON im.created_by = u.id
		%s
		ORDER BY im.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list inventory movements: %w", err)
	}
	defer rows.Close()

	var movements []models.InventoryMovement
	for rows.Next() {
		var m models.InventoryMovement
		if err := rows.Scan(
			&m.ID, &m.BatchID, &m.BatchNumber, &m.ProductID, &m.ProductName,
			&m.MovementType, &m.QuantityDelta, &m.ReferenceID, &m.Reason,
			&m.CreatedBy, &m.CreatedByName, &m.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan movement row: %w", err)
		}
		movements = append(movements, m)
	}

	return movements, total, nil
}
