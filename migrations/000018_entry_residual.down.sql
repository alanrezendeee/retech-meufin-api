DROP INDEX IF EXISTS idx_financial_entries_residual_of;
ALTER TABLE financial_entries DROP COLUMN IF EXISTS residual_of_id;
