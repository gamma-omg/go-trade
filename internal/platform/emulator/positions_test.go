package emulator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	acc := defaultAccount{balance: decimal.NewFromInt(10000)}
	pm := newPositionManager(l, &noCommission{}, &acc)

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			a := market.NewAssetWithBars(c.symbol, []market.Bar{{
				Time:  c.time,
				Close: decimal.NewFromFloat(c.price),
			}})
			p, err := pm.Open(context.Background(), a, decimal.NewFromFloat(c.size))
			require.NoError(t, err)

			assert.Equal(t, a, p.Asset)
			assert.Equal(t, c.symbol, p.Asset.Symbol)
			assert.Equal(t, c.time, p.OpenTime)
			assert.True(t, decimal.NewFromFloat(c.qty).Equal(p.Qty))
			assert.True(t, decimal.NewFromFloat(c.price).Equal(p.EntryPrice))
		})
	}
}

func TestOpen_withdrawsMoney(t *testing.T) {
	acc := defaultAccount{balance: decimal.NewFromInt(1000)}
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	pm := newPositionManager(l, &noCommission{}, &acc)

	a := market.NewAssetWithBars("BTC", []market.Bar{{Close: decimal.NewFromInt(1000)}})
	_, err := pm.Open(context.Background(), a, decimal.NewFromInt(100))
	require.NoError(t, err)

	assert.True(t, acc.balance.Equal(decimal.NewFromInt(900)))
}

func TestOpen_failsWhenCalledTwice(t *testing.T) {
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	pm := newPositionManager(l, &noCommission{}, &defaultAccount{balance: decimal.NewFromInt(10000)})
	a := market.NewAssetWithBars("BTC", []market.Bar{{Close: decimal.NewFromInt(100)}})

	_, err := pm.Open(context.Background(), a, decimal.NewFromFloat(100))
	require.NoError(t, err)
}

func TestClose(t *testing.T) {
	ts := time.Now()
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	pm := newPositionManager(l, &noCommission{}, &defaultAccount{balance: decimal.NewFromInt(100000)})
	a := market.NewAssetWithBars("BTC", []market.Bar{{Time: ts, Close: decimal.NewFromInt(100)}})
	p, err := pm.Open(context.Background(), a, decimal.NewFromFloat(200))
	require.NoError(t, err)

	a.Receive(market.Bar{
		Time:  ts.Add(1 * time.Minute),
		Close: decimal.NewFromFloat(120),
	})
	d, err := pm.Close(context.Background(), p)
	require.NoError(t, err)

	assert.Equal(t, "BTC", d.Symbol)
	assert.Equal(t, ts, d.BuyTime)
	assert.Equal(t, ts.Add(1*time.Minute), d.SellTime)
	assert.True(t, decimal.NewFromFloat(100).Equal(d.BuyPrice))
	assert.True(t, decimal.NewFromFloat(120).Equal(d.SellPrice))
}
