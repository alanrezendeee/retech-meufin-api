ALTER TABLE financial_entries DROP COLUMN IF EXISTS fiscal_document_id;
DROP INDEX IF EXISTS idx_finance_fiscal_items_entry;
DROP TABLE IF EXISTS finance_fiscal_items;
