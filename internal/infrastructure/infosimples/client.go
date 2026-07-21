// Package infosimples é o adaptador para a API de consultas públicas da
// Infosimples. Hoje expõe a consulta unificada de NFC-e (cupom fiscal), que
// resolve o captcha da SEFAZ e devolve os itens da nota em JSON estruturado.
//
// Token e base URL vêm de env (padrão por-adapter do projeto):
//
//	INFOSIMPLES_TOKEN     — token da conta (segredo; nunca versionar)
//	INFOSIMPLES_BASE_URL  — default https://api.infosimples.com
//	INFOSIMPLES_TIMEOUT_S — timeout da consulta em segundos (default 90)
package infosimples

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL  = "https://api.infosimples.com"
	nfcePath        = "/api/v2/consultas/sefaz/nfce"
	defaultTimeoutS = 90
)

// Config define os parâmetros do adaptador.
type Config struct {
	Token      string
	BaseURL    string
	TimeoutSec int
}

// ConfigFromEnv monta a Config a partir das variáveis de ambiente.
func ConfigFromEnv() Config {
	timeout := defaultTimeoutS
	if v := strings.TrimSpace(os.Getenv("INFOSIMPLES_TIMEOUT_S")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = n
		}
	}
	base := strings.TrimSpace(os.Getenv("INFOSIMPLES_BASE_URL"))
	if base == "" {
		base = defaultBaseURL
	}
	return Config{
		Token:      strings.TrimSpace(os.Getenv("INFOSIMPLES_TOKEN")),
		BaseURL:    base,
		TimeoutSec: timeout,
	}
}

// NFCeProduto é um item da nota, com quantidade em milésimos e valores em centavos.
type NFCeProduto struct {
	Codigo        string
	Descricao     string
	QuantityMilli int64
	UnitCents     int64
	AmountCents   int64
	Unidade       string
}

// NFCeResult é o resultado normalizado da consulta de uma NFC-e.
type NFCeResult struct {
	ChaveAcesso      string
	EmitenteCNPJ     string
	EmitenteNome     string
	EmitenteEndereco string
	DataEmissao      string // YYYY-MM-DD ("" quando indisponível)
	ValorTotalCents  int64
	TributosCents    int64
	Produtos         []NFCeProduto
	// PaymentMethod é a forma de pagamento normalizada (credito|debito|dinheiro|
	// pix|outros); "" quando indisponível. Base para conciliar cupom × fatura.
	PaymentMethod string
	Warnings      []string
	// PriceBRL é o custo cobrado pela Infosimples nesta consulta (auditoria).
	PriceBRL string
}

// Client é o cliente HTTP da Infosimples.
type Client struct {
	cfg  Config
	http *http.Client
}

// New constrói o cliente. Timeout do HTTP com margem sobre o timeout da consulta.
func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = defaultTimeoutS
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: time.Duration(cfg.TimeoutSec+15) * time.Second},
	}
}

// Enabled indica se há token configurado.
func (c *Client) Enabled() bool { return c.cfg.Token != "" }

// Provider identifica a origem do dado.
func (c *Client) Provider() string { return "sefaz" }

// respostaNFCe espelha o envelope da Infosimples (campos usados).
type respostaNFCe struct {
	Code        int             `json:"code"`
	CodeMessage string          `json:"code_message"`
	Header      json.RawMessage `json:"header"`
	Data        []dadoNFCe      `json:"data"`
	Errors      []string        `json:"errors"`
}

type headerNFCe struct {
	Price string `json:"price"`
}

type dadoNFCe struct {
	Emitente struct {
		CNPJ      string `json:"cnpj"`
		NomeRazao string `json:"nome_razao_social"`
		Endereco  string `json:"endereco"`
	} `json:"emitente"`
	ChaveAcesso              string          `json:"chave_acesso"`
	InformacoesNota          json.RawMessage `json:"informacoes_nota"`
	NormalizadoValorAPagar   *float64        `json:"normalizado_valor_a_pagar"`
	NormalizadoValorTotal    *float64        `json:"normalizado_valor_total"`
	NormalizadoTributosTotal *float64        `json:"normalizado_tributos_totais"`
	Produtos                 []produtoNFCe   `json:"produtos"`
	// FormasPagamento varia por estado; parseado de forma tolerante (só o texto).
	FormasPagamento json.RawMessage `json:"formas_pagamento"`
}

type produtoNFCe struct {
	Codigo                       string   `json:"codigo"`
	Nome                         string   `json:"nome"`
	Unidade                      string   `json:"unidade"`
	NormalizadoQuantidade        *float64 `json:"normalizado_quantidade"`
	NormalizadoValorUnitario     *float64 `json:"normalizado_valor_unitario"`
	NormalizadoValorTotalProduto *float64 `json:"normalizado_valor_total_produto"`
}

// ConsultarNFCe consulta uma NFC-e pela chave de acesso (44 dígitos) ou pela URL
// do QR Code. Retorna o resultado normalizado ou um erro amigável em caso de
// falha da consulta (SEFAZ fora, chave inválida, etc.).
func (c *Client) ConsultarNFCe(ctx context.Context, nfce string) (*NFCeResult, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("infosimples: token não configurado")
	}
	nfce = strings.TrimSpace(nfce)
	if nfce == "" {
		return nil, fmt.Errorf("infosimples: nfce (chave ou URL) é obrigatório")
	}

	form := url.Values{}
	form.Set("token", c.cfg.Token)
	form.Set("nfce", nfce)
	form.Set("timeout", strconv.Itoa(c.cfg.TimeoutSec))

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + nfcePath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("infosimples: montar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("infosimples: consulta NFC-e falhou: %w", err)
	}
	defer resp.Body.Close()

	var body respostaNFCe
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("infosimples: resposta inválida: %w", err)
	}

	// code 200 = sucesso; qualquer outro é erro de negócio da consulta.
	if body.Code != 200 {
		msg := body.CodeMessage
		if msg == "" && len(body.Errors) > 0 {
			msg = strings.Join(body.Errors, "; ")
		}
		if msg == "" {
			msg = fmt.Sprintf("code %d", body.Code)
		}
		return nil, fmt.Errorf("infosimples: consulta NFC-e não concluída: %s", msg)
	}
	if len(body.Data) == 0 {
		return nil, fmt.Errorf("infosimples: consulta NFC-e sem dados")
	}

	d := body.Data[0]
	out := &NFCeResult{
		ChaveAcesso:      firstNonEmpty(d.ChaveAcesso, digitsOnly(nfce)),
		EmitenteCNPJ:     d.Emitente.CNPJ,
		EmitenteNome:     d.Emitente.NomeRazao,
		EmitenteEndereco: d.Emitente.Endereco,
		DataEmissao:      extrairDataEmissao(d.InformacoesNota),
		ValorTotalCents:  reaisToCents(firstNonNil(d.NormalizadoValorAPagar, d.NormalizadoValorTotal)),
		TributosCents:    reaisToCents(d.NormalizadoTributosTotal),
		Produtos:         make([]NFCeProduto, 0, len(d.Produtos)),
		PaymentMethod:    detectPaymentMethod(rawText(d.FormasPagamento)),
		Warnings:         []string{},
	}
	var h headerNFCe
	if len(body.Header) > 0 {
		_ = json.Unmarshal(body.Header, &h)
		out.PriceBRL = h.Price
	}
	for _, p := range d.Produtos {
		qty := int64(1000) // default 1 unidade quando não normalizado
		if p.NormalizadoQuantidade != nil {
			qty = int64(math.Round(*p.NormalizadoQuantidade * 1000))
		}
		out.Produtos = append(out.Produtos, NFCeProduto{
			Codigo:        p.Codigo,
			Descricao:     strings.TrimSpace(p.Nome),
			QuantityMilli: qty,
			UnitCents:     reaisToCents(p.NormalizadoValorUnitario),
			AmountCents:   reaisToCents(p.NormalizadoValorTotalProduto),
			Unidade:       p.Unidade,
		})
	}
	return out, nil
}

// extrairDataEmissao tenta achar uma data (YYYY-MM-DD) dentro de informacoes_nota,
// cujo formato varia por estado. Tolera ausência retornando "".
func extrairDataEmissao(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, k := range []string{"data_emissao", "data_autorizacao", "emissao", "data"} {
		if v, ok := m[k].(string); ok {
			if d := parseDataBR(v); d != "" {
				return d
			}
		}
	}
	return ""
}

// parseDataBR normaliza datas comuns da SEFAZ para YYYY-MM-DD.
func parseDataBR(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Recorta a parte de data quando vem com hora ("16/07/2026 18:48:02").
	if i := strings.IndexByte(raw, ' '); i > 0 {
		raw = raw[:i]
	}
	layouts := []string{"2006-01-02", "02/01/2006", "02-01-2006", "02/01/06"}
	for _, l := range layouts {
		if t, err := time.Parse(l, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

func reaisToCents(v *float64) int64 {
	if v == nil {
		return 0
	}
	return int64(math.Round(*v * 100))
}

// rawText concatena todos os valores string de um JSON arbitrário (o formato de
// formas_pagamento varia por estado) — usado só para detectar palavras-chave.
func rawText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	var b strings.Builder
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case string:
			b.WriteString(t)
			b.WriteByte(' ')
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(v)
	return b.String()
}

// detectPaymentMethod normaliza a forma de pagamento a partir de texto livre.
// Prioriza cartão (crédito/débito) e pix; "" quando não reconhece.
func detectPaymentMethod(s string) string {
	l := strings.ToLower(s)
	switch {
	case l == "":
		return ""
	case strings.Contains(l, "credito") || strings.Contains(l, "crédito"):
		return "credito"
	case strings.Contains(l, "debito") || strings.Contains(l, "débito"):
		return "debito"
	case strings.Contains(l, "pix"):
		return "pix"
	case strings.Contains(l, "dinheiro") || strings.Contains(l, "espécie") || strings.Contains(l, "especie"):
		return "dinheiro"
	default:
		return "outros"
	}
}

func firstNonNil(vs ...*float64) *float64 {
	for _, v := range vs {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	// Uma chave de acesso tem 44 dígitos; URLs de QR contêm a chave embutida.
	out := b.String()
	if len(out) >= 44 {
		return out[:44]
	}
	return out
}
