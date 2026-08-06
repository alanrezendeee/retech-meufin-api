package finance

// CancelReason é o catálogo global (fixo, compartilhado por todos os tenants)
// de motivos de cancelamento de um lançamento.
//
// O motivo não é só rótulo: ele carrega a INTENÇÃO do usuário sobre a série
// recorrente. Sem ele, cancelar uma ocorrência é ambíguo — pode significar
// "encerrei esta assinatura" ou "neste mês não houve cobrança" — e o extensor
// de recorrência precisava adivinhar pela posição da ocorrência na série
// (a mais futura era tratada como encerramento), o que matava séries em
// silêncio quando o usuário cancelava justamente o mês mais distante.
type CancelReason struct {
	Slug        string
	Name        string
	Description string
	// EndsRecurrence indica que o cancelamento encerra a série: o extensor
	// para de gerar novas ocorrências para o grupo.
	EndsRecurrence bool
}

// CancelReasonRenegotiation é o motivo aplicado às cobranças encerradas por
// uma renegociação. Constante porque a repactuação o grava sem passar pela UI.
const CancelReasonRenegotiation = "renegociacao"

// CancelReasons é a lista curada, na ordem de exibição na UI.
var CancelReasons = []CancelReason{
	{Slug: "encerramento", Name: "Encerrei este compromisso", Description: "Assinatura, contrato ou serviço encerrado — não haverá novas cobranças", EndsRecurrence: true},
	{Slug: "sem_cobranca_no_mes", Name: "Não houve cobrança neste mês", Description: "Cobrança pontualmente ausente; o compromisso continua nos próximos meses"},
	{Slug: "cobranca_indevida", Name: "Cobrança indevida", Description: "Cobrança que não deveria ter existido"},
	{Slug: "renegociacao", Name: "Renegociação da dívida", Description: "Cobrança encerrada e substituída por um novo acordo de parcelamento"},
	{Slug: "duplicidade", Name: "Lançamento duplicado", Description: "Já existe outro lançamento para esta mesma cobrança"},
	{Slug: "erro_lancamento", Name: "Erro no lançamento", Description: "Lançamento criado por engano ou com dados incorretos"},
	{Slug: "outros", Name: "Outros", Description: "Motivo de cancelamento não listado"},
}

// ValidCancelReason informa se o slug pertence ao catálogo.
func ValidCancelReason(slug string) bool {
	for i := range CancelReasons {
		if CancelReasons[i].Slug == slug {
			return true
		}
	}
	return false
}

// CancelReasonEndsRecurrence informa se o motivo encerra a série recorrente.
//
// Cancelamento SEM motivo (dados anteriores a este catálogo) preserva o
// comportamento antigo — encerra a série — para não ressuscitar recorrências
// que o usuário já considera encerradas.
func CancelReasonEndsRecurrence(slug *string) bool {
	if slug == nil || *slug == "" {
		return true
	}
	for i := range CancelReasons {
		if CancelReasons[i].Slug == *slug {
			return CancelReasons[i].EndsRecurrence
		}
	}
	return true
}
