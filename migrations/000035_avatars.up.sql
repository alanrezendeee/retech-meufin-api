-- Avatares: foto do membro da família + foto de perfil do usuário logado.
-- 100% idempotente: ADD COLUMN IF NOT EXISTS / CREATE TABLE|INDEX IF NOT EXISTS.

-- 1) Foto (avatar) do membro da família — object key no storage de saúde.
ALTER TABLE health_family_members
    ADD COLUMN IF NOT EXISTS avatar_object_key VARCHAR(500);

-- 2) Perfil do usuário logado — guarda o avatar do usuário (1:1 com o usuário).
CREATE TABLE IF NOT EXISTS user_profiles (
    user_id           UUID         PRIMARY KEY,
    workspace_id      UUID         NOT NULL,
    avatar_object_key VARCHAR(500),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_profiles_workspace ON user_profiles (workspace_id);
