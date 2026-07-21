package finance

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	appent "github.com/retechfin/retechfin-api/internal/application/entitlement"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
	"github.com/retechfin/retechfin-api/internal/infrastructure/cache"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
	"github.com/retechfin/retechfin-api/internal/infrastructure/infosimples"
	"github.com/retechfin/retechfin-api/internal/infrastructure/queue"
)

// nfceConsulter é a porta (nil-safe) para a consulta de NFC-e na SEFAZ via
// Infosimples. Mantida como interface local para permitir ausência do provider.
type nfceConsulter interface {
	Enabled() bool
	ConsultarNFCe(ctx context.Context, nfce string) (*infosimples.NFCeResult, error)
}

// qrDecoder é a porta (nil-safe) para leitura server-side do QR Code do cupom.
type qrDecoder interface {
	DecodeNFCe(content []byte) (string, bool)
}

// MessageTypeFiscalIngestion roteia as mensagens de ingestão fiscal na fila.
const MessageTypeFiscalIngestion = "fiscal_ingestion"

// FiscalIngestionMessage é o payload enfileirado para processar um cupom. Não
// carrega o conteúdo do arquivo (recarregado do storage pelo worker) — mantém a
// mensagem pequena e durável.
type FiscalIngestionMessage struct {
	JobID       uuid.UUID `json:"job_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	DocumentID  uuid.UUID `json:"document_id"`
	InputType   string    `json:"input_type"`
	MimeType    string    `json:"mime_type"`
	// Chave opcional (colada pelo usuário ou lida no navegador). Vazia → o worker
	// tenta decodificar o QR Code server-side a partir da imagem.
	Chave string `json:"chave,omitempty"`
}

// FinanceExtractionService orquestra a criação e execução (assíncrona) de jobs
// de extração de faturas (LLM) e de cupons/notas fiscais (SEFAZ com fallback
// IA). Ao concluir, atualiza tanto o job quanto o documento.
type FinanceExtractionService struct {
	jobs         dom.FinanceExtractionJobRepository
	docs         dom.FinanceDocumentRepository
	extractor    extraction.Extractor
	infosimples  nfceConsulter              // opcional (nil = sem SEFAZ)
	entitlements *appent.Service            // opcional (nil = sem cota)
	cache        *cache.Cache               // opcional (nil = sem cache por chave)
	categorizer  *FiscalCategorizer         // opcional (nil = itens sem categoria)
	queue        queue.Publisher            // opcional (nil = processa inline)
	qr           qrDecoder                  // opcional (nil = sem decode QR server-side)
	keyReader    extraction.FiscalKeyReader // opcional (nil = sem leitura de chave por IA)
}

// NewFinanceExtractionService constrói o serviço de extração.
// infosimples/entitlements/cache/categorizer/queue/qr são opcionais (podem ser
// nil): sem queue, a ingestão fiscal roda inline (goroutine); sem qr, não há
// leitura server-side do QR Code.
func NewFinanceExtractionService(
	jobs dom.FinanceExtractionJobRepository,
	docs dom.FinanceDocumentRepository,
	extractor extraction.Extractor,
	nfce nfceConsulter,
	entitlements *appent.Service,
	c *cache.Cache,
	categorizer *FiscalCategorizer,
	q queue.Publisher,
	qr qrDecoder,
	keyReader extraction.FiscalKeyReader,
) *FinanceExtractionService {
	return &FinanceExtractionService{
		jobs:         jobs,
		docs:         docs,
		extractor:    extractor,
		infosimples:  nfce,
		entitlements: entitlements,
		cache:        c,
		categorizer:  categorizer,
		queue:        q,
		qr:           qr,
		keyReader:    keyReader,
	}
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

	// Perfil por tipo de documento: fatura (default) ou cupom/nota fiscal.
	profile := extraction.CreditCardInvoiceProfile()
	if doc, derr := s.docs.GetByID(ctx, workspaceID, documentID); derr == nil && doc.Kind == dom.DocumentFiscal {
		profile = extraction.FiscalReceiptProfile()
	}
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
		extErr = friendlyExtractionErr(extErr)
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

// StartFiscalExtraction inicia a ingestão de um cupom/nota fiscal (kind=fiscal).
// Cria o job (pending) e ENFILEIRA o processamento; o worker resolve SEFAZ
// (verificado) com fallback IA. Sem fila configurada, roda inline (goroutine).
// A chave é opcional: sem ela, o worker tenta ler o QR Code server-side.
func (s *FinanceExtractionService) StartFiscalExtraction(
	ctx context.Context,
	workspaceID, documentID uuid.UUID,
	inputType, mimeType string,
	content []byte,
	chave string,
) (*dom.FinanceExtractionJob, error) {
	now := time.Now().UTC()
	chave = strings.TrimSpace(chave)
	infosimplesEnabled := s.infosimples != nil && s.infosimples.Enabled()

	provider := s.extractor.Provider()
	if infosimplesEnabled {
		provider = dom.FiscalSourceSEFAZ // otimista; corrigido no processamento
	}

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

	// Nenhum caminho disponível (sem SEFAZ e sem LLM): falha controlada.
	if !infosimplesEnabled && !s.extractor.Enabled() {
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

	// Fila: publica a mensagem (durável via job + sweeper) e retorna já.
	if s.queue != nil {
		body, err := json.Marshal(FiscalIngestionMessage{
			JobID: job.ID, WorkspaceID: workspaceID, DocumentID: documentID,
			InputType: inputType, MimeType: mimeType, Chave: chave,
		})
		if err == nil {
			if perr := s.queue.Publish(ctx, queue.Message{Type: MessageTypeFiscalIngestion, Body: body}); perr == nil {
				return job, nil
			} else {
				slog.Warn("falha ao publicar na fila — caindo para inline", slog.String("error", perr.Error()))
			}
		}
	}

	// Fallback inline (sem fila): goroutine própria.
	buf := make([]byte, len(content))
	copy(buf, content)
	go func() {
		if err := s.ProcessFiscal(context.Background(), job.ID, workspaceID, documentID, inputType, mimeType, buf, chave); err != nil {
			slog.Error("ingestão fiscal inline falhou", slog.String("error", err.Error()))
		}
	}()
	return job, nil
}

// ProcessFiscal executa a ingestão fiscal de forma síncrona (chamada pelo worker
// da fila ou inline). Retorna erro quando a falha é retriável (a fila reagenda);
// falhas terminais (sem caminho de processamento) marcam o job como failed e
// retornam nil. Grava a procedência (fiscal_source) no documento.
func (s *FinanceExtractionService) ProcessFiscal(
	ctx context.Context,
	jobID, workspaceID, documentID uuid.UUID,
	inputType, mimeType string,
	content []byte,
	chave string,
) error {
	job, err := s.jobs.GetByID(ctx, workspaceID, jobID)
	if err != nil {
		return err // transitório: a fila tenta de novo
	}
	// Já concluído (ex.: reenfileirado pelo sweeper após completar): no-op.
	if job.Status == dom.JobCompleted {
		return nil
	}

	started := time.Now().UTC()
	job.Status = dom.JobProcessing
	job.StartedAt = &started
	job.UpdatedAt = started
	if err := s.jobs.Update(ctx, job); err != nil {
		return err
	}
	s.updateDocumentStatus(ctx, workspaceID, documentID, dom.ExtractionProcessing, nil, nil)

	// Sem chave explícita: tenta ler o QR Code da imagem no servidor.
	chave = strings.TrimSpace(chave)
	isImage := dom.ExtractionInputType(inputType) == dom.ExtractionInputImage
	if chave == "" && isImage && s.qr != nil {
		if decoded, ok := s.qr.DecodeNFCe(content); ok {
			chave = decoded
			slog.Info("QR Code lido server-side", slog.String("document_id", documentID.String()))
		}
	}

	// QR ilegível (foto borrada/comprimida): a IA lê a CHAVE IMPRESSA de 44
	// dígitos, que sobrevive ao borrão melhor que o QR. Se achar, ainda vai pra
	// SEFAZ (dado exato) em vez de cair na extração probabilística de itens.
	if chave == "" && isImage && s.keyReader != nil && s.keyReader.Enabled() {
		if raw, err := s.keyReader.ReadFiscalKey(ctx, content, mimeType); err == nil {
			if d := onlyDigits(raw); len(d) == 44 {
				chave = d
				slog.Info("chave lida por IA (QR ilegível)", slog.String("document_id", documentID.String()))
			}
		}
	}

	// 1) Caminho de ouro: SEFAZ (verificado). Só debita cota em sucesso.
	if chave != "" && s.infosimples != nil && s.infosimples.Enabled() {
		if structured, ok := s.trySEFAZ(ctx, workspaceID, chave); ok {
			finished := time.Now().UTC()
			job.Provider = dom.FiscalSourceSEFAZ
			job.Status = dom.JobCompleted
			job.FinishedAt = &finished
			job.UpdatedAt = finished
			_ = s.jobs.Update(ctx, job)
			s.updateDocumentFiscalResult(ctx, workspaceID, documentID, structured, dom.FiscalSourceSEFAZ)
			slog.Info("✅ ingestão fiscal via SEFAZ concluída",
				slog.String("document_id", documentID.String()),
				slog.Duration("duration", finished.Sub(started)),
			)
			return nil
		}
		// SEFAZ indisponível/sem cota → degrada para IA (abaixo).
	}

	// 2) Fallback IA (LLM). Requer extractor habilitado.
	if !s.extractor.Enabled() {
		s.failFiscalJob(ctx, job, workspaceID, documentID, extraction.ErrExtractionDisabled, started)
		return nil // terminal: sem LLM não adianta reenfileirar
	}
	profile := extraction.FiscalReceiptProfile()
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
		s.failFiscalJob(ctx, job, workspaceID, documentID, friendlyExtractionErr(extErr), started)
		return extErr // pode ser transitório (rate limit): a fila reagenda
	}
	job.Provider = s.extractor.Provider()
	job.Status = dom.JobCompleted
	_ = s.jobs.Update(ctx, job)
	// Categoriza tenant-aware (ignora a category_suggestion crua do LLM de
	// extração, que não conhece as categorias reais da tenant).
	structured := []byte(res.StructuredJSON)
	if sf, ok := storedFiscalFromLLM(structured); ok {
		structured = s.categorizeAndMarshal(ctx, workspaceID, sf)
	}
	s.updateDocumentFiscalResult(ctx, workspaceID, documentID, structured, dom.FiscalSourceOCRLLM)
	slog.Info("✅ ingestão fiscal via IA (fallback) concluída",
		slog.String("document_id", documentID.String()),
		slog.Duration("duration", finished.Sub(started)),
	)
	return nil
}

// RecoverStaleFiscalJobs reenfileira jobs fiscais travados (pending/processing
// antigos) — recuperação após restart/crash. Retorna quantos foram reenfileirados.
// Só reenfileira jobs de documentos kind=fiscal.
func (s *FinanceExtractionService) RecoverStaleFiscalJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	if s.queue == nil {
		return 0, nil
	}
	before := time.Now().UTC().Add(-olderThan)
	jobs, err := s.jobs.ListStale(ctx, []dom.ExtractionJobStatus{dom.JobPending, dom.JobProcessing}, before, 200)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range jobs {
		j := jobs[i]
		doc, derr := s.docs.GetByID(ctx, j.WorkspaceID, j.DocumentID)
		if derr != nil || doc.Kind != dom.DocumentFiscal {
			continue // não-fiscal (ou doc sumiu): fora do escopo deste worker
		}
		body, merr := json.Marshal(FiscalIngestionMessage{
			JobID: j.ID, WorkspaceID: j.WorkspaceID, DocumentID: j.DocumentID,
			InputType: string(j.InputType), MimeType: doc.MimeType,
			// Sem chave: o worker relê o QR server-side a partir da imagem.
		})
		if merr != nil {
			continue
		}
		if perr := s.queue.Publish(ctx, queue.Message{Type: MessageTypeFiscalIngestion, Body: body}); perr == nil {
			n++
		}
	}
	if n > 0 {
		slog.Info("♻️ jobs fiscais travados reenfileirados", slog.Int("count", n))
	}
	return n, nil
}

// trySEFAZ consulta a NFC-e na Infosimples e devolve o JSON no schema fiscal
// (fiscal-extract-v1, o mesmo do LLM). Usa cache por chave (não paga 2×) e
// reserva de cota (estornada em falha). ok=false → chamador degrada para IA.
func (s *FinanceExtractionService) trySEFAZ(ctx context.Context, workspaceID uuid.UUID, chave string) ([]byte, bool) {
	cacheKey := "nfce:" + workspaceID.String() + ":" + chave
	if cached, _ := s.cache.Get(ctx, cacheKey); cached != "" {
		return []byte(cached), true // reuso: não debita cota nem consulta de novo
	}

	if s.entitlements != nil {
		allowed, _, err := s.entitlements.ReserveFiscalSEFAZ(ctx, workspaceID)
		if err == nil && !allowed {
			slog.Info("cota SEFAZ do mês esgotada — degradando para IA",
				slog.String("workspace_id", workspaceID.String()))
			return nil, false
		}
	}

	res, err := s.infosimples.ConsultarNFCe(ctx, chave)
	if err != nil {
		if s.entitlements != nil {
			s.entitlements.RefundFiscalSEFAZ(ctx, workspaceID) // só sucesso debita
		}
		slog.Warn("consulta SEFAZ falhou — fallback IA",
			slog.String("error", err.Error()),
			slog.String("workspace_id", workspaceID.String()))
		return nil, false
	}

	sf := storedFiscalFromNFCe(res)
	structured := s.categorizeAndMarshal(ctx, workspaceID, sf)
	if len(structured) == 0 {
		if s.entitlements != nil {
			s.entitlements.RefundFiscalSEFAZ(ctx, workspaceID)
		}
		return nil, false
	}
	_ = s.cache.Set(ctx, cacheKey, string(structured), 720*time.Hour) // 30 dias
	return structured, true
}

// failFiscalJob marca o job fiscal como falho e o documento como failed.
func (s *FinanceExtractionService) failFiscalJob(
	ctx context.Context,
	job *dom.FinanceExtractionJob,
	workspaceID, documentID uuid.UUID,
	cause error,
	started time.Time,
) {
	finished := time.Now().UTC()
	msg := cause.Error()
	job.Status = dom.JobFailed
	job.ErrorMessage = &msg
	job.FinishedAt = &finished
	job.UpdatedAt = finished
	_ = s.jobs.Update(ctx, job)
	s.markDocumentFailed(ctx, workspaceID, documentID)
	slog.Error("❌ ingestão fiscal falhou",
		slog.String("error", msg),
		slog.String("document_id", documentID.String()),
		slog.String("workspace_id", workspaceID.String()),
		slog.Duration("duration", finished.Sub(started)),
	)
}

// updateDocumentFiscalResult grava o detalhamento extraído + a procedência.
func (s *FinanceExtractionService) updateDocumentFiscalResult(
	ctx context.Context,
	workspaceID, documentID uuid.UUID,
	structured []byte,
	source string,
) {
	doc, err := s.docs.GetByID(ctx, workspaceID, documentID)
	if err != nil {
		return
	}
	doc.ExtractionStatus = dom.ExtractionExtracted
	if len(structured) > 0 {
		doc.ExtractedJSON = structured
	}
	src := source
	doc.FiscalSource = &src
	doc.UpdatedAt = time.Now().UTC()
	_ = s.docs.UpdateExtraction(ctx, doc)
}

// storedFiscalItem/storedFiscal são o schema fiscal-extract-v1 gravado no
// documento (o mesmo lido por ParseFiscal), agora com os campos de categoria
// preenchidos pela categorização tenant-aware.
type storedFiscalItem struct {
	Description        string  `json:"description"`
	Quantity           float64 `json:"quantity"`
	UnitAmount         float64 `json:"unit_amount"`
	Amount             float64 `json:"amount"`
	CategorySuggestion string  `json:"category_suggestion,omitempty"`
	CategoryName       string  `json:"category_name,omitempty"`
	CategoryGroup      string  `json:"category_group,omitempty"`
	CategoryIsNew      bool    `json:"category_is_new,omitempty"`
	RawText            string  `json:"raw_text,omitempty"`
}

type storedFiscal struct {
	Merchant    string             `json:"merchant"`
	CNPJ        string             `json:"cnpj"`
	Date        string             `json:"date"`
	TotalAmount float64            `json:"total_amount"`
	Items       []storedFiscalItem `json:"items"`
	Warnings    []string           `json:"warnings"`
}

// storedFiscalFromNFCe monta o cupom (sem categoria) a partir do resultado SEFAZ.
func storedFiscalFromNFCe(r *infosimples.NFCeResult) storedFiscal {
	sf := storedFiscal{
		Merchant:    r.EmitenteNome,
		CNPJ:        r.EmitenteCNPJ,
		Date:        r.DataEmissao,
		TotalAmount: float64(r.ValorTotalCents) / 100.0,
		Items:       make([]storedFiscalItem, 0, len(r.Produtos)),
		Warnings:    r.Warnings,
	}
	for _, p := range r.Produtos {
		sf.Items = append(sf.Items, storedFiscalItem{
			Description: p.Descricao,
			Quantity:    float64(p.QuantityMilli) / 1000.0,
			UnitAmount:  float64(p.UnitCents) / 100.0,
			Amount:      float64(p.AmountCents) / 100.0,
			RawText:     p.Codigo,
		})
	}
	return sf
}

// storedFiscalFromLLM reconstrói o cupom a partir do JSON do LLM de extração.
// Ignora a category_suggestion crua (será substituída pela categorização
// tenant-aware). ok=false quando o JSON não é parseável.
func storedFiscalFromLLM(raw []byte) (storedFiscal, bool) {
	var f fiscalExtraction
	if err := json.Unmarshal(raw, &f); err != nil {
		return storedFiscal{}, false
	}
	sf := storedFiscal{
		Merchant: strings.TrimSpace(f.Merchant),
		CNPJ:     strings.TrimSpace(f.CNPJ),
		Date:     f.Date,
		Warnings: f.Warnings,
		Items:    make([]storedFiscalItem, 0, len(f.Items)),
	}
	if f.TotalAmount != nil {
		sf.TotalAmount = *f.TotalAmount
	}
	for _, it := range f.Items {
		sfi := storedFiscalItem{Description: it.Description, RawText: it.RawText, Quantity: 1}
		if it.Quantity != nil {
			sfi.Quantity = *it.Quantity
		}
		if it.UnitAmount != nil {
			sfi.UnitAmount = *it.UnitAmount
		}
		if it.Amount != nil {
			sfi.Amount = *it.Amount
		}
		sf.Items = append(sf.Items, sfi)
	}
	return sf, true
}

// categorizeAndMarshal preenche as categorias (tenant-aware, validadas) dos
// itens e serializa o cupom no schema fiscal-extract-v1. Sem categorizador,
// serializa sem categoria. Retorna nil só em erro de marshal.
func (s *FinanceExtractionService) categorizeAndMarshal(ctx context.Context, workspaceID uuid.UUID, sf storedFiscal) []byte {
	if s.categorizer != nil && s.categorizer.Enabled() && len(sf.Items) > 0 {
		descs := make([]string, len(sf.Items))
		for i := range sf.Items {
			descs[i] = sf.Items[i].Description
		}
		cats := s.categorizer.Categorize(ctx, workspaceID, descs)
		for i := range sf.Items {
			if i < len(cats) && cats[i].Slug != "" {
				sf.Items[i].CategorySuggestion = cats[i].Slug
				sf.Items[i].CategoryGroup = cats[i].Group
				sf.Items[i].CategoryName = cats[i].Name
				sf.Items[i].CategoryIsNew = cats[i].IsNew
			}
		}
	}
	b, err := json.Marshal(sf)
	if err != nil {
		return nil
	}
	return b
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
	CardIssuer      string            `json:"card_issuer"`
	StatementMonth  string            `json:"statement_month"`
	DueDate         string            `json:"due_date"`
	TotalAmount     *float64          `json:"total_amount"`
	PreviousBalance *float64          `json:"previous_balance"`
	Purchases       []invoicePurchase `json:"purchases"`
	Credits         []invoiceCredit   `json:"credits"`
	Warnings        []string          `json:"warnings"`
}

type invoiceCredit struct {
	Description string   `json:"description"`
	Date        string   `json:"date"`
	Amount      *float64 `json:"amount"`
}

// CreditSuggestion é um pagamento/estorno/crédito do ciclo (não é compra).
type CreditSuggestion struct {
	Description string
	Date        string // YYYY-MM-DD ("" quando ilegível)
	AmountCents int64  // valor absoluto
}

// InvoiceMeta são os agregados extraídos da fatura para reconciliação:
// total a pagar, fatura anterior e créditos do ciclo.
type InvoiceMeta struct {
	TotalCents           *int64
	PreviousBalanceCents *int64
	Credits              []CreditSuggestion
}

// ParseInvoiceMeta extrai os agregados da fatura do extracted_json. Retorna
// meta vazia (não erro) quando não há JSON — compatível com extrações v1/v2,
// que não traziam esses campos.
func (s *FinanceExtractionService) ParseInvoiceMeta(doc *dom.FinanceDocument) (*InvoiceMeta, error) {
	meta := &InvoiceMeta{Credits: []CreditSuggestion{}}
	if doc == nil || len(doc.ExtractedJSON) == 0 {
		return meta, nil
	}
	var inv InvoiceExtraction
	if err := json.Unmarshal(doc.ExtractedJSON, &inv); err != nil {
		return nil, &dom.ValidationError{Msg: "extracted_json inválido: " + err.Error()}
	}
	if inv.TotalAmount != nil {
		v := reaisToCents(inv.TotalAmount)
		meta.TotalCents = &v
	}
	if inv.PreviousBalance != nil {
		v := reaisToCents(inv.PreviousBalance)
		meta.PreviousBalanceCents = &v
	}
	for _, c := range inv.Credits {
		cents := reaisToCents(c.Amount)
		if cents < 0 {
			cents = -cents // defensivo: LLM pode mandar negativo
		}
		meta.Credits = append(meta.Credits, CreditSuggestion{
			Description: c.Description,
			Date:        normalizePurchaseDate(c.Date, inv.DueDate),
			AmountCents: cents,
		})
	}
	return meta, nil
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

// onlyDigits mantém apenas os dígitos de uma string (para validar a chave de 44).
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// reaisToCents converte um valor em reais (float) para centavos inteiros,
// arredondando para o centavo mais próximo.
func reaisToCents(v *float64) int64 {
	if v == nil {
		return 0
	}
	return int64(math.Round(*v * 100))
}

// FiscalItemSuggestion é um item de cupom/nota fiscal sugerido pela extração.
// Valores em centavos; quantidade em milésimos (1un = 1000).
type FiscalItemSuggestion struct {
	Description   string
	QuantityMilli int64
	UnitCents     int64
	AmountCents   int64
	// Category é o slug da categoria sugerida (validada contra o catálogo da
	// tenant). CategoryGroup é o grupo global. CategoryIsNew indica que a
	// categoria ainda não existe na tenant (sugestão a confirmar). CategoryName
	// é o nome de exibição (usado no auto-cadastro de categorias novas).
	Category      string
	CategoryName  string
	CategoryGroup string
	CategoryIsNew bool
	RawText       string
}

// FiscalSuggestion é o cupom/nota fiscal estruturado sugerido pela extração.
type FiscalSuggestion struct {
	Merchant   string
	CNPJ       string
	Date       string // YYYY-MM-DD ("" quando ilegível)
	TotalCents int64
	Items      []FiscalItemSuggestion
	Warnings   []string
}

// fiscalExtraction espelha o schema fiscal-extract-v1 (profile.go).
type fiscalExtraction struct {
	Merchant    string       `json:"merchant"`
	CNPJ        string       `json:"cnpj"`
	Date        string       `json:"date"`
	TotalAmount *float64     `json:"total_amount"`
	Items       []fiscalItem `json:"items"`
	Warnings    []string     `json:"warnings"`
}

type fiscalItem struct {
	Description        string   `json:"description"`
	Quantity           *float64 `json:"quantity"`
	UnitAmount         *float64 `json:"unit_amount"`
	Amount             *float64 `json:"amount"`
	CategorySuggestion string   `json:"category_suggestion"`
	CategoryName       string   `json:"category_name"`
	CategoryGroup      string   `json:"category_group"`
	CategoryIsNew      bool     `json:"category_is_new"`
	RawText            string   `json:"raw_text"`
}

// ParseFiscal faz o unmarshal do extracted_json de um documento fiscal
// (cupom/nota) e retorna a sugestão com valores em centavos.
func (s *FinanceExtractionService) ParseFiscal(doc *dom.FinanceDocument) (*FiscalSuggestion, error) {
	if doc == nil || len(doc.ExtractedJSON) == 0 {
		return &FiscalSuggestion{Items: []FiscalItemSuggestion{}}, nil
	}
	var f fiscalExtraction
	if err := json.Unmarshal(doc.ExtractedJSON, &f); err != nil {
		return nil, &dom.ValidationError{Msg: "extracted_json inválido: " + err.Error()}
	}
	out := &FiscalSuggestion{
		Merchant:   strings.TrimSpace(f.Merchant),
		CNPJ:       strings.TrimSpace(f.CNPJ),
		Date:       normalizePurchaseDate(f.Date, ""),
		TotalCents: reaisToCents(f.TotalAmount),
		Items:      make([]FiscalItemSuggestion, 0, len(f.Items)),
		Warnings:   f.Warnings,
	}
	for _, it := range f.Items {
		qty := int64(1000) // default 1 unidade
		if it.Quantity != nil {
			qty = int64(math.Round(*it.Quantity * 1000))
		}
		out.Items = append(out.Items, FiscalItemSuggestion{
			Description:   it.Description,
			QuantityMilli: qty,
			UnitCents:     reaisToCents(it.UnitAmount),
			AmountCents:   reaisToCents(it.Amount),
			Category:      it.CategorySuggestion,
			CategoryName:  it.CategoryName,
			CategoryGroup: it.CategoryGroup,
			CategoryIsNew: it.CategoryIsNew,
			RawText:       it.RawText,
		})
	}
	return out, nil
}

// friendlyExtractionErr traduz erros crípticos do provedor LLM em mensagens
// acionáveis para o usuário (armazenadas em job.error_message).
func friendlyExtractionErr(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "password protected") || strings.Contains(msg, "password-protected") {
		return errors.New("Este PDF é protegido por senha — informe a senha do arquivo e tente novamente.")
	}
	return err
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
