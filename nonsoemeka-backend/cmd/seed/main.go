package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"nonsoemeka-backend/internal/auth"
	"nonsoemeka-backend/internal/config"
	"nonsoemeka-backend/internal/database"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Seed failed: %v", err)
	}
	log.Println("Seed completed successfully.")
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer pool.Close()

	// Check if already seeded to prevent duplicate movements/audit logs
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE username = 'admin'`).Scan(&count)
	if err != nil {
		return fmt.Errorf("check existing admin: %w", err)
	}
	if count > 0 {
		log.Println("Database already seeded (admin user exists). Skipping seed script.")
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Users
	adminHash, err := auth.HashPassword(cfg.Seed.AdminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}
	staffHash, err := auth.HashPassword(cfg.Seed.StaffPassword)
	if err != nil {
		return fmt.Errorf("hash staff password: %w", err)
	}

	adminID := uuid.New()
	staffID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, username, email, password_hash, role, is_active)
		VALUES 
			($1, 'admin', 'admin@example.com', $2, 'ADMIN', true),
			($3, 'staff', 'staff@example.com', $4, 'STAFF', true)
		ON CONFLICT (username) DO NOTHING
	`, adminID, adminHash, staffID, staffHash)
	if err != nil {
		return fmt.Errorf("insert users: %w", err)
	}

	// Retrieve actual IDs in case they already existed
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE username = 'admin'`).Scan(&adminID)
	if err != nil {
		return fmt.Errorf("get admin id: %w", err)
	}
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE username = 'staff'`).Scan(&staffID)
	if err != nil {
		return fmt.Errorf("get staff id: %w", err)
	}

	// 2. Settings
	settingsQuery := `
		INSERT INTO settings (key, value)
		VALUES 
			('default_markup_percentage', '"30.00"'),
			('expiry_alert_days', '90'),
			('low_stock_threshold', '50'),
			('receipt_header', '"Nonsoemeka Pharmacy"'),
			('receipt_footer', '"Thank you for your business!"')
		ON CONFLICT (key) DO NOTHING
	`
	if _, err := tx.Exec(ctx, settingsQuery); err != nil {
		return fmt.Errorf("insert settings: %w", err)
	}

	// 3. Products
	product1ID := uuid.New()
	product2ID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO products (id, name, sku, description, is_active)
		VALUES 
			($1, 'Paracetamol 500mg', 'PRC-500', 'Pain reliever', true),
			($2, 'Amoxicillin 250mg', 'AMX-250', 'Antibiotic', true)
		ON CONFLICT (sku) DO NOTHING
	`, product1ID, product2ID)
	if err != nil {
		return fmt.Errorf("insert products: %w", err)
	}
	tx.QueryRow(ctx, `SELECT id FROM products WHERE sku = 'PRC-500'`).Scan(&product1ID)
	tx.QueryRow(ctx, `SELECT id FROM products WHERE sku = 'AMX-250'`).Scan(&product2ID)

	// 4. Batches
	batch1ID := uuid.New()
	batch2ID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO batches (id, product_id, batch_number, quantity_received, quantity_remaining, expiry_date, cost_price, markup_percentage)
		VALUES 
			($1, $2, 'BATCH-001', 1000, 950, CURRENT_DATE + INTERVAL '1 year', 5.00, 30.00),
			($3, $4, 'BATCH-002', 500, 500, CURRENT_DATE + INTERVAL '6 months', 12.00, 40.00)
		ON CONFLICT (product_id, batch_number) DO NOTHING
	`, batch1ID, product1ID, batch2ID, product2ID)
	if err != nil {
		return fmt.Errorf("insert batches: %w", err)
	}
	tx.QueryRow(ctx, `SELECT id FROM batches WHERE batch_number = 'BATCH-001'`).Scan(&batch1ID)
	tx.QueryRow(ctx, `SELECT id FROM batches WHERE batch_number = 'BATCH-002'`).Scan(&batch2ID)

	// 5. Movements & Audit logs
	// Seed one ADJUSTMENT and one EXPIRED_WRITE_OFF
	movement1ID := uuid.New()
	movement2ID := uuid.New()

	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_movements (id, batch_id, movement_type, quantity_delta, reason, created_by)
		VALUES 
			($1, $2, 'ADJUSTMENT', -50, 'Stock count discrepancy', $3),
			($4, $5, 'EXPIRED_WRITE_OFF', -10, 'Initial write off for testing', $6)
		ON CONFLICT (id) DO NOTHING
	`, movement1ID, batch1ID, adminID, movement2ID, batch2ID, adminID)
	if err != nil {
		return fmt.Errorf("insert movements: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (id, actor_id, action, target_table, target_id, metadata)
		VALUES 
			($1, $2, 'STOCK_ADJUSTED', 'batches', $3, '{"delta": -50, "reason": "Stock count discrepancy"}'),
			($4, $5, 'STOCK_WRITTEN_OFF', 'batches', $6, '{"reason": "Initial write off for testing"}')
		ON CONFLICT (id) DO NOTHING
	`, uuid.New(), adminID, batch1ID, uuid.New(), adminID, batch2ID)
	if err != nil {
		return fmt.Errorf("insert audit logs: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Println("Database seed completed successfully.")
	return nil
}
