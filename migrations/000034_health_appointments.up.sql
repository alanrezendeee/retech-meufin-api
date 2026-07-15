-- Módulo: Saúde Familiar — Consultas & Agenda + Planos de Saúde.
-- Análogo à manutenção de veículos, porém para saúde: agenda de consultas,
-- exames, retornos, vacinas etc., e o cadastro dos planos de saúde da família.
-- 100% idempotente: CREATE TABLE/INDEX IF NOT EXISTS e ADD COLUMN IF NOT EXISTS.

-- 1) health_labs ganha "kind" (tipo de local de saúde).
ALTER TABLE health_labs
    ADD COLUMN IF NOT EXISTS kind VARCHAR(20) NOT NULL DEFAULT 'laboratorio';

-- 2) Planos de saúde da família.
CREATE TABLE IF NOT EXISTS health_plans (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      UUID         NOT NULL,
    name              VARCHAR(255) NOT NULL,
    operator          VARCHAR(120),
    plan_type         VARCHAR(20)  NOT NULL DEFAULT 'familiar',
    ans_code          VARCHAR(30),
    monthly_fee_cents BIGINT       NOT NULL DEFAULT 0,
    coverage_notes    TEXT,
    active            BOOLEAN      NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_health_plans_workspace ON health_plans (workspace_id);

-- 3) Membros cobertos por um plano de saúde.
CREATE TABLE IF NOT EXISTS health_plan_members (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID       NOT NULL,
    plan_id     UUID        NOT NULL REFERENCES health_plans (id) ON DELETE CASCADE,
    member_id   UUID        NOT NULL REFERENCES health_family_members (id),
    card_number VARCHAR(60),
    holder      BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_health_plan_members_unique   ON health_plan_members (plan_id, member_id);
CREATE INDEX        IF NOT EXISTS idx_health_plan_members_workspace ON health_plan_members (workspace_id);
CREATE INDEX        IF NOT EXISTS idx_health_plan_members_member    ON health_plan_members (member_id);

-- 4) Consultas / agenda de saúde.
CREATE TABLE IF NOT EXISTS health_appointments (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID         NOT NULL,
    family_member_id UUID         NOT NULL REFERENCES health_family_members (id),
    kind             VARCHAR(20)  NOT NULL DEFAULT 'consulta',
    specialty        VARCHAR(30),
    professional_name VARCHAR(255),
    lab_id           UUID         REFERENCES health_labs (id),
    exam_request_id  UUID         REFERENCES health_exam_requests (id),
    plan_id          UUID         REFERENCES health_plans (id),
    scheduled_at     TIMESTAMPTZ  NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'agendada',
    reason           TEXT,
    outcome          TEXT,
    price_cents      BIGINT       NOT NULL DEFAULT 0,
    covered_by_plan  BOOLEAN      NOT NULL DEFAULT false,
    notes            TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_health_appointments_workspace    ON health_appointments (workspace_id);
CREATE INDEX IF NOT EXISTS idx_health_appointments_scheduled_at ON health_appointments (workspace_id, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_health_appointments_member       ON health_appointments (workspace_id, family_member_id);
CREATE INDEX IF NOT EXISTS idx_health_appointments_status       ON health_appointments (workspace_id, status);
