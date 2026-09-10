-- Volta ao vínculo único: origens apontam de novo para o acordo que as encerrou.
UPDATE financial_entries
   SET renegotiation_id = settled_by_renegotiation_id
 WHERE settled_by_renegotiation_id IS NOT NULL;

DROP INDEX IF EXISTS idx_finance_renegotiations_new_group;
DROP INDEX IF EXISTS idx_finance_renegotiations_origin_group;
ALTER TABLE finance_renegotiations
    DROP COLUMN IF EXISTS new_group_id,
    DROP COLUMN IF EXISTS origin_group_id;

DROP INDEX IF EXISTS idx_financial_entries_settled_by_renegotiation;
ALTER TABLE financial_entries DROP COLUMN IF EXISTS settled_by_renegotiation_id;
