package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxLabNameLen = 255

// Lab representa um laboratório de exames vinculado a um workspace.
// Não há campos de login/senha por decisão de segurança.
type Lab struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	Name           string
	WebsiteURL     *string
	ExamResultsURL *string
	ContactPhone   *string
	Address        *string
	Notes          *string
	Active         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (l *Lab) Validate() error {
	if l.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(l.Name)
	if name == "" {
		return &ValidationError{Msg: "nome do laboratório é obrigatório"}
	}
	if len(name) > maxLabNameLen {
		return &ValidationError{Msg: "nome do laboratório excede o tamanho máximo"}
	}
	l.Name = name
	return nil
}

// LabRepository abstrai a persistência dos laboratórios, sempre no escopo do tenant.
type LabRepository interface {
	Create(ctx context.Context, l *Lab) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Lab, error)
	Update(ctx context.Context, l *Lab) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Lab, int64, error)
}
