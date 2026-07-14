DROP TABLE IF EXISTS vehicle_maintenance_schedules;
DROP TABLE IF EXISTS vehicle_maintenance_items;

ALTER TABLE vehicle_maintenance
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS total_cents,
    DROP COLUMN IF EXISTS total_services_cents,
    DROP COLUMN IF EXISTS total_products_cents,
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS technician,
    DROP COLUMN IF EXISTS os_number,
    DROP COLUMN IF EXISTS status;

ALTER TABLE vehicle_maintenance ALTER COLUMN service_date SET NOT NULL;
