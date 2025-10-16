package agent

import (
	"fmt"
	"testing"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeedClose(t *testing.T) {
	tbl := []struct {
		entryPrice float64
		price      float64
		takeProfit float64
		stopLoss   float64
		close      bool
	}{
		{entryPrice: 100, price: 105, takeProfit: 1.1, stopLoss: 0.9, close: false},
		{entryPrice: 100, price: 112, takeProfit: 1.1, stopLoss: 0.9, close: true},
		{entryPrice: 100, price: 98, takeProfit: 1.1, stopLoss: 0.9, close: false},
		{entryPrice: 100, price: 89, takeProfit: 2, stopLoss: 0.9, close: true},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			v := defaultPositionValidator{
				takeProfit: c.takeProfit,
				stopLoss:   c.stopLoss,
			}

			a := market.NewAssetWithBars("sym", []market.Bar{{Close: decimal.NewFromFloat(c.price)}})
			p := market.Position{
				Asset:      a,
				EntryPrice: decimal.NewFromFloat(c.entryPrice),
			}

			cls, err := v.NeedClose(&p)
			require.NoError(t, err)
			assert.Equal(t, c.close, cls)
		})
	}
}

func TestNeedClose_Err(t *testing.T) {
	v := defaultPositionValidator{}
	a := market.NewAsset("sym", 1)
	p := market.Position{Asset: a}

	_, err := v.NeedClose(&p)
	require.Error(t, err)
}
