package sync

import (
	"context"
	"fmt"

	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/models"
)

// DBTX matches the repository-layer convention: works with both pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// SyncTrackingRepository handles the cross-cutting sync-status bookkeeping
// that no single domain repository owns. It reads PENDING rows for outbound push
// and marks them SYNCED or FAILED after processing.
type SyncTrackingRepository interface {
	FetchPendingProducts(ctx context.Context, db DBTX, limit int) ([]SyncProduct, error)
	FetchPendingBatches(ctx context.Context, db DBTX, limit int) ([]SyncBatch, error)
	FetchPendingSettings(ctx context.Context, db DBTX, limit int) ([]models.Setting, error)
	FetchPendingSales(ctx context.Context, db DBTX, limit int) ([]SalePushItem, error)
	FetchPendingMovements(ctx context.Context, db DBTX, limit int) ([]SyncMovement, error)
	FetchPendingAuditLogs(ctx context.Context, db DBTX, limit int) ([]SyncAuditLog, error)
	FetchPendingUserStates(ctx context.Context, db DBTX, limit int) ([]UserSecurityState, error)

	MarkSynced(ctx context.Context, db DBTX, table string, ids []uuid.UUID) error
	MarkSettingsSynced(ctx context.Context, db DBTX, keys []string) error
	MarkFailed(ctx context.Context, db DBTX, table string, ids []uuid.UUID, reason string) error

	ListFailedMovements(ctx context.Context, db DBTX, offset, limit int) ([]models.InventoryMovement, int, error)
}

type syncTrackingRepository struct{}

func NewSyncTrackingRepository() SyncTrackingRepository {
	return &syncTrackingRepository{}
}

// allowedSyncTables prevents SQL injection in the dynamic table name used by MarkSynced/MarkFailed.
var allowedSyncTables = map[string]bool{
	"products":            true,
	"batches":             true,
	"settings":            true,
	"sales":               true,
	"inventory_movements": true,
	"audit_logs":          true,
	"users":               true,
}

func (r *syncTrackingRepository) FetchPendingProducts(ctx context.Context, db DBTX, limit int) ([]SyncProduct, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, sku, description, is_active, created_at, updated_at
		FROM products
		WHERE sync_status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []SyncProduct
	for rows.Next() {
		var p SyncProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Description, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *syncTrackingRepository) FetchPendingBatches(ctx context.Context, db DBTX, limit int) ([]SyncBatch, error) {
	rows, err := db.Query(ctx, `
		SELECT id, product_id, batch_number, quantity_received, quantity_remaining,
		       expiry_date, cost_price, markup_percentage, received_at
		FROM batches
		WHERE sync_status = 'PENDING'
		ORDER BY received_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []SyncBatch
	for rows.Next() {
		var b SyncBatch
		if err := rows.Scan(&b.ID, &b.ProductID, &b.BatchNumber, &b.QuantityReceived, &b.QuantityRemaining,
			&b.ExpiryDate, &b.CostPrice, &b.MarkupPercentage, &b.ReceivedAt); err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

func (r *syncTrackingRepository) FetchPendingSettings(ctx context.Context, db DBTX, limit int) ([]models.Setting, error) {
	rows, err := db.Query(ctx, `
		SELECT key, value, updated_by, updated_at
		FROM settings
		WHERE sync_status = 'PENDING'
		ORDER BY updated_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []models.Setting
	for rows.Next() {
		var s models.Setting
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedBy, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}

func (r *syncTrackingRepository) FetchPendingSales(ctx context.Context, db DBTX, limit int) ([]SalePushItem, error) {
	// Fetch pending sales
	saleRows, err := db.Query(ctx, `
		SELECT id, staff_id, total_amount, idempotency_key, created_at
		FROM sales
		WHERE sync_status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer saleRows.Close()

	var sales []SalePushItem
	var saleIDs []uuid.UUID
	saleIndexMap := make(map[uuid.UUID]int) // maps sale ID to index in sales slice

	for saleRows.Next() {
		var s SyncSale
		if err := saleRows.Scan(&s.ID, &s.StaffID, &s.TotalAmount, &s.IdempotencyKey, &s.CreatedAt); err != nil {
			return nil, err
		}
		saleIndexMap[s.ID] = len(sales)
		saleIDs = append(saleIDs, s.ID)
		sales = append(sales, SalePushItem{Sale: s})
	}
	if err := saleRows.Err(); err != nil {
		return nil, err
	}
	if len(saleIDs) == 0 {
		return nil, nil
	}

	// Fetch all sale_items for these sales in one query
	itemRows, err := db.Query(ctx, `
		SELECT id, sale_id, product_id, batch_id, quantity, unit_price
		FROM sale_items
		WHERE sale_id = ANY($1)
		ORDER BY sale_id, id
	`, saleIDs)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item SyncSaleItem
		if err := itemRows.Scan(&item.ID, &item.SaleID, &item.ProductID, &item.BatchID, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, err
		}
		idx := saleIndexMap[item.SaleID]
		sales[idx].Items = append(sales[idx].Items, item)
	}
	return sales, itemRows.Err()
}

func (r *syncTrackingRepository) FetchPendingMovements(ctx context.Context, db DBTX, limit int) ([]SyncMovement, error) {
	rows, err := db.Query(ctx, `
		SELECT id, batch_id, movement_type, quantity_delta, reference_id, reason, created_by, created_at
		FROM inventory_movements
		WHERE sync_status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movements []SyncMovement
	for rows.Next() {
		var m SyncMovement
		if err := rows.Scan(&m.ID, &m.BatchID, &m.MovementType, &m.QuantityDelta, &m.ReferenceID, &m.Reason, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

func (r *syncTrackingRepository) FetchPendingAuditLogs(ctx context.Context, db DBTX, limit int) ([]SyncAuditLog, error) {
	rows, err := db.Query(ctx, `
		SELECT id, actor_id, action, target_table, target_id, metadata, created_at
		FROM audit_logs
		WHERE sync_status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []SyncAuditLog
	for rows.Next() {
		var a SyncAuditLog
		if err := rows.Scan(&a.ID, &a.ActorID, &a.Action, &a.TargetTable, &a.TargetID, &a.Metadata, &a.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, a)
	}
	return logs, rows.Err()
}

func (r *syncTrackingRepository) FetchPendingUserStates(ctx context.Context, db DBTX, limit int) ([]UserSecurityState, error) {
	rows, err := db.Query(ctx, `
		SELECT id, is_active, locked_until, updated_at
		FROM users
		WHERE sync_status = 'PENDING'
		ORDER BY updated_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []UserSecurityState
	for rows.Next() {
		var s UserSecurityState
		if err := rows.Scan(&s.ID, &s.IsActive, &s.LockedUntil, &s.UpdatedAt); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}

func (r *syncTrackingRepository) MarkSynced(ctx context.Context, db DBTX, table string, ids []uuid.UUID) error {
	if !allowedSyncTables[table] {
		return fmt.Errorf("invalid sync table: %s", table)
	}
	if len(ids) == 0 {
		return nil
	}

	query := fmt.Sprintf(`UPDATE %s SET sync_status = 'SYNCED', synced_at = NOW() WHERE id = ANY($1)`, table)
	_, err := db.Exec(ctx, query, ids)
	return err
}

func (r *syncTrackingRepository) MarkSettingsSynced(ctx context.Context, db DBTX, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	query := `UPDATE settings SET sync_status = 'SYNCED', synced_at = NOW() WHERE key = ANY($1)`
	_, err := db.Exec(ctx, query, keys)
	return err
}

func (r *syncTrackingRepository) MarkFailed(ctx context.Context, db DBTX, table string, ids []uuid.UUID, reason string) error {
	if !allowedSyncTables[table] {
		return fmt.Errorf("invalid sync table: %s", table)
	}
	if len(ids) == 0 {
		return nil
	}

	query := fmt.Sprintf(`UPDATE %s SET sync_status = 'FAILED', synced_at = NOW() WHERE id = ANY($1)`, table)
	_, err := db.Exec(ctx, query, ids)
	return err
}

func (r *syncTrackingRepository) ListFailedMovements(ctx context.Context, db DBTX, offset, limit int) ([]models.InventoryMovement, int, error) {
	query := `
		SELECT 
			id, batch_id, movement_type, quantity_delta, reference_id, reason, created_by, created_at,
			sync_status, synced_at, sync_failure_reason
		FROM inventory_movements
		WHERE sync_status = 'FAILED'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var movements []models.InventoryMovement
	for rows.Next() {
		var m models.InventoryMovement
		err := rows.Scan(
			&m.ID, &m.BatchID, &m.MovementType, &m.QuantityDelta, &m.ReferenceID, &m.Reason, &m.CreatedBy, &m.CreatedAt,
			&m.SyncStatus, &m.SyncedAt, &m.SyncFailureReason,
		)
		if err != nil {
			return nil, 0, err
		}
		movements = append(movements, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	err = db.QueryRow(ctx, `SELECT COUNT(*) FROM inventory_movements WHERE sync_status = 'FAILED'`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return movements, total, nil
}

// Ensure unused imports don't cause build failures — these types are used in the interface.
var (
	_ json.RawMessage
	_ decimal.Decimal
)
