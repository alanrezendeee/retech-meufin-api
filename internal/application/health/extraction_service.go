package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
)

// ExtractionService orquestra a criação e execução (assíncrona) de jobs de
// extração OCR/LLM de documentos de saúde.
type ExtractionService struct {
	jobs      dom.ExtractionJobRepository
	docs      dom.DocumentRepository
	members   dom.FamilyMemberRepository
	extractor extraction.Extractor
}

// NewExtractionService constrói o serviço de extração.
func NewExtractionService(jobs dom.ExtractionJobRepository, docs dom.DocumentRepository, members dom.FamilyMemberRepository, extractor extraction.Extractor) *ExtractionService {
	return &ExtractionService{jobs: jobs, docs: docs, members: members, extractor: extractor}
}

// StartExtraction cria um job (status=pending) e dispara a extração em
// background. O conteúdo do arquivo (content) é fornecido pelo chamador — este
// serviço não conhece o storage.
//
// Se o extractor estiver desabilitado, o job é criado já como "failed" com um
// erro claro e o próprio erro é retornado ao chamador.
//
// Quando habilitado, uma goroutine roda extractor.Extract usando
// context.Background() (não o ctx da requisição) e atualiza job e documento
// (processing -> completed/failed; extracted_text/extracted_json no documento).
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

	go s.runExtraction(job.ID, workspaceID, documentID, dom.ExtractionInputType(inputType), mimeType, buf)

	return job, nil
}

// runExtraction executa a extração e atualiza job + documento. Usa
// context.Background() pois o ciclo de vida é independente da requisição.
func (s *ExtractionService) runExtraction(
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
	job.Status = dom.ExtractionProcessing
	job.StartedAt = &started
	job.UpdatedAt = started
	if err := s.jobs.Update(ctx, job); err != nil {
		return
	}
	s.updateDocumentStatus(ctx, workspaceID, documentID, dom.ExtractionProcessing, nil, nil)

	profile := extraction.LabExamProfile()
	// Sexo/idade do membro dono do documento: laudos trazem faixas de
	// referência distintas por sexo/idade e, sem o contexto, o modelo chuta.
	profile.WithPatientContext(s.patientContext(ctx, workspaceID, documentID))
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
		extErr = friendlyHealthExtractionErr(extErr)
		msg := extErr.Error()
		job.Status = dom.ExtractionFailed
		job.ErrorMessage = &msg
		_ = s.jobs.Update(ctx, job)
		s.updateDocumentStatus(ctx, workspaceID, documentID, dom.ExtractionFailed, nil, nil)
		// Além do error_message no job (que o front mostra), o servidor precisa
		// registrar a falha — crédito esgotado/rate limit têm que aparecer no log.
		slog.Error("❌ extração LLM de exame falhou",
			slog.String("error", msg),
			slog.String("document_id", documentID.String()),
			slog.String("workspace_id", workspaceID.String()),
			slog.Duration("duration", finished.Sub(started)),
		)
		return
	}

	job.Status = dom.ExtractionCompleted
	_ = s.jobs.Update(ctx, job)
	slog.Info("✅ extração LLM de exame concluída",
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

// patientContext monta a descrição de sexo/idade do membro vinculado ao
// documento ("" quando o documento não tem membro ou o cadastro não tem os
// dados — a extração segue sem contexto, comportamento anterior).
func (s *ExtractionService) patientContext(ctx context.Context, workspaceID, documentID uuid.UUID) string {
	doc, err := s.docs.GetByID(ctx, workspaceID, documentID)
	if err != nil || doc.FamilyMemberID == nil {
		return ""
	}
	member, err := s.members.GetByID(ctx, workspaceID, *doc.FamilyMemberID)
	if err != nil {
		return ""
	}
	var parts []string
	if member.Gender != nil {
		if g := describeGender(*member.Gender); g != "" {
			parts = append(parts, "sexo "+g)
		}
	}
	if age := member.Age(); age != nil {
		parts = append(parts, fmt.Sprintf("%d anos", *age))
	}
	return strings.Join(parts, ", ")
}

// describeGender traduz o valor do cadastro para o texto do prompt.
func describeGender(g string) string {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case "male", "m", "masculino":
		return "masculino"
	case "female", "f", "feminino":
		return "feminino"
	default:
		return ""
	}
}

func (s *ExtractionService) updateDocumentStatus(
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

// friendlyHealthExtractionErr traduz erros técnicos do provider para mensagens
// acionáveis pelo usuário final.
func friendlyHealthExtractionErr(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "password protected") || strings.Contains(msg, "password-protected") {
		return errors.New("Este PDF é protegido por senha — informe a senha do arquivo e tente novamente.")
	}
	return err
}

// GetStatus retorna o job de extração mais recente do documento.
func (s *ExtractionService) GetStatus(ctx context.Context, workspaceID, documentID uuid.UUID) (*dom.ExtractionJob, error) {
	return s.jobs.GetByDocument(ctx, workspaceID, documentID)
}
