package fipe

// Tipos de veículo aceitos pela API FIPE.
const (
	VehicleCarros    = "carros"
	VehicleMotos     = "motos"
	VehicleCaminhoes = "caminhoes"
)

// Brand representa uma marca na tabela FIPE.
type Brand struct {
	Code string `json:"codigo"`
	Name string `json:"nome"`
}

// Model representa um modelo de veículo.
type Model struct {
	Code int    `json:"codigo"`
	Name string `json:"nome"`
}

// ModelsResponse é o envelope retornado em /marcas/:code/modelos.
type ModelsResponse struct {
	Models []Model `json:"modelos"`
	Years  []Year  `json:"anos"`
}

// Year representa um ano-combustível disponível (ex: código "2020-1", nome "2020 Gasolina").
type Year struct {
	Code string `json:"codigo"`
	Name string `json:"nome"`
}

// Price é o preço FIPE para uma combinação veículo/marca/modelo/ano.
type Price struct {
	Value          string `json:"Valor"`           // ex: "R$ 62.839,00"
	Brand          string `json:"Marca"`
	Model          string `json:"Modelo"`
	YearModel      int    `json:"AnoModelo"`
	Fuel           string `json:"Combustivel"`
	FipeCode       string `json:"CodigoFipe"`
	ReferenceMonth string `json:"MesReferencia"`   // ex: "julho de 2026"
	VehicleType    int    `json:"TipoVeiculo"`
	FuelAcronym    string `json:"SiglaCombustivel"` // G|A|D|E
}
