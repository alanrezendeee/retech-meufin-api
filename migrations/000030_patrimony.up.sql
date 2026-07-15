-- Módulo: Patrimônio (imóveis + impostos de bens da família).
-- 100% idempotente: pode rodar múltiplas vezes sem erro.

-- Cadastro de imóveis da família.
CREATE TABLE IF NOT EXISTS properties (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID          NOT NULL,
    name                VARCHAR(150)  NOT NULL,
    property_type       VARCHAR(20)   NOT NULL DEFAULT 'casa',
    address             VARCHAR(255),
    city                VARCHAR(120),
    state               VARCHAR(40),
    zip_code            VARCHAR(20),
    registration_number VARCHAR(80),
    area_m2             NUMERIC(10, 2),
    purchase_date       DATE,
    purchase_value_cents BIGINT,
    current_value_cents  BIGINT,
    notes               TEXT,
    active              BOOLEAN       NOT NULL DEFAULT true,
    created_at          TIMESTAMP     NOT NULL DEFAULT now(),
    updated_at          TIMESTAMP     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_properties_workspace ON properties (workspace_id);
CREATE INDEX IF NOT EXISTS idx_properties_active    ON properties (workspace_id, active);

-- Documentos anexados a um imóvel (escritura, matrícula, IPTU, etc.).
CREATE TABLE IF NOT EXISTS property_documents (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id  UUID         NOT NULL REFERENCES properties (id) ON DELETE CASCADE,
    workspace_id UUID         NOT NULL,
    doc_type     VARCHAR(30)  NOT NULL DEFAULT 'outros',
    file_name    VARCHAR(255) NOT NULL,
    object_key   VARCHAR(500) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    size_bytes   BIGINT       NOT NULL DEFAULT 0,
    notes        TEXT,
    created_at   TIMESTAMP    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_property_documents_property  ON property_documents (property_id);
CREATE INDEX IF NOT EXISTS idx_property_documents_workspace ON property_documents (workspace_id);

-- Impostos e taxas de bens (imóveis e veículos): IPTU, IPVA, condomínio, etc.
CREATE TABLE IF NOT EXISTS asset_taxes (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id       UUID        NOT NULL,
    asset_type         VARCHAR(20) NOT NULL DEFAULT 'property',
    property_id        UUID        REFERENCES properties (id) ON DELETE CASCADE,
    vehicle_id         UUID        REFERENCES vehicles (id) ON DELETE CASCADE,
    tax_type           VARCHAR(30) NOT NULL DEFAULT 'outros',
    reference_year     INT         NOT NULL,
    description        VARCHAR(255),
    due_date           DATE,
    amount_cents       BIGINT      NOT NULL DEFAULT 0,
    paid_cents         BIGINT      NOT NULL DEFAULT 0,
    paid_date          DATE,
    status             VARCHAR(20) NOT NULL DEFAULT 'pending',
    installments_total INT         NOT NULL DEFAULT 1,
    installment_number INT         NOT NULL DEFAULT 1,
    notes              TEXT,
    created_at         TIMESTAMP   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMP   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_asset_taxes_workspace      ON asset_taxes (workspace_id);
CREATE INDEX IF NOT EXISTS idx_asset_taxes_workspace_year ON asset_taxes (workspace_id, reference_year);
CREATE INDEX IF NOT EXISTS idx_asset_taxes_workspace_due  ON asset_taxes (workspace_id, due_date);
CREATE INDEX IF NOT EXISTS idx_asset_taxes_property       ON asset_taxes (property_id);
CREATE INDEX IF NOT EXISTS idx_asset_taxes_vehicle        ON asset_taxes (vehicle_id);
