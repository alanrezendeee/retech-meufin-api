package handlers

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/retechfin/retechfin-api/internal/application/finance"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/errrespond"
	"github.com/retechfin/retechfin-api/internal/interfaces/http/middleware"
)

// FinanceExtractionHandler cobre o polling de status da extração e a
// confirmação (criação da fatura + compras a partir das sugestões).
//
// Injeta:
//   - ext:     FinanceExtractionService (status + parse das compras sugeridas)
//   - docs:    FinanceDocumentService   (carregar o documento e atualizar entry_id)
//   - entries: FinancialEntryService    (CreateInvoiceWithItems)
type FinanceExtractionHandler struct {
	ext     *app.FinanceExtractionService
	docs    *app.FinanceDocumentService
	entries *app.FinancialEntryService
}

func NewFinanceExtractionHandler(
	ext *app.FinanceExtractionService,
	docs *app.FinanceDocumentService,
	entries *app.FinancialEntryService,
) *FinanceExtractionHandler {
	return &FinanceExtractionHandler{ext: ext, docs: docs, entries: entries}
}

type purchaseSuggestionResponse struct {
	Description        string `json:"description"`
	AmountCents        int64  `json:"amount_cents"`
	Date               string `json:"date,omitempty"`
	Category           string `json:"category,omitempty"`
	InstallmentCurrent *int   `json:"installment_current,omitempty"`
	InstallmentTotal   *int   `json:"installment_total,omitempty"`
	RawText            string `json:"raw_text,omitempty"`
}

type financeExtractionStatusResponse struct {
	ID           uuid.UUID                    `json:"id"`
	DocumentID   uuid.UUID                    `json:"document_id"`
	Provider     string                       `json:"provider"`
	Model        *string                      `json:"model,omitempty"`
	Status       string                       `json:"status"`
	InputType    string                       `json:"input_type"`
	ErrorMessage *string                      `json:"error_message,omitempty"`
	StartedAt    *string                      `json:"started_at,omitempty"`
	FinishedAt   *string                      `json:"finished_at,omitempty"`
	CreatedAt    string                       `json:"created_at"`
	UpdatedAt    string                       `json:"updated_at"`
	Purchases    []purchaseSuggestionResponse `json:"purchases,omitempty"`
	Fiscal       *fiscalSuggestionResponse    `json:"fiscal,omitempty"`
}

type fiscalItemSuggestionResponse struct {
	Description   string `json:"description"`
	QuantityMilli int64  `json:"quantity_milli"`
	UnitCents     int64  `json:"unit_cents"`
	AmountCents   int64  `json:"amount_cents"`
	Category      string `json:"category,omitempty"`
	RawText       string `json:"raw_text,omitempty"`
}

type fiscalSuggestionResponse struct {
	Merchant   string                         `json:"merchant,omitempty"`
	CNPJ       string                         `json:"cnpj,omitempty"`
	Date       string                         `json:"date,omitempty"`
	TotalCents int64                          `json:"total_cents"`
	Items      []fiscalItemSuggestionResponse `json:"items"`
	Warnings   []string                       `json:"warnings,omitempty"`
}

// Status responde GET /documents/:id/extraction-status. Quando o documento já
// foi extraído, inclui as compras sugeridas (parseadas do extracted_json).
func (h *FinanceExtractionHandler) Status(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}

	job, err := h.ext.GetStatus(c.Request.Context(), ws, documentID)
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	out := financeExtractionStatusResponse{
		ID:           job.ID,
		DocumentID:   job.DocumentID,
		Provider:     job.Provider,
		Model:        job.Model,
		Status:       string(job.Status),
		InputType:    string(job.InputType),
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:    job.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if job.StartedAt != nil {
		s := job.StartedAt.UTC().Format(time.RFC3339Nano)
		out.StartedAt = &s
	}
	if job.FinishedAt != nil {
		f := job.FinishedAt.UTC().Format(time.RFC3339Nano)
		out.FinishedAt = &f
	}

	// Sugestões quando o documento já foi extraído: compras (fatura) ou
	// itens (cupom/nota fiscal), conforme o tipo do documento.
	doc, derr := h.docs.Get(c.Request.Context(), ws, documentID)
	if derr == nil && doc.ExtractionStatus == dom.ExtractionExtracted {
		if doc.Kind == dom.DocumentFiscal {
			if fiscal, ferr := h.ext.ParseFiscal(doc); ferr == nil {
				out.Fiscal = mapFiscalSuggestion(fiscal)
			}
		} else if purchases, perr := h.ext.ParsePurchases(doc); perr == nil {
			out.Purchases = mapPurchaseSuggestions(purchases)
		}
	}

	c.JSON(http.StatusOK, out)
}

func mapFiscalSuggestion(f *app.FiscalSuggestion) *fiscalSuggestionResponse {
	if f == nil {
		return nil
	}
	items := make([]fiscalItemSuggestionResponse, len(f.Items))
	for i, it := range f.Items {
		items[i] = fiscalItemSuggestionResponse{
			Description:   it.Description,
			QuantityMilli: it.QuantityMilli,
			UnitCents:     it.UnitCents,
			AmountCents:   it.AmountCents,
			Category:      it.Category,
			RawText:       it.RawText,
		}
	}
	return &fiscalSuggestionResponse{
		Merchant:   f.Merchant,
		CNPJ:       f.CNPJ,
		Date:       f.Date,
		TotalCents: f.TotalCents,
		Items:      items,
		Warnings:   f.Warnings,
	}
}

func mapPurchaseSuggestions(ps []app.PurchaseSuggestion) []purchaseSuggestionResponse {
	out := make([]purchaseSuggestionResponse, len(ps))
	for i, p := range ps {
		out[i] = purchaseSuggestionResponse{
			Description:        p.Description,
			AmountCents:        p.AmountCents,
			Date:               p.Date,
			Category:           p.Category,
			InstallmentCurrent: p.InstallmentCurrent,
			InstallmentTotal:   p.InstallmentTotal,
			RawText:            p.RawText,
		}
	}
	return out
}

type confirmInvoiceItemRequest struct {
	Description string `json:"description"`
	// AmountCents é o contrato canônico: dinheiro trafega como inteiro de
	// centavos (menor unidade da moeda), nunca float — regra do sistema todo.
	AmountCents *int64 `json:"amount_cents"`
	// Amount em reais (float) é o contrato legado, mantido por compatibilidade
	// de deploy; usado apenas quando amount_cents não vier. Remover na v2.
	Amount            *float64 `json:"amount"`
	Date              *string  `json:"date"` // YYYY-MM-DD (opcional)
	Category          *string  `json:"category"`
	InstallmentNumber *int     `json:"installment_number"`
	InstallmentTotal  *int     `json:"installment_total"`
}

// cents resolve o valor do item priorizando o contrato inteiro.
func (it confirmInvoiceItemRequest) cents() int64 {
	if it.AmountCents != nil {
		return *it.AmountCents
	}
	return reaisToCents(it.Amount)
}

type confirmInvoiceRequest struct {
	CardID      *string                     `json:"card_id"`
	DueDate     string                      `json:"due_date"` // YYYY-MM-DD
	Description string                      `json:"description"`
	Status      string                      `json:"status"`
	Items       []confirmInvoiceItemRequest `json:"items"`
}

// Confirm responde POST /documents/:id/confirm: cria a fatura + compras a
// partir do corpo enviado (tipicamente as sugestões revisadas pelo usuário),
// vincula o documento à fatura criada (entry_id) e retorna a fatura.
func (h *FinanceExtractionHandler) Confirm(c *gin.Context) {
	ws, ok := middleware.WorkspaceID(c)
	if !ok {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeWorkspaceRequired, "workspace inválido")
		return
	}
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "id inválido")
		return
	}

	var body confirmInvoiceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "corpo inválido")
		return
	}
	if len(body.Items) == 0 {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "a fatura precisa de ao menos um item")
		return
	}

	dueDate, err := time.Parse("2006-01-02", body.DueDate)
	if err != nil {
		errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "due_date inválida (use YYYY-MM-DD)")
		return
	}

	var cardID *uuid.UUID
	if body.CardID != nil && *body.CardID != "" {
		id, perr := uuid.Parse(*body.CardID)
		if perr != nil {
			errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "card_id inválido")
			return
		}
		cardID = &id
	}

	items := make([]app.InvoiceItemInput, 0, len(body.Items))
	for _, it := range body.Items {
		var date *time.Time
		if it.Date != nil && *it.Date != "" {
			d, derr := time.Parse("2006-01-02", *it.Date)
			if derr != nil {
				errrespond.Message(c, http.StatusBadRequest, errrespond.CodeBadRequest, "date de item inválida (use YYYY-MM-DD)")
				return
			}
			date = &d
		}
		items = append(items, app.InvoiceItemInput{
			Description:       it.Description,
			AmountCents:       it.cents(),
			Date:              date,
			Category:          it.Category,
			InstallmentNumber: it.InstallmentNumber,
			InstallmentTotal:  it.InstallmentTotal,
		})
	}

	invoice, children, err := h.entries.CreateInvoiceWithItems(c.Request.Context(), app.CreateInvoiceInput{
		WorkspaceID: ws,
		CardID:      cardID,
		DueDate:     dueDate,
		Description: body.Description,
		Status:      body.Status,
		Items:       items,
	})
	if err != nil {
		errrespond.Write(c, err)
		return
	}

	// Vincula o documento à fatura criada (entry_id) e marca como extraído.
	if doc, derr := h.docs.Get(c.Request.Context(), ws, documentID); derr == nil {
		doc.EntryID = &invoice.ID
		if doc.ExtractionStatus != dom.ExtractionExtracted {
			doc.ExtractionStatus = dom.ExtractionExtracted
		}
		doc.UpdatedAt = time.Now().UTC()
		_ = h.docs.UpdateExtraction(c.Request.Context(), doc)
	}

	c.JSON(http.StatusCreated, gin.H{
		"invoice":     mapInvoiceEntry(invoice),
		"items":       mapInvoiceEntries(children),
		"document_id": documentID,
	})
}

type invoiceEntryResponse struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	CardID      *uuid.UUID `json:"card_id,omitempty"`
	Kind        string     `json:"kind"`
	Status      string     `json:"status"`
	AmountCents int64      `json:"amount_cents"`
	DueDate     string     `json:"due_date"`
	Type        *string    `json:"type,omitempty"`
	Description string     `json:"description"`
}

func mapInvoiceEntry(e *dom.FinancialEntry) invoiceEntryResponse {
	return invoiceEntryResponse{
		ID:          e.ID,
		ParentID:    e.ParentID,
		CardID:      e.CardID,
		Kind:        string(e.Kind),
		Status:      string(e.Status),
		AmountCents: e.AmountCents,
		DueDate:     e.DueDate.UTC().Format("2006-01-02"),
		Type:        e.Type,
		Description: e.Description,
	}
}

func mapInvoiceEntries(es []dom.FinancialEntry) []invoiceEntryResponse {
	out := make([]invoiceEntryResponse, len(es))
	for i := range es {
		out[i] = mapInvoiceEntry(&es[i])
	}
	return out
}

// reaisToCents converte reais (float) para centavos inteiros.
func reaisToCents(v *float64) int64 {
	if v == nil {
		return 0
	}
	return int64(math.Round(*v * 100))
}
