package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/models"
)

type ProductRepository interface {
	Create(ctx context.Context, db DBTX, p models.Product) (models.Product, error)
	FindByID(ctx context.Context, db DBTX, id uuid.UUID) (models.Product, error)
	FindBySKU(ctx context.Context, db DBTX, sku string) (models.Product, error)
	List(ctx context.Context, db DBTX, search string, activeOnly bool, page, pageSize int) ([]models.Product, int, error)
}

type postgresProductRepository struct{}

func NewProductRepository() ProductRepository {
	return &postgresProductRepository{}
}

func (r *postgresProductRepository) Create(ctx context.Context, db DBTX, p models.Product) (models.Product, error) {
	query := `
		INSERT INTO products (name, sku, description, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, sku, description, is_active, created_at, updated_at
	`
	var created models.Product
	err := db.QueryRow(ctx, query, p.Name, p.SKU, p.Description, p.IsActive).Scan(
		&created.ID, &created.Name, &created.SKU, &created.Description, &created.IsActive, &created.CreatedAt, &created.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Product{}, apperrors.ErrDuplicateSKU
		}
		return models.Product{}, fmt.Errorf("failed to create product: %w", err)
	}

	return created, nil
}

func (r *postgresProductRepository) FindByID(ctx context.Context, db DBTX, id uuid.UUID) (models.Product, error) {
	query := `
		SELECT id, name, sku, description, is_active, created_at, updated_at
		FROM products WHERE id = $1
	`
	var p models.Product
	err := db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.SKU, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, apperrors.ErrNotFound
		}
		return models.Product{}, fmt.Errorf("failed to find product by id: %w", err)
	}

	return p, nil
}

func (r *postgresProductRepository) FindBySKU(ctx context.Context, db DBTX, sku string) (models.Product, error) {
	query := `
		SELECT id, name, sku, description, is_active, created_at, updated_at
		FROM products WHERE sku = $1
	`
	var p models.Product
	err := db.QueryRow(ctx, query, sku).Scan(
		&p.ID, &p.Name, &p.SKU, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Product{}, apperrors.ErrNotFound
		}
		return models.Product{}, fmt.Errorf("failed to find product by sku: %w", err)
	}

	return p, nil
}

func (r *postgresProductRepository) List(ctx context.Context, db DBTX, search string, activeOnly bool, page, pageSize int) ([]models.Product, int, error) {
	offset := (page - 1) * pageSize

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if activeOnly {
		whereClause += fmt.Sprintf(" AND p.is_active = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if search != "" {
		whereClause += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.sku ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM products p %s`, whereClause)
	var total int
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			p.id, p.name, p.sku, p.description, p.is_active, p.created_at, p.updated_at,
			COALESCE(SUM(b.quantity_remaining) FILTER (WHERE b.expiry_date > CURRENT_DATE), 0) as total_quantity,
			MIN(b.selling_price) FILTER (WHERE b.quantity_remaining > 0 AND b.expiry_date > CURRENT_DATE) as min_selling_price
		FROM products p
		LEFT JOIN batches b ON p.id = b.product_id
		%s
		GROUP BY p.id
		ORDER BY p.name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		var totalQty int
		var sellingPrice *decimal.Decimal

		if err := rows.Scan(
			&p.ID, &p.Name, &p.SKU, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
			&totalQty, &sellingPrice,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan product row: %w", err)
		}
		p.TotalQuantity = totalQty
		p.SellingPrice = sellingPrice
		products = append(products, p)
	}

	return products, total, nil
}
