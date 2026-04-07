CREATE TABLE financial_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'BRL',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_financial_accounts_workspace ON financial_accounts (workspace_id);

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(20) NOT NULL,
    parent_id UUID REFERENCES categories (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_categories_workspace ON categories (workspace_id);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    account_id UUID NOT NULL REFERENCES financial_accounts (id) ON DELETE RESTRICT,
    category_id UUID NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
    amount_cents BIGINT NOT NULL,
    flow VARCHAR(10) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_workspace_occurred ON transactions (workspace_id, occurred_at);
CREATE INDEX idx_transactions_category ON transactions (category_id);

CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    category_id UUID NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
    year INT NOT NULL,
    month INT NOT NULL,
    limit_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_budget_workspace_category_period UNIQUE (workspace_id, category_id, year, month)
);

CREATE INDEX idx_budgets_workspace ON budgets (workspace_id);
