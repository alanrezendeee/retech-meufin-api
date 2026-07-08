-- Volta os valores de bank para brand (estado anterior) e remove a coluna.
UPDATE credit_cards SET brand = bank WHERE bank IS NOT NULL;
ALTER TABLE credit_cards DROP COLUMN IF EXISTS bank;
