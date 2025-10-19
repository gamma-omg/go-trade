package indicator

import (
	"errors"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type mockBarsProvider struct {
	closePrices []float64
}

func (m *mockBarsProvider) GetBars(count int) ([]market.Bar, error) {
	n := len(m.closePrices)
	if n < count {
		return nil, errors.New("not enought data")
	}

	bars := make([]market.Bar, count)
	for i := 0; i < count; i++ {
		bars[i] = market.Bar{Close: decimal.NewFromFloat(m.closePrices[n-count+i])}
	}

	return bars, nil
}

func (m *mockBarsProvider) HasBars(count int) bool {
	return len(m.closePrices) >= count
}
