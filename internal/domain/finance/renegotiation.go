package finance

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RenegotiationKind é o desfecho de um evento de dívida.
type RenegotiationKind string

const (
	// KindRenegotiation: série antiga → série nova (novação).
	KindRenegotiation RenegotiationKind = "renegociacao"
	// KindPayoff: série antiga → um lançamento realizado de quitação.
	KindPayoff RenegotiationKind = "quitacao"
	// KindAssetSwap: troca de bem financiado — quitação do contrato antigo
	// (paga pelo terceiro que recebe o bem), venda do bem usado e, opcional,
	// série nova no bem novo. Um único evento amarra tudo.
	KindAssetSwap RenegotiationKind = "troca_bem"
)

// Payer diz de onde saiu o dinheiro da quitação.
type Payer string

const (
	// PayerSelf: caixa do usuário (conta/forma de pagamento reais).
	PayerSelf Payer = "proprio"
	// PayerThirdParty: terceiro (concessionária, comprador) pagou ao credor
	// abatendo do valor do bem — não há movimento de caixa do usuário.
	PayerThirdParty Payer = "terceiro"
)

// Renegotiation registra um EVENTO sobre uma dívida parcelada: as cobranças
// em aberto da dívida antiga são encerradas e algo nasce no lugar — uma série
// nova (renegociação), um lançamento de quitação (quitação) ou os dois mais
// a venda do bem (troca).
//
// É modelada como evento, e não como um ponteiro de um lançamento para outro,
// porque a relação é N→M: dezenas de parcelas previstas mais os residuais de
// pagamentos parciais se consolidam num único desfecho. O evento é o que
// permite, a partir de qualquer parcela nova, chegar a todas as origens — e
// vice-versa —, além de tornar natural encadear eventos sucessivos.
type Renegotiation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Kind        RenegotiationKind
	Date        time.Time
	// Description identifica a dívida (herda a descrição da origem).
	Description string
	// SettledAmountCents é o saldo devedor apurado nas origens: parcelas
	// previstas + residuais em aberto. NÃO inclui parcelas já realizadas.
	SettledAmountCents int64
	// NewAmountCents é o total da série nova (parcelas × valor); zero em
	// quitação sem contrato novo.
	NewAmountCents int64
	// AdjustmentCents mede o que o evento custou ou economizou sobre o saldo
	// apurado. Em renegociação = NewAmount − Settled; em quitação/troca =
	// Payoff − Settled. Positivo é encargo/juros; negativo é desconto.
	AdjustmentCents int64
	// OriginCount / NewCount: quantos lançamentos entraram e quantas
	// parcelas nasceram.
	OriginCount int
	NewCount    int
	// OriginGroupID é o parcelamento encerrado; NewGroupID é o grupo da
	// série nova (nil em quitação). Juntos encadeiam eventos sucessivos e
	// ancoram a linhagem da dívida.
	OriginGroupID *uuid.UUID
	NewGroupID    *uuid.UUID
	Notes         *string

	// --- quitação (KindPayoff / KindAssetSwap) ---
	// PayoffCents: quanto foi efetivamente pago para encerrar o saldo.
	PayoffCents *int64
	Payer       *Payer
	// PayoffEntryID: o lançamento realizado que representa a quitação.
	PayoffEntryID *uuid.UUID
	// AssetType/AssetID: o bem financiado pelo contrato encerrado.
	AssetType *AssetType
	AssetID   *uuid.UUID

	// --- troca (KindAssetSwap) ---
	// NewAssetID: o bem novo, financiado pela série nova.
	NewAssetID *uuid.UUID
	// TradeInCents: valor de avaliação do bem usado dado como parte do
	// pagamento; TradeInEntryID é a receita de venda correspondente.
	TradeInCents   *int64
	TradeInEntryID *uuid.UUID
	// CashDownpaymentCents: entrada paga em dinheiro pelo usuário além do
	// bem usado; DownpaymentEntryID é a despesa realizada correspondente.
	CashDownpaymentCents *int64
	DownpaymentEntryID   *uuid.UUID

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsDiscount informa se o evento resultou em abatimento.
func (r *Renegotiation) IsDiscount() bool { return r.AdjustmentCents < 0 }

// ClosesDebt informa se o evento encerra a dívida sem continuação na mesma
// linhagem (quitação e troca; a série nova de uma troca é OUTRA dívida, de
// outro bem).
func (r *Renegotiation) ClosesDebt() bool { return r.Kind != KindRenegotiation }

// NetDownpaymentCents é a entrada líquida numa troca: o que o bem usado
// valeu menos o que foi gasto para quitá-lo, mais o dinheiro colocado.
// Negativo significa que o bem valia menos que a dívida — a diferença foi
// coberta em dinheiro ou rolada no financiamento novo.
func (r *Renegotiation) NetDownpaymentCents() int64 {
	var tradeIn, payoff, cash int64
	if r.TradeInCents != nil {
		tradeIn = *r.TradeInCents
	}
	if r.PayoffCents != nil {
		payoff = *r.PayoffCents
	}
	if r.CashDownpaymentCents != nil {
		cash = *r.CashDownpaymentCents
	}
	return tradeIn - payoff + cash
}

func (r *Renegotiation) Validate() error {
	if r.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	if r.Kind == "" {
		r.Kind = KindRenegotiation
	}
	r.Description = strings.TrimSpace(r.Description)
	if r.Description == "" {
		return &ValidationError{Msg: "descrição é obrigatória"}
	}
	if r.AssetType != nil && !ValidAssetType(*r.AssetType) {
		return &ValidationError{Msg: "asset_type inválido"}
	}
	if (r.AssetType == nil) != (r.AssetID == nil) {
		return &ValidationError{Msg: "asset_type e asset_id devem vir juntos"}
	}
	if r.Payer != nil && *r.Payer != PayerSelf && *r.Payer != PayerThirdParty {
		return &ValidationError{Msg: "payer inválido"}
	}

	switch r.Kind {
	case KindRenegotiation:
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

	case KindPayoff:
		if r.SettledAmountCents <= 0 || r.OriginCount == 0 {
			return &ValidationError{Msg: "não há saldo em aberto para quitar"}
		}
		if r.PayoffCents == nil || *r.PayoffCents <= 0 {
			return &ValidationError{Msg: "informe o valor pago na quitação"}
		}
		if r.Payer == nil {
			return &ValidationError{Msg: "informe quem pagou a quitação"}
		}
		r.NewAmountCents = 0
		r.NewCount = 0
		r.NewGroupID = nil
		r.AdjustmentCents = *r.PayoffCents - r.SettledAmountCents

	case KindAssetSwap:
		if r.AssetID == nil || r.NewAssetID == nil {
			return &ValidationError{Msg: "a troca precisa do bem antigo e do bem novo"}
		}
		if *r.AssetID == *r.NewAssetID {
			return &ValidationError{Msg: "o bem novo precisa ser diferente do bem antigo"}
		}
		if r.TradeInCents == nil || *r.TradeInCents < 0 {
			return &ValidationError{Msg: "informe o valor do bem usado na troca"}
		}
		if r.CashDownpaymentCents != nil && *r.CashDownpaymentCents < 0 {
			return &ValidationError{Msg: "a entrada em dinheiro não pode ser negativa"}
		}
		// Contrato antigo é opcional (o bem trocado pode já estar quitado).
		if r.OriginCount > 0 {
			if r.SettledAmountCents <= 0 {
				return &ValidationError{Msg: "saldo apurado inválido para o contrato antigo"}
			}
			if r.PayoffCents == nil || *r.PayoffCents <= 0 {
				return &ValidationError{Msg: "informe quanto foi pago para quitar o contrato antigo"}
			}
			if r.Payer == nil {
				return &ValidationError{Msg: "informe quem pagou a quitação do contrato antigo"}
			}
			r.AdjustmentCents = *r.PayoffCents - r.SettledAmountCents
		} else {
			r.SettledAmountCents = 0
			r.PayoffCents = nil
			r.Payer = nil
			r.AdjustmentCents = 0
		}
		// Série nova é opcional (compra à vista).
		if (r.NewCount > 0) != (r.NewAmountCents > 0) {
			return &ValidationError{Msg: "série nova inconsistente"}
		}
		if r.NewCount == 0 && r.OriginCount == 0 && *r.TradeInCents == 0 {
			return &ValidationError{Msg: "a troca precisa envolver ao menos um contrato ou um valor de bem usado"}
		}

	default:
		return &ValidationError{Msg: "kind do evento inválido"}
	}
	return nil
}

// AssetSale descreve a baixa de um bem dentro de um evento (troca): o bem
// passa a vendido, com data e valor. Aplicada na mesma transação do evento.
type AssetSale struct {
	AssetType  AssetType
	AssetID    uuid.UUID
	SoldAt     time.Time
	PriceCents int64
}

// AssetAcquisition registra a aquisição de um bem novo dentro de um evento
// (troca): data e, quando informado, preço. Só preenche o que estiver vazio
// no cadastro — o usuário pode já ter cadastrado o bem com esses dados.
type AssetAcquisition struct {
	AssetType  AssetType
	AssetID    uuid.UUID
	AcquiredAt time.Time
	PriceCents *int64
}

// ApplyInput reúne tudo que um evento precisa persistir atomicamente.
type ApplyInput struct {
	Event *Renegotiation
	// OriginIDs: cobranças em aberto a encerrar (canceladas com CancelReason
	// e vínculo settled_by ao evento). Todas precisam estar previstas no
	// momento da gravação — é a trava de concorrência.
	OriginIDs    []uuid.UUID
	CancelReason string
	// NewEntries: tudo que o evento cria — parcelas novas, lançamento de
	// quitação, receita de venda, entrada em dinheiro.
	NewEntries []*FinancialEntry
	// Sale/Acquisition: efeitos no cadastro do bem (troca).
	Sale        *AssetSale
	Acquisition *AssetAcquisition
}

// RenegotiationRepository persiste o evento e aplica o desfecho.
type RenegotiationRepository interface {
	// Apply grava o evento, encerra as origens e cria os lançamentos novos
	// (e atualiza o bem, na troca) — tudo em uma única transação. Um evento
	// parcialmente aplicado deixaria a dívida duplicada ou sumida.
	Apply(ctx context.Context, in ApplyInput) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*Renegotiation, error)
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]Renegotiation, int64, error)
	// ListEntries devolve os lançamentos vinculados ao evento, separados
	// entre origens (canceladas) e os criados por ele.
	ListEntries(ctx context.Context, workspaceID, renegotiationID uuid.UUID) (origins, created []FinancialEntry, err error)
	// FindByNewGroup devolve o evento que criou o grupo; nil sem erro quando
	// o grupo é um parcelamento original.
	FindByNewGroup(ctx context.Context, workspaceID, groupID uuid.UUID) (*Renegotiation, error)
	// FindByOriginGroup devolve o evento que encerrou o grupo; nil sem erro
	// quando o grupo ainda está vigente.
	FindByOriginGroup(ctx context.Context, workspaceID, groupID uuid.UUID) (*Renegotiation, error)
	// ListByAsset devolve os eventos em que o bem aparece (como antigo ou
	// novo), do mais recente ao mais antigo.
	ListByAsset(ctx context.Context, workspaceID uuid.UUID, assetType AssetType, assetID uuid.UUID) ([]Renegotiation, error)
}
