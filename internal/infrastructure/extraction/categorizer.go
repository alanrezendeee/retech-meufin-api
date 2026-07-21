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

// CategorizerPromptVersion versiona o prompt de categorização.
const CategorizerPromptVersion = "fiscal-categorize-v1"

const categorizeToolName = "classificar_itens"

// CategoryOption é uma opção apresentada ao classificador (categoria existente
// da tenant, ou grupo global).
type CategoryOption struct {
	Slug  string
	Name  string
	Group string // grupo global ao qual a categoria pertence (vazio para grupos)
}

// CategorizeInput pede a classificação de uma lista de descrições de itens.
type CategorizeInput struct {
	Descriptions []string
	Categories   []CategoryOption // categorias EXISTENTES da tenant (reuse-first)
	Groups       []CategoryOption // grupos globais válidos (Slug + Name)
}

// CategorizedItem é a classificação de um item (por índice).
type CategorizedItem struct {
	Index   int
	Slug    string // slug da categoria escolhida/proposta
	NewName string // nome curto quando é proposta nova
	Group   string // grupo global
	IsNew   bool   // true = não está na lista da tenant (proposta)
}

// CategorizeResult é o resultado da classificação.
type CategorizeResult struct {
	Items []CategorizedItem
	Model string
}

// Categorizer classifica descrições de itens em categorias da tenant,
// preferindo reusar as existentes e só propondo novas quando necessário.
type Categorizer interface {
	Enabled() bool
	Categorize(ctx context.Context, in CategorizeInput) (CategorizeResult, error)
}

// NewCategorizer devolve o classificador Anthropic quando há credencial, senão
// um classificador desabilitado (no-op).
func NewCategorizer(cfg Config) Categorizer {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Provider == ProviderAnthropic && cfg.APIKey != "" {
		return NewAnthropicExtractor(cfg)
	}
	return disabledCategorizer{}
}

type disabledCategorizer struct{}

func (disabledCategorizer) Enabled() bool { return false }
func (disabledCategorizer) Categorize(context.Context, CategorizeInput) (CategorizeResult, error) {
	return CategorizeResult{}, ErrExtractionDisabled
}

const categorizeSystemPrompt = `Você classifica itens de cupons/notas fiscais em categorias de despesa.

REGRAS OBRIGATÓRIAS:
- PREFIRA SEMPRE reusar uma categoria da lista de "CATEGORIAS EXISTENTES" (retorne o slug dela, is_new=false).
- Só proponha uma categoria NOVA (is_new=true) quando NENHUMA existente servir razoavelmente.
- Ao propor nova: use um nome CURTO e GENÉRICO (ex.: "Padaria", não "Pão francês da manhã") para evitar duplicatas; escolha group_slug SEMPRE da lista de "GRUPOS GLOBAIS".
- group_slug é OBRIGATÓRIO em todo item (o da categoria existente, ou o da nova).
- Em caso de dúvida, use a categoria "outros".
- Classifique TODOS os itens, um resultado por índice.

Use SEMPRE a ferramenta ` + categorizeToolName + `.`

func categorizeInputSchema() map[string]any {
	item := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"index":         map[string]any{"type": "integer"},
			"category_slug": map[string]any{"type": "string"},
			"is_new":        map[string]any{"type": "boolean"},
			"new_name":      map[string]any{"type": "string"},
			"group_slug":    map[string]any{"type": "string"},
		},
		"required": []string{"index", "is_new", "group_slug"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{"type": "array", "items": item},
		},
		"required": []string{"items"},
	}
}

type categorizeToolOutput struct {
	Items []struct {
		Index        int    `json:"index"`
		CategorySlug string `json:"category_slug"`
		IsNew        bool   `json:"is_new"`
		NewName      string `json:"new_name"`
		GroupSlug    string `json:"group_slug"`
	} `json:"items"`
}

// Categorize implementa Categorizer no AnthropicExtractor (chamada texto→tool).
func (a *AnthropicExtractor) Categorize(ctx context.Context, in CategorizeInput) (CategorizeResult, error) {
	if !a.Enabled() {
		return CategorizeResult{}, ErrExtractionDisabled
	}
	if len(in.Descriptions) == 0 {
		return CategorizeResult{Model: a.cfg.Model}, nil
	}

	var b strings.Builder
	b.WriteString("CATEGORIAS EXISTENTES (reuse sempre que possível) — slug | nome | grupo:\n")
	for _, c := range in.Categories {
		fmt.Fprintf(&b, "- %s | %s | %s\n", c.Slug, c.Name, c.Group)
	}
	b.WriteString("\nGRUPOS GLOBAIS válidos — slug | nome:\n")
	for _, g := range in.Groups {
		fmt.Fprintf(&b, "- %s | %s\n", g.Slug, g.Name)
	}
	b.WriteString("\nITENS a classificar (índice: descrição):\n")
	for i, d := range in.Descriptions {
		fmt.Fprintf(&b, "%d: %s\n", i, d)
	}

	reqBody := anthropicRequest{
		Model:     a.cfg.Model,
		MaxTokens: 4096,
		System:    categorizeSystemPrompt,
		Tools: []anthropicTool{{
			Name:        categorizeToolName,
			Description: "Classifica cada item numa categoria de despesa (existente ou nova).",
			InputSchema: categorizeInputSchema(),
		}},
		ToolChoice: &anthropicToolChoice{Type: "tool", Name: categorizeToolName},
		Messages: []anthropicMessage{{
			Role:    "user",
			Content: []anthropicContentBlock{{Type: "text", Text: b.String()}},
		}},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return CategorizeResult{}, fmt.Errorf("categorizer: montar payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.BaseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return CategorizeResult{}, fmt.Errorf("categorizer: montar requisição: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", a.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return CategorizeResult{}, fmt.Errorf("categorizer: chamada à Anthropic: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CategorizeResult{}, fmt.Errorf("categorizer: ler resposta: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr anthropicResponse
		_ = json.Unmarshal(rawBody, &apiErr)
		if apiErr.Error != nil {
			return CategorizeResult{}, fmt.Errorf("categorizer: anthropic %d (%s): %s", resp.StatusCode, apiErr.Error.Type, apiErr.Error.Message)
		}
		return CategorizeResult{}, fmt.Errorf("categorizer: anthropic status %d", resp.StatusCode)
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return CategorizeResult{}, fmt.Errorf("categorizer: decodificar resposta: %w", err)
	}

	var toolOut categorizeToolOutput
	for _, block := range parsed.Content {
		if block.Type == "tool_use" && block.Name == categorizeToolName && len(block.Input) > 0 {
			if err := json.Unmarshal(block.Input, &toolOut); err != nil {
				return CategorizeResult{}, fmt.Errorf("categorizer: decodificar tool: %w", err)
			}
			break
		}
	}

	out := CategorizeResult{Model: parsed.Model, Items: make([]CategorizedItem, 0, len(toolOut.Items))}
	if out.Model == "" {
		out.Model = a.cfg.Model
	}
	for _, it := range toolOut.Items {
		out.Items = append(out.Items, CategorizedItem{
			Index:   it.Index,
			Slug:    strings.TrimSpace(it.CategorySlug),
			NewName: strings.TrimSpace(it.NewName),
			Group:   strings.TrimSpace(it.GroupSlug),
			IsNew:   it.IsNew,
		})
	}
	return out, nil
}
