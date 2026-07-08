-- Pagamento parcial (baixa parcial com desdobramento): o saldo não pago vira
-- um novo lançamento previsto, ligado à origem por residual_of_id.
ALTER TABLE financial_entries ADD COLUMN IF NOT EXISTS residual_of_id UUID;

CREATE INDEX IF NOT EXISTS idx_financial_entries_residual_of
    ON financial_entries (workspace_id, residual_of_id)
    WHERE residual_of_id IS NOT NULL;
