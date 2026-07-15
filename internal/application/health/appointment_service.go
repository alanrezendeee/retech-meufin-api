package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type AppointmentService struct {
	repo dom.AppointmentRepository
}

func NewAppointmentService(repo dom.AppointmentRepository) *AppointmentService {
	return &AppointmentService{repo: repo}
}

type CreateAppointmentInput struct {
	WorkspaceID      uuid.UUID
	FamilyMemberID   uuid.UUID
	Kind             dom.AppointmentKind
	Specialty        *dom.Specialty
	ProfessionalName *string
	LabID            *uuid.UUID
	ExamRequestID    *uuid.UUID
	PlanID           *uuid.UUID
	ScheduledAt      time.Time
	Status           dom.AppointmentStatus
	Reason           *string
	Outcome          *string
	PriceCents       int64
	CoveredByPlan    bool
	Notes            *string
}

type UpdateAppointmentInput struct {
	WorkspaceID      uuid.UUID
	ID               uuid.UUID
	FamilyMemberID   uuid.UUID
	Kind             dom.AppointmentKind
	Specialty        *dom.Specialty
	ProfessionalName *string
	LabID            *uuid.UUID
	ExamRequestID    *uuid.UUID
	PlanID           *uuid.UUID
	ScheduledAt      time.Time
	Status           dom.AppointmentStatus
	Reason           *string
	Outcome          *string
	PriceCents       int64
	CoveredByPlan    bool
	Notes            *string
}

func (s *AppointmentService) Create(ctx context.Context, in CreateAppointmentInput) (*dom.Appointment, error) {
	now := time.Now().UTC()
	a := &dom.Appointment{
		ID:               uuid.New(),
		WorkspaceID:      in.WorkspaceID,
		FamilyMemberID:   in.FamilyMemberID,
		Kind:             in.Kind,
		Specialty:        in.Specialty,
		ProfessionalName: in.ProfessionalName,
		LabID:            in.LabID,
		ExamRequestID:    in.ExamRequestID,
		PlanID:           in.PlanID,
		ScheduledAt:      in.ScheduledAt,
		Status:           in.Status,
		Reason:           in.Reason,
		Outcome:          in.Outcome,
		PriceCents:       in.PriceCents,
		CoveredByPlan:    in.CoveredByPlan,
		Notes:            in.Notes,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *AppointmentService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Appointment, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListAppointmentsResult struct {
	Items []dom.AppointmentEnriched
	Total int64
}

func (s *AppointmentService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.AppointmentFilter, limit, offset int) (*ListAppointmentsResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListAppointmentsResult{Items: items, Total: total}, nil
}

func (s *AppointmentService) Update(ctx context.Context, in UpdateAppointmentInput) (*dom.Appointment, error) {
	cur, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	// Bloqueia a máquina de estados quando o status muda via edição direta.
	if in.Status != "" && in.Status != cur.Status && !cur.Status.CanTransitionTo(in.Status) {
		return nil, &dom.ValidationError{Msg: "transição de status inválida a partir de " + string(cur.Status)}
	}

	cur.FamilyMemberID = in.FamilyMemberID
	cur.Kind = in.Kind
	cur.Specialty = in.Specialty
	cur.ProfessionalName = in.ProfessionalName
	cur.LabID = in.LabID
	cur.ExamRequestID = in.ExamRequestID
	cur.PlanID = in.PlanID
	cur.ScheduledAt = in.ScheduledAt
	if in.Status != "" {
		cur.Status = in.Status
	}
	cur.Reason = in.Reason
	cur.Outcome = in.Outcome
	cur.PriceCents = in.PriceCents
	cur.CoveredByPlan = in.CoveredByPlan
	cur.Notes = in.Notes
	cur.UpdatedAt = time.Now().UTC()

	if err := cur.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, cur); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
}

func (s *AppointmentService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

// transition aplica uma mudança de status validando a máquina de estados.
func (s *AppointmentService) transition(ctx context.Context, workspaceID, id uuid.UUID, to dom.AppointmentStatus, mutate func(a *dom.Appointment)) (*dom.Appointment, error) {
	a, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if !a.Status.CanTransitionTo(to) {
		return nil, &dom.ValidationError{Msg: "transição de status inválida: " + string(a.Status) + " → " + string(to)}
	}
	a.Status = to
	if mutate != nil {
		mutate(a)
	}
	a.UpdatedAt = time.Now().UTC()
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, workspaceID, id)
}

func (s *AppointmentService) Confirm(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Appointment, error) {
	return s.transition(ctx, workspaceID, id, dom.AppointmentStatusConfirmada, nil)
}

func (s *AppointmentService) Cancel(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Appointment, error) {
	return s.transition(ctx, workspaceID, id, dom.AppointmentStatusCancelada, nil)
}

func (s *AppointmentService) NoShow(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Appointment, error) {
	return s.transition(ctx, workspaceID, id, dom.AppointmentStatusFaltou, nil)
}

type CompleteAppointmentInput struct {
	Outcome    *string
	PriceCents *int64
}

func (s *AppointmentService) Complete(ctx context.Context, workspaceID, id uuid.UUID, in CompleteAppointmentInput) (*dom.Appointment, error) {
	return s.transition(ctx, workspaceID, id, dom.AppointmentStatusRealizada, func(a *dom.Appointment) {
		if in.Outcome != nil {
			a.Outcome = in.Outcome
		}
		if in.PriceCents != nil {
			a.PriceCents = *in.PriceCents
		}
	})
}

// --- Agenda ---

type Agenda struct {
	Year                 int
	Upcoming             []dom.AppointmentEnriched
	Next7Count           int
	Next30Count          int
	StatusCounts         []dom.AgendaStatusCount
	YearSpendCents       int64
	PlansMonthlyFeeCents int64
	PlansAnnualFeeCents  int64
	BySpecialty          []dom.AgendaSpecialtyCount
	ByMember             []dom.AgendaMemberCount
}

func (s *AppointmentService) GetAgenda(ctx context.Context, workspaceID uuid.UUID) (*Agenda, error) {
	now := time.Now().UTC()
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.UTC)
	in7 := now.AddDate(0, 0, 7)
	in30 := now.AddDate(0, 0, 30)

	upcoming, err := s.repo.Upcoming(ctx, workspaceID, now, in30)
	if err != nil {
		return nil, err
	}
	next7 := 0
	for i := range upcoming {
		if !upcoming[i].ScheduledAt.After(in7) {
			next7++
		}
	}

	statusCounts, err := s.repo.StatusCounts(ctx, workspaceID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	yearSpend, err := s.repo.RealizedSpendCents(ctx, workspaceID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	bySpecialty, err := s.repo.SpecialtyCounts(ctx, workspaceID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	byMember, err := s.repo.MemberCounts(ctx, workspaceID, yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	plansMonthly, err := s.repo.ActivePlansMonthlyFeeCents(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	return &Agenda{
		Year:                 now.Year(),
		Upcoming:             upcoming,
		Next7Count:           next7,
		Next30Count:          len(upcoming),
		StatusCounts:         statusCounts,
		YearSpendCents:       yearSpend,
		PlansMonthlyFeeCents: plansMonthly,
		PlansAnnualFeeCents:  plansMonthly * 12,
		BySpecialty:          bySpecialty,
		ByMember:             byMember,
	}, nil
}
