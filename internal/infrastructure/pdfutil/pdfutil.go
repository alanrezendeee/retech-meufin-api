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

func init() {
	// Uso headless/servidor: sem isso o pdfcpu tenta criar ~/.config/pdfcpu
	// no primeiro NewDefaultConfiguration e dá PANIC em container sem HOME
	// gravável (Railway) — era a causa do 500 ao informar a senha.
	api.DisableConfigDir()
}

// ErrPasswordRequired: o PDF é protegido e nenhuma senha foi informada.
var ErrPasswordRequired = errors.New("pdf protegido por senha")

// ErrWrongPassword: a senha informada não abre o PDF.
var ErrWrongPassword = errors.New("senha do pdf incorreta")

// EnsureDecrypted devolve o conteúdo pronto para o LLM:
//   - PDF sem proteção → conteúdo original (senha ignorada);
//   - PDF protegido + senha correta → conteúdo descriptografado;
//   - PDF protegido sem senha → ErrPasswordRequired;
//   - senha errada ou arquivo corrompido → ErrWrongPassword.
//
// pdfcpu usa panic para alguns erros internos; recuperamos e devolvemos
// erro normal para nunca derrubar a requisição com 500.
func EnsureDecrypted(content []byte, password string) (out []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			out, err = nil, ErrWrongPassword
		}
	}()
	return ensureDecrypted(content, password)
}

func ensureDecrypted(content []byte, password string) ([]byte, error) {
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
