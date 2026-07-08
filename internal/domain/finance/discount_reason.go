package finance

// DiscountReason é o catálogo global (fixo, compartilhado por todos os
// tenants) de motivos de desconto na liquidação de um lançamento. O slug é
// gravado no lançamento e vira indicador para dashboards/insights futuros.
type DiscountReason struct {
	Slug        string
	Name        string
	Description string
}

// DiscountReasons é a lista curada, na ordem de exibição na UI.
var DiscountReasons = []DiscountReason{
	{Slug: "pagamento_antecipado", Name: "Pagamento antecipado", Description: "Desconto por pagar antes do vencimento"},
	{Slug: "pagamento_a_vista", Name: "Pagamento à vista", Description: "Desconto por pagar de uma vez, sem parcelar"},
	{Slug: "quitacao_antecipada", Name: "Quitação antecipada", Description: "Abatimento de juros por quitar parcelas futuras (financiamento/empréstimo)"},
	{Slug: "primeira_ultima_parcela", Name: "Primeira e última parcela", Description: "Condição comercial ao pagar a primeira e a última parcela juntas"},
	{Slug: "pontualidade", Name: "Pontualidade", Description: "Desconto por pagamento em dia (ex.: condomínio, mensalidade escolar)"},
	{Slug: "promocao", Name: "Promoção/campanha", Description: "Oferta pontual do fornecedor ou campanha sazonal"},
	{Slug: "cupom", Name: "Cupom/voucher", Description: "Cupom de desconto, vale ou cashback aplicado no pagamento"},
	{Slug: "fidelidade", Name: "Fidelidade", Description: "Benefício por tempo de relacionamento ou plano de assinatura"},
	{Slug: "negociacao", Name: "Negociação/acordo", Description: "Desconto negociado, inclusive renegociação de dívida"},
	{Slug: "convenio", Name: "Convênio/parceria", Description: "Desconto por convênio de empresa, sindicato ou associação"},
	{Slug: "cortesia", Name: "Cortesia/isenção", Description: "Abono parcial concedido pelo credor"},
	{Slug: "outros", Name: "Outros", Description: "Motivo de desconto não listado"},
}

// ValidDiscountReason informa se o slug pertence ao catálogo.
func ValidDiscountReason(slug string) bool {
	for i := range DiscountReasons {
		if DiscountReasons[i].Slug == slug {
			return true
		}
	}
	return false
}
