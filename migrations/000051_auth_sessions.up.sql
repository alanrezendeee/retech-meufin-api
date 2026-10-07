-- Sessões de browser do gateway de autenticação (docs/auth-session-gateway.md).
-- Padrão "token handler" (BFF): o browser recebe só um id aleatório em cookie
-- HttpOnly; access/refresh tokens do retech-auth-api ficam aqui, cifrados em
-- repouso (AES-256-GCM, chave SESSION_ENCRYPTION_KEY). Nenhum JWT chega ao JS.
CREATE TABLE auth_sessions (
    id                 VARCHAR(64) PRIMARY KEY,          -- sha256(hex) do token do cookie
    user_id            UUID NOT NULL,
    email              VARCHAR(255) NOT NULL,
    name               VARCHAR(255) NOT NULL DEFAULT '',
    tenant_id          UUID NULL,
    access_token_enc   BYTEA NOT NULL,
    refresh_token_enc  BYTEA NOT NULL,
    access_expires_at  TIMESTAMPTZ NOT NULL,
    expires_at         TIMESTAMPTZ NOT NULL,             -- expiração absoluta da sessão
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at         TIMESTAMPTZ NULL,
    ip                 VARCHAR(64) NOT NULL DEFAULT '',
    user_agent         VARCHAR(512) NOT NULL DEFAULT ''
);

CREATE INDEX idx_auth_sessions_user ON auth_sessions (user_id);
CREATE INDEX idx_auth_sessions_expires ON auth_sessions (expires_at);
