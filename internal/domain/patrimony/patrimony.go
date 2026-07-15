// Package patrimony modela o patrimônio da família: imóveis, seus documentos
// e os impostos/taxas incidentes sobre bens (imóveis e veículos).
package patrimony

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ─── Property ──────────────────────────────────────────────────────────────────

// PropertyType classifica o imóvel.
type PropertyType string

const (
	PropertyCasa        PropertyType = "casa"
	PropertyApartamento PropertyType = "apartamento"
	PropertyTerreno     PropertyType = "terreno"
	PropertyComercial   PropertyType = "comercial"
	PropertyRural       PropertyType = "rural"
	PropertyOutros      PropertyType = "outros"
)

// Property é um imóvel do patrimônio familiar.
type Property struct {
	ID                 uuid.UUID
	WorkspaceID        uuid.UUID
	Name               string
	PropertyType       PropertyType
	Address            *string
	City               *string
	State              *string
	ZipCode            *string
	RegistrationNumber *string // matrícula
	AreaM2             *float64
	PurchaseDate       *time.Time
	PurchaseValueCents *int64
	CurrentValueCents  *int64
	Notes              *string
	Active             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Validate verifica as regras de domínio do imóvel.
func (p *Property) Validate() error {
	if p.Name == "" {
		return &ValidationError{Msg: "name é obrigatório"}
	}
	switch p.PropertyType {
	case PropertyCasa, PropertyApartamento, PropertyTerreno, PropertyComercial, PropertyRural, PropertyOutros:
	default:
		return &ValidationError{Msg: fmt.Sprintf("property_type inválido: %s", p.PropertyType)}
	}
	return nil
}

// ─── Property document ─────────────────────────────────────────────────────────

// PropertyDocType classifica o documento anexado a um imóvel.
type PropertyDocType string

const (
	DocEscritura PropertyDocType = "escritura"
	DocMatricula PropertyDocType = "matricula"
	DocIPTU      PropertyDocType = "iptu"
	DocContrato  PropertyDocType = "contrato"
	DocSeguro    PropertyDocType = "seguro"
	DocPlanta    PropertyDocType = "planta"
	DocOutros    PropertyDocType = "outros"
)

// PropertyDocument é um arquivo (PDF/imagem) anexado a um imóvel.
type PropertyDocument struct {
	ID          uuid.UUID
	PropertyID  uuid.UUID
	WorkspaceID uuid.UUID
	DocType     PropertyDocType
	FileName    string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	Notes       *string
	CreatedAt   time.Time
}

// ─── Asset tax ─────────────────────────────────────────────────────────────────

// AssetType indica sobre qual tipo de bem o imposto incide.
type AssetType string

const (
	AssetProperty AssetType = "property"
	AssetVehicle  AssetType = "vehicle"
)

// TaxType é o tipo de imposto/taxa.
type TaxType string

const (
	TaxIPTU          TaxType = "iptu"
	TaxIPVA          TaxType = "ipva"
	TaxLicenciamento TaxType = "licenciamento"
	TaxDPVAT         TaxType = "dpvat"
	TaxCondominio    TaxType = "condominio"
	TaxSeguroPredial TaxType = "seguro_predial"
	TaxTaxaLixo      TaxType = "taxa_lixo"
	TaxTaxaBombeiros TaxType = "taxa_bombeiros"
	TaxOutros        TaxType = "outros"
)

// TaxStatus é o estado de pagamento do imposto.
type TaxStatus string

const (
	TaxStatusPending TaxStatus = "pending"
	TaxStatusPaid    TaxStatus = "paid"
	TaxStatusOverdue TaxStatus = "overdue"
	TaxStatusPartial TaxStatus = "partial"
)

// AssetTax é um imposto/taxa incidente sobre um bem (imóvel ou veículo).
type AssetTax struct {
	ID                uuid.UUID
	WorkspaceID       uuid.UUID
	AssetType         AssetType
	PropertyID        *uuid.UUID
	VehicleID         *uuid.UUID
	TaxType           TaxType
	ReferenceYear     int
	Description       *string
	DueDate           *time.Time
	AmountCents       int64
	PaidCents         int64
	PaidDate          *time.Time
	Status            TaxStatus
	InstallmentsTotal int
	InstallmentNumber int
	Notes             *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Validate verifica as regras de domínio do imposto.
func (t *AssetTax) Validate() error {
	switch t.AssetType {
	case AssetProperty:
		if t.PropertyID == nil {
			return &ValidationError{Msg: "property_id é obrigatório para asset_type=property"}
		}
	case AssetVehicle:
		if t.VehicleID == nil {
			return &ValidationError{Msg: "vehicle_id é obrigatório para asset_type=vehicle"}
		}
	default:
		return &ValidationError{Msg: fmt.Sprintf("asset_type inválido: %s", t.AssetType)}
	}
	if t.ReferenceYear < 1900 || t.ReferenceYear > 2200 {
		return &ValidationError{Msg: fmt.Sprintf("reference_year inválido: %d", t.ReferenceYear)}
	}
	if t.AmountCents < 0 {
		return &ValidationError{Msg: "amount_cents não pode ser negativo"}
	}
	if t.InstallmentsTotal < 1 {
		t.InstallmentsTotal = 1
	}
	if t.InstallmentNumber < 1 {
		t.InstallmentNumber = 1
	}
	return nil
}

// ComputeStatus deriva o status a partir dos valores pagos e da data de vencimento.
// Preserva um status explícito de "paid" quando totalmente quitado.
func (t *AssetTax) ComputeStatus(now time.Time) TaxStatus {
	if t.PaidCents >= t.AmountCents && t.AmountCents > 0 {
		return TaxStatusPaid
	}
	if t.PaidCents > 0 {
		return TaxStatusPartial
	}
	if t.DueDate != nil && t.DueDate.Before(now) {
		return TaxStatusOverdue
	}
	return TaxStatusPending
}

// ─── Overview (agregados de dashboard) ─────────────────────────────────────────

// YearTotals agrega o previsto e o pago de um ano.
type YearTotals struct {
	Year         int
	PlannedCents int64
	PaidCents    int64
}

// TaxTypeYearTotals agrega o previsto/pago por tipo de imposto em um ano.
type TaxTypeYearTotals struct {
	TaxType      TaxType
	Year         int
	PlannedCents int64
	PaidCents    int64
}

// InflationEntry compara o valor de um tipo de imposto entre anos consecutivos.
type InflationEntry struct {
	TaxType       TaxType
	Year          int
	PreviousYear  int
	AmountCents   int64
	PreviousCents int64
	VariationPct  *float64 // variação percentual YoY (nil quando não há base)
}

// TaxOverview é o retorno agregado para o dashboard de patrimônio.
type TaxOverview struct {
	ByYear             []YearTotals
	ByTaxTypeYear      []TaxTypeYearTotals
	Inflation          []InflationEntry
	Upcoming           []AssetTax // vencimentos nos próximos 90 dias
	Overdue            []AssetTax
	TotalProperties    int
	TotalPropertyValue int64 // soma de current_value_cents (fallback purchase_value)
}

// ─── Params / filtros ──────────────────────────────────────────────────────────

// ListPropertiesParams filtra a listagem de imóveis.
type ListPropertiesParams struct {
	OnlyActive bool
	Limit      int
	Offset     int
}

// ListTaxesParams filtra a listagem de impostos.
type ListTaxesParams struct {
	AssetType     string
	TaxType       string
	ReferenceYear int
	Status        string
	PropertyID    *uuid.UUID
	VehicleID     *uuid.UUID
	Limit         int
	Offset        int
}

// ─── Errors ────────────────────────────────────────────────────────────────────

// ValidationError é retornado quando a entidade viola regras de domínio.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// ErrNotFound é retornado quando o recurso não existe (ou não pertence ao workspace).
var ErrNotFound = &ValidationError{Msg: "não encontrado"}

// ─── Repositórios ──────────────────────────────────────────────────────────────

// Repository define a persistência de imóveis e impostos.
type Repository interface {
	// Properties
	CreateProperty(ctx context.Context, p *Property) error
	GetProperty(ctx context.Context, workspaceID, id uuid.UUID) (*Property, error)
	ListProperties(ctx context.Context, workspaceID uuid.UUID, params ListPropertiesParams) ([]Property, int64, error)
	UpdateProperty(ctx context.Context, p *Property) error
	DeleteProperty(ctx context.Context, workspaceID, id uuid.UUID) error

	// Taxes
	CreateTax(ctx context.Context, t *AssetTax) error
	GetTax(ctx context.Context, workspaceID, id uuid.UUID) (*AssetTax, error)
	ListTaxes(ctx context.Context, workspaceID uuid.UUID, params ListTaxesParams) ([]AssetTax, int64, error)
	UpdateTax(ctx context.Context, t *AssetTax) error
	DeleteTax(ctx context.Context, workspaceID, id uuid.UUID) error
	ListAllTaxes(ctx context.Context, workspaceID uuid.UUID) ([]AssetTax, error)
}

// DocumentRepository define a persistência de documentos de imóveis.
type DocumentRepository interface {
	Create(ctx context.Context, d *PropertyDocument) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*PropertyDocument, error)
	ListByProperty(ctx context.Context, workspaceID, propertyID uuid.UUID) ([]PropertyDocument, error)
	Delete(ctx context.Context, workspaceID, id uuid.UUID) error
}
