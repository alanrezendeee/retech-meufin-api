package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxIncomeSourceNameLen = 255

// IncomeSourceKind classifica a origem de uma receita.
type IncomeSourceKind string

const (
	IncomeSourceCLT        IncomeSourceKind = "clt"
	IncomeSourcePJ         IncomeSourceKind = "pj"
	IncomeSourceFreelance  IncomeSourceKind = "freelance"
	IncomeSourceRental     IncomeSourceKind = "rental"
	IncomeSourceInvestment IncomeSourceKind = "investment"
	IncomeSourceBenefit    IncomeSourceKind = "benefit"
	IncomeSourceOther      IncomeSourceKind = "other"
)

// IncomeSource é uma fonte de receita cadastrada pelo tenant.
type IncomeSource struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Kind        IncomeSourceKind
	Active      bool
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Validate normaliza campos e valida invariantes.
func (s *IncomeSource) Validate() error {
	if s.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(s.Name)
	if name == "" {
		return &ValidationError{Msg: "nome da fonte de receita é obrigatório"}
	}
	if len(name) > maxIncomeSourceNameLen {
		return &ValidationError{Msg: "nome da fonte de receita excede o tamanho máximo"}
	}
	s.Name = name

	switch s.Kind {
	case IncomeSourceCLT, IncomeSourcePJ, IncomeSourceFreelance, IncomeSourceRental,
		IncomeSourceInvestment, IncomeSourceBenefit, IncomeSourceOther:
	default:
		return &ValidationError{Msg: "kind da fonte de receita inválido"}
	}
	return nil
}

// IncomeSourceRepository persiste fontes de receita com escopo de workspace.
type IncomeSourceRepository interface {
	Create(ctx context.Context, s *IncomeSource) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*IncomeSource, error)
	Update(ctx context.Context, s *IncomeSource) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]IncomeSource, int64, error)
}
