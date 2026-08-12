-- Curadoria do catálogo: VLDL não recebe faixa dos laboratórios ("não dispomos
-- de valor de referência"). Grava o desejável de literatura como default do
-- marcador (fallback de interpretação; nunca substitui a faixa do laudo).
-- O seed cobre instalações novas; este UPDATE cobre a linha já existente.
UPDATE health_markers
SET default_ref_max = 30,
    default_ref_text = 'Desejável: inferior a 30 mg/dL (valor de literatura; laboratórios geralmente não informam faixa)'
WHERE scope = 'system'
  AND normalized_key = 'colesterol vldl'
  AND default_ref_min IS NULL
  AND default_ref_max IS NULL;
