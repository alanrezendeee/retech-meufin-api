-- Agendamentos de manutenção — derivados de itens de OS ou criados manualmente
CREATE TABLE IF NOT EXISTS vehicle_maintenance_schedules (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id            UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    workspace_id          UUID NOT NULL,
    service_order_item_id UUID REFERENCES vehicle_service_order_items(id) ON DELETE SET NULL,
    description           TEXT NOT NULL,
    category              TEXT NOT NULL DEFAULT 'outros',
    scheduled_km          INT,
    scheduled_date        DATE,
    alert_status          TEXT NOT NULL DEFAULT 'pending' CHECK (alert_status IN ('pending','due_soon','overdue','done','cancelled')),
    completed_at          DATE,
    notes                 TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vms_vehicle   ON vehicle_maintenance_schedules(vehicle_id);
CREATE INDEX idx_vms_workspace ON vehicle_maintenance_schedules(workspace_id);
CREATE INDEX idx_vms_status    ON vehicle_maintenance_schedules(alert_status);
CREATE INDEX idx_vms_km        ON vehicle_maintenance_schedules(scheduled_km) WHERE scheduled_km IS NOT NULL;
CREATE INDEX idx_vms_date      ON vehicle_maintenance_schedules(scheduled_date) WHERE scheduled_date IS NOT NULL;
