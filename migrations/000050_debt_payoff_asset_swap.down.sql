-- Eventos de quitação/troca não têm representação no esquema anterior: as
-- cobranças encerradas por eles continuam canceladas (motivo 'quitacao'),
-- mas o evento em si é removido.
DELETE FROM finance_renegotiations WHERE kind <> 'renegociacao';

UPDATE financial_entries SET type = 'outro' WHERE kind = 'credit' AND type = 'venda_bem';
ALTER TABLE financial_entries DROP CONSTRAINT IF EXISTS chk_financial_entries_type;
ALTER TABLE financial_entries
    ADD CONSTRAINT chk_financial_entries_type CHECK (
        type IS NULL
        OR kind = 'debit'
        OR (kind = 'credit' AND type IN (
            'salario','pj_contrato','pro_labore','dividendos','rendimento',
            'aluguel','freela','ferias_13','beneficio','reembolso','outro'
        ))
    );

DROP INDEX IF EXISTS idx_financial_entries_asset;
ALTER TABLE financial_entries DROP CONSTRAINT IF EXISTS chk_financial_entries_asset_type;
ALTER TABLE financial_entries
    DROP COLUMN IF EXISTS asset_id,
    DROP COLUMN IF EXISTS asset_type;

DROP INDEX IF EXISTS idx_finance_renegotiations_new_asset;
DROP INDEX IF EXISTS idx_finance_renegotiations_asset;
ALTER TABLE finance_renegotiations DROP CONSTRAINT IF EXISTS chk_finance_renegotiations_payer;
ALTER TABLE finance_renegotiations DROP CONSTRAINT IF EXISTS chk_finance_renegotiations_kind;
ALTER TABLE finance_renegotiations
    DROP COLUMN IF EXISTS downpayment_entry_id,
    DROP COLUMN IF EXISTS cash_downpayment_cents,
    DROP COLUMN IF EXISTS trade_in_entry_id,
    DROP COLUMN IF EXISTS trade_in_cents,
    DROP COLUMN IF EXISTS new_asset_id,
    DROP COLUMN IF EXISTS asset_id,
    DROP COLUMN IF EXISTS asset_type,
    DROP COLUMN IF EXISTS payoff_entry_id,
    DROP COLUMN IF EXISTS payer,
    DROP COLUMN IF EXISTS payoff_cents,
    DROP COLUMN IF EXISTS kind;
