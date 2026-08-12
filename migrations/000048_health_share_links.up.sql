-- Links públicos temporários de saúde (docs/health-share-links.md).
-- MVP: scope member_panels — médico vê os painéis evolutivos de UM membro,
-- somente leitura, sem login. Token 256 bits; expira; revogável.
CREATE TABLE health_share_links (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID NOT NULL,
    token            VARCHAR(64) NOT NULL,
    scope            VARCHAR(32) NOT NULL DEFAULT 'member_panels',
    family_member_id UUID NOT NULL REFERENCES health_family_members (id) ON DELETE CASCADE,
    title            VARCHAR(255) NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    view_count       INT NOT NULL DEFAULT 0,
    last_viewed_at   TIMESTAMPTZ NULL,
    created_by       UUID NULL,
    revoked_at       TIMESTAMPTZ NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX uq_health_share_links_token
    ON health_share_links (token) WHERE deleted_at IS NULL;
CREATE INDEX idx_health_share_links_workspace
    ON health_share_links (workspace_id, expires_at) WHERE deleted_at IS NULL;
