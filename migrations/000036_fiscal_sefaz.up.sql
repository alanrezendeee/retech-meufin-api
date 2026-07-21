-- Ingestão fiscal SEFAZ (Infosimples) + fallback IA — entitlement/cota + procedência.
-- Regras de negócio em DEC-0001 (vault retech-meufin).

-- Cota/tier por workspace. Sem tabela de workspaces local: workspace_id é o UUID
-- opaco do JWT, standalone (mesmo padrão das demais tabelas do produto).
-- Linha ausente = tier 'free' implícito (sem backfill).
CREATE TABLE IF NOT EXISTS workspace_entitlements (
    workspace_id       UUID        PRIMARY KEY,
    tier               VARCHAR(30) NOT NULL DEFAULT 'free',
    fiscal_sefaz_quota INTEGER,                 -- NULL = usa o default do tier
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Procedência do detalhamento fiscal de um documento:
--   'sefaz'   → verificado na Receita (Infosimples), dado exato
--   'ocr_llm' → leitura por IA (fallback), requer conferência do usuário
ALTER TABLE finance_documents ADD COLUMN IF NOT EXISTS fiscal_source VARCHAR(20);
