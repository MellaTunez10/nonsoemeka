package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"nonsoemeka-backend/internal/models"
)

type InventoryMovementRepository interface {
	Create(ctx context.Context, db DBTX, m models.InventoryMovement) error
	ListByBatch(ctx context.Context, db DBTX, batchID uuid.UUID, page, pageSize int) ([]models.InventoryMovement, int, error)
	List(ctx context.Context, db DBTX, movementType *string, page, pageSize int) ([]models.InventoryMovement, int, error)
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
		       im.created_by, u.username as created_by_name, im.created_at
		FROM inventory_movements im
		JOIN batches b ON im.batch_id = b.id
		JOIN products p ON b.product_id = p.id
		JOIN users u ON im.created_by = u.id
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
		       im.created_by, u.username as created_by_name, im.created_at
		FROM inventory_movements im
		JOIN batches b ON im.batch_id = b.id
		JOIN products p ON b.product_id = p.id
		JOIN users u ON im.created_by = u.id
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
