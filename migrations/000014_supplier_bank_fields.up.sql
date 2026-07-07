ALTER TABLE suppliers
    ADD COLUMN IF NOT EXISTS bank_agency      VARCHAR(20),
    ADD COLUMN IF NOT EXISTS bank_account     VARCHAR(30),
    ADD COLUMN IF NOT EXISTS bank_account_type VARCHAR(20);
