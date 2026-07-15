// Package education modela o domínio de Educação / Material Escolar: matrículas
// (ano letivo por membro da família), listas de material e seus itens.
package education

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Stage é a etapa escolar da matrícula.
type Stage string

const (
	StageBercario     Stage = "bercario"
	StageInfantil     Stage = "infantil"
	StageFundamental1 Stage = "fundamental1"
	StageFundamental2 Stage = "fundamental2"
	StageMedio        Stage = "medio"
	StageTecnico      Stage = "tecnico"
	StagePreVestibular Stage = "pre_vestibular"
	StageSuperior     Stage = "superior"
	StagePos          Stage = "pos"
)

// Shift é o turno da matrícula.
type Shift string

const (
	ShiftManha    Shift = "manha"
	ShiftTarde    Shift = "tarde"
	ShiftIntegral Shift = "integral"
	ShiftNoite    Shift = "noite"
	ShiftEAD      Shift = "ead"
)

// ListStatus é o estado de uma lista de material.
type ListStatus string

const (
	ListStatusPlanejada ListStatus = "planejada"
	ListStatusEmCompra  ListStatus = "em_compra"
	ListStatusConcluida ListStatus = "concluida"
)

// ItemCategory é a categoria de um item de material.
type ItemCategory string

const (
	ItemCategoryPapelaria   ItemCategory = "papelaria"
	ItemCategoryLivros      ItemCategory = "livros"
	ItemCategoryUniforme    ItemCategory = "uniforme"
	ItemCategoryMochila     ItemCategory = "mochila"
	ItemCategoryEletronicos ItemCategory = "eletronicos"
	ItemCategoryArte        ItemCategory = "arte"
	ItemCategoryHigiene     ItemCategory = "higiene"
	ItemCategoryOutros      ItemCategory = "outros"
)

// SchoolEnrollment é a matrícula (ano letivo) de um membro da família.
type SchoolEnrollment struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	MemberID         uuid.UUID
	SchoolYear       int
	Stage            Stage
	SchoolName       *string
	Grade            *string
	Shift            *Shift
	MonthlyFeeCents  int64
	EnrollmentFeeCents int64
	Notes            *string
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// MemberName é enriquecido pela camada de aplicação (não persistido aqui).
	MemberName *string
}

// SchoolSupplyList é uma lista de material vinculada a uma matrícula.
type SchoolSupplyList struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	EnrollmentID uuid.UUID
	Title        string
	Status       ListStatus
	Notes        *string
	Items        []SchoolSupplyItem
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Enriquecidos pela camada de aplicação (não persistidos aqui).
	MemberID   *uuid.UUID
	MemberName *string
	SchoolYear *int
}

// SchoolSupplyItem é um item de uma lista de material.
type SchoolSupplyItem struct {
	ID                  uuid.UUID
	WorkspaceID         uuid.UUID
	ListID              uuid.UUID
	Name                string
	Category            ItemCategory
	Quantity            float64
	ReferencePriceCents int64
	Purchased           bool
	PaidPriceCents      int64
	PurchasedAt         *time.Time
	Store               *string
	Notes               *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ─── Dashboard / analytics ─────────────────────────────────────────────────────

// MemberSpend agrega o gasto com material por membro da família.
type MemberSpend struct {
	MemberID       string
	MemberName     string
	TotalPaidCents int64
	ItemCount      int
	PurchasedCount int
	PurchasedPct   float64
}

// CategoryAvg agrega custo médio por item por categoria.
type CategoryAvg struct {
	Category        string
	ItemCount       int
	PurchasedCount  int
	TotalPaidCents  int64
	AvgPaidCents    int64
}

// YearSpend é a evolução anual do gasto com educação (mensalidades + material).
type YearSpend struct {
	SchoolYear        int
	MonthlyFeesCents  int64 // mensalidade anualizada (mensal × 12)
	EnrollmentFeesCents int64
	SuppliesPaidCents int64
	TotalCents        int64
}

// Dashboard consolida os indicadores de educação de um ano letivo.
type Dashboard struct {
	SchoolYear          int
	TotalReferenceCents int64 // referência de todos os itens planejados do ano
	TotalPaidCents      int64 // pago dos itens comprados
	ListCount           int
	ItemCount           int
	PurchasedCount      int
	PurchasedPct        float64
	// Economia/estouro sobre os itens comprados (referência − pago).
	SavingsCents        int64
	SavingsPct          float64
	MonthlyFeesCents    int64 // soma das mensalidades do ano (valor mensal)
	EnrollmentFeesCents int64
	ByMember            []MemberSpend
	ByCategory          []CategoryAvg
	AnnualEvolution     []YearSpend
}

// ─── Errors ────────────────────────────────────────────────────────────────────

// ValidationError é retornado quando a entidade viola regras de domínio.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// ErrNotFound é retornado quando uma entidade não existe (ou não pertence ao workspace).
var ErrNotFound = &ValidationError{Msg: "não encontrado"}

// ─── Validation ──────────────────────────────────────────────────────────────

func (e *SchoolEnrollment) Validate() error {
	if e.MemberID == uuid.Nil {
		return &ValidationError{Msg: "member_id é obrigatório"}
	}
	if e.SchoolYear < 1900 || e.SchoolYear > 2200 {
		return &ValidationError{Msg: fmt.Sprintf("school_year inválido: %d", e.SchoolYear)}
	}
	switch e.Stage {
	case StageBercario, StageInfantil, StageFundamental1, StageFundamental2, StageMedio,
		StageTecnico, StagePreVestibular, StageSuperior, StagePos:
	default:
		return &ValidationError{Msg: fmt.Sprintf("stage inválido: %s", e.Stage)}
	}
	if e.Shift != nil {
		switch *e.Shift {
		case ShiftManha, ShiftTarde, ShiftIntegral, ShiftNoite, ShiftEAD:
		default:
			return &ValidationError{Msg: fmt.Sprintf("shift inválido: %s", *e.Shift)}
		}
	}
	if e.MonthlyFeeCents < 0 || e.EnrollmentFeeCents < 0 {
		return &ValidationError{Msg: "valores não podem ser negativos"}
	}
	return nil
}

func (l *SchoolSupplyList) Validate() error {
	if l.EnrollmentID == uuid.Nil {
		return &ValidationError{Msg: "enrollment_id é obrigatório"}
	}
	if l.Title == "" {
		return &ValidationError{Msg: "title é obrigatório"}
	}
	switch l.Status {
	case ListStatusPlanejada, ListStatusEmCompra, ListStatusConcluida:
	default:
		return &ValidationError{Msg: fmt.Sprintf("status inválido: %s", l.Status)}
	}
	return nil
}

func (i *SchoolSupplyItem) Validate() error {
	if i.Name == "" {
		return &ValidationError{Msg: "name é obrigatório"}
	}
	switch i.Category {
	case ItemCategoryPapelaria, ItemCategoryLivros, ItemCategoryUniforme, ItemCategoryMochila,
		ItemCategoryEletronicos, ItemCategoryArte, ItemCategoryHigiene, ItemCategoryOutros:
	default:
		return &ValidationError{Msg: fmt.Sprintf("category inválida: %s", i.Category)}
	}
	if i.Quantity <= 0 {
		return &ValidationError{Msg: "quantity deve ser maior que zero"}
	}
	if i.ReferencePriceCents < 0 || i.PaidPriceCents < 0 {
		return &ValidationError{Msg: "preços não podem ser negativos"}
	}
	return nil
}

// ─── Repository ──────────────────────────────────────────────────────────────

// ListEnrollmentsParams filtra a listagem de matrículas.
type ListEnrollmentsParams struct {
	MemberID   *uuid.UUID
	SchoolYear *int
}

// ListSupplyListsParams filtra a listagem de listas de material.
type ListSupplyListsParams struct {
	EnrollmentID *uuid.UUID
	SchoolYear   *int
	Status       string
}

// Repository define a persistência do módulo de educação.
type Repository interface {
	// Enrollments
	CreateEnrollment(ctx context.Context, e *SchoolEnrollment) error
	GetEnrollment(ctx context.Context, workspaceID, id uuid.UUID) (*SchoolEnrollment, error)
	ListEnrollments(ctx context.Context, workspaceID uuid.UUID, p ListEnrollmentsParams) ([]SchoolEnrollment, error)
	UpdateEnrollment(ctx context.Context, e *SchoolEnrollment) error
	DeleteEnrollment(ctx context.Context, workspaceID, id uuid.UUID) error

	// Supply lists
	CreateList(ctx context.Context, l *SchoolSupplyList) error
	GetList(ctx context.Context, workspaceID, id uuid.UUID) (*SchoolSupplyList, error)
	ListSupplyLists(ctx context.Context, workspaceID uuid.UUID, p ListSupplyListsParams) ([]SchoolSupplyList, error)
	UpdateList(ctx context.Context, l *SchoolSupplyList) error
	DeleteList(ctx context.Context, workspaceID, id uuid.UUID) error

	// Supply items
	CreateItem(ctx context.Context, i *SchoolSupplyItem) error
	GetItem(ctx context.Context, workspaceID, id uuid.UUID) (*SchoolSupplyItem, error)
	UpdateItem(ctx context.Context, i *SchoolSupplyItem) error
	DeleteItem(ctx context.Context, workspaceID, id uuid.UUID) error

	// Dashboard raw data (workspace-wide; agregação feita na aplicação)
	AllEnrollments(ctx context.Context, workspaceID uuid.UUID) ([]SchoolEnrollment, error)
	AllLists(ctx context.Context, workspaceID uuid.UUID) ([]SchoolSupplyList, error)
	AllItems(ctx context.Context, workspaceID uuid.UUID) ([]SchoolSupplyItem, error)
	MemberNames(ctx context.Context, workspaceID uuid.UUID) (map[string]string, error)
}
