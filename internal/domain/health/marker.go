package health

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxMarkerNameLen = 255

// Scope define se o marcador é do catálogo base (system) ou custom do tenant.
type Scope string

const (
	ScopeSystem Scope = "system"
	ScopeTenant Scope = "tenant"
)

// ComparabilityClass indica se o valor do marcador é comparável entre laboratórios.
// Dirige o modo default do gráfico de evolução (ver docs/modulo-saude-familiar-plan.md).
type ComparabilityClass string

const (
	// Padronizado/harmonizado: valor comparável entre labs (ex.: hematócrito, glicose).
	ComparabilityStandardized ComparabilityClass = "standardized"
	// Dependente de método: valor cru varia com o método (ex.: ferritina, vitamina D).
	ComparabilityMethodDependent ComparabilityClass = "method_dependent"
	// Qualitativo: sem valor numérico contínuo (ex.: positivo/negativo).
	ComparabilityQualitative ComparabilityClass = "qualitative"
)

// Marker é a identidade canônica de um analito. Resultados apontam para ele por FK,
// garantindo histórico/evolução confiável e dedup sério.
type Marker struct {
	ID             uuid.UUID
	Scope          Scope
	WorkspaceID    *uuid.UUID // nil quando system
	CanonicalName  string
	NormalizedKey  string
	LoincCode      *string
	Category       string
	Comparability  ComparabilityClass
	CanonicalUnit  *string
	DefaultRefMin  *float64
	DefaultRefMax  *float64
	DefaultRefText *string
	Active         bool
	Aliases        []MarkerAlias
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MarkerAlias mapeia sinônimos/abreviações para o marcador canônico (TGO->AST etc.).
type MarkerAlias struct {
	ID              uuid.UUID
	MarkerID        uuid.UUID
	Scope           Scope
	WorkspaceID     *uuid.UUID
	Alias           string
	NormalizedAlias string
	Source          *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Validate normaliza campos e valida invariantes. Preenche NormalizedKey.
func (m *Marker) Validate() error {
	if m.Scope != ScopeSystem && m.Scope != ScopeTenant {
		return &ValidationError{Msg: "scope inválido"}
	}
	if m.Scope == ScopeTenant && (m.WorkspaceID == nil || *m.WorkspaceID == uuid.Nil) {
		return &ValidationError{Msg: "workspace_id é obrigatório para marcador do tenant"}
	}
	if m.Scope == ScopeSystem && m.WorkspaceID != nil {
		return &ValidationError{Msg: "marcador do sistema não deve ter workspace_id"}
	}

	name := strings.TrimSpace(m.CanonicalName)
	if name == "" {
		return &ValidationError{Msg: "nome do marcador é obrigatório"}
	}
	if len(name) > maxMarkerNameLen {
		return &ValidationError{Msg: "nome do marcador excede o tamanho máximo"}
	}
	m.CanonicalName = name
	m.NormalizedKey = Normalize(name)
	if m.NormalizedKey == "" {
		return &ValidationError{Msg: "nome do marcador inválido após normalização"}
	}

	switch m.Comparability {
	case ComparabilityStandardized, ComparabilityMethodDependent, ComparabilityQualitative:
	case "":
		m.Comparability = ComparabilityStandardized
	default:
		return &ValidationError{Msg: "comparability_class inválido"}
	}

	cat := strings.TrimSpace(strings.ToLower(m.Category))
	if cat == "" {
		return &ValidationError{Msg: "categoria é obrigatória"}
	}
	m.Category = cat

	if m.DefaultRefMin != nil && m.DefaultRefMax != nil && *m.DefaultRefMin > *m.DefaultRefMax {
		return &ValidationError{Msg: "referência mínima não pode ser maior que a máxima"}
	}

	for i := range m.Aliases {
		a := strings.TrimSpace(m.Aliases[i].Alias)
		if a == "" {
			return &ValidationError{Msg: "alias vazio"}
		}
		m.Aliases[i].Alias = a
		m.Aliases[i].NormalizedAlias = Normalize(a)
		if m.Aliases[i].NormalizedAlias == "" {
			return &ValidationError{Msg: "alias inválido após normalização"}
		}
	}
	return nil
}

// IsSystem indica marcador do catálogo base (imutável para o tenant).
func (m *Marker) IsSystem() bool { return m.Scope == ScopeSystem }
