package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxExamNameLen    = 255
	maxExamCodeLen    = 50
	maxBodyAreaLen    = 100
	maxRequestedByLen = 255
)

// ExamRequestStatus representa o ciclo de vida de uma solicitação de exame.
type ExamRequestStatus string

const (
	ExamRequestStatusDraft             ExamRequestStatus = "draft"
	ExamRequestStatusRequested         ExamRequestStatus = "requested"
	ExamRequestStatusCollected         ExamRequestStatus = "collected"
	ExamRequestStatusPartiallyResulted ExamRequestStatus = "partially_resulted"
	ExamRequestStatusResulted          ExamRequestStatus = "resulted"
	ExamRequestStatusCanceled          ExamRequestStatus = "canceled"
)

// ExamRequestItemStatus representa o ciclo de vida de um item da solicitação.
type ExamRequestItemStatus string

const (
	ExamRequestItemStatusPending   ExamRequestItemStatus = "pending"
	ExamRequestItemStatusCollected ExamRequestItemStatus = "collected"
	ExamRequestItemStatusResulted  ExamRequestItemStatus = "resulted"
	ExamRequestItemStatusCanceled  ExamRequestItemStatus = "canceled"
)

// ExamRequest é a solicitação de exames de um membro da família, com seus itens.
type ExamRequest struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	FamilyMemberID uuid.UUID
	LabID          *uuid.UUID
	RequestedBy    *string
	RequestDate    time.Time
	Status         ExamRequestStatus
	Notes          *string
	Items          []ExamRequestItem
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ExamRequestItem é um exame individual dentro de uma solicitação.
type ExamRequestItem struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	ExamRequestID uuid.UUID
	MarkerID      *uuid.UUID
	ExamName      string
	ExamCode      *string
	BodyArea      *string
	Notes         *string
	Status        ExamRequestItemStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Validate normaliza e valida a solicitação e todos os seus itens.
func (r *ExamRequest) Validate() error {
	if r.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if r.FamilyMemberID == uuid.Nil {
		return &ValidationError{Msg: "family_member_id é obrigatório"}
	}
	if r.RequestDate.IsZero() {
		return &ValidationError{Msg: "request_date é obrigatório"}
	}
	if r.LabID != nil && *r.LabID == uuid.Nil {
		r.LabID = nil
	}
	if r.RequestedBy != nil {
		v := strings.TrimSpace(*r.RequestedBy)
		if v == "" {
			r.RequestedBy = nil
		} else {
			if len(v) > maxRequestedByLen {
				return &ValidationError{Msg: "requested_by excede o tamanho máximo"}
			}
			r.RequestedBy = &v
		}
	}

	switch r.Status {
	case ExamRequestStatusDraft, ExamRequestStatusRequested, ExamRequestStatusCollected,
		ExamRequestStatusPartiallyResulted, ExamRequestStatusResulted, ExamRequestStatusCanceled:
	case "":
		r.Status = ExamRequestStatusDraft
	default:
		return &ValidationError{Msg: "status inválido"}
	}

	for i := range r.Items {
		if err := r.Items[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Validate normaliza e valida um item da solicitação.
func (it *ExamRequestItem) Validate() error {
	if it.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(it.ExamName)
	if name == "" {
		return &ValidationError{Msg: "exam_name é obrigatório"}
	}
	if len(name) > maxExamNameLen {
		return &ValidationError{Msg: "exam_name excede o tamanho máximo"}
	}
	it.ExamName = name

	if it.MarkerID != nil && *it.MarkerID == uuid.Nil {
		it.MarkerID = nil
	}
	if it.ExamCode != nil {
		v := strings.TrimSpace(*it.ExamCode)
		if v == "" {
			it.ExamCode = nil
		} else {
			if len(v) > maxExamCodeLen {
				return &ValidationError{Msg: "exam_code excede o tamanho máximo"}
			}
			it.ExamCode = &v
		}
	}
	if it.BodyArea != nil {
		v := strings.TrimSpace(*it.BodyArea)
		if v == "" {
			it.BodyArea = nil
		} else {
			if len(v) > maxBodyAreaLen {
				return &ValidationError{Msg: "body_area excede o tamanho máximo"}
			}
			it.BodyArea = &v
		}
	}
	if it.Notes != nil {
		if v := strings.TrimSpace(*it.Notes); v == "" {
			it.Notes = nil
		} else {
			it.Notes = &v
		}
	}

	switch it.Status {
	case ExamRequestItemStatusPending, ExamRequestItemStatusCollected,
		ExamRequestItemStatusResulted, ExamRequestItemStatusCanceled:
	case "":
		it.Status = ExamRequestItemStatusPending
	default:
		return &ValidationError{Msg: "status do item inválido"}
	}
	return nil
}

// ExamRequestRepository abstrai a persistência de solicitações e seus itens.
// Todas as operações são escopadas por workspace.
type ExamRequestRepository interface {
	// Create persiste a solicitação junto de seus itens.
	Create(ctx context.Context, r *ExamRequest) error
	// GetByID retorna a solicitação (com Items) do workspace informado.
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*ExamRequest, error)
	// Update atualiza os campos da solicitação (não mexe nos itens).
	Update(ctx context.Context, r *ExamRequest) error
	// SoftDelete marca a solicitação como removida.
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	// List retorna solicitações paginadas do workspace.
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]ExamRequest, int64, error)

	// AddItem adiciona um item a uma solicitação existente.
	AddItem(ctx context.Context, it *ExamRequestItem) error
	// UpdateItem atualiza um item existente.
	UpdateItem(ctx context.Context, it *ExamRequestItem) error
	// SoftDeleteItem marca um item como removido.
	SoftDeleteItem(ctx context.Context, workspaceID, requestID, itemID uuid.UUID) error
}
