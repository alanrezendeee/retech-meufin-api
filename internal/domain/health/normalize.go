package health

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Normalize gera a chave canônica para dedup e resolução:
// minúsculo, sem acento, sem pontuação, espaços colapsados.
// Ex.: "Vitamina D (25-OH)" -> "vitamina d 25 oh"; "TGO/AST" -> "tgo ast".
func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	// remove acentos (decompõe e descarta marcas combinantes)
	var noAccent strings.Builder
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		noAccent.WriteRune(r)
	}

	// troca tudo que não é letra/dígito por espaço
	var cleaned strings.Builder
	for _, r := range noAccent.String() {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cleaned.WriteRune(r)
		} else {
			cleaned.WriteRune(' ')
		}
	}

	// colapsa espaços
	return strings.Join(strings.Fields(cleaned.String()), " ")
}
