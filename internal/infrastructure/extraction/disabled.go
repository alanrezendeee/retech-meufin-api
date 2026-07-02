package extraction

import (
	"context"
	"errors"
)

// ErrExtractionDisabled é retornado quando a extração está desabilitada
// (provider "disabled" ou ausência de credenciais). Erro controlado: a API
// sobe normalmente e a extração falha de forma previsível.
var ErrExtractionDisabled = errors.New("extração por LLM desabilitada (configure EXTRACTION_PROVIDER=anthropic e EXTRACTION_API_KEY)")

// DisabledExtractor é o extrator no-op usado como fallback.
type DisabledExtractor struct{}

// NewDisabledExtractor cria um extrator desabilitado.
func NewDisabledExtractor() *DisabledExtractor { return &DisabledExtractor{} }

func (d *DisabledExtractor) Enabled() bool    { return false }
func (d *DisabledExtractor) Provider() string { return ProviderDisabled }

func (d *DisabledExtractor) Extract(ctx context.Context, in ExtractInput) (ExtractResult, error) {
	return ExtractResult{}, ErrExtractionDisabled
}
