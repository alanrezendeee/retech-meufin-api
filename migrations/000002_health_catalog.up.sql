-- Módulo Saúde Familiar — Fase 0: catálogo canônico de marcadores.

CREATE TABLE health_markers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope VARCHAR(10) NOT NULL,                       -- 'system' | 'tenant'
    workspace_id UUID NULL,                           -- NULL quando system
    canonical_name VARCHAR(255) NOT NULL,
    normalized_key VARCHAR(255) NOT NULL,             -- gerado no app (unaccent+lower+trim)
    loinc_code VARCHAR(20) NULL,
    category VARCHAR(50) NOT NULL,
    comparability_class VARCHAR(20) NOT NULL DEFAULT 'standardized',
    canonical_unit VARCHAR(30) NULL,
    default_ref_min NUMERIC NULL,
    default_ref_max NUMERIC NULL,
    default_ref_text VARCHAR(255) NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

-- Dedup: chave única por escopo, ignorando registros soft-deletados.
CREATE UNIQUE INDEX uq_markers_system_key ON health_markers (normalized_key)
    WHERE scope = 'system' AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_markers_tenant_key ON health_markers (workspace_id, normalized_key)
    WHERE scope = 'tenant' AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_markers_system_loinc ON health_markers (loinc_code)
    WHERE scope = 'system' AND loinc_code IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_health_markers_workspace ON health_markers (workspace_id);
CREATE INDEX idx_health_markers_category ON health_markers (category);

CREATE TABLE health_marker_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    marker_id UUID NOT NULL REFERENCES health_markers (id) ON DELETE CASCADE,
    scope VARCHAR(10) NOT NULL,
    workspace_id UUID NULL,
    alias VARCHAR(255) NOT NULL,
    normalized_alias VARCHAR(255) NOT NULL,
    source VARCHAR(50) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX uq_alias_system ON health_marker_aliases (normalized_alias)
    WHERE scope = 'system' AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_alias_tenant ON health_marker_aliases (workspace_id, normalized_alias)
    WHERE scope = 'tenant' AND deleted_at IS NULL;
CREATE INDEX idx_health_marker_aliases_marker ON health_marker_aliases (marker_id);
