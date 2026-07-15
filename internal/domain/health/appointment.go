package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	maxProfessionalNameLen = 255
)

// AppointmentKind enumera o tipo de compromisso de saúde.
type AppointmentKind string

const (
	AppointmentKindConsulta     AppointmentKind = "consulta"
	AppointmentKindExame        AppointmentKind = "exame"
	AppointmentKindRetorno      AppointmentKind = "retorno"
	AppointmentKindVacina       AppointmentKind = "vacina"
	AppointmentKindProcedimento AppointmentKind = "procedimento"
	AppointmentKindTerapia      AppointmentKind = "terapia"
	AppointmentKindOdontologia  AppointmentKind = "odontologia"
)

func validAppointmentKinds() map[AppointmentKind]struct{} {
	return map[AppointmentKind]struct{}{
		AppointmentKindConsulta:     {},
		AppointmentKindExame:        {},
		AppointmentKindRetorno:      {},
		AppointmentKindVacina:       {},
		AppointmentKindProcedimento: {},
		AppointmentKindTerapia:      {},
		AppointmentKindOdontologia:  {},
	}
}

// Specialty enumera as especialidades médicas suportadas (campo opcional).
type Specialty string

const (
	SpecialtyClinicaGeral   Specialty = "clinica_geral"
	SpecialtyCardiologia    Specialty = "cardiologia"
	SpecialtyPediatria      Specialty = "pediatria"
	SpecialtyGinecologia    Specialty = "ginecologia"
	SpecialtyDermatologia   Specialty = "dermatologia"
	SpecialtyOrtopedia      Specialty = "ortopedia"
	SpecialtyOftalmologia   Specialty = "oftalmologia"
	SpecialtyOtorrino       Specialty = "otorrino"
	SpecialtyPsicologia     Specialty = "psicologia"
	SpecialtyPsiquiatria    Specialty = "psiquiatria"
	SpecialtyNutricao       Specialty = "nutricao"
	SpecialtyEndocrinologia Specialty = "endocrinologia"
	SpecialtyUrologia       Specialty = "urologia"
	SpecialtyGeriatria      Specialty = "geriatria"
	SpecialtyOdontologia    Specialty = "odontologia"
	SpecialtyFisioterapia   Specialty = "fisioterapia"
	SpecialtyOutros         Specialty = "outros"
)

func validSpecialties() map[Specialty]struct{} {
	return map[Specialty]struct{}{
		SpecialtyClinicaGeral: {}, SpecialtyCardiologia: {}, SpecialtyPediatria: {},
		SpecialtyGinecologia: {}, SpecialtyDermatologia: {}, SpecialtyOrtopedia: {},
		SpecialtyOftalmologia: {}, SpecialtyOtorrino: {}, SpecialtyPsicologia: {},
		SpecialtyPsiquiatria: {}, SpecialtyNutricao: {}, SpecialtyEndocrinologia: {},
		SpecialtyUrologia: {}, SpecialtyGeriatria: {}, SpecialtyOdontologia: {},
		SpecialtyFisioterapia: {}, SpecialtyOutros: {},
	}
}

// AppointmentStatus representa o ciclo de vida de uma consulta.
type AppointmentStatus string

const (
	AppointmentStatusAgendada   AppointmentStatus = "agendada"
	AppointmentStatusConfirmada AppointmentStatus = "confirmada"
	AppointmentStatusRealizada  AppointmentStatus = "realizada"
	AppointmentStatusCancelada  AppointmentStatus = "cancelada"
	AppointmentStatusFaltou     AppointmentStatus = "faltou"
)

func validAppointmentStatuses() map[AppointmentStatus]struct{} {
	return map[AppointmentStatus]struct{}{
		AppointmentStatusAgendada:   {},
		AppointmentStatusConfirmada: {},
		AppointmentStatusRealizada:  {},
		AppointmentStatusCancelada:  {},
		AppointmentStatusFaltou:     {},
	}
}

// IsTerminal indica se o status não admite mais transições.
func (s AppointmentStatus) IsTerminal() bool {
	switch s {
	case AppointmentStatusRealizada, AppointmentStatusCancelada, AppointmentStatusFaltou:
		return true
	default:
		return false
	}
}

// CanTransitionTo valida a máquina de estados das consultas.
// agendada e confirmada podem avançar; agendada pode ser confirmada;
// realizada/cancelada/faltou são terminais.
func (s AppointmentStatus) CanTransitionTo(to AppointmentStatus) bool {
	if s == to {
		return true
	}
	switch s {
	case AppointmentStatusAgendada:
		switch to {
		case AppointmentStatusConfirmada, AppointmentStatusRealizada,
			AppointmentStatusCancelada, AppointmentStatusFaltou:
			return true
		}
	case AppointmentStatusConfirmada:
		switch to {
		case AppointmentStatusRealizada, AppointmentStatusCancelada, AppointmentStatusFaltou:
			return true
		}
	}
	return false
}

// Appointment é um compromisso de saúde de um membro da família.
type Appointment struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	FamilyMemberID   uuid.UUID
	Kind             AppointmentKind
	Specialty        *Specialty
	ProfessionalName *string
	LabID            *uuid.UUID
	ExamRequestID    *uuid.UUID
	PlanID           *uuid.UUID
	ScheduledAt      time.Time
	Status           AppointmentStatus
	Reason           *string
	Outcome          *string
	PriceCents       int64
	CoveredByPlan    bool
	Notes            *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Validate normaliza e valida a consulta.
func (a *Appointment) Validate() error {
	if a.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if a.FamilyMemberID == uuid.Nil {
		return &ValidationError{Msg: "family_member_id é obrigatório"}
	}
	if a.ScheduledAt.IsZero() {
		return &ValidationError{Msg: "scheduled_at é obrigatório"}
	}

	if a.Kind == "" {
		a.Kind = AppointmentKindConsulta
	}
	if _, ok := validAppointmentKinds()[a.Kind]; !ok {
		return &ValidationError{Msg: "kind inválido (consulta|exame|retorno|vacina|procedimento|terapia|odontologia)"}
	}

	if a.Specialty != nil {
		sp := Specialty(strings.TrimSpace(string(*a.Specialty)))
		if sp == "" {
			a.Specialty = nil
		} else if _, ok := validSpecialties()[sp]; !ok {
			return &ValidationError{Msg: "specialty inválida"}
		} else {
			a.Specialty = &sp
		}
	}

	if a.Status == "" {
		a.Status = AppointmentStatusAgendada
	}
	if _, ok := validAppointmentStatuses()[a.Status]; !ok {
		return &ValidationError{Msg: "status inválido (agendada|confirmada|realizada|cancelada|faltou)"}
	}

	if a.ProfessionalName != nil {
		v := strings.TrimSpace(*a.ProfessionalName)
		if v == "" {
			a.ProfessionalName = nil
		} else {
			if len(v) > maxProfessionalNameLen {
				return &ValidationError{Msg: "professional_name excede o tamanho máximo"}
			}
			a.ProfessionalName = &v
		}
	}
	if a.LabID != nil && *a.LabID == uuid.Nil {
		a.LabID = nil
	}
	if a.ExamRequestID != nil && *a.ExamRequestID == uuid.Nil {
		a.ExamRequestID = nil
	}
	if a.PlanID != nil && *a.PlanID == uuid.Nil {
		a.PlanID = nil
	}
	if a.Reason != nil {
		if v := strings.TrimSpace(*a.Reason); v == "" {
			a.Reason = nil
		} else {
			a.Reason = &v
		}
	}
	if a.Outcome != nil {
		if v := strings.TrimSpace(*a.Outcome); v == "" {
			a.Outcome = nil
		} else {
			a.Outcome = &v
		}
	}
	if a.Notes != nil {
		if v := strings.TrimSpace(*a.Notes); v == "" {
			a.Notes = nil
		} else {
			a.Notes = &v
		}
	}
	if a.PriceCents < 0 {
		return &ValidationError{Msg: "price_cents não pode ser negativo"}
	}
	return nil
}

// AppointmentFilter recorta a listagem/agenda de consultas.
type AppointmentFilter struct {
	FamilyMemberID *uuid.UUID
	Status         AppointmentStatus
	Kind           AppointmentKind
	LabID          *uuid.UUID
	PlanID         *uuid.UUID
	From           *time.Time
	To             *time.Time
}

// AppointmentEnriched acompanha nomes desnormalizados para exibição.
type AppointmentEnriched struct {
	Appointment
	MemberName string
	LabName    *string
	PlanName   *string
}

// AgendaSpecialtyCount agrega consultas por especialidade no período.
type AgendaSpecialtyCount struct {
	Specialty string
	Count     int64
}

// AgendaMemberCount agrega consultas por membro no período.
type AgendaMemberCount struct {
	MemberID   uuid.UUID
	MemberName string
	Count      int64
}

// AgendaStatusCount agrega consultas por status no período.
type AgendaStatusCount struct {
	Status string
	Count  int64
}

// AppointmentRepository abstrai a persistência das consultas (workspace-scoped).
type AppointmentRepository interface {
	Create(ctx context.Context, a *Appointment) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Appointment, error)
	Update(ctx context.Context, a *Appointment) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter AppointmentFilter, limit, offset int) ([]AppointmentEnriched, int64, error)

	// Agregações da agenda (todas no ano informado, salvo Upcoming).
	Upcoming(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]AppointmentEnriched, error)
	StatusCounts(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]AgendaStatusCount, error)
	RealizedSpendCents(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) (int64, error)
	SpecialtyCounts(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]AgendaSpecialtyCount, error)
	MemberCounts(ctx context.Context, workspaceID uuid.UUID, from, to time.Time) ([]AgendaMemberCount, error)
	ActivePlansMonthlyFeeCents(ctx context.Context, workspaceID uuid.UUID) (int64, error)
}
