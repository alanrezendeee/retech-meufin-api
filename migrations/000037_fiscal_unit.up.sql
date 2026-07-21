-- Unidade de medida do item fiscal (kg, un, L, g, ...), vinda da SEFAZ/leitura.
-- Permite normalizar o preço por unidade canônica (R$/kg vs R$/un) e separar
-- séries de preço/inflação por (produto, unidade). NULL nas linhas antigas.
ALTER TABLE finance_fiscal_items ADD COLUMN IF NOT EXISTS unit_of_measure VARCHAR(10);
