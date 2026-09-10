-- Linhagem da dívida: encadeia renegociações sucessivas e separa os dois
-- papéis que um lançamento pode ter em relação a um acordo.
--
-- Antes, financial_entries.renegotiation_id servia tanto para "criado por"
-- quanto para "encerrado por". Renegociar um acordo por cima do outro
-- sobrescrevia o vínculo das parcelas criadas pelo primeiro, e o detalhe do
-- acordo antigo perdia a seção "parcelas criadas". Agora:
--   renegotiation_id            = acordo que CRIOU o lançamento
--   settled_by_renegotiation_id = acordo que ENCERROU o lançamento
ALTER TABLE financial_entries
    ADD COLUMN IF NOT EXISTS settled_by_renegotiation_id UUID
    REFERENCES finance_renegotiations (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_financial_entries_settled_by_renegotiation
    ON financial_entries (workspace_id, settled_by_renegotiation_id)
    WHERE settled_by_renegotiation_id IS NOT NULL;

-- O acordo passa a saber de que grupo veio e que grupo criou: é o que
-- permite andar a cadeia original → acordo 1 → acordo 2 nos dois sentidos.
ALTER TABLE finance_renegotiations
    ADD COLUMN IF NOT EXISTS origin_group_id UUID,
    ADD COLUMN IF NOT EXISTS new_group_id    UUID;

CREATE INDEX IF NOT EXISTS idx_finance_renegotiations_origin_group
    ON finance_renegotiations (workspace_id, origin_group_id)
    WHERE origin_group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_finance_renegotiations_new_group
    ON finance_renegotiations (workspace_id, new_group_id)
    WHERE new_group_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Backfill
-- ---------------------------------------------------------------------------

-- 1. Origens encerradas: o vínculo antigo vira "encerrado por".
UPDATE financial_entries
   SET settled_by_renegotiation_id = renegotiation_id
 WHERE status = 'cancelada'
   AND cancel_reason = 'renegociacao'
   AND renegotiation_id IS NOT NULL
   AND settled_by_renegotiation_id IS NULL;

-- 2. Grupo criado por cada acordo: o grupo mais frequente entre os
--    lançamentos que apontam para ele como criador.
UPDATE finance_renegotiations r
   SET new_group_id = sub.gid
  FROM (
        SELECT DISTINCT ON (renegotiation_id) renegotiation_id, recurrence_group_id AS gid
          FROM (
                SELECT renegotiation_id, recurrence_group_id, COUNT(*) AS n
                  FROM financial_entries
                 WHERE renegotiation_id IS NOT NULL
                   AND recurrence_group_id IS NOT NULL
                   AND (settled_by_renegotiation_id IS NULL
                        OR settled_by_renegotiation_id <> renegotiation_id)
                 GROUP BY renegotiation_id, recurrence_group_id
               ) c
         ORDER BY renegotiation_id, n DESC
       ) sub
 WHERE sub.renegotiation_id = r.id
   AND r.new_group_id IS NULL;

-- 3. Grupo de origem de cada acordo: o grupo mais frequente entre as
--    parcelas que ele encerrou (residuais não têm grupo e ficam de fora).
UPDATE finance_renegotiations r
   SET origin_group_id = sub.gid
  FROM (
        SELECT DISTINCT ON (settled_by_renegotiation_id) settled_by_renegotiation_id, recurrence_group_id AS gid
          FROM (
                SELECT settled_by_renegotiation_id, recurrence_group_id, COUNT(*) AS n
                  FROM financial_entries
                 WHERE settled_by_renegotiation_id IS NOT NULL
                   AND recurrence_group_id IS NOT NULL
                 GROUP BY settled_by_renegotiation_id, recurrence_group_id
               ) c
         ORDER BY settled_by_renegotiation_id, n DESC
       ) sub
 WHERE sub.settled_by_renegotiation_id = r.id
   AND r.origin_group_id IS NULL;

-- 4. Repara o "criado por" das origens: se o grupo delas foi criado por um
--    acordo anterior, o vínculo volta para ele; senão, eram lançamentos
--    originais (ou residuais) e não foram criados por acordo nenhum.
UPDATE financial_entries e
   SET renegotiation_id = r1.id
  FROM finance_renegotiations r1
 WHERE e.settled_by_renegotiation_id IS NOT NULL
   AND e.recurrence_group_id IS NOT NULL
   AND e.recurrence_group_id = r1.new_group_id
   AND e.renegotiation_id = e.settled_by_renegotiation_id;

UPDATE financial_entries
   SET renegotiation_id = NULL
 WHERE settled_by_renegotiation_id IS NOT NULL
   AND renegotiation_id = settled_by_renegotiation_id;
