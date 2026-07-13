-- Ordens de serviço de veículos (OS de auto centers, mecânicas, etc.)
CREATE TABLE IF NOT EXISTS vehicle_service_orders (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id            UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    workspace_id          UUID NOT NULL,
    supplier_id           UUID,
    os_number             TEXT,
    service_date          DATE NOT NULL,
    km_at_service         INT NOT NULL DEFAULT 0,
    total_products_cents  BIGINT NOT NULL DEFAULT 0,
    total_services_cents  BIGINT NOT NULL DEFAULT 0,
    total_cents           BIGINT NOT NULL DEFAULT 0,
    payment_method        TEXT,
    technician            TEXT,
    notes                 TEXT,
    status                TEXT NOT NULL DEFAULT 'completed' CHECK (status IN ('draft','completed','cancelled')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vso_vehicle    ON vehicle_service_orders(vehicle_id);
CREATE INDEX idx_vso_workspace  ON vehicle_service_orders(workspace_id);
CREATE INDEX idx_vso_supplier   ON vehicle_service_orders(supplier_id) WHERE supplier_id IS NOT NULL;
CREATE INDEX idx_vso_date       ON vehicle_service_orders(service_date DESC);
