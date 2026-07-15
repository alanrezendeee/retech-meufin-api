// Package homesafety modela o controle de itens de segurança do lar — itens de
// segurança física, química, biológica, elétrica e de incêndio da casa que
// possuem validade e/ou manutenção periódica (mangueira de gás, extintor,
// limpeza de caixa d'água, dedetização, revisão elétrica, para-raios, etc).
package homesafety

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Category agrupa os itens por natureza do sistema da casa.
type Category string

const (
	CategoryGas                 Category = "gas"
	CategoryAgua                Category = "agua"
	CategoryIncendio            Category = "incendio"
	CategoryEletrica            Category = "eletrica"
	CategoryClimatizacao        Category = "climatizacao"
	CategoryPragas              Category = "pragas"
	CategoryEstrutura           Category = "estrutura"
	CategorySegurancaEletronica Category = "seguranca_eletronica"
	CategoryPiscina             Category = "piscina"
	CategorySaude               Category = "saude"
	CategoryOutros              Category = "outros"
)

// RiskType classifica o risco predominante do item.
type RiskType string

const (
	RiskFisico    RiskType = "fisico"
	RiskQuimico   RiskType = "quimico"
	RiskBiologico RiskType = "biologico"
	RiskEletrico  RiskType = "eletrico"
	RiskIncendio  RiskType = "incendio"
	RiskOutros    RiskType = "outros"
)

// Priority indica a urgência de acompanhamento do item.
type Priority string

const (
	PriorityAlta  Priority = "alta"
	PriorityMedia Priority = "media"
	PriorityBaixa Priority = "baixa"
)

// EventType é o tipo de um evento registrado no histórico de um item.
type EventType string

const (
	EventInstalacao EventType = "instalacao"
	EventTroca      EventType = "troca"
	EventManutencao EventType = "manutencao"
	EventInspecao   EventType = "inspecao"
	EventRecarga    EventType = "recarga"
	EventLimpeza    EventType = "limpeza"
)

// Status é o estado derivado do item em relação ao próximo vencimento.
type Status string

const (
	StatusVencido     Status = "vencido"      // next_due_date já passou
	StatusAtencao     Status = "atencao"      // vence em <= 30 dias
	StatusProximo     Status = "proximo"      // vence em <= 90 dias
	StatusOK          Status = "ok"           // vence em mais de 90 dias
	StatusSemControle Status = "sem_controle" // sem next_due_date definido
)

// Item é o agregado central do módulo de segurança do lar.
type Item struct {
	ID                    uuid.UUID
	WorkspaceID           uuid.UUID
	Name                  string
	Category              Category
	RiskType              RiskType
	Location              *string
	Brand                 *string
	Model                 *string
	InstalledAt           *time.Time
	LifespanMonths        *int
	ServiceIntervalMonths *int
	LastServiceAt         *time.Time
	NextDueDate           *time.Time
	Priority              Priority
	Responsible           *string
	LastCostCents         int64
	Active                bool
	Notes                 *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Event é um evento do histórico de um item (instalação, troca, manutenção…).
type Event struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ItemID      uuid.UUID
	EventType   EventType
	EventDate   time.Time
	CostCents   int64
	Provider    *string
	Notes       *string
	CreatedAt   time.Time
}

// RecalcNextDue recalcula NextDueDate a partir da validade (installed_at +
// lifespan_months) e da periodicidade de manutenção (last_service_at + interval,
// ou installed_at + interval quando ainda não houve manutenção). Segue a regra do
// módulo: o vencimento armazenado é o mais distante entre as datas candidatas.
func (i *Item) RecalcNextDue() {
	var candidates []time.Time

	if i.InstalledAt != nil && i.LifespanMonths != nil && *i.LifespanMonths > 0 {
		candidates = append(candidates, i.InstalledAt.AddDate(0, *i.LifespanMonths, 0))
	}
	if i.ServiceIntervalMonths != nil && *i.ServiceIntervalMonths > 0 {
		base := i.LastServiceAt
		if base == nil {
			base = i.InstalledAt
		}
		if base != nil {
			candidates = append(candidates, base.AddDate(0, *i.ServiceIntervalMonths, 0))
		}
	}

	if len(candidates) == 0 {
		i.NextDueDate = nil
		return
	}
	max := candidates[0]
	for _, c := range candidates[1:] {
		if c.After(max) {
			max = c
		}
	}
	i.NextDueDate = &max
}

// Status calcula o estado do item na data de referência.
func (i *Item) Status(now time.Time) Status {
	return ComputeStatus(i.NextDueDate, now)
}

// DaysUntilDue retorna os dias restantes até o vencimento (negativo se vencido),
// ou nil quando o item não tem controle de vencimento.
func (i *Item) DaysUntilDue(now time.Time) *int {
	if i.NextDueDate == nil {
		return nil
	}
	days := int(i.NextDueDate.Sub(truncateDay(now)).Hours() / 24)
	return &days
}

// ComputeStatus deriva o status a partir de uma data de vencimento.
func ComputeStatus(nextDue *time.Time, now time.Time) Status {
	if nextDue == nil {
		return StatusSemControle
	}
	days := int(nextDue.Sub(truncateDay(now)).Hours() / 24)
	switch {
	case days < 0:
		return StatusVencido
	case days <= 30:
		return StatusAtencao
	case days <= 90:
		return StatusProximo
	default:
		return StatusOK
	}
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ─── Filtros e dashboard ──────────────────────────────────────────────────────

// ListItemsParams filtra a listagem de itens.
type ListItemsParams struct {
	Category string
	Location string
	Query    string
}

// StatusCount é a contagem de itens em um status.
type StatusCount struct {
	Status Status
	Count  int
}

// CategoryCount é a distribuição de itens por categoria.
type CategoryCount struct {
	Category Category
	Count    int
}

// RiskCount é a distribuição de itens por tipo de risco.
type RiskCount struct {
	RiskType RiskType
	Count    int
}

// YearCost agrega o custo de manutenção por ano.
type YearCost struct {
	Year      int
	CostCents int64
}

// CategoryCost agrega o custo de manutenção por categoria.
type CategoryCost struct {
	Category  Category
	CostCents int64
}

// Dashboard consolida os indicadores do módulo.
type Dashboard struct {
	StatusCounts    []StatusCount
	Overdue         []Item          // itens vencidos
	DueNext30       []Item          // vencem em até 30 dias
	DueNext90       []Item          // vencem em até 90 dias
	CostByYear      []YearCost      // custo de manutenção por ano
	CostByCategory  []CategoryCost  // custo de manutenção por categoria
	ByCategory      []CategoryCount // distribuição por categoria
	ByRisk          []RiskCount     // distribuição por risco
	TotalItems      int
	AnnualCostCents int64 // custo dos últimos 12 meses
}

// ─── Catálogo sugerido ────────────────────────────────────────────────────────

// CatalogEntry é uma sugestão de item para o botão "adicionar do catálogo".
type CatalogEntry struct {
	Key                   string   `json:"key"`
	Name                  string   `json:"name"`
	Category              Category `json:"category"`
	RiskType              RiskType `json:"risk_type"`
	LifespanMonths        *int     `json:"lifespan_months"`
	ServiceIntervalMonths *int     `json:"service_interval_months"`
	Priority              Priority `json:"priority"`
	DefaultLocation       string   `json:"default_location"`
	Notes                 string   `json:"notes"`
}

func months(n int) *int { return &n }

// Catalog retorna o catálogo sugerido de itens de segurança do lar (hardcoded).
func Catalog() []CatalogEntry {
	return []CatalogEntry{
		{Key: "mangueira_gas", Name: "Mangueira de gás", Category: CategoryGas, RiskType: RiskIncendio, LifespanMonths: months(60), Priority: PriorityAlta, DefaultLocation: "Cozinha", Notes: "Validade de 5 anos impressa na mangueira. Troque mesmo sem sinais de desgaste."},
		{Key: "regulador_gas", Name: "Regulador de gás", Category: CategoryGas, RiskType: RiskIncendio, LifespanMonths: months(60), Priority: PriorityAlta, DefaultLocation: "Cozinha", Notes: "Vida útil de 5 anos. Verifique data de fabricação no corpo do regulador."},
		{Key: "botijao_gas", Name: "Botijão de gás (GLP)", Category: CategoryGas, RiskType: RiskIncendio, Priority: PriorityMedia, DefaultLocation: "Área de serviço", Notes: "Requalificação é responsabilidade do distribuidor. Verifique lacre e validade ao trocar."},
		{Key: "aquecedor_gas", Name: "Aquecedor a gás (revisão)", Category: CategoryGas, RiskType: RiskIncendio, ServiceIntervalMonths: months(12), Priority: PriorityAlta, DefaultLocation: "Banheiro / Área externa", Notes: "Revisão anual obrigatória — risco de monóxido de carbono (CO). Exige exaustão adequada."},
		{Key: "limpeza_caixa_dagua", Name: "Limpeza de caixa d'água", Category: CategoryAgua, RiskType: RiskBiologico, ServiceIntervalMonths: months(6), Priority: PriorityAlta, DefaultLocation: "Reservatório", Notes: "Recomendada a cada 6 meses para prevenir contaminação."},
		{Key: "filtro_agua", Name: "Troca de filtro / refil de água", Category: CategoryAgua, RiskType: RiskBiologico, ServiceIntervalMonths: months(6), Priority: PriorityMedia, DefaultLocation: "Cozinha", Notes: "Troque o refil a cada 6 meses ou conforme volume filtrado."},
		{Key: "extintor", Name: "Extintor de incêndio", Category: CategoryIncendio, RiskType: RiskIncendio, ServiceIntervalMonths: months(12), Priority: PriorityAlta, DefaultLocation: "Corredor / Garagem", Notes: "Recarga anual. Verifique manômetro na faixa verde e validade do cilindro."},
		{Key: "detector_fumaca", Name: "Detector de fumaça", Category: CategoryIncendio, RiskType: RiskIncendio, ServiceIntervalMonths: months(12), Priority: PriorityMedia, DefaultLocation: "Quartos / Corredor", Notes: "Teste periódico e troca anual da bateria."},
		{Key: "chamine_churrasqueira", Name: "Chaminé / churrasqueira", Category: CategoryEstrutura, RiskType: RiskIncendio, ServiceIntervalMonths: months(12), Priority: PriorityBaixa, DefaultLocation: "Área gourmet", Notes: "Limpeza anual para evitar acúmulo de fuligem e risco de incêndio."},
		{Key: "revisao_eletrica", Name: "Revisão elétrica / disjuntores / DR", Category: CategoryEletrica, RiskType: RiskEletrico, ServiceIntervalMonths: months(12), Priority: PriorityAlta, DefaultLocation: "Quadro de distribuição", Notes: "Teste o botão do DR mensalmente; revisão geral anual por eletricista."},
		{Key: "para_raios", Name: "Para-raios (SPDA)", Category: CategoryEletrica, RiskType: RiskEletrico, ServiceIntervalMonths: months(12), Priority: PriorityAlta, DefaultLocation: "Cobertura", Notes: "Inspeção anual do sistema de proteção contra descargas atmosféricas."},
		{Key: "alarme_cftv", Name: "Alarme / CFTV (bateria)", Category: CategorySegurancaEletronica, RiskType: RiskEletrico, ServiceIntervalMonths: months(12), Priority: PriorityMedia, DefaultLocation: "Central de segurança", Notes: "Troca anual da bateria da central e verificação das câmeras."},
		{Key: "ar_condicionado", Name: "Limpeza de ar-condicionado / filtros", Category: CategoryClimatizacao, RiskType: RiskBiologico, ServiceIntervalMonths: months(3), Priority: PriorityMedia, DefaultLocation: "Quartos / Sala", Notes: "Higienização trimestral dos filtros; limpeza técnica conforme uso."},
		{Key: "dedetizacao", Name: "Dedetização", Category: CategoryPragas, RiskType: RiskBiologico, ServiceIntervalMonths: months(12), Priority: PriorityMedia, DefaultLocation: "Casa toda", Notes: "Controle de pragas anual, ou semestral em áreas críticas."},
		{Key: "limpeza_calhas", Name: "Limpeza de calhas", Category: CategoryEstrutura, RiskType: RiskOutros, ServiceIntervalMonths: months(6), Priority: PriorityBaixa, DefaultLocation: "Telhado", Notes: "Limpeza semestral, principalmente antes do período de chuvas."},
		{Key: "corrimaos", Name: "Corrimãos e guarda-corpos", Category: CategoryEstrutura, RiskType: RiskFisico, ServiceIntervalMonths: months(12), Priority: PriorityBaixa, DefaultLocation: "Escadas / Sacadas", Notes: "Inspeção anual de fixação e integridade estrutural."},
		{Key: "piscina", Name: "Piscina (tratamento químico)", Category: CategoryPiscina, RiskType: RiskQuimico, ServiceIntervalMonths: months(1), Priority: PriorityMedia, DefaultLocation: "Área de lazer", Notes: "Controle mensal de pH e cloro; armazene produtos químicos com segurança."},
		{Key: "kit_primeiros_socorros", Name: "Kit de primeiros socorros", Category: CategorySaude, RiskType: RiskBiologico, ServiceIntervalMonths: months(12), Priority: PriorityMedia, DefaultLocation: "Cozinha / Banheiro", Notes: "Revisão anual das validades de medicamentos e itens do kit."},
		{Key: "vacina_pet_antirrabica", Name: "Vacinação antirrábica de pets", Category: CategorySaude, RiskType: RiskBiologico, ServiceIntervalMonths: months(12), Priority: PriorityAlta, DefaultLocation: "—", Notes: "Reforço anual da vacina antirrábica dos animais domésticos."},
	}
}

// ─── Erros e validação ────────────────────────────────────────────────────────

// ValidationError é retornado quando a entidade viola regras de domínio.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

// ErrNotFound é retornado quando um item ou evento não existe (ou não pertence ao workspace).
var ErrNotFound = &ValidationError{Msg: "não encontrado"}

// Validate verifica as regras de domínio do Item.
func (i *Item) Validate() error {
	if i.Name == "" {
		return &ValidationError{Msg: "name é obrigatório"}
	}
	switch i.Category {
	case CategoryGas, CategoryAgua, CategoryIncendio, CategoryEletrica, CategoryClimatizacao,
		CategoryPragas, CategoryEstrutura, CategorySegurancaEletronica, CategoryPiscina,
		CategorySaude, CategoryOutros:
	default:
		return &ValidationError{Msg: "category inválida: " + string(i.Category)}
	}
	switch i.RiskType {
	case RiskFisico, RiskQuimico, RiskBiologico, RiskEletrico, RiskIncendio, RiskOutros:
	default:
		return &ValidationError{Msg: "risk_type inválido: " + string(i.RiskType)}
	}
	switch i.Priority {
	case PriorityAlta, PriorityMedia, PriorityBaixa:
	default:
		return &ValidationError{Msg: "priority inválida: " + string(i.Priority)}
	}
	return nil
}

// Validate verifica as regras de domínio do Event.
func (e *Event) Validate() error {
	switch e.EventType {
	case EventInstalacao, EventTroca, EventManutencao, EventInspecao, EventRecarga, EventLimpeza:
	default:
		return &ValidationError{Msg: "event_type inválido: " + string(e.EventType)}
	}
	if e.EventDate.IsZero() {
		return &ValidationError{Msg: "event_date é obrigatório"}
	}
	return nil
}

// Repository define as operações de persistência do módulo.
type Repository interface {
	CreateItem(ctx context.Context, i *Item) error
	GetItemByID(ctx context.Context, workspaceID, id uuid.UUID) (*Item, error)
	ListItems(ctx context.Context, workspaceID uuid.UUID, p ListItemsParams) ([]Item, error)
	UpdateItem(ctx context.Context, i *Item) error
	DeleteItem(ctx context.Context, workspaceID, id uuid.UUID) error

	CreateEvent(ctx context.Context, e *Event) error
	ListEvents(ctx context.Context, workspaceID, itemID uuid.UUID) ([]Event, error)
	DeleteEvent(ctx context.Context, workspaceID, itemID, eventID uuid.UUID) (*Event, error)

	// ListMaintenanceCosts retorna os eventos com custo (> 0) do workspace para
	// agregação de custo no dashboard.
	ListMaintenanceCosts(ctx context.Context, workspaceID uuid.UUID) ([]Event, error)
}
