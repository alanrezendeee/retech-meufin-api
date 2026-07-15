package health

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAppointmentStatusTransitions(t *testing.T) {
	cases := []struct {
		from AppointmentStatus
		to   AppointmentStatus
		want bool
	}{
		// agendada pode confirmar, realizar, cancelar ou faltar
		{AppointmentStatusAgendada, AppointmentStatusConfirmada, true},
		{AppointmentStatusAgendada, AppointmentStatusRealizada, true},
		{AppointmentStatusAgendada, AppointmentStatusCancelada, true},
		{AppointmentStatusAgendada, AppointmentStatusFaltou, true},
		// confirmada avança para os estados terminais
		{AppointmentStatusConfirmada, AppointmentStatusRealizada, true},
		{AppointmentStatusConfirmada, AppointmentStatusCancelada, true},
		{AppointmentStatusConfirmada, AppointmentStatusFaltou, true},
		// não pode "voltar" de confirmada para agendada
		{AppointmentStatusConfirmada, AppointmentStatusAgendada, false},
		// realizada é terminal
		{AppointmentStatusRealizada, AppointmentStatusCancelada, false},
		{AppointmentStatusRealizada, AppointmentStatusConfirmada, false},
		{AppointmentStatusRealizada, AppointmentStatusAgendada, false},
		// cancelada e faltou são terminais
		{AppointmentStatusCancelada, AppointmentStatusRealizada, false},
		{AppointmentStatusFaltou, AppointmentStatusRealizada, false},
		// mesmo estado é sempre permitido (idempotente)
		{AppointmentStatusRealizada, AppointmentStatusRealizada, true},
		{AppointmentStatusAgendada, AppointmentStatusAgendada, true},
	}
	for _, c := range cases {
		if got := c.from.CanTransitionTo(c.to); got != c.want {
			t.Errorf("CanTransitionTo(%s → %s) = %v, quer %v", c.from, c.to, got, c.want)
		}
	}
}

func TestAppointmentStatusIsTerminal(t *testing.T) {
	terminal := []AppointmentStatus{AppointmentStatusRealizada, AppointmentStatusCancelada, AppointmentStatusFaltou}
	for _, s := range terminal {
		if !s.IsTerminal() {
			t.Errorf("esperava %s terminal", s)
		}
	}
	nonTerminal := []AppointmentStatus{AppointmentStatusAgendada, AppointmentStatusConfirmada}
	for _, s := range nonTerminal {
		if s.IsTerminal() {
			t.Errorf("não esperava %s terminal", s)
		}
	}
}

func TestAppointmentValidateDefaults(t *testing.T) {
	a := &Appointment{
		WorkspaceID:    uuid.New(),
		FamilyMemberID: uuid.New(),
		ScheduledAt:    time.Now(),
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("Validate() erro inesperado: %v", err)
	}
	if a.Kind != AppointmentKindConsulta {
		t.Errorf("kind padrão = %s, quer consulta", a.Kind)
	}
	if a.Status != AppointmentStatusAgendada {
		t.Errorf("status padrão = %s, quer agendada", a.Status)
	}
}

func TestAppointmentValidateRejectsBadEnums(t *testing.T) {
	base := func() *Appointment {
		return &Appointment{WorkspaceID: uuid.New(), FamilyMemberID: uuid.New(), ScheduledAt: time.Now()}
	}
	a := base()
	a.Kind = AppointmentKind("invalido")
	if err := a.Validate(); err == nil {
		t.Error("esperava erro para kind inválido")
	}
	a = base()
	bad := Specialty("nao_existe")
	a.Specialty = &bad
	if err := a.Validate(); err == nil {
		t.Error("esperava erro para specialty inválida")
	}
	a = base()
	a.Status = AppointmentStatus("qualquer")
	if err := a.Validate(); err == nil {
		t.Error("esperava erro para status inválido")
	}
	a = base()
	a.PriceCents = -1
	if err := a.Validate(); err == nil {
		t.Error("esperava erro para price_cents negativo")
	}
}
