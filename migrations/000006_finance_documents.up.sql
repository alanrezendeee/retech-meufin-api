-- Financeiro — documentos (faturas em PDF) e jobs de extração LLM.

CREATE TABLE finance_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    card_id UUID NULL REFERENCES credit_cards (id) ON DELETE SET NULL,
    entry_id UUID NULL REFERENCES financial_entries (id) ON DELETE SET NULL, -- a fatura criada a partir do doc
    file_name VARCHAR(255) NOT NULL,
    original_file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_provider VARCHAR(20) NOT NULL DEFAULT 'minio',
    bucket VARCHAR(255) NOT NULL,
    object_key VARCHAR(500) NOT NULL,
    checksum VARCHAR(128) NULL,
    uploaded_by_user_id UUID NOT NULL,
    extraction_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending|processing|extracted|failed|not_required
    extracted_text TEXT NULL,
    extracted_json JSONB NULL,
    metadata JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_finance_documents_workspace ON finance_documents (workspace_id);
CREATE INDEX idx_finance_documents_card ON finance_documents (card_id);

CREATE TABLE finance_extraction_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    document_id UUID NOT NULL REFERENCES finance_documents (id) ON DELETE CASCADE,
    provider VARCHAR(30) NOT NULL,
    model VARCHAR(100) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending|processing|completed|failed
    input_type VARCHAR(10) NOT NULL,               -- pdf|image
    prompt_version VARCHAR(30) NULL,
    raw_response JSONB NULL,
    error_message TEXT NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_finance_extraction_jobs_document ON finance_extraction_jobs (document_id);
