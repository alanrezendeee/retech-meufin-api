package ledger

import "errors"

var (
	ErrNotFound              = errors.New("recurso não encontrado")
	ErrConflict              = errors.New("conflito de recurso")
	ErrValidation            = errors.New("validação de domínio")
	ErrCategoryKindMismatch  = errors.New("categoria incompatível com o fluxo do lançamento")
)

// ValidationError envolve mensagem específica mantendo sentinel ErrValidation.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }
func (e *ValidationError) Unwrap() error { return ErrValidation }
