DROP INDEX IF EXISTS idx_financial_entries_discount_reason;
ALTER TABLE financial_entries DROP COLUMN IF EXISTS discount_reason;
ALTER TABLE financial_entries DROP COLUMN IF EXISTS discount_cents;
