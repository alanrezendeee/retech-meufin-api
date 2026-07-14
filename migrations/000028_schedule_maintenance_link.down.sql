DROP INDEX IF EXISTS idx_vms_maintenance;
ALTER TABLE vehicle_maintenance_schedules DROP COLUMN IF EXISTS maintenance_id;
