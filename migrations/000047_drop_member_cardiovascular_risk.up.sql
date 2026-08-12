-- Reverte a 000046: risco cardiovascular no membro foi removido do produto.
-- A classificação de metas condicionais (ex.: LDL) é feita confrontando o
-- valor com a tabela do catálogo (default_ref_tiers), linha a linha, sem
-- depender de dado clínico digitado pelo usuário.
ALTER TABLE health_family_members DROP COLUMN IF EXISTS cardiovascular_risk;
