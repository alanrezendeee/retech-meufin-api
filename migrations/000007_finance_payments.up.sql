-- Financeiro — contas, liquidação de lançamentos (forma de pagamento + comprovante)
-- e documentos de membros da família.

-- Contas do tenant (corrente, poupança, carteira/dinheiro, digital).
CREATE TABLE finance_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(20) NOT NULL,       -- corrente|poupanca|carteira|digital
    bank_name VARCHAR(255) NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_finance_accounts_workspace ON finance_accounts (workspace_id);

-- Liquidação: quando/quanto/como o lançamento foi pago (ou recebido).
-- paid_amount_cents pode diferir de amount_cents (juros/multa/desconto).
ALTER TABLE financial_entries
    ADD COLUMN paid_at TIMESTAMPTZ NULL,
    ADD COLUMN paid_amount_cents BIGINT NULL,
    ADD COLUMN payment_method VARCHAR(20) NULL,  -- pix|debito|transferencia|boleto|dinheiro|cartao_credito
    ADD COLUMN payment_account_id UUID NULL REFERENCES finance_accounts (id) ON DELETE SET NULL,
    ADD COLUMN payment_card_id UUID NULL REFERENCES credit_cards (id) ON DELETE SET NULL;

-- Comprovantes reutilizam finance_documents: kind diferencia fatura importada
-- de comprovante de pagamento (entry_id aponta para o lançamento liquidado).
ALTER TABLE finance_documents
    ADD COLUMN kind VARCHAR(20) NOT NULL DEFAULT 'import';  -- import|receipt
CREATE INDEX idx_finance_documents_entry ON finance_documents (entry_id);

-- Documentos de membros da família (cpf, rg, cnh, passaporte, ...).
CREATE TABLE family_member_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    family_member_id UUID NOT NULL REFERENCES health_family_members (id) ON DELETE CASCADE,
    doc_type VARCHAR(30) NOT NULL,   -- cpf|rg|cnh|passaporte|carteira_trabalho|certidao_nascimento|titulo_eleitor|cartao_sus|plano_saude|outro
    label VARCHAR(255) NULL,         -- rótulo livre (obrigatório quando doc_type=outro)
    doc_number VARCHAR(100) NULL,
    valid_until DATE NULL,           -- cnh/passaporte vencem
    notes TEXT NULL,
    file_name VARCHAR(255) NOT NULL,
    original_file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_provider VARCHAR(20) NOT NULL DEFAULT 'minio',
    bucket VARCHAR(255) NOT NULL,
    object_key VARCHAR(500) NOT NULL,
    uploaded_by_user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_family_member_documents_workspace ON family_member_documents (workspace_id);
CREATE INDEX idx_family_member_documents_member ON family_member_documents (family_member_id);
