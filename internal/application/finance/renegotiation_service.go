package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// RenegotiationService apura o saldo em aberto de uma dívida parcelada e
// aplica a repactuação (novação): encerra as cobranças em aberto e cria a
// série nova, vinculando os dois lados ao mesmo evento.
type RenegotiationService struct {
	entries dom.FinancialEntryRepository
	renegs  dom.RenegotiationRepository
}

func NewRenegotiationService(entries dom.FinancialEntryRepository, renegs dom.RenegotiationRepository) *RenegotiationService {
	return &RenegotiationService{entries: entries, renegs: renegs}
}

// OpenChargeKind distingue a natureza de cada cobrança em aberto apurada.
type OpenChargeKind string

const (
	// ChargeInstallment: parcela prevista do parcelamento.
	ChargeInstallment OpenChargeKind = "installment"
	// ChargeResidual: saldo não pago de uma parcela liquidada parcialmente.
	ChargeResidual OpenChargeKind = "residual"
)

// ChargeStatus é a situação da cobrança na data da apuração.
type ChargeStatus string

const (
	// ChargePaid: quitada integralmente. Não entra na renegociação.
	ChargePaid ChargeStatus = "paid"
	// ChargePartiallyPaid: paga em parte. Não entra pelo valor cheio — o que
	// faltou existe como residual próprio, que entra no lugar.
	ChargePartiallyPaid ChargeStatus = "partially_paid"
	// ChargeOverdue: em aberto e vencida.
	ChargeOverdue ChargeStatus = "overdue"
	// ChargeUpcoming: em aberto, ainda a vencer.
	ChargeUpcoming ChargeStatus = "upcoming"
)

// OpenCharge é uma cobrança do parcelamento. A apuração devolve a série
// inteira — inclusive o que já foi pago — para dar contexto na tela; o campo
// Included é o que distingue o que efetivamente entra na renegociação.
type OpenCharge struct {
	ID          uuid.UUID
	Kind        OpenChargeKind
	Status      ChargeStatus
	Description string
	AmountCents int64
	// PaidAmountCents: quanto foi pago (em quitadas e parcialmente pagas).
	PaidAmountCents *int64
	DueDate         time.Time
	// Included indica se a cobrança compõe o saldo renegociado.
	Included bool
	// InstallmentNumber é o número da parcela; em residual, o da parcela de origem.
	InstallmentNumber *int
	// OriginDescription: em residual, a parcela que o originou.
	OriginDescription *string
}

// RenegotiationPreview é a apuração do saldo devedor, antes de qualquer
// alteração. É o principal produto da tela: o usuário não sabe de cabeça
// quanto ainda deve, sobretudo quando há residuais de pagamentos parciais.
type RenegotiationPreview struct {
	GroupID     uuid.UUID
	Description string
	// Contexto do parcelamento original.
	InstallmentTotal int
	PaidCount        int
	PaidCents        int64
	// Cobranças que entram na renegociação.
	Charges           []OpenCharge
	InstallmentCount  int
	InstallmentCents  int64
	ResidualCount     int
	ResidualCents     int64
	OpenTotalCents    int64
	OverdueCount      int
	OverdueCents      int64
	NextDueDate       *time.Time
	SuggestedDueDate  time.Time
	TypicalAmountCent int64
}

// Preview apura o saldo em aberto do parcelamento a que o lançamento pertence.
//
// Regra que evita contagem em dobro: parcela paga parcialmente NÃO entra pelo
// seu valor cheio — ela está realizada, e o que faltou dela já existe como
// residual. Entram apenas as parcelas previstas e os residuais em aberto.
func (s *RenegotiationService) Preview(ctx context.Context, workspaceID, entryID uuid.UUID) (*RenegotiationPreview, error) {
	anchor, err := s.entries.GetByID(ctx, workspaceID, entryID)
	if err != nil {
		return nil, err
	}
	if anchor.RecurrenceGroupID == nil || anchor.InstallmentTotal == nil {
		return nil, &dom.ValidationError{Msg: "o lançamento não é uma parcela de parcelamento"}
	}
	return s.previewGroup(ctx, workspaceID, *anchor.RecurrenceGroupID)
}

// PreviewGroup apura pelo identificador do grupo (usado pela tela de
// parcelamentos, que trabalha com o grupo e não com uma parcela).
func (s *RenegotiationService) PreviewGroup(ctx context.Context, workspaceID, groupID uuid.UUID) (*RenegotiationPreview, error) {
	return s.previewGroup(ctx, workspaceID, groupID)
}

func (s *RenegotiationService) previewGroup(ctx context.Context, workspaceID, groupID uuid.UUID) (*RenegotiationPreview, error) {
	series, err := s.entries.ListGroup(ctx, workspaceID, groupID)
	if err != nil {
		return nil, err
	}
	if len(series) == 0 {
		return nil, dom.ErrNotFound
	}

	out := &RenegotiationPreview{GroupID: groupID, Description: series[0].Description}
	if series[0].InstallmentTotal != nil {
		out.InstallmentTotal = *series[0].InstallmentTotal
	}

	// Hoje, normalizado por data: parcela que vence hoje não está atrasada.
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// A série inteira vai para a resposta (contexto na tela); só as previstas
	// entram no saldo. Realizadas contam para o histórico e são a origem
	// possível de residuais.
	originIDs := make([]uuid.UUID, 0, len(series))
	for i := range series {
		e := &series[i]
		originIDs = append(originIDs, e.ID)
		switch e.Status {
		case dom.StatusPrevista:
			out.Charges = append(out.Charges, OpenCharge{
				ID:                e.ID,
				Kind:              ChargeInstallment,
				Status:            chargeStatusFor(e.DueDate, today),
				Description:       e.Description,
				AmountCents:       e.AmountCents,
				DueDate:           e.DueDate,
				Included:          true,
				InstallmentNumber: e.InstallmentNumber,
			})
			out.InstallmentCount++
			out.InstallmentCents += e.AmountCents
			if out.TypicalAmountCent == 0 {
				out.TypicalAmountCent = e.AmountCents
			}
		case dom.StatusRealizada:
			out.PaidCount++
			paid := e.AmountCents
			if e.PaidAmountCents != nil {
				paid = *e.PaidAmountCents
			}
			out.PaidCents += paid
			paidCopy := paid
			// Pago a menor sinaliza liquidação parcial: o saldo dela está num
			// residual próprio, e é ele que entra na renegociação.
			status := ChargePaid
			if paid < e.AmountCents {
				status = ChargePartiallyPaid
			}
			out.Charges = append(out.Charges, OpenCharge{
				ID:                e.ID,
				Kind:              ChargeInstallment,
				Status:            status,
				Description:       e.Description,
				AmountCents:       e.AmountCents,
				PaidAmountCents:   &paidCopy,
				DueDate:           e.DueDate,
				Included:          false,
				InstallmentNumber: e.InstallmentNumber,
			})
		}
	}

	// Residuais em aberto de parcelas pagas parcialmente.
	residuals, err := s.entries.ListResidualsOf(ctx, workspaceID, originIDs)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]*dom.FinancialEntry{}
	for i := range series {
		byID[series[i].ID] = &series[i]
	}
	for i := range residuals {
		r := &residuals[i]
		if r.Status != dom.StatusPrevista {
			continue
		}
		charge := OpenCharge{
			ID:          r.ID,
			Kind:        ChargeResidual,
			Status:      chargeStatusFor(r.DueDate, today),
			Description: r.Description,
			AmountCents: r.AmountCents,
			DueDate:     r.DueDate,
			Included:    true,
		}
		if r.ResidualOfID != nil {
			if origin, ok := byID[*r.ResidualOfID]; ok {
				d := origin.Description
				charge.OriginDescription = &d
				charge.InstallmentNumber = origin.InstallmentNumber
			}
		}
		out.Charges = append(out.Charges, charge)
		out.ResidualCount++
		out.ResidualCents += r.AmountCents
	}

	out.OpenTotalCents = out.InstallmentCents + out.ResidualCents
	for i := range out.Charges {
		if !out.Charges[i].Included {
			continue
		}
		d := out.Charges[i].DueDate
		if out.NextDueDate == nil || d.Before(*out.NextDueDate) {
			out.NextDueDate = &d
		}
		if out.Charges[i].Status == ChargeOverdue {
			out.OverdueCount++
			out.OverdueCents += out.Charges[i].AmountCents
		}
	}
	// Sugestão de primeiro vencimento: o mês seguinte ao atual, no dia da
	// próxima cobrança em aberto (ou no dia de hoje, se não houver).
	day := now.Day()
	if out.NextDueDate != nil {
		day = out.NextDueDate.Day()
	}
	out.SuggestedDueDate = dom.WithDayClamped(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0), day)

	return out, nil
}

// RenegotiateInput descreve o novo acordo.
type RenegotiateInput struct {
	WorkspaceID uuid.UUID
	GroupID     uuid.UUID
	// Date é a data do acordo (default: hoje).
	Date *time.Time
	// InstallmentCount e InstallmentCents definem a série nova.
	InstallmentCount int
	InstallmentCents int64
	FirstDueDate     time.Time
	// Description opcional para as parcelas novas (default: a da dívida).
	Description string
	Notes       *string
}

type RenegotiateResult struct {
	Renegotiation *dom.Renegotiation
	Created       []dom.FinancialEntry
}

// Renegotiate aplica a novação: apura o saldo, encerra as cobranças em aberto
// e cria a série nova, tudo vinculado a um evento e numa única transação.
func (s *RenegotiationService) Renegotiate(ctx context.Context, in RenegotiateInput) (*RenegotiateResult, error) {
	if in.InstallmentCount < 1 {
		return nil, &dom.ValidationError{Msg: "informe ao menos uma parcela para o novo acordo"}
	}
	if in.InstallmentCents <= 0 {
		return nil, &dom.ValidationError{Msg: "o valor da parcela deve ser maior que zero"}
	}
	if in.FirstDueDate.IsZero() {
		return nil, &dom.ValidationError{Msg: "informe o vencimento da primeira parcela"}
	}

	preview, err := s.previewGroup(ctx, in.WorkspaceID, in.GroupID)
	if err != nil {
		return nil, err
	}
	// Charges traz a série inteira (contexto da tela); só as incluídas são
	// as cobranças em aberto que efetivamente entram no acordo.
	originIDs := make([]uuid.UUID, 0, len(preview.Charges))
	for _, c := range preview.Charges {
		if c.Included {
			originIDs = append(originIDs, c.ID)
		}
	}
	if len(originIDs) == 0 {
		return nil, &dom.ValidationError{Msg: "não há cobranças em aberto para renegociar neste parcelamento"}
	}

	// Modelo da nova série: herda categoria, membro e fornecedor de uma
	// cobrança em aberto, para o lançamento novo nascer classificado.
	template, err := s.entries.GetByID(ctx, in.WorkspaceID, originIDs[0])
	if err != nil {
		return nil, err
	}

	date := time.Now().UTC()
	if in.Date != nil {
		date = in.Date.UTC()
	}
	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = preview.Description
	}

	reneg := &dom.Renegotiation{
		ID:                 uuid.New(),
		WorkspaceID:        in.WorkspaceID,
		Date:               date,
		Description:        description,
		SettledAmountCents: preview.OpenTotalCents,
		NewAmountCents:     in.InstallmentCents * int64(in.InstallmentCount),
		OriginCount:        len(originIDs),
		NewCount:           in.InstallmentCount,
		Notes:              in.Notes,
		CreatedAt:          date,
		UpdatedAt:          date,
	}
	if err := reneg.Validate(); err != nil {
		return nil, err
	}

	base := dom.FinancialEntry{
		WorkspaceID:    in.WorkspaceID,
		Kind:           template.Kind,
		Status:         dom.StatusPrevista,
		AmountCents:    in.InstallmentCents,
		DueDate:        in.FirstDueDate.UTC(),
		FamilyMemberID: template.FamilyMemberID,
		SourceID:       template.SourceID,
		Type:           template.Type,
		Description:    description,
		SupplierID:     template.SupplierID,
	}
	occurrences := dom.GenerateInstallments(base, in.InstallmentCount)

	now := time.Now().UTC()
	batch := make([]*dom.FinancialEntry, len(occurrences))
	for i := range occurrences {
		occurrences[i].ID = uuid.New()
		occurrences[i].RenegotiationID = &reneg.ID
		occurrences[i].CreatedAt = now
		occurrences[i].UpdatedAt = now
		if err := occurrences[i].Validate(); err != nil {
			return nil, err
		}
		batch[i] = &occurrences[i]
	}

	if err := s.renegs.Apply(ctx, reneg, originIDs, batch); err != nil {
		return nil, err
	}

	created := make([]dom.FinancialEntry, len(batch))
	for i := range batch {
		created[i] = *batch[i]
	}
	return &RenegotiateResult{Renegotiation: reneg, Created: created}, nil
}

// Get devolve o evento com os lançamentos dos dois lados — a trilha que liga
// as cobranças encerradas às parcelas novas.
type RenegotiationDetail struct {
	Renegotiation *dom.Renegotiation
	Origins       []dom.FinancialEntry
	Created       []dom.FinancialEntry
}

func (s *RenegotiationService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*RenegotiationDetail, error) {
	reneg, err := s.renegs.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	origins, created, err := s.renegs.ListEntries(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	return &RenegotiationDetail{Renegotiation: reneg, Origins: origins, Created: created}, nil
}

type ListRenegotiationsResult struct {
	Items []dom.Renegotiation
	Total int64
}

func (s *RenegotiationService) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListRenegotiationsResult, error) {
	items, total, err := s.renegs.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListRenegotiationsResult{Items: items, Total: total}, nil
}

// chargeStatusFor classifica uma cobrança em aberto pelo vencimento.
func chargeStatusFor(due, today time.Time) ChargeStatus {
	d := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
	if d.Before(today) {
		return ChargeOverdue
	}
	return ChargeUpcoming
}
