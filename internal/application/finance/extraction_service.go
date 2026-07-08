package finance

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
)

// FinanceExtractionService orquestra a criação e execução (assíncrona) de jobs
// de extração LLM de faturas de cartão. Ao concluir, atualiza tanto o job
// quanto o documento (extraction_status/extracted_text/extracted_json).
type FinanceExtractionService struct {
	jobs      dom.FinanceExtractionJobRepository
	docs      dom.FinanceDocumentRepository
	extractor extraction.Extractor
}

// NewFinanceExtractionService constrói o serviço de extração de faturas.
func NewFinanceExtractionService(
	jobs dom.FinanceExtractionJobRepository,
	docs dom.FinanceDocumentRepository,
	extractor extraction.Extractor,
) *FinanceExtractionService {
	return &FinanceExtractionService{jobs: jobs, docs: docs, extractor: extractor}
}

// StartExtraction cria um job (status=pending) e dispara a extração em
// background com o perfil de fatura de cartão. O conteúdo do arquivo é
// fornecido pelo chamador — este serviço não conhece o storage.
//
// Se o extractor estiver desabilitado, o job é criado já como "failed" com um
// erro claro e o próprio erro é retornado ao chamador.
func (s *FinanceExtractionService) StartExtraction(
	ctx context.Context,
	workspaceID, documentID uuid.UUID,
	inputType, mimeType string,
	content []byte,
) (*dom.FinanceExtractionJob, error) {
	now := time.Now().UTC()
	provider := s.extractor.Provider()

	job := &dom.FinanceExtractionJob{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		DocumentID:  documentID,
		Provider:    provider,
		Status:      dom.JobPending,
		InputType:   dom.ExtractionInputType(inputType),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Extractor desabilitado: registra job failed e retorna erro controlado.
	if !s.extractor.Enabled() {
		msg := extraction.ErrExtractionDisabled.Error()
		job.Status = dom.JobFailed
		job.ErrorMessage = &msg
		job.FinishedAt = &now
		if err := s.jobs.Create(ctx, job); err != nil {
			return nil, err
		}
		s.markDocumentFailed(ctx, workspaceID, documentID)
		return job, extraction.ErrExtractionDisabled
	}

	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, err
	}

	// Cópia defensiva do conteúdo para a goroutine.
	buf := make([]byte, len(content))
	copy(buf, content)

	go s.runExtraction(job.ID, workspaceID, documentID, dom.ExtractionInputType(inputType), mimeType, buf)

	return job, nil
}

// runExtraction executa a extração e atualiza o job E o documento. Usa
// context.Background() pois o ciclo de vida é independente da requisição.
func (s *FinanceExtractionService) runExtraction(
	jobID, workspaceID, documentID uuid.UUID,
	inputType dom.ExtractionInputType,
	mimeType string,
	content []byte,
) {
	ctx := context.Background()

	job, err := s.jobs.GetByID(ctx, workspaceID, jobID)
	if err != nil {
		return
	}

	started := time.Now().UTC()
	job.Status = dom.JobProcessing
	job.StartedAt = &started
	job.UpdatedAt = started
	if err := s.jobs.Update(ctx, job); err != nil {
		return
	}
	s.updateDocumentStatus(ctx, workspaceID, documentID, dom.ExtractionProcessing, nil, nil)

	profile := extraction.CreditCardInvoiceProfile()
	res, extErr := s.extractor.Extract(ctx, extraction.ExtractInput{
		InputType: string(inputType),
		MimeType:  mimeType,
		Content:   content,
		Profile:   &profile,
	})

	finished := time.Now().UTC()
	job.FinishedAt = &finished
	job.UpdatedAt = finished
	if len(res.RawResponse) > 0 {
		job.RawResponse = res.RawResponse
	}
	if res.Model != "" {
		m := res.Model
		job.Model = &m
	}
	if res.PromptVersion != "" {
		pv := res.PromptVersion
		job.PromptVersion = &pv
	}

	if extErr != nil {
		msg := extErr.Error()
		job.Status = dom.JobFailed
		job.ErrorMessage = &msg
		_ = s.jobs.Update(ctx, job)
		s.markDocumentFailed(ctx, workspaceID, documentID)
		// Além do error_message no job (que o front mostra), o servidor precisa
		// registrar a falha — crédito esgotado/rate limit têm que aparecer no log.
		slog.Error("❌ extração LLM de fatura falhou",
			slog.String("error", msg),
			slog.String("document_id", documentID.String()),
			slog.String("workspace_id", workspaceID.String()),
			slog.Duration("duration", finished.Sub(started)),
		)
		return
	}

	job.Status = dom.JobCompleted
	_ = s.jobs.Update(ctx, job)
	slog.Info("✅ extração LLM de fatura concluída",
		slog.String("document_id", documentID.String()),
		slog.Duration("duration", finished.Sub(started)),
	)

	var text *string
	if res.Text != "" {
		t := res.Text
		text = &t
	}
	var structured []byte
	if len(res.StructuredJSON) > 0 {
		structured = []byte(res.StructuredJSON)
	}
	s.updateDocumentStatus(ctx, workspaceID, documentID, dom.ExtractionExtracted, text, structured)
}

func (s *FinanceExtractionService) updateDocumentStatus(
	ctx context.Context,
	workspaceID, documentID uuid.UUID,
	status dom.ExtractionStatus,
	text *string,
	structured []byte,
) {
	doc, err := s.docs.GetByID(ctx, workspaceID, documentID)
	if err != nil {
		return
	}
	doc.ExtractionStatus = status
	if text != nil {
		doc.ExtractedText = text
	}
	if len(structured) > 0 {
		doc.ExtractedJSON = structured
	}
	doc.UpdatedAt = time.Now().UTC()
	_ = s.docs.UpdateExtraction(ctx, doc)
}

func (s *FinanceExtractionService) markDocumentFailed(ctx context.Context, workspaceID, documentID uuid.UUID) {
	s.updateDocumentStatus(ctx, workspaceID, documentID, dom.ExtractionFailed, nil, nil)
}

// GetStatus retorna o job de extração mais recente do documento.
func (s *FinanceExtractionService) GetStatus(ctx context.Context, workspaceID, documentID uuid.UUID) (*dom.FinanceExtractionJob, error) {
	return s.jobs.GetByDocument(ctx, workspaceID, documentID)
}

// PurchaseSuggestion é uma compra sugerida a partir da extração da fatura.
// AmountCents já está em centavos (o JSON traz "amount" em reais).
type PurchaseSuggestion struct {
	Description        string
	AmountCents        int64
	Date               string
	Category           string
	InstallmentCurrent *int
	InstallmentTotal   *int
	RawText            string
}

// InvoiceExtraction é o schema estruturado da fatura conforme
// CreditCardInvoiceProfile (extraction.invoiceInputSchema).
type InvoiceExtraction struct {
	CardIssuer     string            `json:"card_issuer"`
	StatementMonth string            `json:"statement_month"`
	DueDate        string            `json:"due_date"`
	TotalAmount    *float64          `json:"total_amount"`
	Purchases      []invoicePurchase `json:"purchases"`
	Warnings       []string          `json:"warnings"`
}

type invoicePurchase struct {
	Description        string   `json:"description"`
	Amount             *float64 `json:"amount"`
	Date               string   `json:"date"`
	CategorySuggestion string   `json:"category_suggestion"`
	InstallmentCurrent *int     `json:"installment_current"`
	InstallmentTotal   *int     `json:"installment_total"`
	RawText            string   `json:"raw_text"`
}

// ParsePurchases faz o unmarshal do extracted_json do documento no schema da
// fatura e retorna as compras sugeridas, com amount convertido de reais para
// centavos. Retorna slice vazio (não nil-erro) quando não há JSON.
func (s *FinanceExtractionService) ParsePurchases(doc *dom.FinanceDocument) ([]PurchaseSuggestion, error) {
	if doc == nil || len(doc.ExtractedJSON) == 0 {
		return []PurchaseSuggestion{}, nil
	}
	var inv InvoiceExtraction
	if err := json.Unmarshal(doc.ExtractedJSON, &inv); err != nil {
		return nil, &dom.ValidationError{Msg: "extracted_json inválido: " + err.Error()}
	}
	out := make([]PurchaseSuggestion, 0, len(inv.Purchases))
	for _, p := range inv.Purchases {
		out = append(out, PurchaseSuggestion{
			Description:        p.Description,
			AmountCents:        reaisToCents(p.Amount),
			Date:               normalizePurchaseDate(p.Date, inv.DueDate),
			Category:           p.CategorySuggestion,
			InstallmentCurrent: p.InstallmentCurrent,
			InstallmentTotal:   p.InstallmentTotal,
			RawText:            p.RawText,
		})
	}
	return out, nil
}

// reaisToCents converte um valor em reais (float) para centavos inteiros,
// arredondando para o centavo mais próximo.
func reaisToCents(v *float64) int64 {
	if v == nil {
		return 0
	}
	return int64(math.Round(*v * 100))
}

var ddmmRe = regexp.MustCompile(`^(\d{1,2})[/.\-](\d{1,2})$`)

// normalizePurchaseDate converte a data da compra para YYYY-MM-DD (formato que
// a UI e o confirm falam). O prompt v2 já pede ISO, mas faturas imprimem
// datas sem ano ("07/06") e o LLM pode transcrever literalmente — aceita
// também DD/MM/YYYY, DD/MM/YY e DD/MM (ano inferido do vencimento da fatura:
// mês da compra posterior ao do vencimento pertence ao ano anterior).
// Retorna "" quando não consegue interpretar — melhor sem data do que errada.
func normalizePurchaseDate(raw, dueDateISO string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t.Format("2006-01-02")
	}
	for _, layout := range []string{"02/01/2006", "02-01-2006", "02.01.2006", "02/01/06"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	m := ddmmRe.FindStringSubmatch(raw)
	if m == nil {
		return ""
	}
	due, err := time.Parse("2006-01-02", dueDateISO)
	if err != nil {
		return ""
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return ""
	}
	year := due.Year()
	if month > int(due.Month()) {
		year--
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Day() != day {
		return "" // dia inexistente no mês (ex.: 31/02)
	}
	return t.Format("2006-01-02")
}
