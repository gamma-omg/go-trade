package emulator

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPriceProvider struct {
	price map[string]market.Bar
	mu    sync.Mutex
}

func (pp *mockPriceProvider) GetLastBar(symbol string) (bar market.Bar, err error) {
	bar, ok := pp.price[symbol]
	if !ok {
		err = fmt.Errorf("unknown symbol %s", symbol)
		return
	}

	return
}

type mockReportBuilder struct {
	deals []Deal
}

func (rb *mockReportBuilder) SubmitDeal(d Deal) {
	rb.deals = append(rb.deals, d)
}

func TestOpen(t *testing.T) {
	t.Parallel()

	tbl := []struct {
		symbol string
		price  float64
		size   float64
		qty    float64
		time   time.Time
	}{
		{symbol: "C1", time: time.Now(), price: 100, size: 500, qty: 5},
		{symbol: "C2", time: time.Now(), price: 100, size: 50, qty: 0.5},
		{symbol: "C3", time: time.Now(), price: 200, size: 200, qty: 1},
		{symbol: "C4", time: time.Now(), price: 1000, size: 200, qty: 0.2},
	}

	prices := mockPriceProvider{price: make(map[string]market.Bar)}
	for _, c := range tbl {
		prices.price[c.symbol] = market.Bar{
			Time:  c.time,
			Close: decimal.NewFromFloat(c.price),
		}
	}

	pm := newPositionManager(&noComission{}, &prices, &mockReportBuilder{})

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_{%d}", i), func(t *testing.T) {
			p, err := pm.Open(context.Background(), c.symbol, decimal.NewFromFloat(c.size))
			require.NoError(t, err)

			assert.Equal(t, c.symbol, p.Symbol)
			assert.Equal(t, c.time, p.OpenTime)
			assert.True(t, decimal.NewFromFloat(c.qty).Equal(p.Qty))
			assert.True(t, decimal.NewFromFloat(c.price).Equal(p.EntryPrice))
		})
	}
}

func TestOpen_failsWhenCalledTwice(t *testing.T) {
	prices := mockPriceProvider{
		price: map[string]market.Bar{"BTC": {Close: decimal.NewFromFloat(100)}},
	}

	pm := newPositionManager(&noComission{}, &prices, &mockReportBuilder{})

	_, err := pm.Open(context.Background(), "BTC", decimal.NewFromFloat(100))
	require.NoError(t, err)

	_, err = pm.Open(context.Background(), "BTC", decimal.NewFromFloat(100))
	require.Error(t, err)
}

func TestClose(t *testing.T) {
	ts := time.Now()
	report := mockReportBuilder{}
	prices := mockPriceProvider{
		price: map[string]market.Bar{"BTC": {
			Close: decimal.NewFromFloat(100),
			Time:  ts,
		}},
	}

	pm := newPositionManager(&noComission{}, &prices, &report)
	_, err := pm.Open(context.Background(), "BTC", decimal.NewFromFloat(100))
	require.NoError(t, err)

	prices.price["BTC"] = market.Bar{
		Close: decimal.NewFromFloat(120),
		Time:  ts.Add(1 * time.Minute),
	}
	err = pm.Close(context.Background(), "BTC")
	require.NoError(t, err)

	assert.Equal(t, 1, len(report.deals))

	d := report.deals[0]
	assert.Equal(t, "BTC", d.Symbol)
	assert.Equal(t, ts, d.BuyTime)
	assert.Equal(t, ts.Add(1*time.Minute), d.SellTime)
	assert.True(t, decimal.NewFromFloat(100).Equal(d.BuyPrice))
	assert.True(t, decimal.NewFromFloat(120).Equal(d.SellPrice))
	assert.True(t, decimal.NewFromFloat(20).Equal(d.Gain))
	assert.InDelta(t, 1.2, d.GainPct, 0.00001)
}

func TestClose_failureScenarious(t *testing.T) {
	prices := mockPriceProvider{
		price: map[string]market.Bar{"BTC": {Close: decimal.NewFromFloat(100)}},
	}

	pm := newPositionManager(&noComission{}, &prices, &mockReportBuilder{})

	err := pm.Close(context.Background(), "BTC")
	require.Error(t, err)

	_, err = pm.Open(context.Background(), "BTC", decimal.NewFromFloat(1))
	require.NoError(t, err)

	err = pm.Close(context.Background(), "BTC")
	require.NoError(t, err)

	err = pm.Close(context.Background(), "BTC")
	require.Error(t, err)
}
