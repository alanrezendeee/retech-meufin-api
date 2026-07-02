package finance

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func baseEntry(due time.Time, rec Recurrence) FinancialEntry {
	return FinancialEntry{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		Kind:        KindDebit,
		Status:      StatusPrevista,
		AmountCents: 1000,
		DueDate:     due,
		Recurrence:  rec,
	}
}

func TestGenerateOccurrences_None(t *testing.T) {
	due := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	occ := GenerateOccurrences(baseEntry(due, RecurrenceNone))
	if len(occ) != 1 {
		t.Fatalf("none: esperado 1 ocorrência, obtido %d", len(occ))
	}
	if !occ[0].DueDate.Equal(due) {
		t.Fatalf("none: due_date esperada %v, obtida %v", due, occ[0].DueDate)
	}
	if occ[0].RecurrenceGroupID != nil {
		t.Fatalf("none: recurrence_group_id deveria ser nil")
	}
}

func TestGenerateOccurrences_Yearly(t *testing.T) {
	due := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC)
	occ := GenerateOccurrences(baseEntry(due, RecurrenceYearly))
	if len(occ) != 1 {
		t.Fatalf("yearly: esperado 1 ocorrência, obtido %d", len(occ))
	}
	if occ[0].RecurrenceGroupID == nil {
		t.Fatalf("yearly: recurrence_group_id não deveria ser nil")
	}
	if occ[0].Status != StatusPrevista {
		t.Fatalf("yearly: status esperado prevista, obtido %s", occ[0].Status)
	}
}

func TestGenerateOccurrences_MonthlyRolling12(t *testing.T) {
	due := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	occ := GenerateOccurrences(baseEntry(due, RecurrenceMonthly))
	if len(occ) != 12 {
		t.Fatalf("monthly rolling: esperado 12 ocorrências (cruzando o ano), obtido %d", len(occ))
	}
	wantMonths := []time.Month{
		time.July, time.August, time.September, time.October, time.November, time.December,
		time.January, time.February, time.March, time.April, time.May, time.June,
	}
	for i, m := range wantMonths {
		if occ[i].DueDate.Month() != m {
			t.Fatalf("monthly: ocorrência %d esperava mês %v, obtido %v", i, m, occ[i].DueDate.Month())
		}
		if occ[i].DueDate.Day() != 15 {
			t.Fatalf("monthly: ocorrência %d esperava dia 15, obtido %d", i, occ[i].DueDate.Day())
		}
	}
	// mesmo group em todas
	group := occ[0].RecurrenceGroupID
	if group == nil {
		t.Fatalf("monthly: group_id nil")
	}
	for i := range occ {
		if occ[i].RecurrenceGroupID == nil || *occ[i].RecurrenceGroupID != *group {
			t.Fatalf("monthly: group_id divergente na ocorrência %d", i)
		}
	}
}

func TestGenerateOccurrences_MonthlyDayOverflow(t *testing.T) {
	// 31 de janeiro -> fevereiro deve virar 28 (2026 não é bissexto).
	due := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	occ := GenerateOccurrences(baseEntry(due, RecurrenceMonthly))
	if len(occ) != 12 {
		t.Fatalf("monthly overflow: esperado 12 ocorrências, obtido %d", len(occ))
	}
	feb := occ[1]
	if feb.DueDate.Month() != time.February {
		t.Fatalf("monthly overflow: segunda ocorrência esperava fevereiro, obtido %v", feb.DueDate.Month())
	}
	if feb.DueDate.Day() != 28 {
		t.Fatalf("monthly overflow: fevereiro esperava dia 28, obtido %d", feb.DueDate.Day())
	}
	// janeiro e março mantêm 31
	if occ[0].DueDate.Day() != 31 {
		t.Fatalf("monthly overflow: janeiro esperava dia 31, obtido %d", occ[0].DueDate.Day())
	}
	if occ[2].DueDate.Day() != 31 {
		t.Fatalf("monthly overflow: março esperava dia 31, obtido %d", occ[2].DueDate.Day())
	}
}

func TestGenerateOccurrences_Weekly(t *testing.T) {
	due := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	occ := GenerateOccurrences(baseEntry(due, RecurrenceWeekly))
	// Rolling: 52 semanas a partir da due_date, cruzando o ano.
	if len(occ) != 52 {
		t.Fatalf("weekly rolling: esperado 52 ocorrências, obtido %d", len(occ))
	}
	prev := occ[0].DueDate
	for i := 1; i < len(occ); i++ {
		diff := occ[i].DueDate.Sub(prev)
		if diff != 7*24*time.Hour {
			t.Fatalf("weekly: ocorrência %d não está a 7 dias da anterior (%v)", i, diff)
		}
		prev = occ[i].DueDate
	}
	last := occ[len(occ)-1].DueDate
	if last.Year() != 2027 {
		t.Fatalf("weekly rolling: última ocorrência deveria cruzar pra 2027, obtido %v", last)
	}
}

func TestGenerateInstallments_Count(t *testing.T) {
	due := time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC)
	occ := GenerateInstallments(baseEntry(due, RecurrenceNone), 15)
	if len(occ) != 15 {
		t.Fatalf("installments: esperado 15, obtido %d", len(occ))
	}
	// datas mensais consecutivas cruzando o ano
	for i := range occ {
		want := due.AddDate(0, i, 0)
		if occ[i].DueDate.Month() != want.Month() || occ[i].DueDate.Year() != want.Year() {
			t.Fatalf("installments: parcela %d esperava %v/%d, obtido %v/%d",
				i, want.Month(), want.Year(), occ[i].DueDate.Month(), occ[i].DueDate.Year())
		}
		if occ[i].DueDate.Day() != 15 {
			t.Fatalf("installments: parcela %d esperava dia 15, obtido %d", i, occ[i].DueDate.Day())
		}
		if occ[i].InstallmentNumber == nil || *occ[i].InstallmentNumber != i+1 {
			t.Fatalf("installments: parcela %d installment_number incorreto", i)
		}
		if occ[i].InstallmentTotal == nil || *occ[i].InstallmentTotal != 15 {
			t.Fatalf("installments: parcela %d installment_total incorreto", i)
		}
		if occ[i].Recurrence != RecurrenceNone {
			t.Fatalf("installments: parcela %d recurrence deveria ser none", i)
		}
		if occ[i].Status != StatusPrevista {
			t.Fatalf("installments: parcela %d status deveria ser prevista", i)
		}
	}
	// mesmo group em todas
	group := occ[0].RecurrenceGroupID
	if group == nil {
		t.Fatalf("installments: group_id nil")
	}
	for i := range occ {
		if occ[i].RecurrenceGroupID == nil || *occ[i].RecurrenceGroupID != *group {
			t.Fatalf("installments: group_id divergente na parcela %d", i)
		}
	}
	// cruza o ano: parcela 6 (índice 5) já em dezembro, parcela 7 em janeiro/2027
	if occ[6].DueDate.Year() != 2027 {
		t.Fatalf("installments: parcela 7 deveria estar em 2027, obtido %d", occ[6].DueDate.Year())
	}
}

func TestGenerateInstallments_ClampJan31(t *testing.T) {
	// 31/jan: fevereiro deve virar 28 (2026 não bissexto).
	due := time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)
	occ := GenerateInstallments(baseEntry(due, RecurrenceNone), 3)
	if len(occ) != 3 {
		t.Fatalf("clamp: esperado 3, obtido %d", len(occ))
	}
	if occ[0].DueDate.Day() != 31 {
		t.Fatalf("clamp: janeiro esperava 31, obtido %d", occ[0].DueDate.Day())
	}
	if occ[1].DueDate.Month() != time.February || occ[1].DueDate.Day() != 28 {
		t.Fatalf("clamp: fevereiro esperava 28, obtido %v/%d", occ[1].DueDate.Month(), occ[1].DueDate.Day())
	}
	if occ[2].DueDate.Month() != time.March || occ[2].DueDate.Day() != 31 {
		t.Fatalf("clamp: março esperava 31, obtido %v/%d", occ[2].DueDate.Month(), occ[2].DueDate.Day())
	}
}

func TestNextOccurrencesAfter_Monthly(t *testing.T) {
	group := uuid.New()
	template := baseEntry(time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC), RecurrenceMonthly)
	template.RecurrenceGroupID = &group

	after := time.Date(2027, time.June, 15, 0, 0, 0, 0, time.UTC) // fronteira atual do grupo
	horizon := time.Date(2027, time.October, 15, 0, 0, 0, 0, time.UTC)
	occ := NextOccurrencesAfter(template, after, horizon)

	if len(occ) != 4 {
		t.Fatalf("extensor: esperado 4 (jul..out/27), obtido %d", len(occ))
	}
	if !occ[0].DueDate.After(after) {
		t.Fatalf("extensor: primeira ocorrência %v não é posterior à fronteira %v", occ[0].DueDate, after)
	}
	for i := range occ {
		if occ[i].Status != StatusPrevista {
			t.Fatalf("extensor: ocorrência %d deveria nascer prevista", i)
		}
		if occ[i].RecurrenceGroupID == nil || *occ[i].RecurrenceGroupID != group {
			t.Fatalf("extensor: ocorrência %d fora do grupo", i)
		}
	}
}
