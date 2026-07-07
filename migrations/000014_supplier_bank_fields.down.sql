ALTER TABLE suppliers
    DROP COLUMN IF EXISTS bank_agency,
    DROP COLUMN IF EXISTS bank_account,
    DROP COLUMN IF EXISTS bank_account_type,
    DROP COLUMN IF EXISTS pix_key_holder;
