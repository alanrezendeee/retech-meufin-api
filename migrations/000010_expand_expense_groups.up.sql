-- Expansão do catálogo curado de grupos (plano de contas familiar completo,
-- alinhado à POF/IBGE e aos apps de referência). Grupos seguem fixos do
-- sistema; categorias do usuário se vinculam a eles.
ALTER TABLE finance_expense_categories DROP CONSTRAINT IF EXISTS finance_expense_categories_group_slug_check;
ALTER TABLE finance_expense_categories
    ADD CONSTRAINT finance_expense_categories_group_slug_check CHECK (group_slug IN (
        'moradia','alimentacao','transporte','saude','educacao',
        'lazer','viagens','vestuario','cuidados_pessoais','pets',
        'presentes_doacoes','contas_servicos','seguros_protecao','impostos_taxas',
        'dividas_financiamentos','equipamentos_bens','trabalho_negocio',
        'investimentos','outros'
    ));
