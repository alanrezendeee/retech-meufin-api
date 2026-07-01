package finance

import (
	"time"

	"github.com/google/uuid"
)

// lastDayOfMonth retorna o último dia do mês/ano informados.
func lastDayOfMonth(year int, month time.Month, loc *time.Location) int {
	// Dia 0 do mês seguinte = último dia do mês corrente.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
}

// dateSameDayClamped monta uma data no ano/mês com o dia desejado, limitando
// ao último dia do mês quando o dia não existe (ex.: 31 em fevereiro).
func dateSameDayClamped(year int, month time.Month, day int, loc *time.Location) time.Time {
	last := lastDayOfMonth(year, month, loc)
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// GenerateOccurrences gera as ocorrências de um lançamento conforme sua recorrência,
// até 31/dez do ano da due_date (exercício = ano-calendário). Todas as ocorrências
// compartilham um recurrence_group_id e são criadas com status 'prevista'.
//
//   - none: retorna [base] (1 ocorrência).
//   - monthly: uma por mês, mesmo dia do mês (com clamp para o último dia), de due_date até dezembro.
//   - weekly: a cada 7 dias a partir de due_date até 31/dez.
//   - yearly: apenas a due_date (1 ocorrência no exercício).
func GenerateOccurrences(base FinancialEntry) []FinancialEntry {
	loc := base.DueDate.Location()
	if loc == nil {
		loc = time.UTC
	}
	year := base.DueDate.Year()
	yearEnd := time.Date(year, time.December, 31, 0, 0, 0, 0, loc)

	if base.Recurrence == RecurrenceNone || base.Recurrence == "" {
		base.Recurrence = RecurrenceNone
		return []FinancialEntry{cloneOccurrence(base, base.DueDate, nil)}
	}

	groupID := uuid.New()

	switch base.Recurrence {
	case RecurrenceYearly:
		return []FinancialEntry{cloneOccurrence(base, base.DueDate, &groupID)}

	case RecurrenceMonthly:
		out := make([]FinancialEntry, 0, 12)
		day := base.DueDate.Day()
		for m := base.DueDate.Month(); m <= time.December; m++ {
			d := dateSameDayClamped(year, m, day, loc)
			out = append(out, cloneOccurrence(base, d, &groupID))
		}
		return out

	case RecurrenceWeekly:
		out := make([]FinancialEntry, 0, 53)
		for d := base.DueDate; !d.After(yearEnd); d = d.AddDate(0, 0, 7) {
			out = append(out, cloneOccurrence(base, d, &groupID))
		}
		return out

	default:
		return []FinancialEntry{cloneOccurrence(base, base.DueDate, &groupID)}
	}
}

// cloneOccurrence copia o lançamento base ajustando due_date, id, group_id e status.
func cloneOccurrence(base FinancialEntry, due time.Time, groupID *uuid.UUID) FinancialEntry {
	occ := base
	occ.ID = uuid.New()
	occ.DueDate = due
	occ.RecurrenceGroupID = groupID
	occ.Status = StatusPrevista
	return occ
}
