ALTER TABLE vehicle_maintenance_schedules
    ADD COLUMN IF NOT EXISTS maintenance_id UUID REFERENCES vehicle_maintenance(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_vms_maintenance ON vehicle_maintenance_schedules(maintenance_id)
    WHERE maintenance_id IS NOT NULL;
