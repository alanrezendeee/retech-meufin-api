package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// KeyReaderPromptVersion versiona o prompt de leitura da chave.
const KeyReaderPromptVersion = "fiscal-key-v1"

const keyReaderToolName = "extrair_chave_nfce"

// FiscalKeyReader lê a CHAVE DE ACESSO (44 dígitos) impressa num cupom/nota a
// partir da imagem — fallback quando o QR Code está ilegível (a chave impressa
// costuma sobreviver melhor ao borrão/compressão que o QR).
type FiscalKeyReader interface {
	Enabled() bool
	ReadFiscalKey(ctx context.Context, content []byte, mimeType string) (string, error)
}

// NewKeyReader devolve o leitor Anthropic quando há credencial, senão um
// leitor desabilitado (no-op).
func NewKeyReader(cfg Config) FiscalKeyReader {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Provider == ProviderAnthropic && cfg.APIKey != "" {
		return NewAnthropicExtractor(cfg)
	}
	return disabledKeyReader{}
}

type disabledKeyReader struct{}

func (disabledKeyReader) Enabled() bool { return false }
func (disabledKeyReader) ReadFiscalKey(context.Context, []byte, string) (string, error) {
	return "", ErrExtractionDisabled
}

const keyReaderSystemPrompt = `Você extrai a CHAVE DE ACESSO de um cupom/nota fiscal brasileiro (NFC-e/NF-e) a partir da imagem.
A chave tem EXATAMENTE 44 dígitos, geralmente impressa perto do QR Code (às vezes em grupos de 4, ex.: "4226 0779 2572 ...").
Regras:
- Transcreva SOMENTE os 44 dígitos, sem espaços nem outros caracteres.
- Não confunda com CNPJ, número do documento, protocolo ou valor.
- Se não houver uma chave de 44 dígitos legível, retorne "".
Use SEMPRE a ferramenta ` + keyReaderToolName + `.`

func keyReaderInputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"chave_acesso": map[string]any{"type": "string", "description": "44 dígitos, sem espaços; \"\" se ausente/ilegível"},
		},
		"required": []string{"chave_acesso"},
	}
}

type keyReaderOutput struct {
	ChaveAcesso string `json:"chave_acesso"`
}

// ReadFiscalKey implementa FiscalKeyReader no AnthropicExtractor.
func (a *AnthropicExtractor) ReadFiscalKey(ctx context.Context, content []byte, mimeType string) (string, error) {
	if !a.Enabled() {
		return "", ErrExtractionDisabled
	}
	if len(content) == 0 {
		return "", fmt.Errorf("keyreader: conteúdo vazio")
	}
	inputType := InputTypeImage
	if strings.EqualFold(mimeType, "application/pdf") {
		inputType = InputTypePDF
	}
	docBlock, err := buildDocumentBlock(ExtractInput{InputType: inputType, MimeType: mimeType, Content: content})
	if err != nil {
		return "", err
	}

	reqBody := anthropicRequest{
		Model:     a.cfg.Model,
		MaxTokens: 512,
		System:    keyReaderSystemPrompt,
		Tools: []anthropicTool{{
			Name:        keyReaderToolName,
			Description: "Extrai a chave de acesso (44 dígitos) impressa no cupom/nota fiscal.",
			InputSchema: keyReaderInputSchema(),
		}},
		ToolChoice: &anthropicToolChoice{Type: "tool", Name: keyReaderToolName},
		Messages: []anthropicMessage{{
			Role: "user",
			Content: []anthropicContentBlock{
				docBlock,
				{Type: "text", Text: "Extraia a chave de acesso de 44 dígitos impressa neste cupom."},
			},
		}},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("keyreader: montar payload: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.BaseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("keyreader: montar requisição: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", a.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("keyreader: chamada à Anthropic: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("keyreader: ler resposta: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr anthropicResponse
		_ = json.Unmarshal(rawBody, &apiErr)
		if apiErr.Error != nil {
			return "", fmt.Errorf("keyreader: anthropic %d (%s): %s", resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return "", fmt.Errorf("keyreader: anthropic status %d", resp.StatusCode)
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return "", fmt.Errorf("keyreader: decodificar resposta: %w", err)
	}
	for _, block := range parsed.Content {
		if block.Type == "tool_use" && block.Name == keyReaderToolName && len(block.Input) > 0 {
			var out keyReaderOutput
			if err := json.Unmarshal(block.Input, &out); err != nil {
				return "", fmt.Errorf("keyreader: decodificar tool: %w", err)
			}
			return strings.TrimSpace(out.ChaveAcesso), nil
		}
	}
	return "", nil
}
