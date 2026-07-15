package homesafety

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/homesafety"
)

// Service orquestra as regras de negócio do módulo de segurança do lar.
type Service struct {
	repo dom.Repository
}

func NewService(repo dom.Repository) *Service {
	return &Service{repo: repo}
}

// ─── Item CRUD ────────────────────────────────────────────────────────────────

// CreateItemInput são os dados para criar um item.
type CreateItemInput struct {
	WorkspaceID           uuid.UUID
	Name                  string
	Category              string
	RiskType              string
	Location              *string
	Brand                 *string
	Model                 *string
	InstalledAt           *time.Time
	LifespanMonths        *int
	ServiceIntervalMonths *int
	LastServiceAt         *time.Time
	Priority              string
	Responsible           *string
	LastCostCents         int64
	Active                *bool
	Notes                 *string
}

// UpdateItemInput são os dados para atualizar um item.
type UpdateItemInput struct {
	WorkspaceID           uuid.UUID
	ID                    uuid.UUID
	Name                  string
	Category              string
	RiskType              string
	Location              *string
	Brand                 *string
	Model                 *string
	InstalledAt           *time.Time
	LifespanMonths        *int
	ServiceIntervalMonths *int
	LastServiceAt         *time.Time
	Priority              string
	Responsible           *string
	LastCostCents         int64
	Active                *bool
	Notes                 *string
}

func (s *Service) Create(ctx context.Context, in CreateItemInput) (*dom.Item, error) {
	now := time.Now().UTC()
	priority := dom.Priority(in.Priority)
	if priority == "" {
		priority = dom.PriorityMedia
	}
	riskType := dom.RiskType(in.RiskType)
	if riskType == "" {
		riskType = dom.RiskOutros
	}
	category := dom.Category(in.Category)
	if category == "" {
		category = dom.CategoryOutros
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}

	item := &dom.Item{
		ID:                    uuid.New(),
		WorkspaceID:           in.WorkspaceID,
		Name:                  in.Name,
		Category:              category,
		RiskType:              riskType,
		Location:              in.Location,
		Brand:                 in.Brand,
		Model:                 in.Model,
		InstalledAt:           in.InstalledAt,
		LifespanMonths:        in.LifespanMonths,
		ServiceIntervalMonths: in.ServiceIntervalMonths,
		LastServiceAt:         in.LastServiceAt,
		Priority:              priority,
		Responsible:           in.Responsible,
		LastCostCents:         in.LastCostCents,
		Active:                active,
		Notes:                 in.Notes,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	item.RecalcNextDue()

	if err := item.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Item, error) {
	return s.repo.GetItemByID(ctx, workspaceID, id)
}

func (s *Service) List(ctx context.Context, workspaceID uuid.UUID, category, status, location, query string) ([]dom.Item, error) {
	items, err := s.repo.ListItems(ctx, workspaceID, dom.ListItemsParams{
		Category: category,
		Location: location,
		Query:    query,
	})
	if err != nil {
		return nil, err
	}
	if status == "" {
		return items, nil
	}
	// Filtro por status derivado (feito na aplicação, não no banco).
	now := time.Now().UTC()
	filtered := items[:0]
	for _, it := range items {
		if string(it.Status(now)) == status {
			filtered = append(filtered, it)
		}
	}
	return filtered, nil
}

func (s *Service) Update(ctx context.Context, in UpdateItemInput) (*dom.Item, error) {
	item, err := s.repo.GetItemByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}

	priority := dom.Priority(in.Priority)
	if priority == "" {
		priority = item.Priority
	}
	riskType := dom.RiskType(in.RiskType)
	if riskType == "" {
		riskType = item.RiskType
	}
	category := dom.Category(in.Category)
	if category == "" {
		category = item.Category
	}

	item.Name = in.Name
	item.Category = category
	item.RiskType = riskType
	item.Location = in.Location
	item.Brand = in.Brand
	item.Model = in.Model
	item.InstalledAt = in.InstalledAt
	item.LifespanMonths = in.LifespanMonths
	item.ServiceIntervalMonths = in.ServiceIntervalMonths
	item.LastServiceAt = in.LastServiceAt
	item.Priority = priority
	item.Responsible = in.Responsible
	item.LastCostCents = in.LastCostCents
	if in.Active != nil {
		item.Active = *in.Active
	}
	item.Notes = in.Notes
	item.UpdatedAt = time.Now().UTC()
	item.RecalcNextDue()

	if err := item.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.DeleteItem(ctx, workspaceID, id)
}

// ─── Eventos ──────────────────────────────────────────────────────────────────

// CreateEventInput são os dados para registrar um evento no item.
type CreateEventInput struct {
	WorkspaceID uuid.UUID
	ItemID      uuid.UUID
	EventType   string
	EventDate   time.Time
	CostCents   int64
	Provider    *string
	Notes       *string
}

// CreateEvent registra um evento e atualiza o item: last_service_at e
// next_due_date são recalculados; installed_at é atualizado em instalação/troca.
func (s *Service) CreateEvent(ctx context.Context, in CreateEventInput) (*dom.Event, *dom.Item, error) {
	item, err := s.repo.GetItemByID(ctx, in.WorkspaceID, in.ItemID)
	if err != nil {
		return nil, nil, err
	}

	eventType := dom.EventType(in.EventType)
	if eventType == "" {
		eventType = dom.EventManutencao
	}
	ev := &dom.Event{
		ID:          uuid.New(),
		WorkspaceID: in.WorkspaceID,
		ItemID:      in.ItemID,
		EventType:   eventType,
		EventDate:   in.EventDate,
		CostCents:   in.CostCents,
		Provider:    in.Provider,
		Notes:       in.Notes,
		CreatedAt:   time.Now().UTC(),
	}
	if err := ev.Validate(); err != nil {
		return nil, nil, err
	}
	if err := s.repo.CreateEvent(ctx, ev); err != nil {
		return nil, nil, err
	}

	// Atualiza o item a partir do evento.
	eventDate := in.EventDate
	item.LastServiceAt = &eventDate
	if in.CostCents > 0 {
		item.LastCostCents = in.CostCents
	}
	if eventType == dom.EventInstalacao || eventType == dom.EventTroca {
		item.InstalledAt = &eventDate
	}
	item.UpdatedAt = time.Now().UTC()
	item.RecalcNextDue()
	if err := s.repo.UpdateItem(ctx, item); err != nil {
		return nil, nil, err
	}
	return ev, item, nil
}

func (s *Service) ListEvents(ctx context.Context, workspaceID, itemID uuid.UUID) ([]dom.Event, error) {
	if _, err := s.repo.GetItemByID(ctx, workspaceID, itemID); err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, workspaceID, itemID)
}

// DeleteEvent remove um evento e recalcula o item a partir do evento mais recente
// remanescente.
func (s *Service) DeleteEvent(ctx context.Context, workspaceID, itemID, eventID uuid.UUID) error {
	if _, err := s.repo.DeleteEvent(ctx, workspaceID, itemID, eventID); err != nil {
		return err
	}

	item, err := s.repo.GetItemByID(ctx, workspaceID, itemID)
	if err != nil {
		return err
	}
	events, err := s.repo.ListEvents(ctx, workspaceID, itemID)
	if err != nil {
		return err
	}
	// events já vem ordenado por event_date DESC.
	var lastService *time.Time
	for _, e := range events {
		if e.EventType == dom.EventInstalacao || e.EventType == dom.EventTroca ||
			e.EventType == dom.EventManutencao || e.EventType == dom.EventRecarga ||
			e.EventType == dom.EventLimpeza || e.EventType == dom.EventInspecao {
			d := e.EventDate
			lastService = &d
			break
		}
	}
	item.LastServiceAt = lastService
	item.UpdatedAt = time.Now().UTC()
	item.RecalcNextDue()
	return s.repo.UpdateItem(ctx, item)
}

// ─── Catálogo ─────────────────────────────────────────────────────────────────

func (s *Service) Catalog() []dom.CatalogEntry {
	return dom.Catalog()
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

func (s *Service) Dashboard(ctx context.Context, workspaceID uuid.UUID) (*dom.Dashboard, error) {
	items, err := s.repo.ListItems(ctx, workspaceID, dom.ListItemsParams{})
	if err != nil {
		return nil, err
	}
	costs, err := s.repo.ListMaintenanceCosts(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	statusCounts := map[dom.Status]int{}
	catCounts := map[dom.Category]int{}
	riskCounts := map[dom.RiskType]int{}

	dash := &dom.Dashboard{TotalItems: len(items)}

	for _, it := range items {
		st := it.Status(now)
		statusCounts[st]++
		catCounts[it.Category]++
		riskCounts[it.RiskType]++

		switch st {
		case dom.StatusVencido:
			dash.Overdue = append(dash.Overdue, it)
			dash.DueNext30 = append(dash.DueNext30, it)
			dash.DueNext90 = append(dash.DueNext90, it)
		case dom.StatusAtencao:
			dash.DueNext30 = append(dash.DueNext30, it)
			dash.DueNext90 = append(dash.DueNext90, it)
		case dom.StatusProximo:
			dash.DueNext90 = append(dash.DueNext90, it)
		}
	}

	// Ordena as listas de vencimento pela data mais próxima primeiro.
	sortByDue := func(list []dom.Item) {
		sort.SliceStable(list, func(a, b int) bool {
			if list[a].NextDueDate == nil {
				return false
			}
			if list[b].NextDueDate == nil {
				return true
			}
			return list[a].NextDueDate.Before(*list[b].NextDueDate)
		})
	}
	sortByDue(dash.Overdue)
	sortByDue(dash.DueNext30)
	sortByDue(dash.DueNext90)

	for st, c := range statusCounts {
		dash.StatusCounts = append(dash.StatusCounts, dom.StatusCount{Status: st, Count: c})
	}
	for cat, c := range catCounts {
		dash.ByCategory = append(dash.ByCategory, dom.CategoryCount{Category: cat, Count: c})
	}
	for rk, c := range riskCounts {
		dash.ByRisk = append(dash.ByRisk, dom.RiskCount{RiskType: rk, Count: c})
	}
	sort.Slice(dash.StatusCounts, func(a, b int) bool { return dash.StatusCounts[a].Count > dash.StatusCounts[b].Count })
	sort.Slice(dash.ByCategory, func(a, b int) bool { return dash.ByCategory[a].Count > dash.ByCategory[b].Count })
	sort.Slice(dash.ByRisk, func(a, b int) bool { return dash.ByRisk[a].Count > dash.ByRisk[b].Count })

	// Custo de manutenção por ano e por categoria.
	itemCategory := map[uuid.UUID]dom.Category{}
	for _, it := range items {
		itemCategory[it.ID] = it.Category
	}
	yearCosts := map[int]int64{}
	catCosts := map[dom.Category]int64{}
	oneYearAgo := now.AddDate(-1, 0, 0)
	for _, e := range costs {
		yearCosts[e.EventDate.Year()] += e.CostCents
		if cat, ok := itemCategory[e.ItemID]; ok {
			catCosts[cat] += e.CostCents
		}
		if e.EventDate.After(oneYearAgo) {
			dash.AnnualCostCents += e.CostCents
		}
	}
	for y, c := range yearCosts {
		dash.CostByYear = append(dash.CostByYear, dom.YearCost{Year: y, CostCents: c})
	}
	for cat, c := range catCosts {
		dash.CostByCategory = append(dash.CostByCategory, dom.CategoryCost{Category: cat, CostCents: c})
	}
	sort.Slice(dash.CostByYear, func(a, b int) bool { return dash.CostByYear[a].Year < dash.CostByYear[b].Year })
	sort.Slice(dash.CostByCategory, func(a, b int) bool { return dash.CostByCategory[a].CostCents > dash.CostByCategory[b].CostCents })

	return dash, nil
}
