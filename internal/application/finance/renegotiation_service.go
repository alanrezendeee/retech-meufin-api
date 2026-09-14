package finance

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/retechfin/retechfin-api/internal/appctx"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// RenegotiationService apura o saldo em aberto de uma dívida parcelada e
// aplica o desfecho — repactuação (novação), quitação antecipada ou troca do
// bem financiado — encerrando as cobranças em aberto e vinculando os dois
// lados ao mesmo evento.
type RenegotiationService struct {
	entries    dom.FinancialEntryRepository
	renegs     dom.RenegotiationRepository
	events     dom.EntryEventRepository
	categories dom.ExpenseCategoryRepository
}

// NewRenegotiationService monta o serviço. Trilha de eventos e catálogo de
// categorias são opcionais (testes e chamadas que não precisam deles).
func NewRenegotiationService(entries dom.FinancialEntryRepository, renegs dom.RenegotiationRepository, opts ...any) *RenegotiationService {
	s := &RenegotiationService{entries: entries, renegs: renegs}
	for _, o := range opts {
		switch v := o.(type) {
		case dom.EntryEventRepository:
			s.events = v
		case dom.ExpenseCategoryRepository:
			s.categories = v
		}
	}
	return s
}

// logEvent grava um evento da trilha do lançamento. Falha de trilha não
// derruba a operação principal (o evento financeiro já foi persistido).
func (s *RenegotiationService) logEvent(ctx context.Context, ev dom.EntryEvent) {
	if s.events == nil {
		return
	}
	ev.ID = uuid.New()
	ev.ActorUserID = appctx.ActorFromContext(ctx)
	ev.CreatedAt = time.Now().UTC()
	if err := s.events.Create(ctx, &ev); err != nil {
		slog.Error("trilha de eventos: falha ao gravar",
			slog.String("entry_id", ev.EntryID.String()),
			slog.String("event", string(ev.Event)),
			slog.String("error", err.Error()),
		)
	}
}

// logApplied registra na trilha o que o evento fez: cancelamento das origens
// e confirmação dos lançamentos que já nascem realizados (quitação, venda,
// entrada). Parcelas novas previstas não geram evento — nascer não é
// transição.
func (s *RenegotiationService) logApplied(ctx context.Context, ws uuid.UUID, originIDs []uuid.UUID, reason string, created []*dom.FinancialEntry) {
	if s.events == nil {
		return
	}
	r := reason
	for _, id := range originIDs {
		s.logEvent(ctx, dom.EntryEvent{
			WorkspaceID:  ws,
			EntryID:      id,
			Event:        dom.EventCancelled,
			FromStatus:   statusPtr(dom.StatusPrevista),
			ToStatus:     statusPtr(dom.StatusCancelada),
			CancelReason: &r,
		})
	}
	for _, e := range created {
		if e.Status != dom.StatusRealizada {
			continue
		}
		s.logEvent(ctx, dom.EntryEvent{
			WorkspaceID:     ws,
			EntryID:         e.ID,
			Event:           dom.EventSettled,
			FromStatus:      statusPtr(dom.StatusPrevista),
			ToStatus:        statusPtr(dom.StatusRealizada),
			PaidAt:          e.PaidAt,
			PaidAmountCents: e.PaidAmountCents,
		})
	}
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
// Included é o que distingue o que efetivamente entra no desfecho.
type OpenCharge struct {
	ID          uuid.UUID
	Kind        OpenChargeKind
	Status      ChargeStatus
	Description string
	AmountCents int64
	// PaidAmountCents: quanto foi pago (em quitadas e parcialmente pagas).
	PaidAmountCents *int64
	DueDate         time.Time
	// Included indica se a cobrança compõe o saldo apurado.
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
	// Cobranças que entram no desfecho.
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
	// Bem vinculado ao contrato, quando houver.
	AssetType *dom.AssetType
	AssetID   *uuid.UUID
	// Category: categoria de despesa das parcelas (herdada pelos lançamentos
	// que o desfecho cria).
	Category *string
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
	out.AssetType = series[0].AssetType
	out.AssetID = series[0].AssetID
	out.Category = series[0].Type

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
			// residual próprio, e é ele que entra no desfecho.
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

// includedIDs devolve as cobranças que efetivamente entram no desfecho.
func (p *RenegotiationPreview) includedIDs() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(p.Charges))
	for _, c := range p.Charges {
		if c.Included {
			out = append(out, c.ID)
		}
	}
	return out
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
	originIDs := preview.includedIDs()
	if len(originIDs) == 0 {
		return nil, &dom.ValidationError{Msg: "não há cobranças em aberto para renegociar neste parcelamento"}
	}

	// Modelo da nova série: herda categoria, membro, fornecedor e bem de uma
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
		Kind:               dom.KindRenegotiation,
		Date:               date,
		Description:        description,
		SettledAmountCents: preview.OpenTotalCents,
		NewAmountCents:     in.InstallmentCents * int64(in.InstallmentCount),
		OriginCount:        len(originIDs),
		NewCount:           in.InstallmentCount,
		Notes:              in.Notes,
		AssetType:          template.AssetType,
		AssetID:            template.AssetID,
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
		AssetType:      template.AssetType,
		AssetID:        template.AssetID,
	}
	occurrences := dom.GenerateInstallments(base, in.InstallmentCount)
	originGroup := in.GroupID
	reneg.OriginGroupID = &originGroup
	if len(occurrences) > 0 && occurrences[0].RecurrenceGroupID != nil {
		newGroup := *occurrences[0].RecurrenceGroupID
		reneg.NewGroupID = &newGroup
	}

	batch, err := stampCreated(occurrences, reneg.ID)
	if err != nil {
		return nil, err
	}

	if err := s.renegs.Apply(ctx, dom.ApplyInput{
		Event:        reneg,
		OriginIDs:    originIDs,
		CancelReason: dom.CancelReasonRenegotiation,
		NewEntries:   batch,
	}); err != nil {
		return nil, err
	}
	s.logApplied(ctx, in.WorkspaceID, originIDs, dom.CancelReasonRenegotiation, batch)

	return &RenegotiateResult{Renegotiation: reneg, Created: deref(batch)}, nil
}

// stampCreated dá identidade, vínculo ao evento e timestamps aos lançamentos
// que o evento cria, validando cada um.
func stampCreated(entries []dom.FinancialEntry, eventID uuid.UUID) ([]*dom.FinancialEntry, error) {
	now := time.Now().UTC()
	out := make([]*dom.FinancialEntry, len(entries))
	for i := range entries {
		if entries[i].ID == uuid.Nil {
			entries[i].ID = uuid.New()
		}
		entries[i].RenegotiationID = &eventID
		entries[i].CreatedAt = now
		entries[i].UpdatedAt = now
		if err := entries[i].Validate(); err != nil {
			return nil, err
		}
		out[i] = &entries[i]
	}
	return out, nil
}

func deref(batch []*dom.FinancialEntry) []dom.FinancialEntry {
	out := make([]dom.FinancialEntry, len(batch))
	for i := range batch {
		out[i] = *batch[i]
	}
	return out
}

// ---------------------------------------------------------------------------
// Quitação antecipada
// ---------------------------------------------------------------------------

// PayoffInput descreve a quitação de um contrato parcelado.
type PayoffInput struct {
	WorkspaceID uuid.UUID
	GroupID     uuid.UUID
	// Date: data da quitação (default: hoje). É a data de pagamento do
	// lançamento de quitação.
	Date *time.Time
	// PayoffCents: quanto foi efetivamente pago. Menor que o saldo apurado
	// vira desconto (quitação antecipada); maior vira encargo.
	PayoffCents int64
	// Payer: quem pagou. Terceiro (concessionária numa troca) liquida por
	// compensação, sem caixa; próprio exige forma de pagamento.
	Payer            dom.Payer
	PaymentMethod    *dom.PaymentMethod
	PaymentAccountID *uuid.UUID
	// Description: rótulo do evento e do lançamento (default: "Quitação — <dívida>").
	Description string
	Notes       *string
	// AssetType/AssetID: bem quitado; quando ausentes, herdados do contrato.
	AssetType *dom.AssetType
	AssetID   *uuid.UUID
}

type PayoffResult struct {
	Event       *dom.Renegotiation
	PayoffEntry *dom.FinancialEntry
}

// Payoff encerra as cobranças em aberto do contrato e registra um único
// lançamento realizado com o valor pago. O saldo apurado fica como valor do
// lançamento e a diferença como desconto (motivo "quitação antecipada"), de
// modo que a economia aparece nos relatórios de desconto sem inventar
// lançamento negativo.
func (s *RenegotiationService) Payoff(ctx context.Context, in PayoffInput) (*PayoffResult, error) {
	if in.PayoffCents <= 0 {
		return nil, &dom.ValidationError{Msg: "informe o valor pago na quitação"}
	}
	preview, err := s.previewGroup(ctx, in.WorkspaceID, in.GroupID)
	if err != nil {
		return nil, err
	}
	originIDs := preview.includedIDs()
	if len(originIDs) == 0 {
		return nil, &dom.ValidationError{Msg: "não há cobranças em aberto para quitar neste parcelamento"}
	}
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
		description = "Quitação — " + preview.Description
	}
	assetType, assetID := in.AssetType, in.AssetID
	if assetID == nil {
		assetType, assetID = template.AssetType, template.AssetID
	}
	payer := in.Payer
	if payer == "" {
		payer = dom.PayerSelf
	}

	payoff := in.PayoffCents
	reneg := &dom.Renegotiation{
		ID:                 uuid.New(),
		WorkspaceID:        in.WorkspaceID,
		Kind:               dom.KindPayoff,
		Date:               date,
		Description:        description,
		SettledAmountCents: preview.OpenTotalCents,
		OriginCount:        len(originIDs),
		OriginGroupID:      &in.GroupID,
		Notes:              in.Notes,
		PayoffCents:        &payoff,
		Payer:              &payer,
		AssetType:          assetType,
		AssetID:            assetID,
		CreatedAt:          date,
		UpdatedAt:          date,
	}
	if err := reneg.Validate(); err != nil {
		return nil, err
	}

	entry, err := buildPayoffEntry(payoffSpec{
		template:    template,
		description: description,
		date:        date,
		settled:     preview.OpenTotalCents,
		payoff:      payoff,
		payer:       payer,
		method:      in.PaymentMethod,
		accountID:   in.PaymentAccountID,
		assetType:   assetType,
		assetID:     assetID,
	})
	if err != nil {
		return nil, err
	}
	batch, err := stampCreated([]dom.FinancialEntry{*entry}, reneg.ID)
	if err != nil {
		return nil, err
	}
	reneg.PayoffEntryID = &batch[0].ID

	if err := s.renegs.Apply(ctx, dom.ApplyInput{
		Event:        reneg,
		OriginIDs:    originIDs,
		CancelReason: dom.CancelReasonPayoff,
		NewEntries:   batch,
	}); err != nil {
		return nil, err
	}
	s.logApplied(ctx, in.WorkspaceID, originIDs, dom.CancelReasonPayoff, batch)

	return &PayoffResult{Event: reneg, PayoffEntry: batch[0]}, nil
}

type payoffSpec struct {
	template    *dom.FinancialEntry
	description string
	date        time.Time
	settled     int64
	payoff      int64
	payer       dom.Payer
	method      *dom.PaymentMethod
	accountID   *uuid.UUID
	assetType   *dom.AssetType
	assetID     *uuid.UUID
}

// buildPayoffEntry monta o lançamento realizado da quitação. Valor = saldo
// apurado; pago = quitação; desconto = a diferença quando a quitação é menor.
// Pago maior que o valor (multa/juros) fica registrado em paid_amount_cents,
// como qualquer outra liquidação com acréscimo.
func buildPayoffEntry(sp payoffSpec) (*dom.FinancialEntry, error) {
	paid := sp.payoff
	paidAt := sp.date
	e := &dom.FinancialEntry{
		WorkspaceID:      sp.template.WorkspaceID,
		Kind:             sp.template.Kind,
		Status:           dom.StatusRealizada,
		AmountCents:      sp.settled,
		DueDate:          sp.date,
		FamilyMemberID:   sp.template.FamilyMemberID,
		SourceID:         sp.template.SourceID,
		Type:             sp.template.Type,
		Description:      sp.description,
		Recurrence:       dom.RecurrenceNone,
		SupplierID:       sp.template.SupplierID,
		PaidAt:           &paidAt,
		PaidAmountCents:  &paid,
		PaymentAccountID: sp.accountID,
		AssetType:        sp.assetType,
		AssetID:          sp.assetID,
	}
	if sp.settled > sp.payoff {
		discount := sp.settled - sp.payoff
		reason := "quitacao_antecipada"
		e.DiscountCents = &discount
		e.DiscountReason = &reason
	}
	switch sp.payer {
	case dom.PayerThirdParty:
		m := dom.PaymentCompensacao
		e.PaymentMethod = &m
		e.PaymentAccountID = nil
	case dom.PayerSelf:
		if sp.method == nil || !dom.ValidPaymentMethod(*sp.method) {
			return nil, &dom.ValidationError{Msg: "informe a forma de pagamento da quitação"}
		}
		e.PaymentMethod = sp.method
	default:
		return nil, &dom.ValidationError{Msg: "payer inválido"}
	}
	return e, nil
}

// ---------------------------------------------------------------------------
// Troca de bem financiado
// ---------------------------------------------------------------------------

// NewContractSpec descreve o financiamento do bem novo.
type NewContractSpec struct {
	InstallmentCount int
	InstallmentCents int64
	FirstDueDate     time.Time
	Description      string
	// Category: categoria de despesa das parcelas (default: a do contrato
	// antigo; senão "financiamentos" se existir no workspace).
	Category       *string
	SupplierID     *uuid.UUID
	FamilyMemberID *uuid.UUID
}

// AssetSwapInput descreve a troca de um bem financiado por outro.
type AssetSwapInput struct {
	WorkspaceID uuid.UUID
	Date        *time.Time
	AssetType   dom.AssetType
	OldAssetID  uuid.UUID
	NewAssetID  uuid.UUID
	// OldAssetLabel/NewAssetLabel: nomes dos bens para as descrições dos
	// lançamentos (o serviço financeiro não conhece o cadastro de veículos).
	OldAssetLabel string
	NewAssetLabel string
	// OldGroupID: contrato do bem antigo; nil quando já estava quitado.
	OldGroupID  *uuid.UUID
	PayoffCents int64
	// Payer da quitação (default: terceiro — a concessionária quita e abate
	// do valor do usado).
	Payer                  dom.Payer
	PayoffPaymentMethod    *dom.PaymentMethod
	PayoffPaymentAccountID *uuid.UUID
	// TradeInCents: valor de avaliação do bem usado.
	TradeInCents int64
	// CashDownpaymentCents: entrada em dinheiro além do usado.
	CashDownpaymentCents int64
	CashPaymentMethod    *dom.PaymentMethod
	CashPaymentAccountID *uuid.UUID
	// NewAssetPriceCents: preço do bem novo (informacional; preenche a
	// aquisição no cadastro quando vazia).
	NewAssetPriceCents *int64
	// NewContract: financiamento do bem novo; nil = compra à vista.
	NewContract *NewContractSpec
	Description string
	Notes       *string
}

type AssetSwapResult struct {
	Event            *dom.Renegotiation
	PayoffEntry      *dom.FinancialEntry
	TradeInEntry     *dom.FinancialEntry
	DownpaymentEntry *dom.FinancialEntry
	Created          []dom.FinancialEntry
}

// AssetSwap aplica a troca num único evento e numa única transação: quita o
// contrato antigo (quando houver), registra a venda do bem usado e a entrada
// em dinheiro, cria o financiamento novo e atualiza o cadastro dos bens.
//
// O caixa só é tocado pelo que de fato saiu do bolso: a entrada em dinheiro
// (e a quitação, se o próprio usuário pagou). Venda do usado e quitação
// pela concessionária são compensadas entre si — existem para a dívida e o
// patrimônio fecharem, não para o fluxo de caixa.
func (s *RenegotiationService) AssetSwap(ctx context.Context, in AssetSwapInput) (*AssetSwapResult, error) {
	if !dom.ValidAssetType(in.AssetType) {
		return nil, &dom.ValidationError{Msg: "asset_type inválido"}
	}
	if in.OldAssetID == uuid.Nil || in.NewAssetID == uuid.Nil {
		return nil, &dom.ValidationError{Msg: "informe o bem antigo e o bem novo"}
	}
	if in.TradeInCents < 0 || in.CashDownpaymentCents < 0 {
		return nil, &dom.ValidationError{Msg: "valores da troca não podem ser negativos"}
	}
	if in.NewContract != nil {
		nc := in.NewContract
		if nc.InstallmentCount < 1 {
			return nil, &dom.ValidationError{Msg: "informe ao menos uma parcela para o financiamento novo"}
		}
		if nc.InstallmentCents <= 0 {
			return nil, &dom.ValidationError{Msg: "o valor da parcela do financiamento novo deve ser maior que zero"}
		}
		if nc.FirstDueDate.IsZero() {
			return nil, &dom.ValidationError{Msg: "informe o vencimento da primeira parcela do financiamento novo"}
		}
	}

	date := time.Now().UTC()
	if in.Date != nil {
		date = in.Date.UTC()
	}
	oldLabel := strings.TrimSpace(in.OldAssetLabel)
	if oldLabel == "" {
		oldLabel = "bem antigo"
	}
	newLabel := strings.TrimSpace(in.NewAssetLabel)
	if newLabel == "" {
		newLabel = "bem novo"
	}
	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = "Troca — " + oldLabel + " → " + newLabel
	}
	payer := in.Payer
	if payer == "" {
		payer = dom.PayerThirdParty
	}
	assetType := in.AssetType
	oldAsset, newAsset := in.OldAssetID, in.NewAssetID
	tradeIn := in.TradeInCents
	cash := in.CashDownpaymentCents

	reneg := &dom.Renegotiation{
		ID:                   uuid.New(),
		WorkspaceID:          in.WorkspaceID,
		Kind:                 dom.KindAssetSwap,
		Date:                 date,
		Description:          description,
		Notes:                in.Notes,
		AssetType:            &assetType,
		AssetID:              &oldAsset,
		NewAssetID:           &newAsset,
		TradeInCents:         &tradeIn,
		CashDownpaymentCents: &cash,
		CreatedAt:            date,
		UpdatedAt:            date,
	}

	var (
		originIDs []uuid.UUID
		template  *dom.FinancialEntry
		created   []dom.FinancialEntry
		payoffIdx = -1
		tradeIdx  = -1
		cashIdx   = -1
	)

	// 1. Contrato antigo: apuração + lançamento de quitação.
	if in.OldGroupID != nil {
		preview, err := s.previewGroup(ctx, in.WorkspaceID, *in.OldGroupID)
		if err != nil {
			return nil, err
		}
		originIDs = preview.includedIDs()
		if len(originIDs) == 0 {
			return nil, &dom.ValidationError{Msg: "o contrato antigo não tem cobranças em aberto — informe a troca sem contrato"}
		}
		template, err = s.entries.GetByID(ctx, in.WorkspaceID, originIDs[0])
		if err != nil {
			return nil, err
		}
		if in.PayoffCents <= 0 {
			return nil, &dom.ValidationError{Msg: "informe quanto foi pago para quitar o contrato antigo"}
		}
		payoff := in.PayoffCents
		reneg.SettledAmountCents = preview.OpenTotalCents
		reneg.OriginCount = len(originIDs)
		reneg.OriginGroupID = in.OldGroupID
		reneg.PayoffCents = &payoff
		reneg.Payer = &payer

		entry, err := buildPayoffEntry(payoffSpec{
			template:    template,
			description: "Quitação — " + preview.Description,
			date:        date,
			settled:     preview.OpenTotalCents,
			payoff:      payoff,
			payer:       payer,
			method:      in.PayoffPaymentMethod,
			accountID:   in.PayoffPaymentAccountID,
			assetType:   &assetType,
			assetID:     &oldAsset,
		})
		if err != nil {
			return nil, err
		}
		payoffIdx = len(created)
		created = append(created, *entry)
	}

	// Categoria das despesas novas (entrada e parcelas).
	var requested *string
	if in.NewContract != nil {
		requested = in.NewContract.Category
	}
	category, err := s.resolveCategory(ctx, in.WorkspaceID, requested, template)
	if err != nil {
		return nil, err
	}

	// 2. Venda do usado: receita realizada por compensação.
	if tradeIn > 0 {
		incomeType := dom.IncomeTypeAssetSale
		paid := tradeIn
		paidAt := date
		m := dom.PaymentCompensacao
		tradeIdx = len(created)
		created = append(created, dom.FinancialEntry{
			WorkspaceID:     in.WorkspaceID,
			Kind:            dom.KindCredit,
			Status:          dom.StatusRealizada,
			AmountCents:     tradeIn,
			DueDate:         date,
			Type:            &incomeType,
			Description:     "Venda (troca) — " + oldLabel,
			Recurrence:      dom.RecurrenceNone,
			PaidAt:          &paidAt,
			PaidAmountCents: &paid,
			PaymentMethod:   &m,
			AssetType:       &assetType,
			AssetID:         &oldAsset,
		})
	}

	// 3. Entrada em dinheiro: despesa realizada com caixa de verdade.
	if cash > 0 {
		if in.CashPaymentMethod == nil || !dom.ValidPaymentMethod(*in.CashPaymentMethod) || *in.CashPaymentMethod == dom.PaymentCompensacao {
			return nil, &dom.ValidationError{Msg: "informe a forma de pagamento da entrada em dinheiro"}
		}
		paid := cash
		paidAt := date
		var familyMember, supplier *uuid.UUID
		if in.NewContract != nil {
			familyMember, supplier = in.NewContract.FamilyMemberID, in.NewContract.SupplierID
		}
		cashIdx = len(created)
		created = append(created, dom.FinancialEntry{
			WorkspaceID:      in.WorkspaceID,
			Kind:             dom.KindDebit,
			Status:           dom.StatusRealizada,
			AmountCents:      cash,
			DueDate:          date,
			Type:             category,
			Description:      "Entrada — " + newLabel,
			Recurrence:       dom.RecurrenceNone,
			FamilyMemberID:   familyMember,
			SupplierID:       supplier,
			PaidAt:           &paidAt,
			PaidAmountCents:  &paid,
			PaymentMethod:    in.CashPaymentMethod,
			PaymentAccountID: in.CashPaymentAccountID,
			AssetType:        &assetType,
			AssetID:          &newAsset,
		})
	}

	// 4. Financiamento novo.
	if nc := in.NewContract; nc != nil {
		desc := strings.TrimSpace(nc.Description)
		if desc == "" {
			desc = "Financiamento — " + newLabel
		}
		familyMember, supplier := nc.FamilyMemberID, nc.SupplierID
		if template != nil {
			if familyMember == nil {
				familyMember = template.FamilyMemberID
			}
			if supplier == nil {
				supplier = template.SupplierID
			}
		}
		base := dom.FinancialEntry{
			WorkspaceID:    in.WorkspaceID,
			Kind:           dom.KindDebit,
			Status:         dom.StatusPrevista,
			AmountCents:    nc.InstallmentCents,
			DueDate:        nc.FirstDueDate.UTC(),
			FamilyMemberID: familyMember,
			Type:           category,
			Description:    desc,
			SupplierID:     supplier,
			AssetType:      &assetType,
			AssetID:        &newAsset,
		}
		installments := dom.GenerateInstallments(base, nc.InstallmentCount)
		if len(installments) > 0 && installments[0].RecurrenceGroupID != nil {
			g := *installments[0].RecurrenceGroupID
			reneg.NewGroupID = &g
		}
		reneg.NewCount = nc.InstallmentCount
		reneg.NewAmountCents = nc.InstallmentCents * int64(nc.InstallmentCount)
		created = append(created, installments...)
	}

	if err := reneg.Validate(); err != nil {
		return nil, err
	}
	batch, err := stampCreated(created, reneg.ID)
	if err != nil {
		return nil, err
	}
	if payoffIdx >= 0 {
		reneg.PayoffEntryID = &batch[payoffIdx].ID
	}
	if tradeIdx >= 0 {
		reneg.TradeInEntryID = &batch[tradeIdx].ID
	}
	if cashIdx >= 0 {
		reneg.DownpaymentEntryID = &batch[cashIdx].ID
	}

	apply := dom.ApplyInput{
		Event:        reneg,
		OriginIDs:    originIDs,
		CancelReason: dom.CancelReasonPayoff,
		NewEntries:   batch,
		Sale: &dom.AssetSale{
			AssetType:  assetType,
			AssetID:    oldAsset,
			SoldAt:     date,
			PriceCents: tradeIn,
		},
		Acquisition: &dom.AssetAcquisition{
			AssetType:  assetType,
			AssetID:    newAsset,
			AcquiredAt: date,
			PriceCents: in.NewAssetPriceCents,
		},
	}
	if err := s.renegs.Apply(ctx, apply); err != nil {
		return nil, err
	}
	s.logApplied(ctx, in.WorkspaceID, originIDs, dom.CancelReasonPayoff, batch)

	out := &AssetSwapResult{Event: reneg, Created: deref(batch)}
	if payoffIdx >= 0 {
		out.PayoffEntry = batch[payoffIdx]
	}
	if tradeIdx >= 0 {
		out.TradeInEntry = batch[tradeIdx]
	}
	if cashIdx >= 0 {
		out.DownpaymentEntry = batch[cashIdx]
	}
	return out, nil
}

// resolveCategory escolhe a categoria de despesa dos lançamentos novos:
// a pedida (se existir no workspace), senão a do contrato antigo, senão
// "financiamentos" quando cadastrada, senão o fallback do catálogo.
func (s *RenegotiationService) resolveCategory(ctx context.Context, ws uuid.UUID, requested *string, template *dom.FinancialEntry) (*string, error) {
	exists := func(slug string) (bool, error) {
		if s.categories == nil {
			return true, nil
		}
		return s.categories.ExistsBySlug(ctx, ws, slug)
	}
	if requested != nil && strings.TrimSpace(*requested) != "" {
		slug := strings.TrimSpace(*requested)
		if slug == dom.CartaoCategorySlug {
			return nil, &dom.ValidationError{Msg: "'cartao' é reservado às faturas do sistema"}
		}
		ok, err := exists(slug)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &dom.ValidationError{Msg: "categoria de despesa não cadastrada no workspace"}
		}
		return &slug, nil
	}
	if template != nil && template.Type != nil && *template.Type != "" {
		t := *template.Type
		return &t, nil
	}
	for _, slug := range []string{"financiamentos", dom.FallbackCategorySlug} {
		ok, err := exists(slug)
		if err != nil {
			return nil, err
		}
		if ok {
			c := slug
			return &c, nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Consulta
// ---------------------------------------------------------------------------

// Get devolve o evento com os lançamentos dos dois lados — a trilha que liga
// as cobranças encerradas ao que nasceu — mais o que já tinha sido pago do
// contrato antigo antes do evento e os vizinhos na cadeia.
type RenegotiationDetail struct {
	Renegotiation *dom.Renegotiation
	Origins       []dom.FinancialEntry
	Created       []dom.FinancialEntry
	// PaidBefore: parcelas (e residuais) do grupo de origem quitadas antes
	// do evento. Não entram no saldo apurado, mas são parte da história da
	// dívida — "quanto eu já tinha pago quando renegociei/quitei".
	PaidBefore      []dom.FinancialEntry
	PaidBeforeCents int64
	// Previous/Next: evento que criou o grupo de origem e evento que
	// encerrou o grupo criado, quando existem.
	PreviousID  *uuid.UUID
	NextID      *uuid.UUID
	RootGroupID *uuid.UUID
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
	out := &RenegotiationDetail{Renegotiation: reneg, Origins: origins, Created: created}

	if reneg.OriginGroupID != nil {
		series, err := s.entries.ListGroup(ctx, workspaceID, *reneg.OriginGroupID)
		if err != nil {
			return nil, err
		}
		ids := make([]uuid.UUID, 0, len(series))
		for i := range series {
			ids = append(ids, series[i].ID)
			if series[i].Status == dom.StatusRealizada {
				out.PaidBefore = append(out.PaidBefore, series[i])
				out.PaidBeforeCents += paidOf(&series[i])
			}
		}
		residuals, err := s.entries.ListResidualsOf(ctx, workspaceID, ids)
		if err != nil {
			return nil, err
		}
		for i := range residuals {
			if residuals[i].Status == dom.StatusRealizada {
				out.PaidBefore = append(out.PaidBefore, residuals[i])
				out.PaidBeforeCents += paidOf(&residuals[i])
			}
		}
		sort.Slice(out.PaidBefore, func(i, j int) bool { return out.PaidBefore[i].DueDate.Before(out.PaidBefore[j].DueDate) })

		prev, err := s.renegs.FindByNewGroup(ctx, workspaceID, *reneg.OriginGroupID)
		if err != nil {
			return nil, err
		}
		if prev != nil {
			out.PreviousID = &prev.ID
		}
		root, err := s.rootGroup(ctx, workspaceID, *reneg.OriginGroupID)
		if err != nil {
			return nil, err
		}
		out.RootGroupID = &root
	}
	if reneg.NewGroupID != nil {
		next, err := s.renegs.FindByOriginGroup(ctx, workspaceID, *reneg.NewGroupID)
		if err != nil {
			return nil, err
		}
		if next != nil {
			out.NextID = &next.ID
		}
	}
	return out, nil
}

// paidOf devolve o valor efetivamente pago de um lançamento realizado.
func paidOf(e *dom.FinancialEntry) int64 {
	if e.PaidAmountCents != nil {
		return *e.PaidAmountCents
	}
	return e.AmountCents
}

// rootGroup anda a cadeia de renegociações para trás até o parcelamento
// original. Uma troca cria OUTRA dívida (de outro bem): a caminhada para em
// eventos que encerram a dívida — o contrato nascido de uma troca é raiz da
// própria linhagem.
func (s *RenegotiationService) rootGroup(ctx context.Context, workspaceID, groupID uuid.UUID) (uuid.UUID, error) {
	cur := groupID
	for hops := 0; hops < 100; hops++ {
		prev, err := s.renegs.FindByNewGroup(ctx, workspaceID, cur)
		if err != nil {
			return uuid.Nil, err
		}
		if prev == nil || prev.ClosesDebt() || prev.OriginGroupID == nil || *prev.OriginGroupID == cur {
			return cur, nil
		}
		cur = *prev.OriginGroupID
	}
	return cur, nil
}

// DebtStage é uma etapa da dívida: o parcelamento original ou a série criada
// por um acordo. Cada etapa fecha as próprias contas — o que foi pago, o que
// foi levado ao acordo seguinte (ou quitado) e o que ainda está em aberto.
type DebtStage struct {
	Index       int
	GroupID     uuid.UUID
	Description string
	// Renegotiation é o acordo que criou esta etapa (nil na original).
	Renegotiation *dom.Renegotiation
	// SettledBy é o evento que encerrou esta etapa (nil na etapa vigente).
	// Pode ser renegociação (a linhagem continua) ou quitação/troca (a
	// linhagem termina aqui).
	SettledBy *dom.Renegotiation
	// PayoffEntry: o lançamento de quitação, quando SettledBy é quitação
	// ou troca. Conta como pago desta etapa.
	PayoffEntry *dom.FinancialEntry
	// TotalCents: soma das parcelas da série (o "valor do acordo" da etapa).
	InstallmentTotal int
	TotalCents       int64
	FirstDueDate     *time.Time
	LastDueDate      *time.Time
	// Pago: parcelas e residuais realizados desta etapa (+ quitação).
	PaidCount int
	PaidCents int64
	// Carregado: cobranças canceladas pelo evento que encerrou a etapa
	// (saldo que foi para o acordo seguinte ou para a quitação).
	CarriedCount int
	CarriedCents int64
	// Cancelado por outro motivo (fora da dívida).
	CancelledCount int
	CancelledCents int64
	// Em aberto: previstas (parcelas + residuais) — só na etapa vigente.
	OpenCount    int
	OpenCents    int64
	OverdueCount int
	OverdueCents int64
	// Bem vinculado às parcelas desta etapa.
	AssetType *dom.AssetType
	AssetID   *uuid.UUID
	// Entries: a série inteira mais os residuais (e a quitação), por vencimento.
	Entries []dom.FinancialEntry
}

// DebtLineage é a história completa de uma dívida através das suas
// renegociações, com o balanço consolidado.
type DebtLineage struct {
	RootGroupID    uuid.UUID
	CurrentGroupID uuid.UUID
	Description    string
	Stages         []DebtStage
	// Balanço.
	OriginalCents     int64 // valor do parcelamento original
	InterestCents     int64 // encargos somados dos eventos
	DiscountCents     int64 // descontos somados dos eventos (inclui a quitação)
	CurrentTotalCents int64 // original + encargos - descontos
	PaidCents         int64 // pago em todas as etapas (inclui a quitação)
	PaidCount         int
	OpenCents         int64 // em aberto na etapa vigente
	OpenCount         int
	OverdueCents      int64
	OverdueCount      int
	RenegotiationCnt  int
	Settled           bool // nada em aberto
	// ClosedBy: evento de quitação/troca que encerrou a etapa vigente.
	ClosedBy *dom.Renegotiation
	// SuccessorGroupID: contrato do bem novo nascido da troca que encerrou
	// esta dívida (é OUTRA linhagem).
	SuccessorGroupID *uuid.UUID
	// OriginEvent: troca que criou o parcelamento raiz desta dívida;
	// PredecessorGroupID é o contrato do bem antigo.
	OriginEvent        *dom.Renegotiation
	PredecessorGroupID *uuid.UUID
	// Bem da etapa vigente.
	AssetType *dom.AssetType
	AssetID   *uuid.UUID
}

// Lineage monta a linhagem a partir de QUALQUER grupo da cadeia: volta até a
// raiz e avança acordo a acordo até a etapa vigente (ou até o evento que
// encerrou a dívida).
func (s *RenegotiationService) Lineage(ctx context.Context, workspaceID, groupID uuid.UUID) (*DebtLineage, error) {
	root, err := s.rootGroup(ctx, workspaceID, groupID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	out := &DebtLineage{RootGroupID: root}
	cur := root
	var createdBy *dom.Renegotiation
	for hops := 0; hops < 100; hops++ {
		stage, err := s.buildStage(ctx, workspaceID, cur, len(out.Stages), createdBy, today)
		if err != nil {
			return nil, err
		}
		next, err := s.renegs.FindByOriginGroup(ctx, workspaceID, cur)
		if err != nil {
			return nil, err
		}
		stage.SettledBy = next
		if next != nil && next.ClosesDebt() {
			// Quitação/troca: o lançamento de quitação fecha as contas da
			// etapa — é o pagamento que encerrou a dívida.
			if next.PayoffEntryID != nil {
				pe, err := s.entries.GetByID(ctx, workspaceID, *next.PayoffEntryID)
				if err == nil && pe.Status == dom.StatusRealizada {
					stage.PayoffEntry = pe
					stage.PaidCount++
					stage.PaidCents += paidOf(pe)
					stage.Entries = append(stage.Entries, *pe)
				}
			}
			out.ClosedBy = next
			out.SuccessorGroupID = next.NewGroupID
			out.Stages = append(out.Stages, *stage)
			break
		}
		out.Stages = append(out.Stages, *stage)
		if next == nil || next.NewGroupID == nil || *next.NewGroupID == cur {
			break
		}
		createdBy = next
		cur = *next.NewGroupID
	}
	if len(out.Stages) == 0 {
		return nil, dom.ErrNotFound
	}

	// Origem: esta dívida nasceu de uma troca?
	if origin, err := s.renegs.FindByNewGroup(ctx, workspaceID, root); err != nil {
		return nil, err
	} else if origin != nil && origin.ClosesDebt() {
		out.OriginEvent = origin
		out.PredecessorGroupID = origin.OriginGroupID
	}

	first := out.Stages[0]
	last := out.Stages[len(out.Stages)-1]
	out.CurrentGroupID = last.GroupID
	out.Description = last.Description
	out.OriginalCents = first.TotalCents
	out.AssetType = last.AssetType
	out.AssetID = last.AssetID
	for i := range out.Stages {
		st := &out.Stages[i]
		out.PaidCents += st.PaidCents
		out.PaidCount += st.PaidCount
		out.OpenCents += st.OpenCents
		out.OpenCount += st.OpenCount
		out.OverdueCents += st.OverdueCents
		out.OverdueCount += st.OverdueCount
		if st.Renegotiation != nil {
			out.RenegotiationCnt++
			out.addAdjustment(st.Renegotiation.AdjustmentCents)
		}
	}
	if out.ClosedBy != nil && out.ClosedBy.OriginCount > 0 {
		out.addAdjustment(out.ClosedBy.AdjustmentCents)
	}
	out.CurrentTotalCents = out.OriginalCents + out.InterestCents - out.DiscountCents
	out.Settled = out.OpenCount == 0
	return out, nil
}

func (l *DebtLineage) addAdjustment(cents int64) {
	if cents > 0 {
		l.InterestCents += cents
	} else {
		l.DiscountCents += -cents
	}
}

func (s *RenegotiationService) buildStage(ctx context.Context, workspaceID, groupID uuid.UUID, index int, createdBy *dom.Renegotiation, today time.Time) (*DebtStage, error) {
	series, err := s.entries.ListGroup(ctx, workspaceID, groupID)
	if err != nil {
		return nil, err
	}
	if len(series) == 0 {
		return nil, dom.ErrNotFound
	}
	ids := make([]uuid.UUID, 0, len(series))
	for i := range series {
		ids = append(ids, series[i].ID)
	}
	residuals, err := s.entries.ListResidualsOf(ctx, workspaceID, ids)
	if err != nil {
		return nil, err
	}

	st := &DebtStage{Index: index, GroupID: groupID, Renegotiation: createdBy}
	// Descrição e total de parcelas: da parcela mais recente (é a exibida
	// como nome do parcelamento e reflete renomeações).
	latest := series[len(series)-1]
	st.Description = latest.Description
	if latest.InstallmentTotal != nil {
		st.InstallmentTotal = *latest.InstallmentTotal
	}
	st.AssetType = latest.AssetType
	st.AssetID = latest.AssetID
	for i := range series {
		e := &series[i]
		st.TotalCents += e.AmountCents
		if st.FirstDueDate == nil || e.DueDate.Before(*st.FirstDueDate) {
			d := e.DueDate
			st.FirstDueDate = &d
		}
		if st.LastDueDate == nil || e.DueDate.After(*st.LastDueDate) {
			d := e.DueDate
			st.LastDueDate = &d
		}
	}
	all := append(append([]dom.FinancialEntry{}, series...), residuals...)
	sort.SliceStable(all, func(i, j int) bool { return all[i].DueDate.Before(all[j].DueDate) })
	for i := range all {
		e := &all[i]
		switch e.Status {
		case dom.StatusRealizada:
			st.PaidCount++
			st.PaidCents += paidOf(e)
		case dom.StatusPrevista:
			st.OpenCount++
			st.OpenCents += e.AmountCents
			if chargeStatusFor(e.DueDate, today) == ChargeOverdue {
				st.OverdueCount++
				st.OverdueCents += e.AmountCents
			}
		case dom.StatusCancelada:
			if e.SettledByRenegotiationID != nil {
				st.CarriedCount++
				st.CarriedCents += e.AmountCents
			} else {
				st.CancelledCount++
				st.CancelledCents += e.AmountCents
			}
		}
	}
	st.Entries = all
	return st, nil
}

// AssetDebts reúne o que o financeiro sabe sobre um bem: os contratos que o
// financiam (cada um com a própria linhagem) e os eventos em que ele
// aparece (quitação, troca — como antigo ou como novo).
type AssetDebts struct {
	Debts  []DebtLineage
	Events []dom.Renegotiation
}

func (s *RenegotiationService) AssetDebts(ctx context.Context, workspaceID uuid.UUID, assetType dom.AssetType, assetID uuid.UUID) (*AssetDebts, error) {
	if !dom.ValidAssetType(assetType) {
		return nil, &dom.ValidationError{Msg: "asset_type inválido"}
	}
	groups, err := s.entries.ListGroupIDsByAsset(ctx, workspaceID, assetType, assetID)
	if err != nil {
		return nil, err
	}
	out := &AssetDebts{Debts: []DebtLineage{}, Events: []dom.Renegotiation{}}
	seen := map[uuid.UUID]bool{}
	for _, g := range groups {
		l, err := s.Lineage(ctx, workspaceID, g)
		if err != nil {
			return nil, err
		}
		if seen[l.RootGroupID] {
			continue
		}
		seen[l.RootGroupID] = true
		out.Debts = append(out.Debts, *l)
	}
	events, err := s.renegs.ListByAsset(ctx, workspaceID, assetType, assetID)
	if err != nil {
		return nil, err
	}
	out.Events = events
	return out, nil
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
