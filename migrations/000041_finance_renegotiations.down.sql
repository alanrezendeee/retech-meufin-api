DROP INDEX IF EXISTS idx_financial_entries_renegotiation;
ALTER TABLE financial_entries DROP COLUMN IF EXISTS renegotiation_id;
DROP TABLE IF EXISTS finance_renegotiations;
