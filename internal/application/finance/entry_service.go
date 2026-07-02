package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

type FinancialEntryService struct {
	repo       dom.FinancialEntryRepository
	categories dom.ExpenseCategoryRepository
}

func NewFinancialEntryService(repo dom.FinancialEntryRepository, categories dom.ExpenseCategoryRepository) *FinancialEntryService {
	return &FinancialEntryService{repo: repo, categories: categories}
}

// validateExpenseCategory garante que a categoria da despesa existe no
// workspace (cadastro gerenciado) e que 'cartao' não é usado pelo usuário.
func (s *FinancialEntryService) validateExpenseCategory(ctx context.Context, workspaceID uuid.UUID, kind dom.Kind, t *string) error {
	if kind != dom.KindDebit || t == nil || *t == "" {
		return nil
	}
	if *t == dom.CartaoCategorySlug {
		return &dom.ValidationError{Msg: "'cartao' é reservado às faturas do sistema"}
	}
	ok, err := s.categories.ExistsBySlug(ctx, workspaceID, *t)
	if err != nil {
		return err
	}
	if !ok {
		return &dom.ValidationError{Msg: "categoria de despesa não cadastrada no workspace"}
	}
	return nil
}

// normalizeCategory mapeia categoria vinda de fora (LLM) para o cadastro do
// workspace; desconhecida vira o fallback 'outros'.
func (s *FinancialEntryService) normalizeCategory(ctx context.Context, workspaceID uuid.UUID, t *string) *string {
	fallback := dom.FallbackCategorySlug
	if t == nil {
		return &fallback
	}
	norm := strings.ToLower(strings.TrimSpace(*t))
	if norm == "" || norm == dom.CartaoCategorySlug {
		return &fallback
	}
	if ok, err := s.categories.ExistsBySlug(ctx, workspaceID, norm); err == nil && ok {
		return &norm
	}
	return &fallback
}

type CreateEntryInput struct {
	WorkspaceID       uuid.UUID
	Kind              string
	Status            string // opcional, default prevista
	AmountCents       int64
	DueDate           time.Time
	FamilyMemberID    *uuid.UUID
	SourceID          *uuid.UUID
	Type              *string
	Description       string
	Recurrence        string
	Notes             *string
	CardID            *uuid.UUID
	ParentID          *uuid.UUID
	InstallmentsTotal *int
	// ConfirmPastOccurrences: em lançamento retroativo (ex.: financiamento
	// começado ano passado), as ocorrências com vencimento até hoje nascem
	// 'realizada' — evita confirmar dezenas de parcelas uma a uma.
	ConfirmPastOccurrences bool
}

type UpdateEntryInput struct {
	WorkspaceID    uuid.UUID
	ID             uuid.UUID
	Kind           string
	Status         string
	AmountCents    int64
	DueDate        time.Time
	FamilyMemberID *uuid.UUID
	SourceID       *uuid.UUID
	Type           *string
	Description    string
	Recurrence     string
	Notes          *string
}

// Create monta o lançamento base, gera as ocorrências recorrentes e persiste em lote.
func (s *FinancialEntryService) Create(ctx context.Context, in CreateEntryInput) ([]dom.FinancialEntry, error) {
	now := time.Now().UTC()
	status := dom.Status(in.Status)
	if status == "" {
		status = dom.StatusPrevista
	}
	recurrence := dom.Recurrence(in.Recurrence)
	if recurrence == "" {
		recurrence = dom.RecurrenceNone
	}
	base := dom.FinancialEntry{
		ID:             uuid.New(),
		WorkspaceID:    in.WorkspaceID,
		Kind:           dom.Kind(in.Kind),
		Status:         status,
		AmountCents:    in.AmountCents,
		DueDate:        in.DueDate,
		FamilyMemberID: in.FamilyMemberID,
		SourceID:       in.SourceID,
		Type:           in.Type,
		Description:    in.Description,
		Recurrence:     recurrence,
		CardID:         in.CardID,
		ParentID:       in.ParentID,
		Notes:          in.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.validateExpenseCategory(ctx, in.WorkspaceID, base.Kind, base.Type); err != nil {
		return nil, err
	}

	// Três caminhos mutuamente exclusivos: parcelado, recorrente ou único.
	var occurrences []dom.FinancialEntry
	switch {
	case in.InstallmentsTotal != nil && *in.InstallmentsTotal > 1:
		// Parcelado: N lançamentos mensais, recurrence forçada para none.
		base.Recurrence = dom.RecurrenceNone
		base.Status = dom.StatusPrevista
		if err := base.Validate(); err != nil {
			return nil, err
		}
		occurrences = dom.GenerateInstallments(base, *in.InstallmentsTotal)
	default:
		// Recorrente (recurrence != none) ou único.
		if err := base.Validate(); err != nil {
			return nil, err
		}
		occurrences = dom.GenerateOccurrences(base)
	}
	if in.ConfirmPastOccurrences {
		today := time.Now().UTC().Truncate(24 * time.Hour)
		for i := range occurrences {
			if occurrences[i].Status == dom.StatusPrevista && !occurrences[i].DueDate.After(today) {
				occurrences[i].Status = dom.StatusRealizada
				paidAt := occurrences[i].DueDate
				occurrences[i].PaidAt = &paidAt
			}
		}
	}

	batch := make([]*dom.FinancialEntry, len(occurrences))
	for i := range occurrences {
		occ := occurrences[i]
		occ.CreatedAt = now
		occ.UpdatedAt = now
		batch[i] = &occ
	}
	if err := s.repo.CreateBatch(ctx, batch); err != nil {
		return nil, err
	}
	out := make([]dom.FinancialEntry, len(batch))
	for i := range batch {
		out[i] = *batch[i]
	}
	return out, nil
}

// InvoiceItemInput é uma compra/lançamento filho de uma fatura.
type InvoiceItemInput struct {
	Description       string
	AmountCents       int64
	Date              *time.Time
	Category          *string
	InstallmentNumber *int
	InstallmentTotal  *int
}

// CreateInvoiceInput descreve a criação de uma fatura de cartão a partir das
// compras confirmadas (import de fatura via PDF/LLM).
type CreateInvoiceInput struct {
	WorkspaceID uuid.UUID
	CardID      *uuid.UUID
	DueDate     time.Time
	Description string
	Status      string // opcional, default prevista
	AmountCents *int64 // opcional; se nil, soma dos itens
	Items       []InvoiceItemInput
}

// CreateInvoiceWithItems cria a FATURA (pai) e cada compra (filho) numa única
// transação. A fatura é kind=debit, type='cartao'; cada item é kind=debit com
// parent_id apontando para a fatura. O amount da fatura é a soma dos itens
// quando não informado. Retorna (fatura, filhos, error).
func (s *FinancialEntryService) CreateInvoiceWithItems(ctx context.Context, in CreateInvoiceInput) (*dom.FinancialEntry, []dom.FinancialEntry, error) {
	if len(in.Items) == 0 {
		return nil, nil, &dom.ValidationError{Msg: "a fatura precisa de ao menos um item"}
	}
	now := time.Now().UTC()

	status := dom.Status(in.Status)
	if status == "" {
		status = dom.StatusPrevista
	}

	var sum int64
	for _, it := range in.Items {
		sum += it.AmountCents
	}
	amount := sum
	if in.AmountCents != nil {
		amount = *in.AmountCents
	}

	cartaoType := "cartao"
	invoice := dom.FinancialEntry{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		Kind:        dom.KindDebit,
		Status:      status,
		AmountCents: amount,
		DueDate:     in.DueDate,
		Type:        &cartaoType,
		Description: in.Description,
		Recurrence:  dom.RecurrenceNone,
		CardID:      in.CardID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := invoice.Validate(); err != nil {
		return nil, nil, err
	}

	batch := make([]*dom.FinancialEntry, 0, len(in.Items)+1)
	batch = append(batch, &invoice)

	for _, it := range in.Items {
		due := in.DueDate
		if it.Date != nil {
			due = *it.Date
		}
		invoiceID := invoice.ID
		child := dom.FinancialEntry{
			ID:                uuid.New(),
			WorkspaceID:       in.WorkspaceID,
			Kind:              dom.KindDebit,
			Status:            status,
			AmountCents:       it.AmountCents,
			DueDate:           due,
			Type:              s.normalizeCategory(ctx, in.WorkspaceID, it.Category),
			Description:       it.Description,
			Recurrence:        dom.RecurrenceNone,
			CardID:            in.CardID,
			ParentID:          &invoiceID,
			InstallmentNumber: it.InstallmentNumber,
			InstallmentTotal:  it.InstallmentTotal,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := child.Validate(); err != nil {
			return nil, nil, err
		}
		batch = append(batch, &child)
	}

	if err := s.repo.CreateBatch(ctx, batch); err != nil {
		return nil, nil, err
	}

	children := make([]dom.FinancialEntry, 0, len(batch)-1)
	for _, e := range batch[1:] {
		children = append(children, *e)
	}
	return &invoice, children, nil
}

func (s *FinancialEntryService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListEntriesResult struct {
	Items []dom.FinancialEntry
	Total int64
}

func (s *FinancialEntryService) List(ctx context.Context, workspaceID uuid.UUID, filter dom.FinancialEntryFilter, limit, offset int) (*ListEntriesResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListEntriesResult{Items: items, Total: total}, nil
}

func (s *FinancialEntryService) Update(ctx context.Context, in UpdateEntryInput) (*dom.FinancialEntry, error) {
	e, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	e.Kind = dom.Kind(in.Kind)
	if in.Status != "" {
		e.Status = dom.Status(in.Status)
	}
	e.AmountCents = in.AmountCents
	e.DueDate = in.DueDate
	e.FamilyMemberID = in.FamilyMemberID
	e.SourceID = in.SourceID
	e.Type = in.Type
	e.Description = in.Description
	if in.Recurrence != "" {
		e.Recurrence = dom.Recurrence(in.Recurrence)
	}
	e.Notes = in.Notes
	e.UpdatedAt = time.Now().UTC()
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.validateExpenseCategory(ctx, in.WorkspaceID, e.Kind, e.Type); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *FinancialEntryService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

// Confirm marca o lançamento como realizado (liquidação rápida, sem detalhes de pagamento).
func (s *FinancialEntryService) Confirm(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	return s.setStatus(ctx, workspaceID, id, dom.StatusRealizada)
}

// Cancel marca o lançamento como cancelado.
func (s *FinancialEntryService) Cancel(ctx context.Context, workspaceID, id uuid.UUID) (*dom.FinancialEntry, error) {
	return s.setStatus(ctx, workspaceID, id, dom.StatusCancelada)
}

func (s *FinancialEntryService) setStatus(ctx context.Context, workspaceID, id uuid.UUID, status dom.Status) (*dom.FinancialEntry, error) {
	e, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return nil, err
	}
	e.Status = status
	e.UpdatedAt = time.Now().UTC()
	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	// Fatura pai/filho: o status propaga para os filhos (um pagamento real = uma ação).
	if err := s.repo.CascadeStatusToChildren(ctx, workspaceID, e.ID, status, e.PaidAt); err != nil {
		return nil, err
	}
	return e, nil
}

// ExtendRecurrences completa o horizonte rolling (12 meses) de todos os
// grupos de recorrência. Roda no boot e diariamente (ticker no main).
// Grupo cuja ocorrência mais recente está cancelada é tratado como encerrado.
func (s *FinancialEntryService) ExtendRecurrences(ctx context.Context) (int, error) {
	frontiers, err := s.repo.ListRecurrenceFrontiers(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	horizon := now.AddDate(0, dom.RollingMonths, 0)
	created := 0
	for i := range frontiers {
		f := frontiers[i]
		if f.Status == dom.StatusCancelada {
			continue // recorrência encerrada pelo usuário
		}
		if !f.DueDate.Before(horizon.AddDate(0, -1, 0)) {
			continue // horizonte ainda completo
		}
		occs := dom.NextOccurrencesAfter(f, f.DueDate, horizon)
		if len(occs) == 0 {
			continue
		}
		batch := make([]*dom.FinancialEntry, len(occs))
		ts := time.Now().UTC()
		for j := range occs {
			occs[j].CreatedAt = ts
			occs[j].UpdatedAt = ts
			batch[j] = &occs[j]
		}
		if err := s.repo.CreateBatch(ctx, batch); err != nil {
			return created, err
		}
		created += len(batch)
	}
	return created, nil
}

// SettleEntryInput detalha a liquidação de um lançamento (pagamento de despesa
// ou recebimento de receita).
type SettleEntryInput struct {
	WorkspaceID     uuid.UUID
	ID              uuid.UUID
	PaidAt          *time.Time // default: agora
	PaidAmountCents *int64     // default: amount_cents; pode diferir (juros/multa/desconto)
	PaymentMethod   string
	AccountID       *uuid.UUID
	CardID          *uuid.UUID
	Notes           *string
}

// Settle liquida o lançamento com forma de pagamento e valores. Liquidar uma
// fatura pai propaga a realização para os filhos (cascata).
func (s *FinancialEntryService) Settle(ctx context.Context, in SettleEntryInput) (*dom.FinancialEntry, error) {
	e, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	if e.Status == dom.StatusCancelada {
		return nil, &dom.ValidationError{Msg: "lançamento cancelado não pode ser liquidado"}
	}

	method := dom.PaymentMethod(in.PaymentMethod)
	if !dom.ValidPaymentMethod(method) {
		return nil, &dom.ValidationError{Msg: "forma de pagamento inválida"}
	}
	if method == dom.PaymentCartaoCredito && in.CardID == nil {
		return nil, &dom.ValidationError{Msg: "informe o cartão de crédito usado no pagamento"}
	}

	now := time.Now().UTC()
	paidAt := now
	if in.PaidAt != nil {
		paidAt = in.PaidAt.UTC()
	}
	paid := e.AmountCents
	if in.PaidAmountCents != nil {
		if *in.PaidAmountCents == 0 {
			return nil, &dom.ValidationError{Msg: "paid_amount_cents não pode ser zero"}
		}
		paid = *in.PaidAmountCents
	}

	e.Status = dom.StatusRealizada
	e.PaidAt = &paidAt
	e.PaidAmountCents = &paid
	e.PaymentMethod = &method
	e.PaymentAccountID = in.AccountID
	e.PaymentCardID = in.CardID
	if in.Notes != nil {
		e.Notes = in.Notes
	}
	e.UpdatedAt = now

	if err := e.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	if err := s.repo.CascadeStatusToChildren(ctx, in.WorkspaceID, e.ID, dom.StatusRealizada, &paidAt); err != nil {
		return nil, err
	}
	return e, nil
}
