package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/models"
)

type BatchRepository interface {
	Create(ctx context.Context, db DBTX, b models.Batch) (models.Batch, error)
	Update(ctx context.Context, db DBTX, b models.Batch) error
	FindByID(ctx context.Context, db DBTX, id uuid.UUID) (models.Batch, error)
	LockAvailableBatches(ctx context.Context, db DBTX, productID uuid.UUID) ([]models.Batch, error)
	List(ctx context.Context, db DBTX, productID *uuid.UUID, page, pageSize int) ([]models.Batch, int, error)
	ListExpiring(ctx context.Context, db DBTX, daysThreshold int, page, pageSize int) ([]models.Batch, int, error)
}

type postgresBatchRepository struct{}

func NewBatchRepository() BatchRepository {
	return &postgresBatchRepository{}
}

func (r *postgresBatchRepository) Create(ctx context.Context, db DBTX, b models.Batch) (models.Batch, error) {
	query := `
		INSERT INTO batches (product_id, batch_number, quantity_received, quantity_remaining, expiry_date, cost_price, markup_percentage)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, product_id, batch_number, quantity_received, quantity_remaining, expiry_date, cost_price, markup_percentage, selling_price, received_at
	`
	var created models.Batch
	err := db.QueryRow(ctx, query,
		b.ProductID, b.BatchNumber, b.QuantityReceived, b.QuantityRemaining, b.ExpiryDate, b.CostPrice, b.MarkupPercentage,
	).Scan(
		&created.ID, &created.ProductID, &created.BatchNumber, &created.QuantityReceived, &created.QuantityRemaining,
		&created.ExpiryDate, &created.CostPrice, &created.MarkupPercentage, &created.SellingPrice, &created.ReceivedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Batch{}, apperrors.ErrDuplicateBatch
		}
		return models.Batch{}, fmt.Errorf("failed to create batch: %w", err)
	}

	return created, nil
}

func (r *postgresBatchRepository) Update(ctx context.Context, db DBTX, b models.Batch) error {
	query := `
		UPDATE batches
		SET quantity_remaining = $1
		WHERE id = $2
	`
	cmd, err := db.Exec(ctx, query, b.QuantityRemaining, b.ID)
	if err != nil {
		return fmt.Errorf("failed to update batch: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *postgresBatchRepository) FindByID(ctx context.Context, db DBTX, id uuid.UUID) (models.Batch, error) {
	query := `
		SELECT b.id, b.product_id, p.name as product_name, p.sku as product_sku, b.batch_number,
		       b.quantity_received, b.quantity_remaining, b.expiry_date, b.cost_price, b.markup_percentage,
		       b.selling_price, b.received_at
		FROM batches b
		JOIN products p ON b.product_id = p.id
		WHERE b.id = $1
	`
	var b models.Batch
	err := db.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.ProductID, &b.ProductName, &b.ProductSKU, &b.BatchNumber,
		&b.QuantityReceived, &b.QuantityRemaining, &b.ExpiryDate, &b.CostPrice, &b.MarkupPercentage,
		&b.SellingPrice, &b.ReceivedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Batch{}, apperrors.ErrNotFound
		}
		return models.Batch{}, fmt.Errorf("failed to find batch by id: %w", err)
	}

	return b, nil
}

func (r *postgresBatchRepository) LockAvailableBatches(ctx context.Context, db DBTX, productID uuid.UUID) ([]models.Batch, error) {
	query := `
		SELECT b.id, b.product_id, p.name as product_name, p.sku as product_sku, b.batch_number,
		       b.quantity_received, b.quantity_remaining, b.expiry_date, b.cost_price, b.markup_percentage,
		       b.selling_price, b.received_at
		FROM batches b
		JOIN products p ON b.product_id = p.id
		WHERE b.product_id = $1 AND b.quantity_remaining > 0 AND b.expiry_date > CURRENT_DATE
		ORDER BY b.expiry_date ASC
		FOR UPDATE OF b
	`
	rows, err := db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock available batches: %w", err)
	}
	defer rows.Close()

	var batches []models.Batch
	for rows.Next() {
		var b models.Batch
		if err := rows.Scan(
			&b.ID, &b.ProductID, &b.ProductName, &b.ProductSKU, &b.BatchNumber,
			&b.QuantityReceived, &b.QuantityRemaining, &b.ExpiryDate, &b.CostPrice, &b.MarkupPercentage,
			&b.SellingPrice, &b.ReceivedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan batch row: %w", err)
		}
		batches = append(batches, b)
	}

	return batches, nil
}

func (r *postgresBatchRepository) List(ctx context.Context, db DBTX, productID *uuid.UUID, page, pageSize int) ([]models.Batch, int, error) {
	offset := (page - 1) * pageSize

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if productID != nil {
		whereClause += fmt.Sprintf(" AND b.product_id = $%d", argIdx)
		args = append(args, *productID)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM batches b %s`, whereClause)
	var total int
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count batches: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT b.id, b.product_id, p.name as product_name, p.sku as product_sku, b.batch_number,
		       b.quantity_received, b.quantity_remaining, b.expiry_date, b.cost_price, b.markup_percentage,
		       b.selling_price, b.received_at
		FROM batches b
		JOIN products p ON b.product_id = p.id
		%s
		ORDER BY b.received_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list batches: %w", err)
	}
	defer rows.Close()

	var batches []models.Batch
	for rows.Next() {
		var b models.Batch
		if err := rows.Scan(
			&b.ID, &b.ProductID, &b.ProductName, &b.ProductSKU, &b.BatchNumber,
			&b.QuantityReceived, &b.QuantityRemaining, &b.ExpiryDate, &b.CostPrice, &b.MarkupPercentage,
			&b.SellingPrice, &b.ReceivedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan batch: %w", err)
		}
		batches = append(batches, b)
	}

	return batches, total, nil
}

func (r *postgresBatchRepository) ListExpiring(ctx context.Context, db DBTX, daysThreshold int, page, pageSize int) ([]models.Batch, int, error) {
	offset := (page - 1) * pageSize

	countQuery := `
		SELECT COUNT(*) FROM batches
		WHERE quantity_remaining > 0 AND expiry_date <= (CURRENT_DATE + ($1 || ' days')::INTERVAL)
	`
	var total int
	if err := db.QueryRow(ctx, countQuery, daysThreshold).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count expiring batches: %w", err)
	}

	query := `
		SELECT b.id, b.product_id, p.name as product_name, p.sku as product_sku, b.batch_number,
		       b.quantity_received, b.quantity_remaining, b.expiry_date, b.cost_price, b.markup_percentage,
		       b.selling_price, b.received_at
		FROM batches b
		JOIN products p ON b.product_id = p.id
		WHERE b.quantity_remaining > 0 AND b.expiry_date <= (CURRENT_DATE + ($1 || ' days')::INTERVAL)
		ORDER BY b.expiry_date ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.Query(ctx, query, daysThreshold, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list expiring batches: %w", err)
	}
	defer rows.Close()

	var batches []models.Batch
	for rows.Next() {
		var b models.Batch
		if err := rows.Scan(
			&b.ID, &b.ProductID, &b.ProductName, &b.ProductSKU, &b.BatchNumber,
			&b.QuantityReceived, &b.QuantityRemaining, &b.ExpiryDate, &b.CostPrice, &b.MarkupPercentage,
			&b.SellingPrice, &b.ReceivedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan expiring batch: %w", err)
		}
		batches = append(batches, b)
	}

	return batches, total, nil
}
