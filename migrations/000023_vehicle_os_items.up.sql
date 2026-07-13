-- Itens discriminados de cada OS (produtos instalados e serviços executados)
CREATE TABLE IF NOT EXISTS vehicle_service_order_items (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_order_id            UUID NOT NULL REFERENCES vehicle_service_orders(id) ON DELETE CASCADE,
    vehicle_id                  UUID NOT NULL,
    workspace_id                UUID NOT NULL,
    item_type                   TEXT NOT NULL CHECK (item_type IN ('product','service')),
    category                    TEXT NOT NULL DEFAULT 'outros',
    description                 TEXT NOT NULL,
    quantity                    NUMERIC(10,3) NOT NULL DEFAULT 1,
    unit_price_cents            BIGINT NOT NULL DEFAULT 0,
    total_price_cents           BIGINT NOT NULL DEFAULT 0,
    -- produto instalado
    km_at_installation          INT,
    replacement_interval_km     INT,
    replacement_interval_months INT,
    next_due_km                 INT GENERATED ALWAYS AS (
        CASE WHEN km_at_installation IS NOT NULL AND replacement_interval_km IS NOT NULL
             THEN km_at_installation + replacement_interval_km
             ELSE NULL END
    ) STORED,
    next_due_date               DATE,   -- calculado pela app (service_date + interval_months)
    warranty_expires_date       DATE,
    warranty_expires_km         INT,
    notes                       TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vsoi_order     ON vehicle_service_order_items(service_order_id);
CREATE INDEX idx_vsoi_vehicle   ON vehicle_service_order_items(vehicle_id);
CREATE INDEX idx_vsoi_workspace ON vehicle_service_order_items(workspace_id);
CREATE INDEX idx_vsoi_next_km   ON vehicle_service_order_items(next_due_km) WHERE next_due_km IS NOT NULL;
CREATE INDEX idx_vsoi_warranty  ON vehicle_service_order_items(warranty_expires_date) WHERE warranty_expires_date IS NOT NULL;
