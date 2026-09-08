package finance

import (
	"sort"
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

// WithDayClamped retorna a data no mesmo ano/mês de d com o dia informado,
// limitado ao último dia do mês (ex.: dia 31 em fevereiro vira 28/29).
func WithDayClamped(d time.Time, day int) time.Time {
	loc := d.Location()
	if loc == nil {
		loc = time.UTC
	}
	return dateSameDayClamped(d.Year(), d.Month(), day, loc)
}

// RollingMonths é o horizonte de previstos das recorrências: sempre existe
// ~1 ano à frente (rolling), independente da virada do ano-calendário. O
// extensor de recorrências (application) completa o horizonte diariamente.
const RollingMonths = 12

// GenerateOccurrences gera as ocorrências de um lançamento conforme sua
// recorrência, num horizonte ROLLING de 12 meses a partir da due_date (a regra
// antiga parava em 31/dez e deixava janeiro vazio). Todas as ocorrências
// compartilham um recurrence_group_id e nascem 'prevista'.
//
//   - none: retorna [base] (1 ocorrência).
//   - monthly: 12 ocorrências, mesmo dia do mês (clamp p/ último dia).
//   - weekly: 52 ocorrências, a cada 7 dias.
//   - yearly: apenas a due_date (o extensor cria a próxima quando entrar no horizonte).
func GenerateOccurrences(base FinancialEntry) []FinancialEntry {
	loc := base.DueDate.Location()
	if loc == nil {
		loc = time.UTC
	}

	if base.Recurrence == RecurrenceNone || base.Recurrence == "" {
		base.Recurrence = RecurrenceNone
		return []FinancialEntry{cloneOccurrence(base, base.DueDate, nil)}
	}

	groupID := uuid.New()
	return append([]FinancialEntry{}, nextOccurrences(base, base.DueDate, &groupID, true, loc)...)
}

// NextOccurrencesAfter gera as ocorrências que faltam para completar o
// horizonte rolling de um grupo existente: tudo estritamente após `after`,
// até `horizon`. Usado pelo extensor diário de recorrências.
func NextOccurrencesAfter(template FinancialEntry, after, horizon time.Time) []FinancialEntry {
	loc := after.Location()
	if loc == nil {
		loc = time.UTC
	}
	groupID := template.RecurrenceGroupID
	if groupID == nil {
		return nil
	}
	out := make([]FinancialEntry, 0, RollingMonths)
	for _, occ := range nextOccurrences(template, after, groupID, false, loc) {
		// cloneOccurrence já zera status e todo o estado de liquidação.
		if occ.DueDate.After(after) && !occ.DueDate.After(horizon) {
			out = append(out, occ)
		}
	}
	return out
}

// nextOccurrences produz a série a partir de `from` (inclusive quando
// includeFrom). O dia-base do mês vem do template (DueDate original).
func nextOccurrences(template FinancialEntry, from time.Time, groupID *uuid.UUID, includeFrom bool, loc *time.Location) []FinancialEntry {
	switch template.Recurrence {
	case RecurrenceYearly:
		if includeFrom {
			return []FinancialEntry{cloneOccurrence(template, template.DueDate, groupID)}
		}
		// próxima ocorrência anual após `from`, no mesmo dia/mês do template
		next := time.Date(from.Year(), template.DueDate.Month(), template.DueDate.Day(), 0, 0, 0, 0, loc)
		for !next.After(from) {
			next = next.AddDate(1, 0, 0)
		}
		return []FinancialEntry{cloneOccurrence(template, next, groupID)}

	case RecurrenceMonthly:
		day := template.DueDate.Day()
		start := from
		if !includeFrom {
			start = from.AddDate(0, 1, 0)
		}
		baseYear := start.Year()
		baseMonth := int(start.Month())
		out := make([]FinancialEntry, 0, RollingMonths)
		for i := 0; i < RollingMonths; i++ {
			total0 := baseMonth - 1 + i
			year := baseYear + total0/12
			month := time.Month(total0%12 + 1)
			out = append(out, cloneOccurrence(template, dateSameDayClamped(year, month, day, loc), groupID))
		}
		return out

	case RecurrenceWeekly:
		start := from
		if !includeFrom {
			start = from.AddDate(0, 0, 7)
		}
		out := make([]FinancialEntry, 0, 52)
		for i, d := 0, start; i < 52; i, d = i+1, d.AddDate(0, 0, 7) {
			out = append(out, cloneOccurrence(template, d, groupID))
		}
		return out

	default:
		return []FinancialEntry{cloneOccurrence(template, template.DueDate, groupID)}
	}
}

// GenerateInstallments gera N lançamentos mensais de uma compra parcelada, um por mês
// a partir da due_date do base (installment_number 1..total, installment_total = total).
// Todos compartilham um recurrence_group_id, são criados com status 'prevista' e
// recurrence 'none'. Diferente das ocorrências recorrentes, as parcelas NÃO param em
// dezembro — cruzam o ano até completar o total. O dia é limitado ao último dia do mês
// (clamp) quando não existe (ex.: 31/jan -> 28/fev).
func GenerateInstallments(base FinancialEntry, total int) []FinancialEntry {
	if total < 1 {
		total = 1
	}
	loc := base.DueDate.Location()
	if loc == nil {
		loc = time.UTC
	}
	groupID := uuid.New()
	day := base.DueDate.Day()
	baseYear := base.DueDate.Year()
	baseMonth := int(base.DueDate.Month())

	out := make([]FinancialEntry, 0, total)
	for i := 0; i < total; i++ {
		// Avança i meses a partir do mês-base sem depender de AddDate (que
		// normaliza overflow de dia, ex.: 31/jan +1 mês -> 03/mar).
		total0 := baseMonth - 1 + i
		year := baseYear + total0/12
		month := time.Month(total0%12 + 1)
		due := dateSameDayClamped(year, month, day, loc)
		occ := cloneOccurrence(base, due, &groupID)
		occ.Recurrence = RecurrenceNone
		num := i + 1
		tot := total
		occ.InstallmentNumber = &num
		occ.InstallmentTotal = &tot
		out = append(out, occ)
	}
	return out
}

// cloneOccurrence copia o lançamento base ajustando due_date, id, group_id e status.
//
// A ocorrência nova nasce SEMPRE prevista, então todo estado de liquidação do
// template é descartado: pagamento, desconto e motivo de cancelamento
// pertencem à ocorrência que os originou, não às futuras. Sem isso, estender
// uma série cuja última ocorrência foi paga com desconto (ou cancelada)
// contaminava todos os meses seguintes.
func cloneOccurrence(base FinancialEntry, due time.Time, groupID *uuid.UUID) FinancialEntry {
	occ := base
	occ.ID = uuid.New()
	occ.DueDate = due
	occ.RecurrenceGroupID = groupID
	occ.Status = StatusPrevista
	occ.PaidAt = nil
	occ.PaidAmountCents = nil
	occ.PaymentMethod = nil
	occ.PaymentAccountID = nil
	occ.PaymentCardID = nil
	occ.DiscountCents = nil
	occ.DiscountReason = nil
	occ.CancelReason = nil
	occ.ResidualOfID = nil
	return occ
}

// InstallmentSpec descreve uma parcela informada pelo usuário no lançamento
// de um parcelamento: vencimento e valor próprios e, opcionalmente, a
// liquidação já ocorrida (parcelamento antigo lançado retroativamente).
type InstallmentSpec struct {
	DueDate     time.Time
	AmountCents int64
	// PaidAt presente: a parcela nasce realizada, paga nessa data.
	PaidAt *time.Time
	// PaidAmountCents: valor efetivamente pago; default é AmountCents.
	PaidAmountCents *int64
}

// GenerateCustomInstallments monta as parcelas a partir da lista informada
// pelo usuário, em vez da progressão mensal automática. As parcelas são
// ordenadas por vencimento e numeradas nessa ordem; todas compartilham o
// mesmo grupo. Parcela com PaidAt nasce realizada com paid_amount_cents
// (default: o valor da parcela) — sem residual nem desconto, que só existem
// pelo fluxo de liquidação.
func GenerateCustomInstallments(base FinancialEntry, specs []InstallmentSpec) []FinancialEntry {
	sorted := make([]InstallmentSpec, len(specs))
	copy(sorted, specs)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].DueDate.Before(sorted[j].DueDate) })

	groupID := uuid.New()
	total := len(sorted)
	out := make([]FinancialEntry, 0, total)
	for i, sp := range sorted {
		occ := cloneOccurrence(base, sp.DueDate, &groupID)
		occ.Recurrence = RecurrenceNone
		occ.AmountCents = sp.AmountCents
		num := i + 1
		tot := total
		occ.InstallmentNumber = &num
		occ.InstallmentTotal = &tot
		if sp.PaidAt != nil {
			paidAt := sp.PaidAt.UTC()
			paid := sp.AmountCents
			if sp.PaidAmountCents != nil {
				paid = *sp.PaidAmountCents
			}
			occ.Status = StatusRealizada
			occ.PaidAt = &paidAt
			occ.PaidAmountCents = &paid
		}
		out = append(out, occ)
	}
	return out
}
