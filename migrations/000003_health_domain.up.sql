-- Módulo Saúde Familiar — Fases 1..3: membros, laboratórios, solicitações,
-- resultados, documentos, jobs de extração e auditoria.

CREATE TABLE health_family_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    relationship VARCHAR(20) NOT NULL,          -- self|spouse|child|parent|other
    birth_date DATE NULL,
    gender VARCHAR(20) NULL,
    document VARCHAR(50) NULL,
    notes TEXT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_health_family_members_workspace ON health_family_members (workspace_id);

CREATE TABLE health_labs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    website_url VARCHAR(500) NULL,
    exam_results_url VARCHAR(500) NULL,
    contact_phone VARCHAR(50) NULL,
    address VARCHAR(500) NULL,
    notes TEXT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_health_labs_workspace ON health_labs (workspace_id);

CREATE TABLE health_exam_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    family_member_id UUID NOT NULL REFERENCES health_family_members (id) ON DELETE RESTRICT,
    lab_id UUID NULL REFERENCES health_labs (id) ON DELETE RESTRICT,
    requested_by VARCHAR(255) NULL,
    request_date DATE NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'draft', -- draft|requested|collected|partially_resulted|resulted|canceled
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_health_exam_requests_workspace ON health_exam_requests (workspace_id);
CREATE INDEX idx_health_exam_requests_member ON health_exam_requests (family_member_id);

CREATE TABLE health_exam_request_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    exam_request_id UUID NOT NULL REFERENCES health_exam_requests (id) ON DELETE CASCADE,
    marker_id UUID NULL REFERENCES health_markers (id) ON DELETE SET NULL,
    exam_name VARCHAR(255) NOT NULL,
    exam_code VARCHAR(50) NULL,
    body_area VARCHAR(100) NULL,
    notes TEXT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending|collected|resulted|canceled
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_health_exam_request_items_request ON health_exam_request_items (exam_request_id);

CREATE TABLE health_exam_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    family_member_id UUID NOT NULL REFERENCES health_family_members (id) ON DELETE RESTRICT,
    lab_id UUID NULL REFERENCES health_labs (id) ON DELETE RESTRICT,
    exam_request_id UUID NULL REFERENCES health_exam_requests (id) ON DELETE SET NULL,
    exam_date DATE NOT NULL,
    collection_date DATE NULL,
    release_date DATE NULL,
    source_type VARCHAR(20) NOT NULL DEFAULT 'manual', -- manual|pdf|image|ocr|llm
    status VARCHAR(20) NOT NULL DEFAULT 'draft',        -- draft|processing|extracted|reviewed|failed
    summary TEXT NULL,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_health_exam_results_workspace ON health_exam_results (workspace_id);
CREATE INDEX idx_health_exam_results_member ON health_exam_results (family_member_id, exam_date);

CREATE TABLE health_exam_result_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    exam_result_id UUID NOT NULL REFERENCES health_exam_results (id) ON DELETE CASCADE,
    marker_id UUID NULL REFERENCES health_markers (id) ON DELETE SET NULL,
    raw_marker_name VARCHAR(255) NULL,
    result_value VARCHAR(255) NOT NULL,
    result_numeric NUMERIC NULL,
    unit VARCHAR(30) NULL,
    reference_min NUMERIC NULL,
    reference_max NUMERIC NULL,
    reference_text VARCHAR(255) NULL,
    interpretation VARCHAR(20) NULL,          -- low|normal|high|critical|inconclusive
    interpretation_computed VARCHAR(20) NULL, -- calculada por valor x referência
    method VARCHAR(100) NULL,
    material VARCHAR(100) NULL,
    raw_text TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_health_exam_result_items_result ON health_exam_result_items (exam_result_id);
CREATE INDEX idx_health_exam_result_items_history ON health_exam_result_items (workspace_id, marker_id);

CREATE TABLE health_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    family_member_id UUID NULL REFERENCES health_family_members (id) ON DELETE SET NULL,
    lab_id UUID NULL REFERENCES health_labs (id) ON DELETE SET NULL,
    exam_request_id UUID NULL REFERENCES health_exam_requests (id) ON DELETE SET NULL,
    exam_result_id UUID NULL REFERENCES health_exam_results (id) ON DELETE SET NULL,
    document_type VARCHAR(30) NOT NULL, -- exam_request|exam_result|image_report|medical_report|prescription|other
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
CREATE INDEX idx_health_documents_workspace ON health_documents (workspace_id);
CREATE INDEX idx_health_documents_result ON health_documents (exam_result_id);

CREATE TABLE health_extraction_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    document_id UUID NOT NULL REFERENCES health_documents (id) ON DELETE CASCADE,
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
CREATE INDEX idx_health_extraction_jobs_document ON health_extraction_jobs (document_id);

CREATE TABLE health_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    user_id UUID NULL,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NULL,
    metadata JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_health_audit_logs_workspace ON health_audit_logs (workspace_id, created_at);
