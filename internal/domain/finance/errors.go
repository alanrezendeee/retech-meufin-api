package finance

import "errors"

var (
	ErrNotFound   = errors.New("recurso financeiro não encontrado")
	ErrConflict   = errors.New("conflito de recurso financeiro")
	ErrValidation = errors.New("validação de domínio financeiro")
)

// ValidationError envolve mensagem específica mantendo o sentinel ErrValidation.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }
func (e *ValidationError) Unwrap() error { return ErrValidation }
