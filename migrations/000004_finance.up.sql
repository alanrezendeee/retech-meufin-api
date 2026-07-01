-- Módulo Financeiro — lançamento único (crédito/débito) + fontes de receita.
-- Receita e despesa vivem na mesma tabela; o que diferencia é `kind`.

CREATE TABLE income_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(20) NOT NULL,          -- clt|pj|freelance|rental|investment|benefit|other
    active BOOLEAN NOT NULL DEFAULT true,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_income_sources_workspace ON income_sources (workspace_id);

CREATE TABLE financial_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    kind VARCHAR(10) NOT NULL,                       -- credit|debit
    status VARCHAR(15) NOT NULL DEFAULT 'prevista',  -- prevista|realizada|cancelada
    amount_cents BIGINT NOT NULL,
    due_date DATE NOT NULL,                          -- data prevista/efetiva
    family_member_id UUID NULL REFERENCES health_family_members (id) ON DELETE SET NULL,
    source_id UUID NULL REFERENCES income_sources (id) ON DELETE SET NULL,
    type VARCHAR(30) NULL,                           -- salario|pro_labore|dividendos|aluguel|freela|ferias_13|beneficio|reembolso|outro
    description TEXT NOT NULL DEFAULT '',
    recurrence VARCHAR(10) NOT NULL DEFAULT 'none',  -- none|weekly|monthly|yearly
    recurrence_group_id UUID NULL,                   -- agrupa as ocorrências geradas
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX idx_financial_entries_workspace ON financial_entries (workspace_id, due_date);
CREATE INDEX idx_financial_entries_kind ON financial_entries (workspace_id, kind, status);
CREATE INDEX idx_financial_entries_recurrence_group ON financial_entries (recurrence_group_id);
CREATE INDEX idx_financial_entries_member ON financial_entries (family_member_id);
