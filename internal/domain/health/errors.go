package health

import "errors"

var (
	ErrNotFound   = errors.New("recurso de saúde não encontrado")
	ErrConflict   = errors.New("conflito de recurso de saúde")
	ErrValidation = errors.New("validação de domínio de saúde")
	ErrDuplicate  = errors.New("marcador duplicado")
	ErrImmutable  = errors.New("recurso do sistema não pode ser alterado pelo tenant")
)

// ValidationError envolve mensagem específica mantendo o sentinel ErrValidation.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }
func (e *ValidationError) Unwrap() error { return ErrValidation }

// DuplicateError carrega o marcador já existente para sugestão na resposta.
type DuplicateError struct {
	Existing *Marker
}

func (e *DuplicateError) Error() string {
	if e.Existing != nil {
		return "marcador já existe: " + e.Existing.CanonicalName
	}
	return "marcador duplicado"
}
func (e *DuplicateError) Unwrap() error { return ErrDuplicate }
