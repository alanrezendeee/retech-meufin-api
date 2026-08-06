package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Renegotiation registra a repactuação de uma dívida: as cobranças em aberto
// da dívida antiga são encerradas e uma nova série de parcelas nasce no lugar.
//
// É modelada como EVENTO, e não como um ponteiro de um lançamento para outro,
// porque a relação é N→M: dezenas de parcelas previstas mais os residuais de
// pagamentos parciais se consolidam num acordo novo com outra quantidade de
// parcelas. O evento é o que permite, a partir de qualquer parcela nova,
// chegar a todas as origens — e vice-versa —, além de tornar natural o caso
// de renegociar de novo mais adiante (cadeia de eventos).
type Renegotiation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Date        time.Time
	// Description identifica a dívida repactuada (herda a descrição da origem).
	Description string
	// SettledAmountCents é o saldo devedor apurado nas origens: parcelas
	// previstas + residuais em aberto. NÃO inclui parcelas já realizadas.
	SettledAmountCents int64
	// NewAmountCents é o total do novo acordo (parcelas × valor).
	NewAmountCents int64
	// AdjustmentCents = NewAmountCents - SettledAmountCents. Positivo é
	// encargo/juros da renegociação; negativo é desconto obtido.
	AdjustmentCents int64
	// OriginCount / NewCount: quantos lançamentos entraram e quantos nasceram.
	OriginCount int
	NewCount    int
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsDiscount informa se a renegociação resultou em abatimento.
func (r *Renegotiation) IsDiscount() bool { return r.AdjustmentCents < 0 }

func (r *Renegotiation) Validate() error {
	if r.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	r.Description = strings.TrimSpace(r.Description)
	if r.Description == "" {
		return &ValidationError{Msg: "descrição é obrigatória"}
	}
	if r.SettledAmountCents <= 0 {
		return &ValidationError{Msg: "não há saldo em aberto para renegociar"}
	}
	if r.NewAmountCents <= 0 {
		return &ValidationError{Msg: "o valor do novo acordo deve ser maior que zero"}
	}
	if r.OriginCount == 0 {
		return &ValidationError{Msg: "nenhuma cobrança em aberto foi encontrada para renegociar"}
	}
	if r.NewCount < 1 {
		return &ValidationError{Msg: "o novo acordo precisa de ao menos uma parcela"}
	}
	r.AdjustmentCents = r.NewAmountCents - r.SettledAmountCents
	return nil
}

// RenegotiationRepository persiste o evento e aplica a repactuação.
type RenegotiationRepository interface {
	// Apply grava o evento, encerra as origens (canceladas, com motivo
	// renegociacao e vínculo ao evento) e cria as parcelas novas — tudo em
	// uma única transação. Uma renegociação parcialmente aplicada deixaria a
	// dívida duplicada ou sumida.
	Apply(ctx context.Context, r *Renegotiation, originIDs []uuid.UUID, newEntries []*FinancialEntry) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Renegotiation, error)
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Renegotiation, int64, error)
	// ListEntries devolve os lançamentos vinculados ao evento, separados
	// entre origens (canceladas) e as parcelas novas.
	ListEntries(ctx context.Context, workspaceID, renegotiationID uuid.UUID) (origins, created []FinancialEntry, err error)
}
