-- ─── 1. Remover tabelas de OS (sem dados relevantes) ──────────────────────────
DROP TABLE IF EXISTS vehicle_service_order_items;
DROP TABLE IF EXISTS vehicle_service_orders;

-- ─── 2. Evoluir vehicle_maintenance ───────────────────────────────────────────
ALTER TABLE vehicle_maintenance
    ALTER COLUMN service_date DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS status              VARCHAR(20)  NOT NULL DEFAULT 'realizado',
    ADD COLUMN IF NOT EXISTS os_number          VARCHAR(50),
    ADD COLUMN IF NOT EXISTS technician         VARCHAR(100),
    ADD COLUMN IF NOT EXISTS payment_method     VARCHAR(30),
    ADD COLUMN IF NOT EXISTS total_products_cents BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_services_cents BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_cents          BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at         TIMESTAMP NOT NULL DEFAULT now();

-- Migrar cost existente → total_cents (registros antigos)
UPDATE vehicle_maintenance
SET total_cents = ROUND(cost * 100)::BIGINT
WHERE cost IS NOT NULL AND total_cents = 0;

-- ─── 3. Itens de manutenção ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS vehicle_maintenance_items (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    maintenance_id              UUID NOT NULL REFERENCES vehicle_maintenance(id) ON DELETE CASCADE,
    vehicle_id                  UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    workspace_id                UUID NOT NULL,
    catalog_item_id             UUID REFERENCES maintenance_catalog_items(id) ON DELETE SET NULL,
    item_type                   VARCHAR(10)   NOT NULL DEFAULT 'service',
    category                    VARCHAR(30)   NOT NULL DEFAULT 'outros',
    description                 TEXT          NOT NULL,
    quantity                    NUMERIC(10,3) NOT NULL DEFAULT 1,
    unit_price_cents            BIGINT        NOT NULL DEFAULT 0,
    total_price_cents           BIGINT        NOT NULL DEFAULT 0,
    km_at_installation          INT,
    replacement_interval_km     INT,
    replacement_interval_months INT,
    next_due_km                 INT GENERATED ALWAYS AS (
                                    CASE
                                        WHEN km_at_installation IS NOT NULL AND replacement_interval_km IS NOT NULL
                                        THEN km_at_installation + replacement_interval_km
                                        ELSE NULL
                                    END
                                ) STORED,
    next_due_date               DATE,
    warranty_expires_date       DATE,
    warranty_expires_km         INT,
    notes                       TEXT,
    created_at                  TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vmi_maintenance ON vehicle_maintenance_items(maintenance_id);
CREATE INDEX IF NOT EXISTS idx_vmi_catalog ON vehicle_maintenance_items(catalog_item_id)
    WHERE catalog_item_id IS NOT NULL;

-- ─── 4. Recriar vehicle_maintenance_schedules com FK correta ──────────────────
DROP TABLE IF EXISTS vehicle_maintenance_schedules;

CREATE TABLE vehicle_maintenance_schedules (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id            UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    workspace_id          UUID NOT NULL,
    maintenance_item_id   UUID REFERENCES vehicle_maintenance_items(id) ON DELETE SET NULL,
    description           TEXT          NOT NULL,
    category              VARCHAR(30)   NOT NULL DEFAULT 'outros',
    scheduled_km          INT,
    scheduled_date        DATE,
    alert_status          VARCHAR(20)   NOT NULL DEFAULT 'pending',
    completed_at          DATE,
    notes                 TEXT,
    created_at            TIMESTAMP NOT NULL DEFAULT now(),
    updated_at            TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vms_vehicle ON vehicle_maintenance_schedules(vehicle_id);
