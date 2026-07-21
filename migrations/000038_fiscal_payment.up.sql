-- Forma de pagamento do cupom (credito|debito|dinheiro|pix|outros), da SEFAZ ou
-- da leitura por IA. Base para conciliar cupom × fatura de cartão (evitar duplo
-- lançamento). NULL nas linhas antigas.
ALTER TABLE finance_documents ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20);
