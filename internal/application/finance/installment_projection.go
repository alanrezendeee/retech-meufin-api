package finance

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// Projeção de compromissos parcelados dentro de faturas de cartão.
//
// A mesma compra parcelada aparece em TODA fatura ("LOJA PARC 03/10" num mês,
// "LOJA PARC 04/10" no seguinte). A projeção agrupa essas ocorrências pela
// identidade do parcelamento (cartão + descrição normalizada + total de
// parcelas + valor), usa a ocorrência mais recente como estado atual e
// projeta as parcelas restantes nos meses seguintes.
//
// IMPORTANTE: é PROJEÇÃO CALCULADA, nunca lançamentos previstos — criar
// previstos duplicaria tudo no import da fatura seguinte.

// InstallmentGroup é um parcelamento ativo identificado nas faturas.
type InstallmentGroup struct {
	Description      string
	CardID           *uuid.UUID
	Category         *string
	InstallmentCents int64
	InstallmentTotal int
	LastKnownNumber  int
	RemainingCount   int
	RemainingCents   int64
	// LastDueDate: vencimento da fatura onde a última parcela conhecida caiu.
	LastDueDate time.Time
	// EndsAt: mês estimado da última parcela.
	EndsAt time.Time
}

// MonthlyCommitment é o total projetado de parcelas num mês futuro.
type MonthlyCommitment struct {
	Month      string // "YYYY-MM"
	TotalCents int64
	Count      int
}

// InstallmentProjection agrega parcelamentos ativos e o comprometimento mensal.
type InstallmentProjection struct {
	Groups              []InstallmentGroup
	Monthly             []MonthlyCommitment
	RemainingTotalCents int64
}

// parcelaMarkRe remove os marcadores de parcela da descrição para que
// "LOJA PARC 03/10" e "LOJA PARC 04/10" caiam no mesmo grupo.
var parcelaMarkRe = regexp.MustCompile(`\b\d{1,2}\s*(?:/|DE\s+)\s*\d{1,2}\b`)
var spacesRe = regexp.MustCompile(`\s+`)

func normalizeInstallmentDesc(desc string) string {
	d := strings.ToUpper(strings.TrimSpace(desc))
	d = parcelaMarkRe.ReplaceAllString(d, "")
	d = spacesRe.ReplaceAllString(d, " ")
	return strings.TrimSpace(d)
}

// InstallmentsProjection calcula a projeção do workspace.
func (s *FinancialEntryService) InstallmentsProjection(ctx context.Context, workspaceID uuid.UUID) (*InstallmentProjection, error) {
	rows, err := s.repo.ListInvoiceInstallments(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	type groupState struct {
		latest dom.FinancialEntry
		norm   string
	}
	groups := map[string]*groupState{}
	for i := range rows {
		e := rows[i]
		if e.InstallmentNumber == nil || e.InstallmentTotal == nil || *e.InstallmentTotal < 1 {
			continue
		}
		norm := normalizeInstallmentDesc(e.Description)
		card := ""
		if e.CardID != nil {
			card = e.CardID.String()
		}
		key := strings.Join([]string{card, norm,
			strconv.Itoa(*e.InstallmentTotal), strconv.FormatInt(e.AmountCents, 10)}, "|")
		g, ok := groups[key]
		if !ok {
			groups[key] = &groupState{latest: e, norm: norm}
			continue
		}
		// Estado mais recente: maior parcela conhecida; empate decide pela data.
		if *e.InstallmentNumber > *g.latest.InstallmentNumber ||
			(*e.InstallmentNumber == *g.latest.InstallmentNumber && e.DueDate.After(g.latest.DueDate)) {
			g.latest = e
		}
	}

	out := &InstallmentProjection{Groups: []InstallmentGroup{}, Monthly: []MonthlyCommitment{}}
	monthly := map[string]*MonthlyCommitment{}
	for _, g := range groups {
		e := g.latest
		remaining := *e.InstallmentTotal - *e.InstallmentNumber
		if remaining <= 0 {
			continue // parcelamento quitado
		}
		grp := InstallmentGroup{
			Description:      strings.TrimSpace(g.norm),
			CardID:           e.CardID,
			Category:         e.Type,
			InstallmentCents: e.AmountCents,
			InstallmentTotal: *e.InstallmentTotal,
			LastKnownNumber:  *e.InstallmentNumber,
			RemainingCount:   remaining,
			RemainingCents:   int64(remaining) * e.AmountCents,
			LastDueDate:      e.DueDate,
			EndsAt:           e.DueDate.AddDate(0, remaining, 0),
		}
		out.Groups = append(out.Groups, grp)
		out.RemainingTotalCents += grp.RemainingCents

		for k := 1; k <= remaining; k++ {
			m := e.DueDate.AddDate(0, k, 0).Format("2006-01")
			b, ok := monthly[m]
			if !ok {
				b = &MonthlyCommitment{Month: m}
				monthly[m] = b
			}
			b.TotalCents += e.AmountCents
			b.Count++
		}
	}

	for _, b := range monthly {
		out.Monthly = append(out.Monthly, *b)
	}
	sort.Slice(out.Monthly, func(i, j int) bool { return out.Monthly[i].Month < out.Monthly[j].Month })
	sort.Slice(out.Groups, func(i, j int) bool {
		if out.Groups[i].RemainingCents != out.Groups[j].RemainingCents {
			return out.Groups[i].RemainingCents > out.Groups[j].RemainingCents
		}
		return out.Groups[i].Description < out.Groups[j].Description
	})
	return out, nil
}
