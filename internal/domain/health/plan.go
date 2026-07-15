package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxPlanNameLen     = 255
	maxPlanOperatorLen = 120
	maxPlanAnsCodeLen  = 30
	maxCardNumberLen   = 60
)

// PlanType enumera a modalidade do plano de saúde.
type PlanType string

const (
	PlanTypeIndividual   PlanType = "individual"
	PlanTypeFamiliar     PlanType = "familiar"
	PlanTypeEmpresarial  PlanType = "empresarial"
	PlanTypeOdontologico PlanType = "odontologico"
)

func validPlanTypes() map[PlanType]struct{} {
	return map[PlanType]struct{}{
		PlanTypeIndividual:   {},
		PlanTypeFamiliar:     {},
		PlanTypeEmpresarial:  {},
		PlanTypeOdontologico: {},
	}
}

// Plan é um plano de saúde da família (operadora, mensalidade e membros cobertos).
type Plan struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	Name            string
	Operator        *string
	PlanType        PlanType
	AnsCode         *string
	MonthlyFeeCents int64
	CoverageNotes   *string
	Active          bool
	Members         []PlanMember
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PlanMember vincula um membro da família a um plano (com número de carteirinha).
type PlanMember struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	PlanID      uuid.UUID
	MemberID    uuid.UUID
	CardNumber  *string
	Holder      bool
	CreatedAt   time.Time
}

// Validate normaliza e valida o plano e seus vínculos de membros.
func (p *Plan) Validate() error {
	if p.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return &ValidationError{Msg: "nome do plano é obrigatório"}
	}
	if len(name) > maxPlanNameLen {
		return &ValidationError{Msg: "nome do plano excede o tamanho máximo"}
	}
	p.Name = name

	if p.PlanType == "" {
		p.PlanType = PlanTypeFamiliar
	}
	if _, ok := validPlanTypes()[p.PlanType]; !ok {
		return &ValidationError{Msg: "plan_type inválido (individual|familiar|empresarial|odontologico)"}
	}

	if p.Operator != nil {
		v := strings.TrimSpace(*p.Operator)
		if v == "" {
			p.Operator = nil
		} else {
			if len(v) > maxPlanOperatorLen {
				return &ValidationError{Msg: "operadora excede o tamanho máximo"}
			}
			p.Operator = &v
		}
	}
	if p.AnsCode != nil {
		v := strings.TrimSpace(*p.AnsCode)
		if v == "" {
			p.AnsCode = nil
		} else {
			if len(v) > maxPlanAnsCodeLen {
				return &ValidationError{Msg: "código ANS excede o tamanho máximo"}
			}
			p.AnsCode = &v
		}
	}
	if p.CoverageNotes != nil {
		if v := strings.TrimSpace(*p.CoverageNotes); v == "" {
			p.CoverageNotes = nil
		} else {
			p.CoverageNotes = &v
		}
	}
	if p.MonthlyFeeCents < 0 {
		return &ValidationError{Msg: "mensalidade não pode ser negativa"}
	}

	for i := range p.Members {
		if err := p.Members[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Validate normaliza e valida um vínculo de membro do plano.
func (m *PlanMember) Validate() error {
	if m.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if m.MemberID == uuid.Nil {
		return &ValidationError{Msg: "member_id é obrigatório"}
	}
	if m.CardNumber != nil {
		v := strings.TrimSpace(*m.CardNumber)
		if v == "" {
			m.CardNumber = nil
		} else {
			if len(v) > maxCardNumberLen {
				return &ValidationError{Msg: "número da carteirinha excede o tamanho máximo"}
			}
			m.CardNumber = &v
		}
	}
	return nil
}

// PlanRepository abstrai a persistência dos planos e seus membros (workspace-scoped).
type PlanRepository interface {
	Create(ctx context.Context, p *Plan) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Plan, error)
	Update(ctx context.Context, p *Plan) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Plan, int64, error)
	// ReplaceMembers substitui todos os vínculos de membros de um plano.
	ReplaceMembers(ctx context.Context, workspaceID, planID uuid.UUID, members []PlanMember) error
}
