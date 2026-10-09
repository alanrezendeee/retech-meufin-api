// Package authsync sincroniza o manifesto de permissions do MeuFin com o
// retech-auth-api no boot (retech-authkit/authsync). Tela nova no admin = entra
// no manifesto aqui = permission aparece no banco do auth no próximo deploy.
package authsync

import (
	"context"

	"github.com/theretechlabs/retech-authkit/authclient"
	"github.com/theretechlabs/retech-authkit/authsync"
)

type permission = authsync.Permission

func perm(subject, action, description string) permission {
	return authsync.Perm(subject, action, description)
}

// buildManifest é a lista canônica de subjects do MeuFin — espelho do que o
// front referencia (rotas guarded + menu). Tela nova => adicionar aqui.
func buildManifest() authsync.Manifest {
	m := authsync.NewManifest("meufin", "Meu Fin", "Gestão financeira e de saúde familiar")
	m.Permissions = []permission{
		// Home
		perm("retechfin.dashboard", "view", "Home do painel"),

		// Financeiro
		perm("finance.dashboard", "view", "Dashboard financeira"),
		perm("finance.payables", "view", "Contas do Dia (a pagar/receber)"),
		perm("finance.payables", "manage", "Liquidar lançamentos e anexar comprovantes"),
		perm("finance.income", "view", "Receitas"),
		perm("finance.income", "manage", "Gerenciar receitas"),
		perm("finance.expenses", "view", "Despesas"),
		perm("finance.expenses", "manage", "Gerenciar despesas"),
		perm("finance.sources", "view", "Fontes de receita"),
		perm("finance.sources", "manage", "Gerenciar fontes de receita"),
		perm("finance.cards", "view", "Cartões de crédito"),
		perm("finance.cards", "manage", "Gerenciar cartões"),
		perm("finance.invoices", "view", "Faturas (import por PDF)"),
		perm("finance.invoices", "manage", "Importar e confirmar faturas"),
		perm("finance.accounts", "view", "Contas (corrente/poupança/carteira)"),
		perm("finance.accounts", "manage", "Gerenciar contas"),
		perm("finance.categories", "view", "Categorias de despesa"),
		perm("finance.categories", "manage", "Gerenciar categorias de despesa"),

		// Saúde Familiar
		perm("health.dashboard", "view", "Dashboard de saúde"),
		perm("health.family_members", "view", "Membros da família (inclui documentos pessoais)"),
		perm("health.family_members", "manage", "Gerenciar membros e documentos"),
		perm("health.labs", "view", "Laboratórios"),
		perm("health.labs", "manage", "Gerenciar laboratórios"),
		perm("health.markers", "view", "Catálogo de exames (marcadores)"),
		perm("health.markers", "manage", "Gerenciar marcadores"),
		perm("health.results", "view", "Resultados de exames"),
		perm("health.results", "manage", "Gerenciar resultados"),
		perm("health.documents", "view", "Documentos de saúde"),
		perm("health.documents", "manage", "Gerenciar documentos de saúde"),
		perm("health.documents", "view", "Documentos de saúde (item futuro do menu)"),
		perm("health.appointments", "view", "Consultas e agenda de saúde"),
		perm("health.appointments", "manage", "Agendar, realizar e cancelar consultas"),
		perm("health.plans", "view", "Planos de saúde"),
		perm("health.plans", "manage", "Gerenciar planos de saúde e carteirinhas"),

		// Administração (IAM)
		perm("admin.users", "view", "Ver usuários"),
		perm("admin.users", "manage", "Gerenciar usuários"),
		perm("admin.roles", "view", "Ver grupos e permissões"),
		perm("admin.roles", "manage", "Gerenciar grupos"),
		perm("admin.permissions", "view", "Ver catálogo de permissões"),
		perm("admin.permissions", "manage", "Gerenciar catálogo de permissões"),

		// Frota Familiar
		perm("vehicles.dashboard", "view", "Dashboard da frota"),
		perm("vehicles.list", "view", "Listar veículos da frota"),
		perm("vehicles.list", "manage", "Cadastrar e remover veículos"),
		perm("vehicles.detail", "view", "Ver detalhes de um veículo"),
		perm("vehicles.detail", "manage", "Editar veículo, registrar manutenções e agendamentos"),

		// Dashboard fiscal (notas/cupons — inflação por item)
		perm("finance.fiscal-dashboard", "view", "Dashboard fiscal (notas/cupons)"),

		// Patrimônio
		perm("patrimony.dashboard", "view", "Dashboard de patrimônio"),
		perm("patrimony.properties", "view", "Imóveis"),
		perm("patrimony.properties", "manage", "Gerenciar imóveis e documentos"),
		perm("patrimony.taxes", "view", "Impostos de bens"),
		perm("patrimony.taxes", "manage", "Gerenciar impostos e pagamentos"),

		// Garantias
		perm("warranties.list", "view", "Listar garantias de bens"),
		perm("warranties.list", "manage", "Cadastrar, editar e remover garantias e documentos"),
		perm("warranties.summary", "view", "Resumo de garantias (vigentes, expirando, valor coberto)"),

		// Educação / Material Escolar
		perm("education.dashboard", "view", "Dashboard de educação"),
		perm("education.enrollments", "view", "Matrículas escolares"),
		perm("education.enrollments", "manage", "Gerenciar matrículas"),
		perm("education.lists", "view", "Listas de material escolar"),
		perm("education.lists", "manage", "Gerenciar listas e itens de material"),

		// Segurança do Lar
		perm("homesafety.dashboard", "view", "Dashboard de segurança do lar"),
		perm("homesafety.items", "view", "Listar itens de segurança do lar"),
		perm("homesafety.items", "manage", "Cadastrar, editar e registrar manutenções dos itens"),

		// Placeholders do menu legado ("em breve")
		perm("retechfin.transactions", "view", "Transações (em breve)"),
		perm("retechfin.accounts", "view", "Contas legado (em breve)"),
		perm("retechfin.categories", "view", "Categorias (em breve)"),
		perm("retechfin.cards", "view", "Cartões legado (em breve)"),
		perm("retechfin.goals", "view", "Metas (em breve)"),
		perm("retechfin.settings", "view", "Configurações (em breve)"),
	}
	return m
}

// Sync envia o manifesto assinado (HMAC + nonce) ao auth. Retorna o resumo para log.
func Sync(ctx context.Context, c *authclient.Client) (string, error) {
	res, err := authsync.Sync(ctx, c, buildManifest())
	if err != nil {
		return "", err
	}
	return res.String(), nil
}
