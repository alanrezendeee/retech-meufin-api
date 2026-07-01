ALTER TABLE financial_entries
    DROP COLUMN IF EXISTS installment_total,
    DROP COLUMN IF EXISTS installment_number,
    DROP COLUMN IF EXISTS parent_id,
    DROP COLUMN IF EXISTS card_id;

DROP TABLE IF EXISTS credit_cards;
