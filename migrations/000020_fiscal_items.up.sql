-- Cupom/nota fiscal: detalhamento item a item de um lançamento.
-- Itens NÃO são lançamentos (o valor já está na despesa que agrupa a compra);
-- são informação vinculada — mesma lógica do desdobramento fatura→compras,
-- mas um nível abaixo e fora das agregações de saldo.

CREATE TABLE IF NOT EXISTS finance_fiscal_items (
    id             UUID PRIMARY KEY,
    workspace_id   UUID NOT NULL,
    entry_id       UUID NOT NULL REFERENCES financial_entries (id) ON DELETE CASCADE,
    document_id    UUID NOT NULL REFERENCES finance_documents (id) ON DELETE CASCADE,
    description    TEXT NOT NULL,
    -- quantidade em milésimos (1un = 1000; 0,455kg = 455) — inteiro sempre
    quantity_milli BIGINT NOT NULL DEFAULT 0,
    unit_cents     BIGINT NOT NULL DEFAULT 0,
    amount_cents   BIGINT NOT NULL,
    category       VARCHAR(30),
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_finance_fiscal_items_entry
    ON finance_fiscal_items (workspace_id, entry_id);

-- Lançamento aponta para o cupom/nota vinculado (1:1 na prática).
ALTER TABLE financial_entries ADD COLUMN IF NOT EXISTS fiscal_document_id UUID;
