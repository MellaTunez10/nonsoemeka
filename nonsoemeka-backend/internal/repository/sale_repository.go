package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/models"
)

type SaleRepository interface {
	Create(ctx context.Context, db DBTX, sale models.Sale, items []models.SaleItem) (models.Sale, error)
	FindByIdempotencyKey(ctx context.Context, db DBTX, key string) (*models.Sale, error)
	FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*models.Sale, error)
	GetFinancialSummary(ctx context.Context, db DBTX, startDate, endDate string) (totalRevenue, totalCost decimal.Decimal, salesCount, itemsCount int, err error)
	GetSalesTrends(ctx context.Context, db DBTX, startDate, endDate string, page, pageSize int) ([]dto.SalesTrendItem, int, error)
	GetTopProducts(ctx context.Context, db DBTX, startDate, endDate string, page, pageSize int) ([]dto.TopProductItem, int, error)
}

type postgresSaleRepository struct{}

func NewSaleRepository() SaleRepository {
	return &postgresSaleRepository{}
}

func (r *postgresSaleRepository) Create(ctx context.Context, db DBTX, sale models.Sale, items []models.SaleItem) (models.Sale, error) {
	saleQuery := `
		INSERT INTO sales (staff_id, total_amount, idempotency_key)
		VALUES ($1, $2, $3)
		RETURNING id, staff_id, total_amount, idempotency_key, created_at
	`
	var createdSale models.Sale
	err := db.QueryRow(ctx, saleQuery, sale.StaffID, sale.TotalAmount, sale.IdempotencyKey).Scan(
		&createdSale.ID, &createdSale.StaffID, &createdSale.TotalAmount, &createdSale.IdempotencyKey, &createdSale.CreatedAt,
	)
	if err != nil {
		return models.Sale{}, fmt.Errorf("failed to insert sale: %w", err)
	}

	itemQuery := `
		INSERT INTO sale_items (sale_id, product_id, batch_id, quantity, unit_price)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	createdItems := make([]models.SaleItem, 0, len(items))
	for _, item := range items {
		item.SaleID = createdSale.ID
		var itemID uuid.UUID
		err := db.QueryRow(ctx, itemQuery, item.SaleID, item.ProductID, item.BatchID, item.Quantity, item.UnitPrice).Scan(&itemID)
		if err != nil {
			return models.Sale{}, fmt.Errorf("failed to insert sale item: %w", err)
		}
		item.ID = itemID
		item.TotalPrice = item.UnitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		createdItems = append(createdItems, item)
	}

	createdSale.Items = createdItems
	return createdSale, nil
}

func (r *postgresSaleRepository) FindByIdempotencyKey(ctx context.Context, db DBTX, key string) (*models.Sale, error) {
	query := `
		SELECT s.id, s.staff_id, u.username as staff_name, s.total_amount, s.idempotency_key, s.created_at
		FROM sales s
		JOIN users u ON s.staff_id = u.id
		WHERE s.idempotency_key = $1
	`
	var sale models.Sale
	err := db.QueryRow(ctx, query, key).Scan(
		&sale.ID, &sale.StaffID, &sale.StaffName, &sale.TotalAmount, &sale.IdempotencyKey, &sale.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query sale by idempotency key: %w", err)
	}

	itemsQuery := `
		SELECT si.id, si.sale_id, si.product_id, p.name as product_name,
		       si.batch_id, b.batch_number, si.quantity, si.unit_price
		FROM sale_items si
		JOIN products p ON si.product_id = p.id
		JOIN batches b ON si.batch_id = b.id
		WHERE si.sale_id = $1
	`
	rows, err := db.Query(ctx, itemsQuery, sale.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sale items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.SaleItem
		if err := rows.Scan(
			&item.ID, &item.SaleID, &item.ProductID, &item.ProductName,
			&item.BatchID, &item.BatchNumber, &item.Quantity, &item.UnitPrice,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sale item: %w", err)
		}
		item.TotalPrice = item.UnitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		sale.Items = append(sale.Items, item)
	}

	return &sale, nil
}

func (r *postgresSaleRepository) FindByID(ctx context.Context, db DBTX, id uuid.UUID) (*models.Sale, error) {
	query := `
		SELECT s.id, s.staff_id, u.username as staff_name, s.total_amount, s.idempotency_key, s.created_at
		FROM sales s
		JOIN users u ON s.staff_id = u.id
		WHERE s.id = $1
	`
	var sale models.Sale
	err := db.QueryRow(ctx, query, id).Scan(
		&sale.ID, &sale.StaffID, &sale.StaffName, &sale.TotalAmount, &sale.IdempotencyKey, &sale.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to query sale by id: %w", err)
	}

	itemsQuery := `
		SELECT si.id, si.sale_id, si.product_id, p.name as product_name,
		       si.batch_id, b.batch_number, si.quantity, si.unit_price
		FROM sale_items si
		JOIN products p ON si.product_id = p.id
		JOIN batches b ON si.batch_id = b.id
		WHERE si.sale_id = $1
	`
	rows, err := db.Query(ctx, itemsQuery, sale.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sale items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.SaleItem
		if err := rows.Scan(
			&item.ID, &item.SaleID, &item.ProductID, &item.ProductName,
			&item.BatchID, &item.BatchNumber, &item.Quantity, &item.UnitPrice,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sale item: %w", err)
		}
		item.TotalPrice = item.UnitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		sale.Items = append(sale.Items, item)
	}

	return &sale, nil
}

func (r *postgresSaleRepository) GetFinancialSummary(ctx context.Context, db DBTX, startDate, endDate string) (totalRevenue, totalCost decimal.Decimal, salesCount, itemsCount int, err error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if startDate != "" {
		whereClause += fmt.Sprintf(" AND s.created_at >= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		whereClause += fmt.Sprintf(" AND s.created_at <= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(si.quantity * si.unit_price), 0) as total_revenue,
			COALESCE(SUM(si.quantity * b.cost_price), 0) as total_cost,
			COUNT(DISTINCT s.id) as sales_count,
			COALESCE(SUM(si.quantity), 0) as items_count
		FROM sales s
		JOIN sale_items si ON s.id = si.sale_id
		JOIN batches b ON si.batch_id = b.id
		%s
	`, whereClause)

	err = db.QueryRow(ctx, query, args...).Scan(&totalRevenue, &totalCost, &salesCount, &itemsCount)
	if err != nil {
		return decimal.Zero, decimal.Zero, 0, 0, fmt.Errorf("failed to get financial summary: %w", err)
	}

	return totalRevenue, totalCost, salesCount, itemsCount, nil
}

func (r *postgresSaleRepository) GetSalesTrends(ctx context.Context, db DBTX, startDate, endDate string, page, pageSize int) ([]dto.SalesTrendItem, int, error) {
	offset := (page - 1) * pageSize
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if startDate != "" {
		whereClause += fmt.Sprintf(" AND created_at >= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		whereClause += fmt.Sprintf(" AND created_at <= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(DISTINCT DATE(created_at)) FROM sales %s`, whereClause)
	var total int
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count sales trends: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			TO_CHAR(DATE(created_at), 'YYYY-MM-DD') as sales_date,
			COALESCE(SUM(total_amount), 0) as total_amount,
			COUNT(id) as sales_count
		FROM sales
		%s
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at) DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sales trends: %w", err)
	}
	defer rows.Close()

	var trends []dto.SalesTrendItem
	for rows.Next() {
		var item dto.SalesTrendItem
		var amt decimal.Decimal
		if err := rows.Scan(&item.Date, &amt, &item.SalesCount); err != nil {
			return nil, 0, fmt.Errorf("failed to scan sales trend item: %w", err)
		}
		item.TotalAmount = amt.StringFixed(2)
		trends = append(trends, item)
	}

	return trends, total, nil
}

func (r *postgresSaleRepository) GetTopProducts(ctx context.Context, db DBTX, startDate, endDate string, page, pageSize int) ([]dto.TopProductItem, int, error) {
	offset := (page - 1) * pageSize
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if startDate != "" {
		whereClause += fmt.Sprintf(" AND s.created_at >= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate != "" {
		whereClause += fmt.Sprintf(" AND s.created_at <= $%d::TIMESTAMPTZ", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT si.product_id) 
		FROM sale_items si
		JOIN sales s ON si.sale_id = s.id
		%s
	`, whereClause)
	var total int
	if err := db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count top products: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			p.id as product_id,
			p.name as product_name,
			p.sku,
			COALESCE(SUM(si.quantity), 0) as total_quantity,
			COALESCE(SUM(si.quantity * si.unit_price), 0) as total_revenue
		FROM sale_items si
		JOIN sales s ON si.sale_id = s.id
		JOIN products p ON si.product_id = p.id
		%s
		GROUP BY p.id, p.name, p.sku
		ORDER BY total_quantity DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, pageSize, offset)

	rows, err := db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query top products: %w", err)
	}
	defer rows.Close()

	var topProducts []dto.TopProductItem
	for rows.Next() {
		var item dto.TopProductItem
		var rev decimal.Decimal
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.SKU, &item.TotalQuantity, &rev); err != nil {
			return nil, 0, fmt.Errorf("failed to scan top product item: %w", err)
		}
		item.TotalRevenue = rev.StringFixed(2)
		topProducts = append(topProducts, item)
	}

	return topProducts, total, nil
}
