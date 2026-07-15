// Package authsync sincroniza o manifesto de permissions do MeuFin com o
// retech-auth-api no boot. Tela nova no admin = entra no manifesto aqui =
// permission aparece no banco do auth no próximo deploy — sem SQL manual.
package authsync

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config vem das envs:
//
//	AUTH_SYNC_URL          URL completa do endpoint (ex.: https://auth.../v1/applications/sync)
//	AUTH_BOOTSTRAP_SECRET  secret compartilhado do HMAC (mesmo BOOTSTRAP_SECRET do auth)
type Config struct {
	URL    string
	Secret string
}

func ConfigFromEnv() Config {
	return Config{
		URL:    strings.TrimSpace(os.Getenv("AUTH_SYNC_URL")),
		Secret: strings.TrimSpace(os.Getenv("AUTH_BOOTSTRAP_SECRET")),
	}
}

// Enabled indica se o sync está configurado.
func (c Config) Enabled() bool { return c.URL != "" && c.Secret != "" }

type permission struct {
	Code        string `json:"code"`
	Subject     string `json:"subject"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
}

type manifest struct {
	Application struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	} `json:"application"`
	Permissions []permission `json:"permissions"`
	Roles       []any        `json:"roles"`
}

func perm(subject, action, description string) permission {
	return permission{Code: subject + ":" + action, Subject: subject, Action: action, Description: description}
}

// buildManifest é a lista canônica de subjects do MeuFin — espelho do que o
// front referencia (rotas guarded + menu). Tela nova => adicionar aqui.
func buildManifest() manifest {
	var m manifest
	m.Application.Code = "meufin"
	m.Application.Name = "Meu Fin"
	m.Application.Description = "Gestão financeira e de saúde familiar"
	m.Roles = []any{}
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

// Sync envia o manifesto ao auth com assinatura HMAC-SHA256(body+timestamp).
// Retorna o resumo (criadas/existentes) para log.
func Sync(ctx context.Context, cfg Config) (string, error) {
	body, err := json.Marshal(buildManifest())
	if err != nil {
		return "", fmt.Errorf("serializar manifesto: %w", err)
	}

	timestamp := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(cfg.Secret))
	mac.Write(body)
	mac.Write([]byte(fmt.Sprintf("%d", timestamp)))
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("montar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("chamada ao auth: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("auth respondeu %d: %s", resp.StatusCode, string(raw))
	}

	// Resumo compacto pro log: quantas permissions criadas vs já existentes.
	var parsed struct {
		Permissions []struct {
			Action string `json:"action"`
		} `json:"permissions"`
	}
	created := 0
	if err := json.Unmarshal(raw, &parsed); err == nil {
		for _, p := range parsed.Permissions {
			if p.Action == "created" {
				created++
			}
		}
		return fmt.Sprintf("%d permissions no manifesto, %d criadas agora", len(parsed.Permissions), created), nil
	}
	return "sincronizado", nil
}
