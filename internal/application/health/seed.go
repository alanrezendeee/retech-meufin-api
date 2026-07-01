package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type seedMarker struct {
	name          string
	category      string
	unit          string
	comparability dom.ComparabilityClass
	aliases       []string
}

// systemMarkerSeeds é o catálogo base (escopo system) de marcadores laboratoriais BR comuns.
// comparability: standardized = valor comparável entre labs; method_dependent = varia com o método.
func systemMarkerSeeds() []seedMarker {
	std := dom.ComparabilityStandardized
	mdp := dom.ComparabilityMethodDependent
	return []seedMarker{
		{"Glicose", "bioquimica", "mg/dL", std, []string{"Glicemia", "Glicemia de jejum", "GLU"}},
		{"Hemoglobina glicada", "bioquimica", "%", std, []string{"HbA1c", "A1C", "Hemoglobina glicosilada"}},
		{"Creatinina", "renal", "mg/dL", std, []string{"CREA"}},
		{"Ureia", "renal", "mg/dL", std, []string{"URE"}},
		{"Ácido úrico", "bioquimica", "mg/dL", std, []string{"Urato"}},
		{"Colesterol total", "lipidico", "mg/dL", std, []string{"Colesterol"}},
		{"Colesterol HDL", "lipidico", "mg/dL", std, []string{"HDL", "HDL-c"}},
		{"Colesterol LDL", "lipidico", "mg/dL", std, []string{"LDL", "LDL-c"}},
		{"Triglicerídeos", "lipidico", "mg/dL", std, []string{"Triglicérides", "TG"}},
		{"AST (TGO)", "hepatico", "U/L", std, []string{"TGO", "AST", "Aspartato aminotransferase", "Transaminase oxalacética"}},
		{"ALT (TGP)", "hepatico", "U/L", std, []string{"TGP", "ALT", "Alanina aminotransferase", "Transaminase pirúvica"}},
		{"Gama GT", "hepatico", "U/L", std, []string{"GGT", "Gama glutamil transferase"}},
		{"Fosfatase alcalina", "hepatico", "U/L", std, []string{"FA", "ALP"}},
		{"Bilirrubina total", "hepatico", "mg/dL", std, []string{"BT"}},
		{"Bilirrubina direta", "hepatico", "mg/dL", std, []string{"BD"}},
		{"Bilirrubina indireta", "hepatico", "mg/dL", std, []string{"BI"}},
		{"Albumina", "bioquimica", "g/dL", std, nil},
		{"Proteína C reativa", "inflamacao", "mg/L", std, []string{"PCR", "CRP"}},
		{"Hemoglobina", "hematologia", "g/dL", std, []string{"Hb"}},
		{"Hematócrito", "hematologia", "%", std, []string{"Ht", "HCT"}},
		{"Leucócitos", "hematologia", "/mm³", std, []string{"Glóbulos brancos", "WBC"}},
		{"Plaquetas", "hematologia", "/mm³", std, []string{"PLT"}},
		{"Hemácias", "hematologia", "milhões/mm³", std, []string{"Eritrócitos", "Glóbulos vermelhos", "RBC"}},
		{"VCM", "hematologia", "fL", std, []string{"Volume corpuscular médio", "MCV"}},
		{"Sódio", "eletrolitos", "mEq/L", std, []string{"Na"}},
		{"Potássio", "eletrolitos", "mEq/L", std, []string{"K"}},
		{"Cálcio", "eletrolitos", "mg/dL", std, []string{"Ca"}},
		{"Ferro sérico", "bioquimica", "µg/dL", std, []string{"Ferro", "Fe"}},
		{"TSH", "hormonios", "µUI/mL", mdp, []string{"Hormônio tireoestimulante", "Tirotrofina"}},
		{"T4 livre", "hormonios", "ng/dL", mdp, []string{"T4L", "Tiroxina livre", "Free T4"}},
		{"Insulina", "hormonios", "µUI/mL", mdp, []string{"Insulina de jejum"}},
		{"Ferritina", "bioquimica", "ng/mL", mdp, []string{"FER"}},
		{"Vitamina D", "vitaminas", "ng/mL", mdp, []string{"25-OH vitamina D", "25 hidroxivitamina D"}},
		{"Vitamina B12", "vitaminas", "pg/mL", mdp, []string{"B12", "Cobalamina"}},
		{"Homocisteína", "bioquimica", "µmol/L", mdp, nil},
	}
}

// SeedSystem popula o catálogo base de forma idempotente. Retorna quantos foram inseridos.
func (s *MarkerService) SeedSystem(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	inserted := 0
	for _, sd := range systemMarkerSeeds() {
		unit := sd.unit
		m := &dom.Marker{
			ID:            uuid.New(),
			Scope:         dom.ScopeSystem,
			CanonicalName: sd.name,
			Category:      sd.category,
			Comparability: sd.comparability,
			CanonicalUnit: &unit,
			Active:        true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		src := "seed"
		for _, a := range sd.aliases {
			m.Aliases = append(m.Aliases, dom.MarkerAlias{
				ID:        uuid.New(),
				MarkerID:  m.ID,
				Scope:     dom.ScopeSystem,
				Alias:     a,
				Source:    &src,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		if err := m.Validate(); err != nil {
			return inserted, err
		}
		ok, err := s.repo.UpsertSystem(ctx, m)
		if err != nil {
			return inserted, err
		}
		if ok {
			inserted++
		}
	}
	return inserted, nil
}
