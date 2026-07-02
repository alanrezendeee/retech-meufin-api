-- Defesa em profundidade do catálogo de tipos/categorias (a 1ª camada é a
-- validação de domínio no Go). VARCHAR + CHECK em vez de ENUM nativo:
-- evoluir a lista é recriar a constraint numa migration transacional,
-- sem as amarras do ALTER TYPE.
ALTER TABLE financial_entries
    ADD CONSTRAINT chk_financial_entries_type CHECK (
        type IS NULL
        OR (kind = 'credit' AND type IN (
            'salario','pj_contrato','pro_labore','dividendos','rendimento',
            'aluguel','freela','ferias_13','beneficio','reembolso','outro'
        ))
        OR (kind = 'debit' AND type IN (
            'cartao','moradia','alimentacao','mercado','saude','transporte',
            'educacao','lazer','contas_fixas','servicos','impostos',
            'equipamentos','outros'
        ))
    );
