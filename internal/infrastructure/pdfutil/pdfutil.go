// Package pdfutil trata PDFs protegidos por senha antes da extração LLM:
// o provedor (Anthropic) rejeita PDF criptografado, então removemos a
// proteção em memória — a senha nunca é persistida.
package pdfutil

import (
	"bytes"
	"errors"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ErrPasswordRequired: o PDF é protegido e nenhuma senha foi informada.
var ErrPasswordRequired = errors.New("pdf protegido por senha")

// ErrWrongPassword: a senha informada não abre o PDF.
var ErrWrongPassword = errors.New("senha do pdf incorreta")

// EnsureDecrypted devolve o conteúdo pronto para o LLM:
//   - PDF sem proteção → conteúdo original (senha ignorada);
//   - PDF protegido + senha correta → conteúdo descriptografado;
//   - PDF protegido sem senha → ErrPasswordRequired;
//   - senha errada ou arquivo corrompido → ErrWrongPassword.
func EnsureDecrypted(content []byte, password string) ([]byte, error) {
	conf := model.NewDefaultConfiguration()
	conf.UserPW = password
	conf.OwnerPW = password
	// Sem validação estrita: PDFs de banco frequentemente têm pequenas
	// não-conformidades e o objetivo aqui é só remover a criptografia.
	conf.ValidationMode = model.ValidationRelaxed

	var out bytes.Buffer
	err := api.Decrypt(bytes.NewReader(content), &out, conf)
	if err == nil {
		return out.Bytes(), nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "not encrypted") {
		return content, nil // sem proteção: segue o original
	}
	if password == "" {
		return nil, ErrPasswordRequired
	}
	return nil, ErrWrongPassword
}
