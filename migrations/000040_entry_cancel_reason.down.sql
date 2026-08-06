DROP INDEX IF EXISTS idx_financial_entries_cancel_reason;
ALTER TABLE financial_entries DROP COLUMN IF EXISTS cancel_reason;
