package emulator

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFixedRateComission(t *testing.T) {
	tbl := []struct {
		buyFee     float64
		sellFee    float64
		buyAfter   float64
		buyBefore  float64
		sellBefore float64
		sellAfter  float64
	}{
		{buyFee: 0.2, sellFee: 0.5, buyBefore: 100, buyAfter: 80, sellBefore: 100, sellAfter: 50},
		{buyFee: 0.002, sellFee: 0.0015, buyBefore: 100, buyAfter: 99.8, sellBefore: 300, sellAfter: 299.55},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			comm := newFixedRateComission(c.buyFee, c.sellFee)
			buy := comm.ApplyOnBuy(decimal.NewFromFloat(c.buyBefore))
			sell := comm.ApplyOnSell(decimal.NewFromFloat(c.sellBefore))
			assert.True(t, decimal.NewFromFloat(c.buyAfter).Equal(buy))
			assert.True(t, decimal.NewFromFloat(c.sellAfter).Equal(sell))
		})
	}
}

func TestNoComission(t *testing.T) {
	comm := noComission{}
	buy := decimal.NewFromFloat(1234)
	sell := decimal.NewFromFloat(4321)
	assert.True(t, buy.Equal(comm.ApplyOnBuy(buy)))
	assert.True(t, sell.Equal(comm.ApplyOnSell(sell)))
}
