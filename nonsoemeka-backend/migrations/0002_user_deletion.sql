-- Alter sales table to make staff_id nullable and set to NULL on delete
ALTER TABLE sales ALTER COLUMN staff_id DROP NOT NULL;
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_staff_id_fkey;
ALTER TABLE sales ADD CONSTRAINT sales_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES users(id) ON DELETE SET NULL;

-- Alter inventory_movements table to make created_by nullable and set to NULL on delete
ALTER TABLE inventory_movements ALTER COLUMN created_by DROP NOT NULL;
ALTER TABLE inventory_movements DROP CONSTRAINT IF EXISTS inventory_movements_created_by_fkey;
ALTER TABLE inventory_movements ADD CONSTRAINT inventory_movements_created_by_fkey FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

-- Alter settings table to set updated_by to NULL on delete
ALTER TABLE settings DROP CONSTRAINT IF EXISTS settings_updated_by_fkey;
ALTER TABLE settings ADD CONSTRAINT settings_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL;

-- Alter audit_logs table to make actor_id nullable and set to NULL on delete
ALTER TABLE audit_logs ALTER COLUMN actor_id DROP NOT NULL;
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_actor_id_fkey;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL;
