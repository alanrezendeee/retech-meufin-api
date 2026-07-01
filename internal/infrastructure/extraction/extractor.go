// Package extraction fornece a porta/adaptador de extração de dados de exames
// a partir de PDFs/imagens (OCR/LLM). O provider é configurável por env, com
// fallback controlado para "disabled" quando não há chave ou provider inválido.
package extraction

import (
	"context"
	"encoding/json"
	"os"
)

// InputType indica o tipo do conteúdo enviado para extração.
const (
	InputTypePDF   = "pdf"
	InputTypeImage = "image"
)

// Providers suportados.
const (
	ProviderAnthropic = "anthropic"
	ProviderDisabled  = "disabled"
)

// Defaults de configuração.
const (
	DefaultModel   = "claude-opus-4-8"
	DefaultBaseURL = "https://api.anthropic.com"
)

// ExtractInput carrega o conteúdo bruto de um documento a ser extraído.
type ExtractInput struct {
	InputType string // pdf|image
	MimeType  string // ex.: application/pdf, image/png, image/jpeg
	Content   []byte // conteúdo bruto do arquivo
}

// ExtractResult é o resultado da extração produzido por um Extractor.
type ExtractResult struct {
	Text           string          // texto livre/summary opcional
	StructuredJSON json.RawMessage // JSON estruturado no schema alvo de exames
	Model          string          // modelo efetivamente utilizado
	PromptVersion  string          // versão do prompt utilizado
	RawResponse    json.RawMessage // resposta bruta do provider (para auditoria)
}

// Extractor é a porta de extração. Implementações: AnthropicExtractor e
// DisabledExtractor.
type Extractor interface {
	// Enabled indica se a extração está operacional (provider + credenciais ok).
	Enabled() bool
	// Provider retorna o identificador do provider ("anthropic"|"disabled"|...).
	Provider() string
	// Extract executa a extração do documento.
	Extract(ctx context.Context, in ExtractInput) (ExtractResult, error)
}

// Config define os parâmetros de configuração do extrator.
type Config struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

// ConfigFromEnv monta a Config a partir das variáveis de ambiente:
//
//	HEALTH_EXTRACTION_PROVIDER  (anthropic|disabled) — default "disabled"
//	HEALTH_EXTRACTION_MODEL     — default "claude-opus-4-8"
//	HEALTH_EXTRACTION_API_KEY   — chave do provider
//	HEALTH_EXTRACTION_BASE_URL  — default "https://api.anthropic.com"
func ConfigFromEnv() Config {
	cfg := Config{
		Provider: getenvDefault("HEALTH_EXTRACTION_PROVIDER", ProviderDisabled),
		Model:    getenvDefault("HEALTH_EXTRACTION_MODEL", DefaultModel),
		APIKey:   os.Getenv("HEALTH_EXTRACTION_API_KEY"),
		BaseURL:  getenvDefault("HEALTH_EXTRACTION_BASE_URL", DefaultBaseURL),
	}
	return cfg
}

// New devolve o Extractor adequado à configuração. Retorna o adaptador
// Anthropic apenas quando provider=anthropic E APIKey não vazia; caso
// contrário devolve o DisabledExtractor (fallback controlado).
func New(cfg Config) Extractor {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Provider == ProviderAnthropic && cfg.APIKey != "" {
		return NewAnthropicExtractor(cfg)
	}
	return NewDisabledExtractor()
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
