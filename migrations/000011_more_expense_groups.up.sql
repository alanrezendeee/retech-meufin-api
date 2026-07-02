-- Grupos expostos pela planilha real do Alan: pensão alimentícia e serviços
-- de profissionais/pessoas não encaixavam em nenhum grupo (cairiam em Outros).
ALTER TABLE finance_expense_categories DROP CONSTRAINT IF EXISTS finance_expense_categories_group_slug_check;
ALTER TABLE finance_expense_categories
    ADD CONSTRAINT finance_expense_categories_group_slug_check CHECK (group_slug IN (
        'moradia','alimentacao','transporte','saude','educacao',
        'lazer','viagens','vestuario','cuidados_pessoais','pets',
        'presentes_doacoes','contas_servicos','seguros_protecao','impostos_taxas',
        'dividas_financiamentos','equipamentos_bens','trabalho_negocio',
        'familia_dependentes','servicos_profissionais',
        'investimentos','outros'
    ));
