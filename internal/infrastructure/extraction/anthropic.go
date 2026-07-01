package extraction

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PromptVersion identifica a versão do prompt de extração (para auditoria e
// reprocessamento). Incremente ao alterar o prompt/schema.
const PromptVersion = "exam-extract-v1"

const anthropicVersion = "2023-06-01"
const extractionToolName = "registrar_exame"

// extractionSystemPrompt orienta o modelo. Reforça: apenas EXTRAÇÃO, sem
// diagnóstico ou interpretação clínica.
const extractionSystemPrompt = `Você é um extrator de dados estruturados de laudos e resultados de exames laboratoriais.
Sua ÚNICA tarefa é TRANSCREVER e ESTRUTURAR os dados presentes no documento.

REGRAS OBRIGATÓRIAS:
- NÃO diagnostique, NÃO interprete clinicamente e NÃO faça recomendações médicas.
- Extraia apenas o que está literalmente no documento. Não invente valores.
- Se um campo não estiver presente, deixe-o vazio/nulo. Não preencha por dedução clínica.
- Preserve os valores exatamente como aparecem (incluindo unidades e faixas de referência).
- Quando houver valor numérico, também preencha numeric_value com o número correspondente.
- Registre no campo warnings qualquer ambiguidade, ilegibilidade ou dado faltante relevante.

Use SEMPRE a ferramenta ` + extractionToolName + ` para retornar o resultado estruturado.`

// AnthropicExtractor implementa Extractor via HTTP contra a Messages API.
type AnthropicExtractor struct {
	cfg    Config
	client *http.Client
}

// NewAnthropicExtractor cria o adaptador Anthropic.
func NewAnthropicExtractor(cfg Config) *AnthropicExtractor {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	return &AnthropicExtractor{
		cfg:    cfg,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (a *AnthropicExtractor) Enabled() bool    { return a.cfg.APIKey != "" }
func (a *AnthropicExtractor) Provider() string { return ProviderAnthropic }

// --- schema alvo do StructuredJSON (tool input_schema) ---

func extractionInputSchema() map[string]any {
	examItem := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"exam_name":      map[string]any{"type": "string"},
			"marker_name":    map[string]any{"type": "string"},
			"result_value":   map[string]any{"type": "string"},
			"numeric_value":  map[string]any{"type": []string{"number", "null"}},
			"unit":           map[string]any{"type": "string"},
			"reference_min":  map[string]any{"type": []string{"number", "null"}},
			"reference_max":  map[string]any{"type": []string{"number", "null"}},
			"reference_text": map[string]any{"type": "string"},
			"material":       map[string]any{"type": "string"},
			"method":         map[string]any{"type": "string"},
			"interpretation": map[string]any{"type": "string"},
			"raw_text":       map[string]any{"type": "string"},
		},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"patient_name":    map[string]any{"type": "string"},
			"exam_date":       map[string]any{"type": "string"},
			"collection_date": map[string]any{"type": "string"},
			"release_date":    map[string]any{"type": "string"},
			"laboratory_name": map[string]any{"type": "string"},
			"doctor_name":     map[string]any{"type": "string"},
			"exams": map[string]any{
				"type":  "array",
				"items": examItem,
			},
			"summary":  map[string]any{"type": "string"},
			"warnings": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"exams"},
	}
}

// --- request/response wire types ---

type anthropicSource struct {
	Type      string `json:"type"`       // base64
	MediaType string `json:"media_type"` // application/pdf, image/png...
	Data      string `json:"data"`       // base64
}

type anthropicContentBlock struct {
	Type   string           `json:"type"`             // text|image|document|tool_use
	Text   string           `json:"text,omitempty"`   // type=text
	Source *anthropicSource `json:"source,omitempty"` // type=image|document
	// campos de resposta (type=tool_use)
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type anthropicRequest struct {
	Model      string               `json:"model"`
	MaxTokens  int                  `json:"max_tokens"`
	System     string               `json:"system,omitempty"`
	Tools      []anthropicTool      `json:"tools,omitempty"`
	ToolChoice *anthropicToolChoice `json:"tool_choice,omitempty"`
	Messages   []anthropicMessage   `json:"messages"`
}

type anthropicResponse struct {
	Content    []anthropicContentBlock `json:"content"`
	Model      string                  `json:"model"`
	StopReason string                  `json:"stop_reason"`
	Type       string                  `json:"type"`
	Error      *anthropicAPIError      `json:"error,omitempty"`
}

type anthropicAPIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Extract envia o documento (PDF via document, imagem via image) e solicita
// saída estruturada via tool use.
func (a *AnthropicExtractor) Extract(ctx context.Context, in ExtractInput) (ExtractResult, error) {
	if !a.Enabled() {
		return ExtractResult{}, ErrExtractionDisabled
	}
	if len(in.Content) == 0 {
		return ExtractResult{}, fmt.Errorf("conteúdo do documento vazio")
	}

	docBlock, err := buildDocumentBlock(in)
	if err != nil {
		return ExtractResult{}, err
	}

	profile := in.Profile
	if profile == nil {
		p := LabExamProfile()
		profile = &p
	}

	reqBody := anthropicRequest{
		Model:     a.cfg.Model,
		MaxTokens: 8192,
		System:    profile.SystemPrompt,
		Tools: []anthropicTool{{
			Name:        profile.ToolName,
			Description: profile.ToolDescription,
			InputSchema: profile.InputSchema,
		}},
		ToolChoice: &anthropicToolChoice{Type: "tool", Name: profile.ToolName},
		Messages: []anthropicMessage{{
			Role: "user",
			Content: []anthropicContentBlock{
				docBlock,
				{Type: "text", Text: profile.UserInstruction},
			},
		}},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("montar payload: %w", err)
	}

	url := a.cfg.BaseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return ExtractResult{}, fmt.Errorf("montar requisição: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", a.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("chamada à Anthropic: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("ler resposta: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr anthropicResponse
		_ = json.Unmarshal(rawBody, &apiErr)
		if apiErr.Error != nil {
			return ExtractResult{}, fmt.Errorf("anthropic %d (%s): %s", resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return ExtractResult{}, fmt.Errorf("anthropic status %d: %s", resp.StatusCode, string(rawBody))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return ExtractResult{}, fmt.Errorf("decodificar resposta: %w", err)
	}

	result := ExtractResult{
		Model:         parsed.Model,
		PromptVersion: profile.PromptVersion,
		RawResponse:   json.RawMessage(rawBody),
	}
	if result.Model == "" {
		result.Model = a.cfg.Model
	}

	// Extrai o input do tool_use e um eventual texto livre.
	for _, block := range parsed.Content {
		switch block.Type {
		case "tool_use":
			if block.Name == profile.ToolName && len(block.Input) > 0 {
				result.StructuredJSON = block.Input
			}
		case "text":
			if result.Text == "" {
				result.Text = block.Text
			}
		}
	}

	if len(result.StructuredJSON) == 0 {
		return result, fmt.Errorf("resposta sem dados estruturados (stop_reason=%s)", parsed.StopReason)
	}
	return result, nil
}

// buildDocumentBlock monta o bloco de conteúdo apropriado (document p/ PDF,
// image p/ imagem), sempre em base64.
func buildDocumentBlock(in ExtractInput) (anthropicContentBlock, error) {
	b64 := base64.StdEncoding.EncodeToString(in.Content)
	switch in.InputType {
	case InputTypePDF:
		mt := in.MimeType
		if mt == "" {
			mt = "application/pdf"
		}
		return anthropicContentBlock{
			Type:   "document",
			Source: &anthropicSource{Type: "base64", MediaType: mt, Data: b64},
		}, nil
	case InputTypeImage:
		mt := in.MimeType
		if mt == "" {
			mt = "image/png"
		}
		return anthropicContentBlock{
			Type:   "image",
			Source: &anthropicSource{Type: "base64", MediaType: mt, Data: b64},
		}, nil
	default:
		return anthropicContentBlock{}, fmt.Errorf("input_type inválido: %q (esperado pdf|image)", in.InputType)
	}
}
