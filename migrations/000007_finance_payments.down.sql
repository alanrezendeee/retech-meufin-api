DROP TABLE IF EXISTS family_member_documents;

DROP INDEX IF EXISTS idx_finance_documents_entry;
ALTER TABLE finance_documents
    DROP COLUMN IF EXISTS kind;

ALTER TABLE financial_entries
    DROP COLUMN IF EXISTS payment_card_id,
    DROP COLUMN IF EXISTS payment_account_id,
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS paid_amount_cents,
    DROP COLUMN IF EXISTS paid_at;

DROP TABLE IF EXISTS finance_accounts;
