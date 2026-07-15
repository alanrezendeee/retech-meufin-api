-- Módulo: Garantias de bens
-- Controle da garantia de tudo que a família compra (TV, geladeira, celular,
-- veículo, imóvel, compras online com garantia estendida, etc.).
-- Idempotente: pode ser reaplicada com segurança.

CREATE TABLE IF NOT EXISTS warranties (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id                UUID         NOT NULL,
    item_name                   VARCHAR(200) NOT NULL,
    category                    VARCHAR(30)  NOT NULL DEFAULT 'outros',
    brand                       VARCHAR(120),
    model                       VARCHAR(120),
    serial_number               VARCHAR(120),
    store                       VARCHAR(150),
    supplier_name               VARCHAR(150),
    purchase_date               DATE         NOT NULL,
    price_cents                 BIGINT,
    invoice_number              VARCHAR(80),
    entry_id                    UUID,
    fiscal_item_id              UUID,
    legal_warranty_days         INT          NOT NULL DEFAULT 90,
    contractual_warranty_months INT          NOT NULL DEFAULT 12,
    extended_warranty_months    INT          NOT NULL DEFAULT 0,
    extended_provider           VARCHAR(150),
    extended_cost_cents         BIGINT       NOT NULL DEFAULT 0,
    coverage_notes              TEXT,
    notes                       TEXT,
    active                      BOOLEAN      NOT NULL DEFAULT true,
    created_at                  TIMESTAMP    NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMP    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_warranties_workspace     ON warranties (workspace_id);
CREATE INDEX IF NOT EXISTS idx_warranties_category      ON warranties (workspace_id, category);
CREATE INDEX IF NOT EXISTS idx_warranties_purchase_date ON warranties (workspace_id, purchase_date);

-- Documentos anexados a uma garantia (nota fiscal, certificado, termo da
-- garantia estendida, manual, etc.).

CREATE TABLE IF NOT EXISTS warranty_documents (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    warranty_id        UUID         NOT NULL REFERENCES warranties (id) ON DELETE CASCADE,
    workspace_id       UUID         NOT NULL,
    doc_type           VARCHAR(30)  NOT NULL DEFAULT 'outros',
    file_name          VARCHAR(255) NOT NULL,
    original_file_name VARCHAR(255) NOT NULL DEFAULT '',
    object_key         VARCHAR(500) NOT NULL,
    content_type       VARCHAR(100) NOT NULL DEFAULT '',
    size_bytes         BIGINT       NOT NULL DEFAULT 0,
    notes              TEXT,
    created_at         TIMESTAMP    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_warranty_documents_warranty  ON warranty_documents (warranty_id);
CREATE INDEX IF NOT EXISTS idx_warranty_documents_workspace ON warranty_documents (workspace_id);
