-- Módulo: Segurança do Lar — itens de segurança física/química/biológica/elétrica
-- da casa com validade e manutenção periódica (mangueira de gás, extintor, caixa
-- d'água, dedetização, revisão elétrica, para-raios, etc).
-- 100% idempotente: pode rodar múltiplas vezes sem erro.

CREATE TABLE IF NOT EXISTS home_safety_items (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id            UUID NOT NULL,
    name                    TEXT NOT NULL,
    category                TEXT NOT NULL DEFAULT 'outros'
                              CHECK (category IN ('gas','agua','incendio','eletrica','climatizacao','pragas','estrutura','seguranca_eletronica','piscina','saude','outros')),
    risk_type               TEXT NOT NULL DEFAULT 'outros'
                              CHECK (risk_type IN ('fisico','quimico','biologico','eletrico','incendio','outros')),
    location                TEXT,
    brand                   TEXT,
    model                   TEXT,
    installed_at            DATE,
    lifespan_months         INT,
    service_interval_months INT,
    last_service_at         DATE,
    next_due_date           DATE,
    priority                TEXT NOT NULL DEFAULT 'media' CHECK (priority IN ('alta','media','baixa')),
    responsible             TEXT,
    last_cost_cents         BIGINT NOT NULL DEFAULT 0,
    active                  BOOLEAN NOT NULL DEFAULT TRUE,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hsi_workspace ON home_safety_items(workspace_id);
CREATE INDEX IF NOT EXISTS idx_hsi_category  ON home_safety_items(workspace_id, category);
CREATE INDEX IF NOT EXISTS idx_hsi_next_due  ON home_safety_items(workspace_id, next_due_date) WHERE next_due_date IS NOT NULL;

CREATE TABLE IF NOT EXISTS home_safety_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    item_id      UUID NOT NULL REFERENCES home_safety_items(id) ON DELETE CASCADE,
    event_type   TEXT NOT NULL DEFAULT 'manutencao'
                   CHECK (event_type IN ('instalacao','troca','manutencao','inspecao','recarga','limpeza')),
    event_date   DATE NOT NULL,
    cost_cents   BIGINT NOT NULL DEFAULT 0,
    provider     TEXT,
    notes        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_hse_workspace ON home_safety_events(workspace_id);
CREATE INDEX IF NOT EXISTS idx_hse_item      ON home_safety_events(item_id);
CREATE INDEX IF NOT EXISTS idx_hse_date      ON home_safety_events(workspace_id, event_date);
