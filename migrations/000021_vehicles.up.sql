-- Módulo: Frota Familiar
-- Planos globais de manutenção (seeds do sistema, scope = 'system').

CREATE TABLE IF NOT EXISTS maintenance_plan_templates (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type                 VARCHAR(50) NOT NULL UNIQUE,
    name                 VARCHAR(150) NOT NULL,
    default_interval_km  INT,
    default_interval_days INT,
    scope                VARCHAR(20) NOT NULL DEFAULT 'system'
);

INSERT INTO maintenance_plan_templates (type, name, default_interval_km, default_interval_days) VALUES
    ('oil_change',        'Troca de óleo',                5000,  180),
    ('preventive_6m',     'Revisão preventiva (6 meses)', NULL,  180),
    ('preventive_1y',     'Revisão preventiva (1 ano)',   NULL,  365),
    ('preventive_2y',     'Revisão preventiva (2 anos)',  NULL,  730),
    ('tire_rotation',     'Rodízio de pneus',            10000,  180),
    ('alignment_balance', 'Alinhamento e balanceamento', 10000,  365),
    ('air_filter',        'Filtro de ar',                15000,  365),
    ('cabin_filter',      'Filtro de cabine',            15000,  365),
    ('brake_fluid',       'Fluido de freio',              NULL,  730),
    ('coolant',           'Fluido de arrefecimento',      NULL,  730),
    ('power_steering',    'Fluido de direção hidráulica', NULL,  730),
    ('timing_belt',       'Correia dentada',             60000, 1825),
    ('spark_plugs',       'Velas de ignição',            20000, 1095),
    ('brake_pads',        'Pastilhas de freio',          30000,  NULL)
ON CONFLICT (type) DO NOTHING;

-- Cadastro de veículos da frota familiar.

CREATE TABLE IF NOT EXISTS vehicles (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      UUID        NOT NULL,
    nickname          VARCHAR(100),
    make              VARCHAR(80)  NOT NULL,
    model             VARCHAR(120) NOT NULL,
    year_manufacture  INT          NOT NULL,
    year_model        INT          NOT NULL,
    color             VARCHAR(40),
    plate             VARCHAR(10),
    fuel_type         VARCHAR(20)  NOT NULL,
    fipe_vehicle_type VARCHAR(20)  NOT NULL DEFAULT 'carros',
    fipe_code         VARCHAR(20),
    fipe_brand_code   VARCHAR(20),
    fipe_model_code   VARCHAR(20),
    fipe_year_code    VARCHAR(20),
    acquisition_date  DATE,
    acquisition_price NUMERIC(14, 2),
    current_odometer  INT          NOT NULL DEFAULT 0,
    status            VARCHAR(20)  NOT NULL DEFAULT 'active',
    sold_at           DATE,
    sold_price        NUMERIC(14, 2),
    notes             TEXT,
    created_at        TIMESTAMP    NOT NULL DEFAULT now(),
    updated_at        TIMESTAMP    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vehicles_workspace ON vehicles (workspace_id);
CREATE INDEX IF NOT EXISTS idx_vehicles_status    ON vehicles (workspace_id, status);

-- Membros familiares vinculados a um veículo (N:N — carro compartilhado).

CREATE TABLE IF NOT EXISTS vehicle_members (
    vehicle_id UUID NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    member_id  UUID NOT NULL,
    role       VARCHAR(20) NOT NULL DEFAULT 'driver',
    PRIMARY KEY (vehicle_id, member_id)
);

-- Customizações por veículo dos planos globais de manutenção.

CREATE TABLE IF NOT EXISTS vehicle_maintenance_plans (
    id            UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id    UUID      NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    workspace_id  UUID      NOT NULL,
    template_id   UUID      NOT NULL REFERENCES maintenance_plan_templates (id),
    interval_km   INT,
    interval_days INT,
    enabled       BOOLEAN   NOT NULL DEFAULT true,
    updated_at    TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE (vehicle_id, template_id)
);

-- Registros de manutenções executadas.

CREATE TABLE IF NOT EXISTS vehicle_maintenance (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id            UUID         NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    workspace_id          UUID         NOT NULL,
    template_id           UUID         REFERENCES maintenance_plan_templates (id),
    type                  VARCHAR(50)  NOT NULL,
    title                 VARCHAR(150) NOT NULL,
    description           TEXT,
    odometer_at_service   INT,
    service_date          DATE         NOT NULL,
    cost                  NUMERIC(14, 2),
    supplier_id           UUID,
    next_service_odometer INT,
    next_service_date     DATE,
    notes                 TEXT,
    created_at            TIMESTAMP    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vehicle_maintenance_vehicle ON vehicle_maintenance (vehicle_id);
CREATE INDEX IF NOT EXISTS idx_vehicle_maintenance_type    ON vehicle_maintenance (vehicle_id, type);

-- Histórico mensal do valor FIPE por veículo (atualizado pela cron mensal).

CREATE TABLE IF NOT EXISTS vehicle_fipe_history (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id      UUID          NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    workspace_id    UUID          NOT NULL,
    reference_month VARCHAR(10)   NOT NULL,   -- "07/2026"
    fipe_value      NUMERIC(14, 2) NOT NULL,
    fipe_fuel       VARCHAR(40),
    recorded_at     TIMESTAMP     NOT NULL DEFAULT now(),
    UNIQUE (vehicle_id, reference_month)
);

CREATE INDEX IF NOT EXISTS idx_vehicle_fipe_history_vehicle ON vehicle_fipe_history (vehicle_id);
