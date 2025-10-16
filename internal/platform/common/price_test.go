package common

import (
	"fmt"
	"testing"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePrice(t *testing.T) {
	t.Parallel()

	tbl := []struct {
		symbol string
		price  int64
	}{
		{"BTC", 42},
		{"ETC", 43},
		{"C1", 100},
		{"C2", 200},
		{"C3", 300},
		{"C4", 400},
		{"C5", 500},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			p := NewDefaultPriceProvider()
			p.UpdatePrice(c.symbol, market.Bar{Close: decimal.NewFromInt(c.price)})

			b, err := p.GetLastBar(c.symbol)
			require.NoError(t, err)
			assert.True(t, b.Close.Equal(decimal.NewFromInt(c.price)))
		})
	}
}

func TestUpdatePrice_returnsError(t *testing.T) {
	p := NewDefaultPriceProvider()
	_, err := p.GetLastBar("BTC")
	require.Error(t, err)
}
