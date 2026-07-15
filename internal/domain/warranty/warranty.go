package warranty

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Category classifica o tipo de bem cuja garantia é controlada.
type Category string

const (
	CategoryEletrodomestico Category = "eletrodomestico"
	CategoryEletronico      Category = "eletronico"
	CategoryInformatica     Category = "informatica"
	CategoryCelular         Category = "celular"
	CategoryMovel           Category = "movel"
	CategoryVeiculo         Category = "veiculo"
	CategoryImovel          Category = "imovel"
	CategoryFerramenta      Category = "ferramenta"
	CategoryBrinquedo       Category = "brinquedo"
	CategoryEsporte         Category = "esporte"
	CategoryOutros          Category = "outros"
)

// ValidCategory informa se a categoria é conhecida.
func ValidCategory(c Category) bool {
	switch c {
	case CategoryEletrodomestico, CategoryEletronico, CategoryInformatica, CategoryCelular,
		CategoryMovel, CategoryVeiculo, CategoryImovel, CategoryFerramenta,
		CategoryBrinquedo, CategoryEsporte, CategoryOutros:
		return true
	}
	return false
}

// Status é a situação calculada (não persistida) da garantia.
type Status string

const (
	StatusVigente       Status = "vigente"
	StatusExpiraEmBreve Status = "expira_em_breve"
	StatusExpirada      Status = "expirada"
)

// expiringSoonDays define a janela (em dias) em que a garantia é considerada
// "expira em breve".
const expiringSoonDays = 60

// Warranty é o agregado central do módulo de garantias de bens da família.
type Warranty struct {
	ID                        uuid.UUID
	WorkspaceID               uuid.UUID
	ItemName                  string
	Category                  Category
	Brand                     *string
	Model                     *string
	SerialNumber              *string
	Store                     *string // loja / marketplace (Mercado Livre, Magalu, ...)
	SupplierName              *string
	PurchaseDate              time.Time
	PriceCents                *int64
	InvoiceNumber             *string
	EntryID                   *uuid.UUID // lançamento financeiro vinculado
	FiscalItemID              *uuid.UUID
	LegalWarrantyDays         int // garantia legal (CDC) — default 90
	ContractualWarrantyMonths int // garantia do fabricante — default 12
	ExtendedWarrantyMonths    int // garantia estendida — default 0
	ExtendedProvider          *string
	ExtendedCostCents         int64
	CoverageNotes             *string
	Notes                     *string
	Active                    bool
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// ExpiresAt calcula a data de expiração da cobertura combinando a garantia
// legal (dias) com a soma da garantia contratual + estendida (meses), a partir
// da data da compra. Retorna a maior das duas coberturas.
func (w *Warranty) ExpiresAt() time.Time {
	legalEnd := w.PurchaseDate.AddDate(0, 0, w.LegalWarrantyDays)
	coverageMonths := w.ContractualWarrantyMonths + w.ExtendedWarrantyMonths
	coverageEnd := w.PurchaseDate.AddDate(0, coverageMonths, 0)
	if coverageEnd.After(legalEnd) {
		return coverageEnd
	}
	return legalEnd
}

// DaysRemaining retorna quantos dias faltam para a garantia expirar (referência
// = ref). Valor negativo indica garantia já expirada.
func (w *Warranty) DaysRemaining(ref time.Time) int {
	expires := w.ExpiresAt()
	refDay := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)
	expDay := time.Date(expires.Year(), expires.Month(), expires.Day(), 0, 0, 0, 0, time.UTC)
	return int(expDay.Sub(refDay).Hours() / 24)
}

// StatusAt retorna o status da garantia numa data de referência.
func (w *Warranty) StatusAt(ref time.Time) Status {
	days := w.DaysRemaining(ref)
	switch {
	case days < 0:
		return StatusExpirada
	case days <= expiringSoonDays:
		return StatusExpiraEmBreve
	default:
		return StatusVigente
	}
}

// Validate verifica as regras de domínio da garantia.
func (w *Warranty) Validate() error {
	if w.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if strings.TrimSpace(w.ItemName) == "" {
		return &ValidationError{Msg: "item_name é obrigatório"}
	}
	if !ValidCategory(w.Category) {
		return &ValidationError{Msg: "category inválida"}
	}
	if w.PurchaseDate.IsZero() {
		return &ValidationError{Msg: "purchase_date é obrigatória"}
	}
	if w.LegalWarrantyDays < 0 {
		return &ValidationError{Msg: "legal_warranty_days inválido"}
	}
	if w.ContractualWarrantyMonths < 0 {
		return &ValidationError{Msg: "contractual_warranty_months inválido"}
	}
	if w.ExtendedWarrantyMonths < 0 {
		return &ValidationError{Msg: "extended_warranty_months inválido"}
	}
	if w.PriceCents != nil && *w.PriceCents < 0 {
		return &ValidationError{Msg: "price_cents inválido"}
	}
	if w.ExtendedCostCents < 0 {
		return &ValidationError{Msg: "extended_cost_cents inválido"}
	}
	return nil
}

// DocType classifica o documento anexado a uma garantia.
type DocType string

const (
	DocNotaFiscal        DocType = "nota_fiscal"
	DocCertificado       DocType = "certificado"
	DocGarantiaEstendida DocType = "garantia_estendida"
	DocManual            DocType = "manual"
	DocOutros            DocType = "outros"
)

// ValidDocType informa se o tipo de documento é conhecido.
func ValidDocType(t DocType) bool {
	switch t {
	case DocNotaFiscal, DocCertificado, DocGarantiaEstendida, DocManual, DocOutros:
		return true
	}
	return false
}

// Document é um arquivo anexado a uma garantia (nota fiscal, certificado, ...).
type Document struct {
	ID               uuid.UUID
	WarrantyID       uuid.UUID
	WorkspaceID      uuid.UUID
	DocType          DocType
	FileName         string
	OriginalFileName string
	ObjectKey        string
	ContentType      string
	SizeBytes        int64
	Notes            *string
	CreatedAt        time.Time
}

// Validate valida invariantes do documento de garantia.
func (d *Document) Validate() error {
	if d.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if d.WarrantyID == uuid.Nil {
		return &ValidationError{Msg: "warranty_id é obrigatório"}
	}
	if !ValidDocType(d.DocType) {
		return &ValidationError{Msg: "doc_type inválido"}
	}
	if strings.TrimSpace(d.FileName) == "" {
		return &ValidationError{Msg: "file_name é obrigatório"}
	}
	if strings.TrimSpace(d.ObjectKey) == "" {
		return &ValidationError{Msg: "object_key é obrigatório"}
	}
	return nil
}

// ─── Summary ──────────────────────────────────────────────────────────────────

// CategoryCount agrega a contagem de garantias ativas por categoria.
type CategoryCount struct {
	Category Category
	Count    int
}

// ExpiringItem é uma linha curta de garantia próxima do vencimento (para o resumo).
type ExpiringItem struct {
	ID            uuid.UUID
	ItemName      string
	Category      Category
	ExpiresAt     time.Time
	DaysRemaining int
}

// Summary agrega os indicadores do módulo de garantias.
type Summary struct {
	TotalActive       int
	TotalCoveredCents int64 // soma de price_cents das garantias vigentes/expira em breve
	ExpiringIn30Count int
	ExpiringIn60Count int
	ExpiringIn90Count int
	ExpiredThisYear   int
	ExpiringSoon      []ExpiringItem // lista curta (ordenada por vencimento)
	ByCategory        []CategoryCount
}

// ─── Errors ───────────────────────────────────────────────────────────────────

// ValidationError é retornado quando a entidade viola regras de domínio.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// ErrNotFound é retornado quando a garantia/documento não existe (ou não
// pertence ao workspace).
var ErrNotFound = &ValidationError{Msg: "não encontrado"}

// ─── Repository ───────────────────────────────────────────────────────────────

// ListParams filtra a listagem de garantias.
type ListParams struct {
	Category string
	Status   string // vigente | expira_em_breve | expirada
	Query    string // busca por item_name / brand
	Limit    int
	Offset   int
}

// Repository define as operações de persistência do módulo de garantias.
type Repository interface {
	Create(ctx context.Context, w *Warranty) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Warranty, error)
	// List retorna as garantias do workspace. O filtro de status é aplicado na
	// camada de serviço (é campo calculado), portanto o repositório devolve o
	// conjunto filtrado por category/query e a contagem correspondente.
	List(ctx context.Context, workspaceID uuid.UUID, p ListParams) ([]Warranty, int64, error)
	// ListActive retorna todas as garantias ativas do workspace (para o resumo).
	ListActive(ctx context.Context, workspaceID uuid.UUID) ([]Warranty, error)
	Update(ctx context.Context, w *Warranty) error
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error

	// Documentos
	CreateDocument(ctx context.Context, d *Document) error
	GetDocumentByID(ctx context.Context, workspaceID, id uuid.UUID) (*Document, error)
	ListDocuments(ctx context.Context, workspaceID, warrantyID uuid.UUID) ([]Document, error)
	DeleteDocument(ctx context.Context, workspaceID, id uuid.UUID) error
}
