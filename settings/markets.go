package settings

const (
	Binancef = "binancef"
)

var Markets = map[string]Market{
	Binancef: {
		Name: Binancef,
		Symbols: map[string]Symbol{
			"btcusdt": {
				Name:     "btcusdt",
				TickSize: 0.10,
			},
			"solusdt": {
				Name:     "solusdt",
				TickSize: 0.001,
			},
			"ethusdt": {
				Name:     "ethusdt",
				TickSize: 0.01,
			},
			"trumpusdt": {
				Name:     "trumpusdt",
				TickSize: 0.001,
			},
		},
	},
}

type Symbol struct {
	Name         string
	InternalName string
	PriceGroup   float64
	TickSize     float64
}

type Market struct {
	Name    string
	Symbols map[string]Symbol
}
