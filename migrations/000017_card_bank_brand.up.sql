-- Separa banco emissor de bandeira no cartão de crédito.
-- O campo brand vinha sendo usado com nomes de banco (nubank, itau...);
-- move esses valores para a nova coluna bank e zera brand, que passa a
-- aceitar apenas slugs do catálogo global de bandeiras (visa, mastercard,
-- elo... — curado em internal/domain/finance/card_brand.go).
ALTER TABLE credit_cards ADD COLUMN IF NOT EXISTS bank VARCHAR(100);

UPDATE credit_cards SET bank = brand WHERE brand IS NOT NULL AND bank IS NULL;
UPDATE credit_cards SET brand = NULL;
