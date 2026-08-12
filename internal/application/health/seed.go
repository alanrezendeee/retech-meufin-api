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

// seedDefaultRef é a faixa de CURADORIA do catálogo, por nome canônico.
// Aplica-se a marcadores que os laboratórios reportam sem referência (ex.:
// VLDL) e serve de fallback de interpretação quando o item do laudo não traz
// faixa própria — nunca substitui a faixa impressa no laudo.
type seedDefaultRef struct {
	min   *float64
	max   *float64
	text  string
	tiers []dom.RefTier
}

func systemMarkerDefaultRefs() map[string]seedDefaultRef {
	f := func(v float64) *float64 { return &v }
	return map[string]seedDefaultRef{
		"Colesterol VLDL": {
			max:  f(30),
			text: "Desejável: inferior a 30 mg/dL (valor de literatura; laboratórios geralmente não informam faixa)",
		},
		// LDL não tem faixa única: metas por risco cardiovascular (diretriz
		// SBC), estimado pelo médico — tiers informativos, sem interpretação
		// automática.
		"Colesterol LDL": {
			text: "Metas por categoria de risco cardiovascular estimada pelo médico (diretriz SBC); crianças e adolescentes: inferior a 110 mg/dL",
			tiers: []dom.RefTier{
				{Key: dom.RiskLow, Label: "Risco baixo", Max: f(130)},
				{Key: dom.RiskIntermediate, Label: "Risco intermediário", Max: f(100)},
				{Key: dom.RiskHigh, Label: "Risco alto", Max: f(70)},
				{Key: dom.RiskVeryHigh, Label: "Risco muito alto", Max: f(50)},
			},
		},
	}
}

// systemMarkerSeeds é o catálogo base (escopo system) de marcadores laboratoriais BR comuns.
// comparability: standardized = valor comparável entre labs; method_dependent = varia com o
// método; qualitative = resultado descritivo (positivo/negativo, presença/ausência).
//
// Os aliases refletem as grafias reais impressas por laboratórios brasileiros
// (Santa Luzia/DASA, Diagnóstico, Hoffmann e afins) — inclusive as abreviações
// com pontos ("C.H.C.M", "V.C.M") e as formas por extenso entre parênteses.
// Cobrir bem os aliases é o que evita duplicata no catálogo do usuário e o que
// impede match fuzzy perigoso (ex.: "Colesterol VLDL" tem 0,93 de similaridade
// com "Colesterol LDL" — só um match EXATO separa os dois com segurança).
func systemMarkerSeeds() []seedMarker {
	std := dom.ComparabilityStandardized
	mdp := dom.ComparabilityMethodDependent
	qlt := dom.ComparabilityQualitative
	return []seedMarker{
		// --- bioquímica geral ---
		{"Glicose", "bioquimica", "mg/dL", std, []string{"Glicemia", "Glicemia de jejum", "GLU", "Glicose de jejum"}},
		{"Hemoglobina glicada", "bioquimica", "%", std, []string{"HbA1c", "A1C", "Hemoglobina glicosilada", "Hemoglobina glicada (A1C)", "Hemoglobina glicada - HbA1c"}},
		{"Glicose média estimada", "bioquimica", "mg/dL", std, []string{"GME", "Glicose média estimada (GME)", "Glicemia média estimada", "Média estimada de glicose"}},
		{"Ácido úrico", "bioquimica", "mg/dL", std, []string{"Urato"}},
		{"Albumina", "bioquimica", "g/dL", std, nil},
		{"Proteínas totais", "bioquimica", "g/dL", std, []string{"Proteína total", "Proteínas totais e frações"}},
		{"Amilase", "bioquimica", "U/L", std, []string{"Amilase sérica"}},
		{"Lipase", "bioquimica", "U/L", std, nil},
		{"CPK", "bioquimica", "U/L", std, []string{"Creatinofosfoquinase", "Creatinofosfoquinase (CPK)", "CK", "Creatinoquinase", "CK total"}},
		{"Ferro sérico", "bioquimica", "µg/dL", std, []string{"Ferro", "Fe"}},
		{"Ferritina", "bioquimica", "ng/mL", mdp, []string{"FER"}},
		{"Homocisteína", "bioquimica", "µmol/L", mdp, nil},
		{"Zinco", "bioquimica", "µg/dL", mdp, []string{"Zinco sanguíneo", "Zinco sérico", "Zn"}},

		// --- renal ---
		{"Creatinina", "renal", "mg/dL", std, []string{"CREA"}},
		{"Ureia", "renal", "mg/dL", std, []string{"URE"}},
		{"Ritmo de filtração glomerular estimado", "renal", "mL/min/1,73m²", mdp, []string{
			"RFGe", "RFG", "eGFR", "TFG", "RFG (CKD-EPI)", "RFG estimado (CKD-EPI)",
			"Ritmo de filtração glomerular estimado (RFGe)", "Taxa de filtração glomerular",
			"Filtração glomerular estimada", "CKD-EPI",
			// Laboratórios imprimem uma linha por variante da fórmula; todas
			// medem a mesma coisa e devem cair no mesmo marcador.
			"RFG estimado (CKD-EPI) - Caucasiano", "RFG estimado (CKD-EPI) - Afrodescendente",
			"Ritmo de filtração glomerular estimado (RFGe) - Homens",
			"Ritmo de filtração glomerular estimado (RFGe) - Mulheres",
		}},

		// --- lipídico ---
		{"Colesterol total", "lipidico", "mg/dL", std, []string{"Colesterol"}},
		{"Colesterol HDL", "lipidico", "mg/dL", std, []string{"HDL", "HDL-c", "HDL - Colesterol"}},
		{"Colesterol LDL", "lipidico", "mg/dL", std, []string{"LDL", "LDL-c", "LDL - Colesterol", "LDL - Colesterol (calculado)"}},
		{"Colesterol VLDL", "lipidico", "mg/dL", std, []string{"VLDL", "VLDL - Colesterol"}},
		{"Colesterol não-HDL", "lipidico", "mg/dL", std, []string{"Não-HDL"}},
		{"Triglicerídeos", "lipidico", "mg/dL", std, []string{"Triglicérides", "TG"}},
		{"Apolipoproteína B", "lipidico", "mg/dL", mdp, []string{"Apo B", "ApoB"}},
		{"Apolipoproteína A1", "lipidico", "mg/dL", mdp, []string{"Apo A1", "ApoA1"}},
		{"Lipoproteína (a)", "lipidico", "mg/dL", mdp, []string{"Lp(a)"}},

		// --- hepático ---
		{"AST (TGO)", "hepatico", "U/L", std, []string{
			"TGO", "AST", "Aspartato aminotransferase", "Transaminase oxalacética", "TGO (AST)",
			"AST (TGO) - Aspartato aminotransferase", "TGO / AST - Transaminase Oxalacética / Aspartato Aminotransferase",
		}},
		{"ALT (TGP)", "hepatico", "U/L", std, []string{
			"TGP", "ALT", "Alanina aminotransferase", "Transaminase pirúvica", "TGP (ALT)",
			"ALT (TGP) - Alanina aminotransferase", "TGP / ALT - Transaminase Pirúvica / Alanina Aminotransferase",
		}},
		{"Gama GT", "hepatico", "U/L", std, []string{"GGT", "Gama glutamil transferase", "Gama-glutamil transferase (GGT)"}},
		{"Fosfatase alcalina", "hepatico", "U/L", std, []string{"FA", "ALP"}},
		{"Bilirrubina total", "hepatico", "mg/dL", std, []string{"BT", "Bilirrubinas", "Bilirrubina"}},
		{"Bilirrubina direta", "hepatico", "mg/dL", std, []string{"BD"}},
		{"Bilirrubina indireta", "hepatico", "mg/dL", std, []string{"BI"}},

		// --- inflamação ---
		{"Proteína C reativa", "inflamacao", "mg/L", std, []string{
			"PCR", "CRP", "Proteína C reativa (PCR)", "PCR ultrassensível", "PCR-us",
			"Proteína C reativa (PCR-us)", "Proteína C reativa ultrassensível",
			"PCR - Proteína C reativa alta sensibilidade", "Proteína C reativa (PCR) alta sensibilidade",
			"Proteína C reativa de alta sensibilidade",
		}},
		{"VHS", "inflamacao", "mm/h", mdp, []string{"Velocidade de hemossedimentação", "VHS - Primeira hora", "Hemossedimentação"}},

		// --- hematologia: série vermelha ---
		{"Hemoglobina", "hematologia", "g/dL", std, []string{"Hb"}},
		{"Hematócrito", "hematologia", "%", std, []string{"Ht", "HCT"}},
		{"Hemácias", "hematologia", "milhões/mm³", std, []string{"Eritrócitos", "Glóbulos vermelhos", "RBC"}},
		{"VCM", "hematologia", "fL", std, []string{
			"Volume corpuscular médio", "MCV", "V.C.M", "VGM", "Volume globular médio",
			"Volume globular médio (VGM)", "Vol. glob. média (VCM)",
		}},
		{"HCM", "hematologia", "pg", std, []string{
			"Hemoglobina corpuscular média", "MCH", "H.C.M", "Hemoglobina globular média",
			"Hemoglobina globular média (HCM)", "Hem. glob. média (HCM)",
		}},
		{"CHCM", "hematologia", "g/dL", std, []string{
			"Concentração de hemoglobina corpuscular média", "MCHC", "C.H.C.M",
			"Concentração de hemoglobina globular média", "Concentração de hemoglobina globular média (CHCM)",
			"C.H glob média (CHCM)",
		}},
		{"RDW", "hematologia", "%", std, []string{"Índice de anisocitose", "RDW-CV", "Amplitude de distribuição dos eritrócitos"}},
		{"Anisocitose", "hematologia", "", qlt, []string{"Anisocitose (observação)"}},
		{"Poiquilocitose", "hematologia", "", qlt, nil},

		// --- hematologia: série branca ---
		{"Leucócitos", "hematologia", "/mm³", std, []string{"Glóbulos brancos", "WBC", "Leucócitos totais"}},
		{"Neutrófilos", "hematologia", "/mm³", std, []string{"Segmentados", "Neutrófilos segmentados"}},
		{"Bastonetes", "hematologia", "/mm³", std, []string{"Bastões", "Neutrófilos bastonetes"}},
		{"Linfócitos", "hematologia", "/mm³", std, nil},
		{"Monócitos", "hematologia", "/mm³", std, nil},
		{"Eosinófilos", "hematologia", "/mm³", std, nil},
		{"Basófilos", "hematologia", "/mm³", std, nil},

		// --- hematologia: série branca, escala percentual ---
		// O diferencial do hemograma imprime % e absoluto; são marcadores
		// distintos (unidade e faixa próprias). "(percentual)" no nome porque
		// "%" é descartado pelo Normalize e colidiria com o absoluto.
		{"Neutrófilos (percentual)", "hematologia", "%", std, []string{
			"Segmentados percentual", "Percentual de neutrófilos",
		}},
		{"Bastonetes (percentual)", "hematologia", "%", std, []string{"Bastões percentual"}},
		{"Linfócitos (percentual)", "hematologia", "%", std, []string{"Percentual de linfócitos"}},
		{"Monócitos (percentual)", "hematologia", "%", std, []string{"Percentual de monócitos"}},
		{"Eosinófilos (percentual)", "hematologia", "%", std, []string{"Percentual de eosinófilos"}},
		{"Basófilos (percentual)", "hematologia", "%", std, []string{"Percentual de basófilos"}},

		// --- hematologia: plaquetas ---
		{"Plaquetas", "hematologia", "/mm³", std, []string{"PLT", "Contagem de plaquetas"}},
		{"Volume plaquetário médio", "hematologia", "fL", std, []string{"VPM", "VMP", "MPV", "M.P.V.", "VMP (Volume plaquetário médio)"}},

		// --- coagulação ---
		{"Tempo de protrombina", "coagulacao", "segundos", mdp, []string{"TP", "TAP", "Tempo de atividade da protrombina"}},
		{"Atividade de protrombina", "coagulacao", "%", mdp, []string{"AP", "Atividade protrombínica"}},
		{"RNI", "coagulacao", "", std, []string{"INR", "R.N.I.", "Razão normatizada internacional"}},
		{"TTPA", "coagulacao", "segundos", mdp, []string{"Tempo de tromboplastina parcial ativada", "KTTP"}},
		{"Dímero-D", "coagulacao", "ng/mL FEU", mdp, []string{"Dímero-D quantitativo", "D-dímero", "DD"}},

		// --- eletrólitos e minerais ---
		{"Sódio", "eletrolitos", "mEq/L", std, []string{"Na"}},
		{"Potássio", "eletrolitos", "mEq/L", std, []string{"K"}},
		{"Cálcio", "eletrolitos", "mg/dL", std, []string{"Ca", "Cálcio total"}},
		{"Magnésio", "eletrolitos", "mg/dL", std, []string{"Mg"}},
		{"Fósforo", "eletrolitos", "mg/dL", std, []string{"P", "Fosfato", "Fósforo inorgânico"}},

		// --- hormônios: tireoide ---
		{"TSH", "hormonios", "µUI/mL", mdp, []string{
			"Hormônio tireoestimulante", "Tirotrofina", "TSH ultrassensível", "TSH ultra sensível",
			"TSH us", "Hormônio tireoestimulante (TSH)",
		}},
		{"T4 livre", "hormonios", "ng/dL", mdp, []string{"T4L", "Tiroxina livre", "Free T4", "T4 livre (tiroxina livre)"}},
		{"T4 total", "hormonios", "µg/dL", mdp, []string{"Tiroxina total", "Tiroxina"}},
		{"T3 livre", "hormonios", "pg/mL", mdp, []string{"T3L", "Triiodotironina livre", "Free T3"}},
		{"T3 total", "hormonios", "ng/dL", mdp, []string{"Triiodotironina", "T3"}},
		{"Anti-TPO", "hormonios", "UI/mL", mdp, []string{
			"Anticorpos anti-TPO", "Anti-tireoperoxidase", "Anticorpo antitireoperoxidase", "ATPO",
		}},
		{"Anti-tireoglobulina", "hormonios", "UI/mL", mdp, []string{"Anti-Tg", "Anticorpos anti-tireoglobulina"}},

		// --- hormônios: sexuais e metabólicos ---
		{"Insulina", "hormonios", "µUI/mL", mdp, []string{"Insulina de jejum"}},
		{"FSH", "hormonios", "mUI/mL", mdp, []string{"Hormônio folículo estimulante", "Folitropina"}},
		{"LH", "hormonios", "mUI/mL", mdp, []string{"Hormônio luteinizante", "Lutropina"}},
		{"Prolactina", "hormonios", "ng/mL", mdp, []string{"PRL"}},
		{"Estradiol", "hormonios", "pg/mL", mdp, []string{"E2", "Estradiol 17 beta"}},
		{"Progesterona", "hormonios", "ng/mL", mdp, nil},
		{"Testosterona total", "hormonios", "ng/dL", mdp, []string{"Testosterona"}},
		{"Testosterona livre", "hormonios", "ng/dL", mdp, []string{"Testosterona livre calculada"}},
		{"SHBG", "hormonios", "nmol/L", mdp, []string{
			"Globulina ligadora de hormônios sexuais", "SHBG - Globulina ligadora de hormônios sexuais",
			"Globulina ligadora de hormônios sexuais - SHBG", "Globulina carreadora de hormônios sexuais",
		}},
		{"DHEA-S", "hormonios", "µg/dL", mdp, []string{
			"SDHEA", "Sulfato de dehidroepiandrosterona", "Sulfato de dehidroepiandrosterona (SDHEA)",
			"Sulfato de deidroepiandrosterona", "DHEAS",
		}},
		{"Dihidrotestosterona", "hormonios", "pg/mL", mdp, []string{"DHT", "Dihidrotestosterona (DHT)", "Di-hidrotestosterona"}},
		{"Cortisol", "hormonios", "µg/dL", mdp, []string{"Cortisol sérico", "Cortisol matinal"}},
		{"Paratormônio", "hormonios", "pg/mL", mdp, []string{"PTH", "PTH intacto"}},

		// --- marcadores tumorais ---
		{"PSA total", "tumoral", "ng/mL", mdp, []string{"PSA", "Antígeno prostático específico"}},
		{"PSA livre", "tumoral", "ng/mL", mdp, []string{"PSA livre (free PSA)"}},
		{"Relação PSA livre/total", "tumoral", "%", mdp, []string{"Relação PSA livre/PSA total", "PSA livre/total", "Percentual de PSA livre"}},

		// --- vitaminas ---
		{"Vitamina D", "vitaminas", "ng/mL", mdp, []string{
			"25-OH vitamina D", "25 hidroxivitamina D", "Vitamina D 25-hidroxi", "Calcidiol",
		}},
		{"Vitamina B12", "vitaminas", "pg/mL", mdp, []string{"B12", "Cobalamina", "Vitamina B-12"}},
		{"Vitamina B6", "vitaminas", "µg/L", mdp, []string{"B6", "Piridoxina", "Vitamina B-6"}},
		{"Vitamina C", "vitaminas", "mg/dL", mdp, []string{"Ácido ascórbico"}},
		{"Vitamina A", "vitaminas", "µg/dL", mdp, []string{"Retinol"}},
		{"Ácido fólico", "vitaminas", "ng/mL", mdp, []string{"Folato", "Vitamina B9"}},

		// --- urina (EAS / urina tipo I): a maioria é qualitativa ---
		{"pH urinário", "urina", "", std, []string{"pH", "pH da urina"}},
		{"Densidade urinária", "urina", "", std, []string{"Densidade", "Densidade da urina"}},
		{"Aspecto da urina", "urina", "", qlt, []string{"Aspecto"}},
		{"Cor da urina", "urina", "", qlt, []string{"Cor"}},
		{"Proteínas na urina", "urina", "", qlt, []string{"Proteínas", "Proteína", "Proteinúria", "Pesquisa de proteínas"}},
		{"Glicose na urina", "urina", "", qlt, []string{"Glicosúria", "Pesquisa de glicose"}},
		{"Corpos cetônicos", "urina", "", qlt, []string{"Cetonas", "Corpos cetônicos, pesquisa", "Cetonúria"}},
		{"Nitrito", "urina", "", qlt, []string{"Nitritos", "Pesquisa de nitrito"}},
		{"Urobilinogênio", "urina", "mg/dL", qlt, nil},
		{"Pigmentos biliares", "urina", "", qlt, []string{"Bilirrubina urinária", "Pesquisa de pigmentos biliares"}},
		{"Leucócitos na urina", "urina", "/µL", std, []string{"Leucócitos (sedimento)", "Piócitos", "Leucócitos no sedimento"}},
		{"Hemácias na urina", "urina", "/µL", std, []string{"Hemácias (sedimento)", "Eritrócitos na urina", "Hematúria"}},
		{"Células epiteliais", "urina", "", qlt, []string{"Células epiteliais escamosas", "Células epiteliais não escamosas"}},
		{"Cilindros", "urina", "", qlt, []string{"Cilindrúria"}},
		{"Cristais", "urina", "", qlt, []string{"Cristalúria"}},
		{"Bactérias na urina", "urina", "", qlt, []string{"Bactérias", "Bacteriúria", "Flora bacteriana"}},
		{"Filamentos de muco", "urina", "", qlt, []string{"Muco", "Filamento de muco"}},
		{"Leveduras", "urina", "", qlt, nil},
		{"Sangue na urina", "urina", "", qlt, []string{"Sangue", "Hemoglobina na urina", "Pesquisa de sangue", "Hemoglobinúria"}},
		{"Urocultura", "urina", "", qlt, []string{"Cultura de urina", "Urocultura com antibiograma"}},

		// --- sorologias (resultado qualitativo: reagente / não reagente) ---
		{"Anti-HIV", "sorologia", "", qlt, []string{"Anti HIV 1/2", "HIV 1/2", "Anti HIV 1/2 - Índice", "Sorologia HIV"}},
		{"Anti-HCV", "sorologia", "", qlt, []string{"Anti-HCV (Índice)", "Hepatite C", "Sorologia hepatite C"}},
		{"HBsAg", "sorologia", "", qlt, []string{"HBsAg (Índice)", "Antígeno de superfície da hepatite B", "Austrália"}},
		{"Anti-HBs", "sorologia", "mUI/mL", mdp, []string{"Anticorpo anti-HBs"}},
		{"Sífilis (anti-T. pallidum)", "sorologia", "", qlt, []string{
			"Anti-T.pallidum", "Anti-T.pallidum (anticorpos totais específicos)",
			"Sífilis - anticorpos totais específicos anti-T.pallidum", "VDRL", "Treponema pallidum",
		}},
		{"Chlamydia trachomatis", "sorologia", "", qlt, []string{"Clamídia", "Chlamydia"}},
		{"Mycoplasma hominis", "sorologia", "", qlt, []string{"Micoplasma hominis"}},
		{"Ureaplasma", "sorologia", "", qlt, []string{"Ureaplasma spp.", "Ureaplasma urealyticum"}},

		// --- espermograma ---
		{"Volume seminal", "espermograma", "mL", std, []string{"Volume do ejaculado", "Volume do sêmen"}},
		{"Concentração de espermatozoides", "espermograma", "milhões/mL", std, []string{
			"Concentração total de espermatozóides", "Concentração espermática", "Contagem de espermatozoides",
		}},
		{"Motilidade progressiva", "espermograma", "%", std, []string{
			"Motilidade progressiva (a+b)", "Espermatozóides móveis (a+b+c)",
			"Progressão linear rápida (a)", "Progressão linear lenta (b)",
		}},
		{"Motilidade não progressiva", "espermograma", "%", std, []string{"Motilidade não progressiva (c)"}},
		{"Espermatozoides imóveis", "espermograma", "%", std, []string{"Imóveis (d)"}},
		{"Morfologia espermática", "espermograma", "%", mdp, []string{
			"Espermatozóides normais ou típicos", "Formas típicas", "Observação morfológica",
		}},
		{"Tempo de liquefação", "espermograma", "min", std, []string{"Liquefação"}},
		{"Viscosidade seminal", "espermograma", "", qlt, []string{"Viscosidade"}},
		{"Período de abstinência", "espermograma", "dias", std, []string{"Abstinência sexual"}},
	}
}

// SeedSystem popula o catálogo base de forma idempotente. Retorna quantos foram inseridos.
func (s *MarkerService) SeedSystem(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	inserted := 0
	defaults := systemMarkerDefaultRefs()
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
		if ref, ok := defaults[sd.name]; ok {
			m.DefaultRefMin = ref.min
			m.DefaultRefMax = ref.max
			m.DefaultRefTiers = ref.tiers
			if ref.text != "" {
				t := ref.text
				m.DefaultRefText = &t
			}
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
