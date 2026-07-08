package finance

// CardBrand é uma bandeira de cartão de crédito — catálogo global fixo,
// compartilhado por todos os tenants. O slug é gravado em credit_cards.brand.
type CardBrand struct {
	Slug string
	Name string
}

// CardBrands é a lista curada, na ordem de exibição na UI.
var CardBrands = []CardBrand{
	{Slug: "visa", Name: "Visa"},
	{Slug: "mastercard", Name: "Mastercard"},
	{Slug: "elo", Name: "Elo"},
	{Slug: "american_express", Name: "American Express"},
	{Slug: "hipercard", Name: "Hipercard"},
	{Slug: "diners", Name: "Diners Club"},
	{Slug: "discover", Name: "Discover"},
	{Slug: "jcb", Name: "JCB"},
	{Slug: "unionpay", Name: "UnionPay"},
	{Slug: "aura", Name: "Aura"},
	{Slug: "cabal", Name: "Cabal"},
	{Slug: "sorocred", Name: "Sorocred"},
	{Slug: "outra", Name: "Outra"},
}

// ValidCardBrand informa se o slug pertence ao catálogo.
func ValidCardBrand(slug string) bool {
	for i := range CardBrands {
		if CardBrands[i].Slug == slug {
			return true
		}
	}
	return false
}
