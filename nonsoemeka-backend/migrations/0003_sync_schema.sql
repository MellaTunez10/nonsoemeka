-- Migration: Add sync tracking columns for hybrid local/cloud synchronization.
-- Applied to: products, batches, settings, sales, inventory_movements, audit_logs, users.
-- NOT applied to: sale_items (inherits parent sale's sync fate), refresh_tokens (not synced).

-- 1. Create the sync status enum
DO $$ BEGIN
    CREATE TYPE sync_status_enum AS ENUM ('PENDING', 'SYNCED', 'FAILED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- 2. Add sync columns and partial indexes to each table

-- products
ALTER TABLE products
  ADD COLUMN IF NOT EXISTS sync_status sync_status_enum NOT NULL DEFAULT 'PENDING',
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_products_sync_pending ON products (created_at ASC) WHERE sync_status = 'PENDING';

-- batches
ALTER TABLE batches
  ADD COLUMN IF NOT EXISTS sync_status sync_status_enum NOT NULL DEFAULT 'PENDING',
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_batches_sync_pending ON batches (received_at ASC) WHERE sync_status = 'PENDING';

-- settings (uses updated_at for the partial index since settings are upserted)
ALTER TABLE settings
  ADD COLUMN IF NOT EXISTS sync_status sync_status_enum NOT NULL DEFAULT 'PENDING',
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_settings_sync_pending ON settings (updated_at ASC) WHERE sync_status = 'PENDING';

-- sales
ALTER TABLE sales
  ADD COLUMN IF NOT EXISTS sync_status sync_status_enum NOT NULL DEFAULT 'PENDING',
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_sales_sync_pending ON sales (created_at ASC) WHERE sync_status = 'PENDING';

-- inventory_movements
ALTER TABLE inventory_movements
  ADD COLUMN IF NOT EXISTS sync_status sync_status_enum NOT NULL DEFAULT 'PENDING',
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS sync_failure_reason TEXT;
CREATE INDEX IF NOT EXISTS idx_inventory_movements_sync_pending ON inventory_movements (created_at ASC) WHERE sync_status = 'PENDING';

-- audit_logs
ALTER TABLE audit_logs
  ADD COLUMN IF NOT EXISTS sync_status sync_status_enum NOT NULL DEFAULT 'PENDING',
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_audit_logs_sync_pending ON audit_logs (created_at ASC) WHERE sync_status = 'PENDING';

-- users
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS sync_status sync_status_enum NOT NULL DEFAULT 'PENDING',
  ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_sync_pending ON users (created_at ASC) WHERE sync_status = 'PENDING';
