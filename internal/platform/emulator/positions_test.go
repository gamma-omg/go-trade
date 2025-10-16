package emulator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
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
	pp.mu.Lock()
	defer pp.mu.Unlock()

	bar, ok := pp.price[symbol]
	if !ok {
		err = fmt.Errorf("unknown symbol %s", symbol)
		return
	}

	return
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

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	acc := defaultAccount{balance: decimal.NewFromInt(10000)}
	pm := newPositionManager(l, &noComission{}, &prices, &acc)

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

func TestOpen_withdrawsMoney(t *testing.T) {

	prices := mockPriceProvider{
		price: map[string]market.Bar{
			"BTC": {Close: decimal.NewFromInt(100)},
		},
	}
	acc := defaultAccount{balance: decimal.NewFromInt(1000)}
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	pm := newPositionManager(l, &noComission{}, &prices, &acc)

	_, err := pm.Open(context.Background(), "BTC", decimal.NewFromInt(100))
	require.NoError(t, err)

	assert.True(t, acc.balance.Equal(decimal.NewFromInt(900)))
}

func TestOpen_failsWhenCalledTwice(t *testing.T) {
	prices := mockPriceProvider{
		price: map[string]market.Bar{"BTC": {Close: decimal.NewFromFloat(100)}},
	}

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	pm := newPositionManager(l, &noComission{}, &prices, &defaultAccount{balance: decimal.NewFromInt(10000)})

	_, err := pm.Open(context.Background(), "BTC", decimal.NewFromFloat(100))
	require.NoError(t, err)

	_, err = pm.Open(context.Background(), "BTC", decimal.NewFromFloat(100))
	require.Error(t, err)
}

func TestClose(t *testing.T) {
	ts := time.Now()
	prices := mockPriceProvider{
		price: map[string]market.Bar{"BTC": {
			Close: decimal.NewFromFloat(100),
			Time:  ts,
		}},
	}

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	pm := newPositionManager(l, &noComission{}, &prices, &defaultAccount{balance: decimal.NewFromInt(100000)})
	_, err := pm.Open(context.Background(), "BTC", decimal.NewFromFloat(200))
	require.NoError(t, err)

	prices.price["BTC"] = market.Bar{
		Close: decimal.NewFromFloat(120),
		Time:  ts.Add(1 * time.Minute),
	}
	d, err := pm.Close(context.Background(), "BTC")
	require.NoError(t, err)

	assert.Equal(t, "BTC", d.Symbol)
	assert.Equal(t, ts, d.BuyTime)
	assert.Equal(t, ts.Add(1*time.Minute), d.SellTime)
	assert.True(t, decimal.NewFromFloat(100).Equal(d.BuyPrice))
	assert.True(t, decimal.NewFromFloat(120).Equal(d.SellPrice))
	assert.True(t, decimal.NewFromFloat(40).Equal(d.Gain))
}

func TestClose_failureScenarious(t *testing.T) {
	prices := mockPriceProvider{
		price: map[string]market.Bar{"BTC": {Close: decimal.NewFromFloat(100)}},
	}

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	pm := newPositionManager(l, &noComission{}, &prices, &defaultAccount{balance: decimal.NewFromInt(100000)})

	_, err := pm.Close(context.Background(), "BTC")
	require.Error(t, err)

	_, err = pm.Open(context.Background(), "BTC", decimal.NewFromFloat(1))
	require.NoError(t, err)

	_, err = pm.Close(context.Background(), "BTC")
	require.NoError(t, err)

	_, err = pm.Close(context.Background(), "BTC")
	require.Error(t, err)
}
