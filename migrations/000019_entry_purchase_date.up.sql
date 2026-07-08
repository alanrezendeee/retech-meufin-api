-- Data da compra (informacional) para itens de fatura de cartão: a compra
-- acontece numa data, mas o vencimento de cada item é SEMPRE o vencimento da
-- fatura (o dinheiro sai no pagamento da fatura — regime de caixa).
ALTER TABLE financial_entries ADD COLUMN IF NOT EXISTS purchase_date DATE;
