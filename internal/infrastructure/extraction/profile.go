package extraction

// ExtractProfile parametriza a extração por TIPO de documento (exame, fatura…),
// tornando o Extractor reutilizável entre módulos. Sem perfil, o adaptador usa
// o perfil de exame laboratorial (compatibilidade com o módulo Saúde).
type ExtractProfile struct {
	SystemPrompt    string
	ToolName        string
	ToolDescription string
	InputSchema     map[string]any
	PromptVersion   string
	UserInstruction string
}

// LabExamProfile é o perfil do módulo Saúde (laudos/exames laboratoriais).
func LabExamProfile() ExtractProfile {
	return ExtractProfile{
		SystemPrompt:    extractionSystemPrompt,
		ToolName:        extractionToolName,
		ToolDescription: "Registra os dados estruturados extraídos do laudo/exame laboratorial. Não interpreta clinicamente.",
		InputSchema:     extractionInputSchema(),
		PromptVersion:   PromptVersion,
		UserInstruction: "Extraia todos os dados do exame acima e retorne via ferramenta " + extractionToolName + ".",
	}
}

// InvoicePromptVersion versiona o prompt/schema de extração de faturas.
// v2: datas normalizadas para YYYY-MM-DD com inferência de ano.
const InvoicePromptVersion = "invoice-extract-v2"

const invoiceToolName = "registrar_fatura"

const invoiceSystemPrompt = `Você é um extrator de dados estruturados de FATURAS DE CARTÃO DE CRÉDITO.
Sua ÚNICA tarefa é TRANSCREVER e ESTRUTURAR as compras/lançamentos presentes na fatura.

REGRAS OBRIGATÓRIAS:
- Extraia apenas o que está literalmente no documento. NÃO invente valores.
- Cada compra vira um item em "purchases" com descrição, valor e data.
- Valores em reais: use ponto decimal no campo numérico "amount" (ex.: 1234.56).
- DATAS sempre no formato YYYY-MM-DD. Faturas costumam imprimir a data da
  compra sem o ano (ex.: "07/06", "07 JUN"): infira o ano a partir do período
  da fatura (statement_month/due_date). Compra de mês posterior ao vencimento
  pertence ao ano anterior (ex.: compra de dezembro em fatura que vence em
  janeiro). Se a data estiver ilegível ou ausente, use "" (string vazia).
- Se a compra for parcelada (ex.: "PARC 03/10", "3/10"), preencha installment_current e installment_total.
- Sugira uma categoria em "category_suggestion" APENAS entre: moradia, alimentacao, mercado, saude, transporte, educacao, lazer, contas_fixas, servicos, impostos, equipamentos, outros.
- Não inclua pagamentos/estornos da fatura anterior como compras; registre-os em warnings se relevante.
- Registre em warnings qualquer ambiguidade, ilegibilidade ou dado faltante.

Use SEMPRE a ferramenta ` + invoiceToolName + ` para retornar o resultado estruturado.`

// invoiceInputSchema é o schema alvo (tool input_schema) da fatura.
func invoiceInputSchema() map[string]any {
	purchase := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"description":         map[string]any{"type": "string"},
			"amount":              map[string]any{"type": []string{"number", "null"}},
			"date":                map[string]any{"type": "string", "description": "Data da compra em YYYY-MM-DD (ano inferido do período da fatura); \"\" se ilegível"},
			"category_suggestion": map[string]any{"type": "string"},
			"installment_current": map[string]any{"type": []string{"integer", "null"}},
			"installment_total":   map[string]any{"type": []string{"integer", "null"}},
			"raw_text":            map[string]any{"type": "string"},
		},
		"required": []string{"description"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"card_issuer":     map[string]any{"type": "string"},
			"statement_month": map[string]any{"type": "string"},
			"due_date":        map[string]any{"type": "string"},
			"total_amount":    map[string]any{"type": []string{"number", "null"}},
			"purchases":       map[string]any{"type": "array", "items": purchase},
			"warnings":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required": []string{"purchases"},
	}
}

// CreditCardInvoiceProfile é o perfil do módulo Financeiro (fatura de cartão).
func CreditCardInvoiceProfile() ExtractProfile {
	return ExtractProfile{
		SystemPrompt:    invoiceSystemPrompt,
		ToolName:        invoiceToolName,
		ToolDescription: "Registra as compras/lançamentos estruturados extraídos da fatura de cartão de crédito.",
		InputSchema:     invoiceInputSchema(),
		PromptVersion:   InvoicePromptVersion,
		UserInstruction: "Extraia todas as compras da fatura acima e retorne via ferramenta " + invoiceToolName + ".",
	}
}
