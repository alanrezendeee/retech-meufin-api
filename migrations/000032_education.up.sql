-- Módulo: Educação / Material Escolar
-- Gestão das listas de material por membro da família, da pré-escola à faculdade.
-- 100% idempotente: pode rodar múltiplas vezes sem erro.

-- Matrícula / ano letivo de um membro da família.
CREATE TABLE IF NOT EXISTS school_enrollments (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id       UUID         NOT NULL,
    member_id          UUID         NOT NULL REFERENCES health_family_members (id) ON DELETE RESTRICT,
    school_year        INT          NOT NULL,
    stage              VARCHAR(20)  NOT NULL,          -- bercario|infantil|fundamental1|fundamental2|medio|tecnico|pre_vestibular|superior|pos
    school_name        VARCHAR(255),
    grade              VARCHAR(60),                    -- ex "3º ano"
    shift              VARCHAR(20),                    -- manha|tarde|integral|noite|ead
    monthly_fee_cents  BIGINT       NOT NULL DEFAULT 0, -- mensalidade
    enrollment_fee_cents BIGINT     NOT NULL DEFAULT 0, -- matrícula
    notes              TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_school_enrollments_workspace ON school_enrollments (workspace_id);
CREATE INDEX IF NOT EXISTS idx_school_enrollments_member    ON school_enrollments (workspace_id, member_id);
CREATE INDEX IF NOT EXISTS idx_school_enrollments_year      ON school_enrollments (workspace_id, school_year);

-- Lista de material vinculada a uma matrícula.
CREATE TABLE IF NOT EXISTS school_supply_lists (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID         NOT NULL,
    enrollment_id UUID         NOT NULL REFERENCES school_enrollments (id) ON DELETE CASCADE,
    title         VARCHAR(255) NOT NULL,               -- ex "Lista de material 2026"
    status        VARCHAR(20)  NOT NULL DEFAULT 'planejada', -- planejada|em_compra|concluida
    notes         TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_school_supply_lists_workspace  ON school_supply_lists (workspace_id);
CREATE INDEX IF NOT EXISTS idx_school_supply_lists_enrollment ON school_supply_lists (enrollment_id);

-- Item de uma lista de material.
CREATE TABLE IF NOT EXISTS school_supply_items (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id          UUID          NOT NULL,
    list_id               UUID          NOT NULL REFERENCES school_supply_lists (id) ON DELETE CASCADE,
    name                  VARCHAR(255)  NOT NULL,
    category              VARCHAR(20)   NOT NULL DEFAULT 'outros', -- papelaria|livros|uniforme|mochila|eletronicos|arte|higiene|outros
    quantity              NUMERIC(10, 2) NOT NULL DEFAULT 1,
    reference_price_cents BIGINT        NOT NULL DEFAULT 0,        -- preço de referência / pesquisado
    purchased             BOOLEAN       NOT NULL DEFAULT false,
    paid_price_cents      BIGINT        NOT NULL DEFAULT 0,
    purchased_at          DATE,
    store                 VARCHAR(255),
    notes                 TEXT,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_school_supply_items_workspace ON school_supply_items (workspace_id);
CREATE INDEX IF NOT EXISTS idx_school_supply_items_list      ON school_supply_items (list_id);
