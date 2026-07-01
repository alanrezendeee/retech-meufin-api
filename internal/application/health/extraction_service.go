package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
)

// ExtractionService orquestra a criação e execução (assíncrona) de jobs de
// extração OCR/LLM de documentos de saúde.
type ExtractionService struct {
	jobs      dom.ExtractionJobRepository
	extractor extraction.Extractor
}

// NewExtractionService constrói o serviço de extração.
func NewExtractionService(jobs dom.ExtractionJobRepository, extractor extraction.Extractor) *ExtractionService {
	return &ExtractionService{jobs: jobs, extractor: extractor}
}

// StartExtraction cria um job (status=pending) e dispara a extração em
// background. O conteúdo do arquivo (content) é fornecido pelo chamador — este
// serviço não conhece o storage.
//
// Se o extractor estiver desabilitado, o job é criado já como "failed" com um
// erro claro e o próprio erro é retornado ao chamador.
//
// Quando habilitado, uma goroutine roda extractor.Extract usando
// context.Background() (não o ctx da requisição) e atualiza o job
// (processing -> completed/failed, started_at/finished_at, raw_response).
func (s *ExtractionService) StartExtraction(
	ctx context.Context,
	workspaceID, documentID uuid.UUID,
	inputType, mimeType string,
	content []byte,
) (*dom.ExtractionJob, error) {
	now := time.Now().UTC()
	provider := s.extractor.Provider()

	job := &dom.ExtractionJob{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		DocumentID:  documentID,
		Provider:    provider,
		Status:      dom.ExtractionPending,
		InputType:   dom.ExtractionInputType(inputType),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Extractor desabilitado: registra job failed e retorna erro controlado.
	if !s.extractor.Enabled() {
		msg := extraction.ErrExtractionDisabled.Error()
		job.Status = dom.ExtractionFailed
		job.ErrorMessage = &msg
		job.FinishedAt = &now
		if err := s.jobs.Create(ctx, job); err != nil {
			return nil, err
		}
		return job, extraction.ErrExtractionDisabled
	}

	if err := s.jobs.Create(ctx, job); err != nil {
		return nil, err
	}

	// Cópia defensiva do conteúdo para a goroutine.
	buf := make([]byte, len(content))
	copy(buf, content)

	go s.runExtraction(job.ID, workspaceID, dom.ExtractionInputType(inputType), mimeType, buf)

	return job, nil
}

// runExtraction executa a extração e atualiza o job. Usa context.Background()
// pois o ciclo de vida é independente da requisição original.
func (s *ExtractionService) runExtraction(
	jobID, workspaceID uuid.UUID,
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
	job.Status = dom.ExtractionProcessing
	job.StartedAt = &started
	job.UpdatedAt = started
	if err := s.jobs.Update(ctx, job); err != nil {
		return
	}

	res, extErr := s.extractor.Extract(ctx, extraction.ExtractInput{
		InputType: string(inputType),
		MimeType:  mimeType,
		Content:   content,
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
		job.Status = dom.ExtractionFailed
		job.ErrorMessage = &msg
	} else {
		job.Status = dom.ExtractionCompleted
	}

	_ = s.jobs.Update(ctx, job)
}

// GetStatus retorna o job de extração mais recente do documento.
func (s *ExtractionService) GetStatus(ctx context.Context, workspaceID, documentID uuid.UUID) (*dom.ExtractionJob, error) {
	return s.jobs.GetByDocument(ctx, workspaceID, documentID)
}
