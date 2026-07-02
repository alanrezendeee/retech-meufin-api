-- Categorias de despesa viram cadastro GERENCIADO por workspace (CRUD, com
-- seed das padrão no primeiro uso) — taxonomia de gasto é da família, não do
-- sistema. Tipos de receita seguem curados (semântica fiscal).
CREATE TABLE finance_expense_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    slug VARCHAR(40) NOT NULL,   -- valor gravado em financial_entries.type
    name VARCHAR(80) NOT NULL,   -- rótulo exibido
    -- Grupo canônico (curado pelo sistema): dimensão ESTÁVEL dos indicadores.
    -- Categoria é do usuário; grupo nunca. Duas camadas = liberdade sem bagunça.
    group_slug VARCHAR(30) NOT NULL CHECK (group_slug IN (
        'moradia','alimentacao','transporte','saude','educacao','lazer',
        'contas_servicos','impostos_taxas','equipamentos_bens','outros'
    )),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    UNIQUE (workspace_id, slug)
);
CREATE INDEX idx_finance_expense_categories_workspace ON finance_expense_categories (workspace_id);

-- O CHECK estático de categorias sai (agora são dinâmicas por workspace;
-- validação no domínio/serviço). Receitas seguem curadas no CHECK; 'cartao'
-- continua reservado à fatura pai (imposto no domínio).
ALTER TABLE financial_entries DROP CONSTRAINT IF EXISTS chk_financial_entries_type;
ALTER TABLE financial_entries
    ADD CONSTRAINT chk_financial_entries_type CHECK (
        type IS NULL
        OR kind = 'debit'
        OR (kind = 'credit' AND type IN (
            'salario','pj_contrato','pro_labore','dividendos','rendimento',
            'aluguel','freela','ferias_13','beneficio','reembolso','outro'
        ))
    );
