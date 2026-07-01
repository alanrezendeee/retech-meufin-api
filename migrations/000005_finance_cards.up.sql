-- Financeiro — cartões de crédito e agrupamento de faturas/compras.

CREATE TABLE credit_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    brand VARCHAR(50) NULL,          -- nubank|inter|santander|... (livre)
    closing_day INT NULL,            -- dia do fechamento (1..31)
    due_day INT NULL,                -- dia do vencimento (1..31)
    active BOOLEAN NOT NULL DEFAULT true,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_credit_cards_workspace ON credit_cards (workspace_id);

-- Lançamentos ganham: vínculo com cartão (fatura), agrupamento pai/filho
-- (fatura -> compras) e controle de parcelas.
ALTER TABLE financial_entries
    ADD COLUMN card_id UUID NULL REFERENCES credit_cards (id) ON DELETE SET NULL,
    ADD COLUMN parent_id UUID NULL REFERENCES financial_entries (id) ON DELETE CASCADE,
    ADD COLUMN installment_number INT NULL,
    ADD COLUMN installment_total INT NULL;

CREATE INDEX idx_financial_entries_card ON financial_entries (card_id);
CREATE INDEX idx_financial_entries_parent ON financial_entries (parent_id);
