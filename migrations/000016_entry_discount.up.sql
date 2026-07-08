-- Desconto na liquidação: valor abatido e motivo (slug do catálogo global
-- curado em código — internal/domain/finance/discount_reason.go).
ALTER TABLE financial_entries ADD COLUMN IF NOT EXISTS discount_cents BIGINT;
ALTER TABLE financial_entries ADD COLUMN IF NOT EXISTS discount_reason VARCHAR(40);

-- Indicador: agregações por motivo de desconto (dashboards/insights).
CREATE INDEX IF NOT EXISTS idx_financial_entries_discount_reason
    ON financial_entries (workspace_id, discount_reason)
    WHERE discount_reason IS NOT NULL;
