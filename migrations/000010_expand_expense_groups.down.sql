ALTER TABLE finance_expense_categories DROP CONSTRAINT IF EXISTS finance_expense_categories_group_slug_check;
ALTER TABLE finance_expense_categories
    ADD CONSTRAINT finance_expense_categories_group_slug_check CHECK (group_slug IN (
        'moradia','alimentacao','transporte','saude','educacao','lazer',
        'contas_servicos','impostos_taxas','equipamentos_bens','outros'
    ));
