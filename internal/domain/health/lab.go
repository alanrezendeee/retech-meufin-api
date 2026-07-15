package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxLabNameLen = 255

// LabKind classifica o tipo de local de saúde (laboratório, clínica, hospital...).
type LabKind string

const (
	LabKindLaboratorio LabKind = "laboratorio"
	LabKindClinica     LabKind = "clinica"
	LabKindHospital    LabKind = "hospital"
	LabKindConsultorio LabKind = "consultorio"
	LabKindOtica       LabKind = "otica"
	LabKindOutros      LabKind = "outros"
)

func validLabKinds() map[LabKind]struct{} {
	return map[LabKind]struct{}{
		LabKindLaboratorio: {},
		LabKindClinica:     {},
		LabKindHospital:    {},
		LabKindConsultorio: {},
		LabKindOtica:       {},
		LabKindOutros:      {},
	}
}

// Lab representa um local de saúde (laboratório, clínica, hospital, ótica...)
// vinculado a um workspace. Não há campos de login/senha por decisão de segurança.
type Lab struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	Name           string
	Kind           LabKind
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

	if l.Kind == "" {
		l.Kind = LabKindLaboratorio
	}
	if _, ok := validLabKinds()[l.Kind]; !ok {
		return &ValidationError{Msg: "tipo de local inválido (laboratorio|clinica|hospital|consultorio|otica|outros)"}
	}
	return nil
}

// LabFilter recorta a listagem da tela de gestão.
type LabFilter struct {
	Query  string // busca por nome (case-insensitive)
	Active *bool
}

// LabRepository abstrai a persistência dos laboratórios, sempre no escopo do tenant.
type LabRepository interface {
	Create(ctx context.Context, l *Lab) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Lab, error)
	Update(ctx context.Context, l *Lab) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter LabFilter, limit, offset int) ([]Lab, int64, error)
}
