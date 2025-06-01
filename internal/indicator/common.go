package indicator

import (
	"github.com/gamma-omg/trading-bot/internal/market"
)

type barsProvider interface {
	GetBars(count int) ([]market.Bar, error)
	HasBars(count int) bool
}

func ema(data []float64, period int) []float64 {
	if len(data) < period {
		panic("not enough data to compute ema")
	}

	ema := make([]float64, len(data))
	ema[0] = data[0]

	a := 2.0 / (float64(period) + 1)
	for i, val := range data[1:] {
		ema[i+1] = val*a + ema[i]*(1-a)
	}

	return ema
}
