package budget

import "errors"

var (
	ErrNotFound   = errors.New("orçamento não encontrado")
	ErrConflict   = errors.New("orçamento em conflito")
	ErrValidation = errors.New("validação de orçamento")
)

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }
func (e *ValidationError) Unwrap() error { return ErrValidation }
