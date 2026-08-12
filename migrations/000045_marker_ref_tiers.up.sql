-- Metas condicionais de curadoria (tiers) para marcadores sem faixa única.
-- Ex.: LDL — metas por categoria de risco cardiovascular (diretriz SBC).
-- Informativo: a interpretação computada não escolhe tier sozinha (o risco do
-- paciente não é inferível do laudo).
ALTER TABLE health_markers ADD COLUMN IF NOT EXISTS default_ref_tiers JSONB NULL;

UPDATE health_markers
SET default_ref_tiers = '[
  {"key": "baixo", "label": "Risco baixo", "max": 130},
  {"key": "intermediario", "label": "Risco intermediário", "max": 100},
  {"key": "alto", "label": "Risco alto", "max": 70},
  {"key": "muito_alto", "label": "Risco muito alto", "max": 50}
]'::jsonb,
    default_ref_text = COALESCE(default_ref_text,
      'Metas por categoria de risco cardiovascular estimada pelo médico (diretriz SBC); crianças e adolescentes: inferior a 110 mg/dL')
WHERE scope = 'system'
  AND normalized_key = 'colesterol ldl'
  AND default_ref_tiers IS NULL;
