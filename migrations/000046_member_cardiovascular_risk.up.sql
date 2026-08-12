-- Categoria de risco cardiovascular do membro (estratificação feita pelo
-- MÉDICO; informada pelo usuário). Habilita interpretação de marcadores com
-- metas por risco (ex.: LDL). Valores: baixo|intermediario|alto|muito_alto.
ALTER TABLE health_family_members
    ADD COLUMN IF NOT EXISTS cardiovascular_risk VARCHAR(20) NULL;
