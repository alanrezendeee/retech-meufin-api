package health

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

// fuzzyThreshold mínimo de similaridade para sugerir um marcador parecido.
const fuzzyThreshold = 0.55
const candidatesCap = 1000

type MarkerService struct {
	repo dom.MarkerRepository
}

func NewMarkerService(repo dom.MarkerRepository) *MarkerService {
	return &MarkerService{repo: repo}
}

type CreateMarkerInput struct {
	WorkspaceID    uuid.UUID
	CanonicalName  string
	Category       string
	Comparability  string
	CanonicalUnit  *string
	LoincCode      *string
	DefaultRefMin  *float64
	DefaultRefMax  *float64
	DefaultRefText *string
	Aliases        []string
}

type UpdateMarkerInput struct {
	WorkspaceID    uuid.UUID
	ID             uuid.UUID
	CanonicalName  string
	Category       string
	Comparability  string
	CanonicalUnit  *string
	LoincCode      *string
	DefaultRefMin  *float64
	DefaultRefMax  *float64
	DefaultRefText *string
	Active         *bool
}

// Create cadastra um marcador do tenant após dedup exato (nome + aliases, escopo system+tenant).
func (s *MarkerService) Create(ctx context.Context, in CreateMarkerInput) (*dom.Marker, error) {
	keys := []string{dom.Normalize(in.CanonicalName)}
	for _, a := range in.Aliases {
		if n := dom.Normalize(a); n != "" {
			keys = append(keys, n)
		}
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		existing, err := s.repo.MatchExact(ctx, in.WorkspaceID, k)
		if err == nil && existing != nil {
			return nil, &dom.DuplicateError{Existing: existing}
		}
		if err != nil && !errors.Is(err, dom.ErrNotFound) {
			return nil, err
		}
	}

	now := time.Now().UTC()
	ws := in.WorkspaceID
	m := &dom.Marker{
		ID:             uuid.New(),
		Scope:          dom.ScopeTenant,
		WorkspaceID:    &ws,
		CanonicalName:  in.CanonicalName,
		Category:       in.Category,
		Comparability:  dom.ComparabilityClass(in.Comparability),
		CanonicalUnit:  in.CanonicalUnit,
		LoincCode:      in.LoincCode,
		DefaultRefMin:  in.DefaultRefMin,
		DefaultRefMax:  in.DefaultRefMax,
		DefaultRefText: in.DefaultRefText,
		Active:         true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for _, a := range in.Aliases {
		at := strings.TrimSpace(a)
		if at == "" {
			continue
		}
		m.Aliases = append(m.Aliases, dom.MarkerAlias{
			ID:          uuid.New(),
			MarkerID:    m.ID,
			Scope:       dom.ScopeTenant,
			WorkspaceID: &ws,
			Alias:       at,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MarkerService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.Marker, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListMarkersResult struct {
	Items []dom.Marker
	Total int64
}

func (s *MarkerService) List(ctx context.Context, workspaceID uuid.UUID, query, category string, limit, offset int) (*ListMarkersResult, error) {
	items, total, err := s.repo.Search(ctx, workspaceID, query, category, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListMarkersResult{Items: items, Total: total}, nil
}

// Update altera um marcador do tenant. Marcadores do sistema são imutáveis.
func (s *MarkerService) Update(ctx context.Context, in UpdateMarkerInput) (*dom.Marker, error) {
	m, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}
	if m.IsSystem() {
		return nil, dom.ErrImmutable
	}

	newKey := dom.Normalize(in.CanonicalName)
	if newKey != "" && newKey != m.NormalizedKey {
		existing, err := s.repo.MatchExact(ctx, in.WorkspaceID, newKey)
		if err == nil && existing != nil && existing.ID != m.ID {
			return nil, &dom.DuplicateError{Existing: existing}
		}
		if err != nil && !errors.Is(err, dom.ErrNotFound) {
			return nil, err
		}
	}

	m.CanonicalName = in.CanonicalName
	m.Category = in.Category
	m.Comparability = dom.ComparabilityClass(in.Comparability)
	m.CanonicalUnit = in.CanonicalUnit
	m.LoincCode = in.LoincCode
	m.DefaultRefMin = in.DefaultRefMin
	m.DefaultRefMax = in.DefaultRefMax
	m.DefaultRefText = in.DefaultRefText
	if in.Active != nil {
		m.Active = *in.Active
	}
	m.UpdatedAt = time.Now().UTC()
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Delete faz soft-delete de um marcador do tenant.
func (s *MarkerService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	if m.IsSystem() {
		return dom.ErrImmutable
	}
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

// --- Resolução raw -> marcador canônico ---

type ResolveItemInput struct {
	RawName string
	Unit    *string
}

type ResolveCandidate struct {
	Marker     dom.Marker
	Similarity float64
}

type ResolveStatus string

const (
	ResolveMatched    ResolveStatus = "matched"
	ResolveAmbiguous  ResolveStatus = "ambiguous"
	ResolveUnresolved ResolveStatus = "unresolved"
)

type ResolveItemResult struct {
	RawName    string
	Status     ResolveStatus
	Matched    *dom.Marker
	Candidates []ResolveCandidate
}

// Resolve mapeia nomes crus (de OCR/LLM ou digitados) para marcadores canônicos.
// Match exato (nome completo ou variantes: parênteses, barras) -> matched;
// variantes batendo em marcadores distintos -> ambiguous; senão candidatos
// fuzzy -> ambiguous; nenhum -> unresolved.
func (s *MarkerService) Resolve(ctx context.Context, workspaceID uuid.UUID, items []ResolveItemInput) ([]ResolveItemResult, error) {
	var pool []dom.Marker // carregado sob demanda para o fuzzy
	poolLoaded := false

	out := make([]ResolveItemResult, 0, len(items))
	for _, it := range items {
		res := ResolveItemResult{RawName: it.RawName, Status: ResolveUnresolved}

		// Laudos imprimem o marcador de várias formas ("Transaminase Oxalacética
		// (TGO)", "TGO/AST"); o Normalize achata tudo e o exato falhava mesmo com
		// o alias no catálogo. Tenta cada variante antes de cair no fuzzy.
		var hits []*dom.Marker
		for _, variant := range nameVariants(it.RawName) {
			exact, err := s.repo.MatchExact(ctx, workspaceID, variant)
			if err != nil {
				if errors.Is(err, dom.ErrNotFound) {
					continue
				}
				return nil, err
			}
			if exact != nil && !containsMarker(hits, exact.ID) {
				hits = append(hits, exact)
			}
		}
		switch {
		case len(hits) == 1:
			res.Status = ResolveMatched
			res.Matched = hits[0]
			out = append(out, res)
			continue
		case len(hits) > 1:
			// Variantes batem em marcadores diferentes (ex.: "TGO/TGP"): o
			// usuário decide.
			res.Status = ResolveAmbiguous
			for _, h := range hits {
				res.Candidates = append(res.Candidates, ResolveCandidate{Marker: *h, Similarity: 1})
			}
			out = append(out, res)
			continue
		}

		norm := dom.Normalize(it.RawName)
		if norm == "" {
			out = append(out, res)
			continue
		}
		if !poolLoaded {
			var err error
			pool, err = s.repo.Candidates(ctx, workspaceID, candidatesCap)
			if err != nil {
				return nil, err
			}
			poolLoaded = true
		}
		cands := rankCandidates(norm, pool)
		if len(cands) > 0 {
			res.Status = ResolveAmbiguous
			res.Candidates = cands
		}
		out = append(out, res)
	}
	return out, nil
}

// nameVariants gera as chaves normalizadas a tentar no match exato, na ordem:
// nome completo, nome sem os parênteses, conteúdo de cada parêntese e cada
// segmento separado por "/". Ex.: "Transaminase Oxalacética (TGO)" ->
// ["transaminase oxalacetica tgo", "transaminase oxalacetica", "tgo"].
func nameVariants(raw string) []string {
	var variants []string
	seen := map[string]bool{}
	add := func(s string) {
		if n := dom.Normalize(s); n != "" && !seen[n] {
			seen[n] = true
			variants = append(variants, n)
		}
	}

	add(raw)

	// Nome sem parênteses + conteúdo de cada parêntese.
	var outside, inside strings.Builder
	depth := 0
	var groups []string
	for _, r := range raw {
		switch {
		case r == '(':
			depth++
		case r == ')':
			if depth > 0 {
				depth--
				if depth == 0 {
					groups = append(groups, inside.String())
					inside.Reset()
				}
			}
		case depth > 0:
			inside.WriteRune(r)
		default:
			outside.WriteRune(r)
		}
	}
	add(outside.String())
	for _, g := range groups {
		add(g)
	}

	// Segmentos por barra ("TGO/AST").
	if strings.Contains(raw, "/") {
		for _, part := range strings.Split(raw, "/") {
			add(part)
		}
	}
	return variants
}

func containsMarker(list []*dom.Marker, id uuid.UUID) bool {
	for _, m := range list {
		if m.ID == id {
			return true
		}
	}
	return false
}

func rankCandidates(norm string, pool []dom.Marker) []ResolveCandidate {
	var cands []ResolveCandidate
	for i := range pool {
		m := pool[i]
		best := nameSimilarity(norm, m.NormalizedKey)
		for _, a := range m.Aliases {
			if s := nameSimilarity(norm, a.NormalizedAlias); s > best {
				best = s
			}
		}
		if best >= fuzzyThreshold {
			cands = append(cands, ResolveCandidate{Marker: m, Similarity: best})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].Similarity > cands[j].Similarity })
	if len(cands) > 5 {
		cands = cands[:5]
	}
	return cands
}

// nameSimilarity combina Levenshtein com sobreposição de tokens. Nomes de
// laudo costumam ser o nome do catálogo com palavras a mais ("transaminase
// oxalacetica tgo" vs alias "transaminase oxalacetica") — quase idênticos em
// tokens, mas distantes em Levenshtein puro.
func nameSimilarity(a, b string) float64 {
	best := dom.Similarity(a, b)
	if s := tokenContainment(a, b); s > best {
		best = s
	}
	return best
}

// tokenContainment pontua alto quando todos os tokens do nome menor aparecem
// no maior (0.6 base + proporção de cobertura); caso contrário devolve a
// fração de tokens compartilhados.
func tokenContainment(a, b string) float64 {
	ta, tb := strings.Fields(a), strings.Fields(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	short, long := ta, tb
	if len(tb) < len(ta) {
		short, long = tb, ta
	}
	set := make(map[string]bool, len(long))
	for _, t := range long {
		set[t] = true
	}
	shared := 0
	for _, t := range short {
		if set[t] {
			shared++
		}
	}
	if shared == len(short) {
		return 0.6 + 0.4*float64(len(short))/float64(len(long))
	}
	return float64(shared) / float64(len(long))
}
