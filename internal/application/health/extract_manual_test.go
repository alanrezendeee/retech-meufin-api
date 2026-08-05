//go:build manual

// Harness manual de validação da extração LLM com laudos reais.
// NÃO roda no CI (build tag manual). Uso:
//
//	EXAMS_DIR=/tmp/exames OUT_DIR=/tmp/exames-out EXTRACTION_API_KEY=sk-... \
//	  go test -tags manual -run TestRealExtraction -timeout 30m -v ./internal/application/health/
//
//	OUT_DIR=/tmp/exames-out go test -tags manual -run TestSeedCoverage -v ./internal/application/health/
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"github.com/retechfin/retechfin-api/internal/infrastructure/extraction"
	"github.com/retechfin/retechfin-api/internal/infrastructure/pdfutil"
)

// TestRealExtraction roda o extractor real (Anthropic) contra todos os PDFs de
// EXAMS_DIR e grava o StructuredJSON de cada um em OUT_DIR/<nome>.json.
func TestRealExtraction(t *testing.T) {
	dir := os.Getenv("EXAMS_DIR")
	outDir := os.Getenv("OUT_DIR")
	apiKey := os.Getenv("EXTRACTION_API_KEY")
	if dir == "" || outDir == "" || apiKey == "" {
		t.Skip("defina EXAMS_DIR, OUT_DIR e EXTRACTION_API_KEY")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ext := extraction.New(extraction.Config{
		Provider: extraction.ProviderAnthropic,
		APIKey:   apiKey,
		Model:    extraction.DefaultModel,
		BaseURL:  extraction.DefaultBaseURL,
	})
	if !ext.Enabled() {
		t.Fatal("extractor desabilitado")
	}

	pdfs, err := filepath.Glob(filepath.Join(dir, "*.pdf"))
	if err != nil || len(pdfs) == 0 {
		t.Fatalf("nenhum PDF em %s (err=%v)", dir, err)
	}
	sort.Strings(pdfs)

	profile := extraction.LabExamProfile()
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	var mu sync.Mutex
	type result struct {
		file, status, note string
		items              int
	}
	var results []result

	for _, p := range pdfs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			name := filepath.Base(path)
			content, err := os.ReadFile(path)
			if err != nil {
				mu.Lock()
				results = append(results, result{name, "ERRO-LEITURA", err.Error(), 0})
				mu.Unlock()
				return
			}
			content, err = pdfutil.EnsureDecrypted(content, "")
			if err != nil {
				mu.Lock()
				results = append(results, result{name, "PDF-PROTEGIDO", err.Error(), 0})
				mu.Unlock()
				return
			}
			res, err := ext.Extract(context.Background(), extraction.ExtractInput{
				InputType: "pdf",
				MimeType:  "application/pdf",
				Content:   content,
				Profile:   &profile,
			})
			if err != nil {
				mu.Lock()
				results = append(results, result{name, "ERRO-EXTRACAO", err.Error(), 0})
				mu.Unlock()
				return
			}
			outPath := filepath.Join(outDir, strings.TrimSuffix(name, ".pdf")+".json")
			_ = os.WriteFile(outPath, res.StructuredJSON, 0o644)
			_ = os.WriteFile(filepath.Join(outDir, strings.TrimSuffix(name, ".pdf")+".raw.json"), res.RawResponse, 0o644)

			var parsed extractedExamJSON
			_ = json.Unmarshal(res.StructuredJSON, &parsed)
			note := fmt.Sprintf("lab=%q data=%q warnings=%d", parsed.LaboratoryName, parsed.ExamDate, len(parsed.Warnings))
			mu.Lock()
			results = append(results, result{name, "OK", note, len(parsed.Exams)})
			mu.Unlock()
		}(p)
	}
	wg.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].file < results[j].file })
	for _, r := range results {
		t.Logf("%-50s %-14s itens=%-3d %s", r.file, r.status, r.items, r.note)
	}
}

// TestSeedCoverage cruza os marcadores extraídos (OUT_DIR/*.json) com o
// catálogo seed (systemMarkerSeeds), simulando o Resolve (Normalize +
// Similarity, mesmo threshold 0.55). Lista o que casaria e o que ficaria
// sem match — insumo para expandir o seed global.
func TestSeedCoverage(t *testing.T) {
	outDir := os.Getenv("OUT_DIR")
	if outDir == "" {
		t.Skip("defina OUT_DIR")
	}
	files, _ := filepath.Glob(filepath.Join(outDir, "*.json"))
	if len(files) == 0 {
		t.Fatalf("nenhum JSON em %s", outDir)
	}

	// Índice do seed: normalized -> canonical (nome + aliases).
	type seedRef struct{ canonical, key string }
	var refs []seedRef
	exact := map[string]string{}
	for _, sd := range systemMarkerSeeds() {
		k := dom.Normalize(sd.name)
		exact[k] = sd.name
		refs = append(refs, seedRef{sd.name, k})
		for _, a := range sd.aliases {
			ak := dom.Normalize(a)
			exact[ak] = sd.name
			refs = append(refs, seedRef{sd.name, ak})
		}
	}

	// raw name -> ocorrências/unidades/datas
	type seen struct {
		count int
		units map[string]int
		files map[string]bool
	}
	all := map[string]*seen{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var parsed extractedExamJSON
		if err := json.Unmarshal(b, &parsed); err != nil {
			t.Errorf("%s: JSON inválido: %v", filepath.Base(f), err)
			continue
		}
		for _, e := range parsed.Exams {
			name := strings.TrimSpace(e.MarkerName)
			if name == "" {
				name = strings.TrimSpace(e.ExamName)
			}
			if name == "" {
				continue
			}
			key := dom.Normalize(name)
			s := all[key]
			if s == nil {
				s = &seen{units: map[string]int{}, files: map[string]bool{}}
				all[key] = s
			}
			s.count++
			if u := strings.TrimSpace(e.Unit); u != "" {
				s.units[u]++
			}
			s.files[filepath.Base(f)] = true
			// guarda um nome de exibição
			if _, ok := displayNames[key]; !ok {
				displayNames[key] = name
			}
		}
	}

	var matched, fuzzy, missing []string
	for key, s := range all {
		name := displayNames[key]
		unitList := unitSummary(s.units)
		if canonical, ok := exact[key]; ok {
			matched = append(matched, fmt.Sprintf("%-40s → %-30s (%dx) %s", name, canonical, s.count, unitList))
			continue
		}
		best, bestScore := "", 0.0
		for _, r := range refs {
			if sc := dom.Similarity(key, r.key); sc > bestScore {
				bestScore, best = sc, r.canonical
			}
		}
		if bestScore >= 0.55 {
			fuzzy = append(fuzzy, fmt.Sprintf("%-40s ~ %-30s (%.2f, %dx) %s", name, best, bestScore, s.count, unitList))
		} else {
			missing = append(missing, fmt.Sprintf("%-40s (%dx em %d arq) %s", name, s.count, len(s.files), unitList))
		}
	}
	sort.Strings(matched)
	sort.Strings(fuzzy)
	sort.Strings(missing)

	t.Logf("\n===== MATCH EXATO (seed cobre) — %d =====", len(matched))
	for _, m := range matched {
		t.Log(m)
	}
	t.Logf("\n===== FUZZY (ambiguous na UI) — %d =====", len(fuzzy))
	for _, m := range fuzzy {
		t.Log(m)
	}
	t.Logf("\n===== SEM MATCH (candidatos a seed novo) — %d =====", len(missing))
	for _, m := range missing {
		t.Log(m)
	}
}

var displayNames = map[string]string{}

func unitSummary(units map[string]int) string {
	if len(units) == 0 {
		return ""
	}
	var parts []string
	for u, n := range units {
		parts = append(parts, fmt.Sprintf("%s×%d", u, n))
	}
	sort.Strings(parts)
	return "[" + strings.Join(parts, " ") + "]"
}
