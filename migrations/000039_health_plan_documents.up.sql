-- Documentos anexados a um plano de saúde (contrato/apólice, carteirinha, tabela
-- de coparticipação, boletos, IR etc.). Espelha family_member_documents + plan_id.
-- Fase 1 do módulo Plano de Saúde. Storage no MinIO (bucket 'health').
CREATE TABLE IF NOT EXISTS health_plan_documents (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id       UUID         NOT NULL,
    plan_id            UUID         NOT NULL REFERENCES health_plans (id) ON DELETE CASCADE,
    doc_type           VARCHAR(40)  NOT NULL,
    label              VARCHAR(255),
    doc_number         VARCHAR(100),
    valid_until        DATE,
    notes              TEXT,
    file_name          VARCHAR(255) NOT NULL,
    original_file_name VARCHAR(255) NOT NULL,
    mime_type          VARCHAR(100) NOT NULL,
    size_bytes         BIGINT       NOT NULL DEFAULT 0,
    storage_provider   VARCHAR(20)  NOT NULL DEFAULT 'minio',
    bucket             VARCHAR(255) NOT NULL,
    object_key         VARCHAR(500) NOT NULL,
    uploaded_by_user_id UUID        NOT NULL,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_health_plan_documents_workspace ON health_plan_documents (workspace_id);
CREATE INDEX IF NOT EXISTS idx_health_plan_documents_plan ON health_plan_documents (workspace_id, plan_id);
