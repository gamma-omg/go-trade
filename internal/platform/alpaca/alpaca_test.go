package alpaca

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAlpacaApi struct {
	getCryptoBars       func(symbol string, req marketdata.GetCryptoBarsRequest) ([]marketdata.CryptoBar, error)
	getCryptoBarsStream func(symbol string, bars chan<- stream.CryptoBar, errs chan<- error)
	placeOrder          func(req alpaca.PlaceOrderRequest) (*alpaca.Order, error)
	getOrder            func(orderId string) (*alpaca.Order, error)
	closePosition       func(symbol string, req alpaca.ClosePositionRequest) (*alpaca.Order, error)
	getAccount          func() (*alpaca.Account, error)
	closeAllPositions   func(req alpaca.CloseAllPositionsRequest) ([]alpaca.Order, error)
}

func (m *mockAlpacaApi) GetCryptoBars(symbol string, req marketdata.GetCryptoBarsRequest) ([]marketdata.CryptoBar, error) {
	return m.getCryptoBars(symbol, req)
}

func (m *mockAlpacaApi) GetCryptoBarsStream(_ context.Context, symbol string) (<-chan stream.CryptoBar, <-chan error) {
	bars := make(chan stream.CryptoBar)
	errs := make(chan error)
	go func() {
		defer close(bars)
		defer close(errs)
		m.getCryptoBarsStream(symbol, bars, errs)
	}()

	return bars, errs
}

func (m *mockAlpacaApi) PlaceOrder(req alpaca.PlaceOrderRequest) (*alpaca.Order, error) {
	return m.placeOrder(req)
}

func (m *mockAlpacaApi) ClosePosition(symbol string, req alpaca.ClosePositionRequest) (*alpaca.Order, error) {
	return m.closePosition(symbol, req)
}

func (m *mockAlpacaApi) GetOrder(orderID string) (*alpaca.Order, error) {
	return m.getOrder(orderID)
}

func (m *mockAlpacaApi) GetAccount() (*alpaca.Account, error) {
	return m.getAccount()
}

func (m *mockAlpacaApi) CloseAllPositions(req alpaca.CloseAllPositionsRequest) ([]alpaca.Order, error) {
	return m.closeAllPositions(req)
}

type testBar struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

func TestPrefetch(t *testing.T) {
	tbl := []struct {
		count   int
		history []marketdata.CryptoBar
		out     []testBar
		err     bool
	}{
		{
			count: 3,
			history: []marketdata.CryptoBar{
				{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
			},
			out: []testBar{
				{Time: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Time: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Time: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
			},
			err: false,
		},
		{
			count: 3,
			history: []marketdata.CryptoBar{
				{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Timestamp: time.Unix(0, 0), Open: 50, High: 109, Low: 150, Close: 200, Volume: 50},
				{Timestamp: time.Unix(0, 0), Open: 100, High: 200, Low: 300, Close: 400, Volume: 100},
			},
			out: []testBar{
				{Time: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
				{Time: time.Unix(0, 0), Open: 50, High: 109, Low: 150, Close: 200, Volume: 50},
				{Time: time.Unix(0, 0), Open: 100, High: 200, Low: 300, Close: 400, Volume: 100},
			},
			err: false,
		},
		{
			count: 3,
			history: []marketdata.CryptoBar{
				{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 1},
			},
			out: []testBar{},
			err: true,
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			a := AlpacaPlatform{
				log: slog.New(slog.DiscardHandler),
				api: &mockAlpacaApi{
					getCryptoBars: func(symbol string, req marketdata.GetCryptoBarsRequest) ([]marketdata.CryptoBar, error) {
						return c.history, nil
					},
				},
			}

			ex := make([]market.Bar, len(c.out))
			for i, b := range c.out {
				ex[i] = market.Bar{
					Time:   b.Time,
					Open:   decimal.NewFromFloat(b.Open),
					High:   decimal.NewFromFloat(b.High),
					Low:    decimal.NewFromFloat(b.Low),
					Close:  decimal.NewFromFloat(b.Close),
					Volume: decimal.NewFromFloat(b.Volume),
				}
			}

			barsCh, err := a.Prefetch("BTC", c.count)
			require.Equal(t, c.err, err != nil)
			if c.err {
				return
			}

			var bars []market.Bar
			for b := range barsCh {
				bars = append(bars, b)
			}

			assert.ElementsMatch(t, ex, bars)
		})
	}
}

func TestGetBars(t *testing.T) {
	type testInput struct {
		bar stream.CryptoBar
		err error
	}

	tbl := []struct {
		in   []testInput
		bars []testBar
		errs []error
	}{
		{
			in: []testInput{
				{bar: stream.CryptoBar{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 0}},
				{bar: stream.CryptoBar{Timestamp: time.Unix(1, 0), Open: 4, High: 5, Low: 6, Close: 7, Volume: 1}},
				{err: errors.New("some error")},
				{bar: stream.CryptoBar{Timestamp: time.Unix(2, 0), Open: 7, High: 8, Low: 9, Close: 6, Volume: 2}},
			},
			bars: []testBar{
				{Time: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 0},
				{Time: time.Unix(1, 0), Open: 4, High: 5, Low: 6, Close: 7, Volume: 1},
				{Time: time.Unix(2, 0), Open: 7, High: 8, Low: 9, Close: 6, Volume: 2},
			},
			errs: []error{
				errors.New("some error"),
			},
		},
		{
			in: []testInput{
				{bar: stream.CryptoBar{Timestamp: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 0}},
				{bar: stream.CryptoBar{Timestamp: time.Unix(1, 0), Open: 4, High: 5, Low: 6, Close: 7, Volume: 1}},
				{bar: stream.CryptoBar{Timestamp: time.Unix(2, 0), Open: 7, High: 8, Low: 9, Close: 6, Volume: 2}},
			},
			bars: []testBar{
				{Time: time.Unix(0, 0), Open: 1, High: 2, Low: 3, Close: 4, Volume: 0},
				{Time: time.Unix(1, 0), Open: 4, High: 5, Low: 6, Close: 7, Volume: 1},
				{Time: time.Unix(2, 0), Open: 7, High: 8, Low: 9, Close: 6, Volume: 2},
			},
			errs: []error{},
		},
		{
			in: []testInput{
				{err: errors.New("error 1")},
				{err: errors.New("error 2")},
				{err: errors.New("error 3")},
			},
			bars: []testBar{},
			errs: []error{
				errors.New("error 1"),
				errors.New("error 2"),
				errors.New("error 3"),
			},
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			a := AlpacaPlatform{
				log: slog.New(slog.DiscardHandler),
				api: &mockAlpacaApi{
					getCryptoBarsStream: func(symbol string, bars chan<- stream.CryptoBar, errs chan<- error) {
						for _, in := range c.in {
							if in.err != nil {
								errs <- in.err
							} else {
								bars <- in.bar
							}
						}
					},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			barsCh, errsCh := a.GetBars(ctx, "BTC")

			var bars []testBar
			var errs []error
			for barsCh != nil || errsCh != nil {
				select {
				case <-ctx.Done():
					break
				case bar, ok := <-barsCh:
					if !ok {
						barsCh = nil
						continue
					}
					o, _ := bar.Open.Float64()
					h, _ := bar.High.Float64()
					l, _ := bar.Low.Float64()
					v, _ := bar.Volume.Float64()
					cl, _ := bar.Close.Float64()
					bars = append(bars, testBar{Time: bar.Time, Open: o, High: h, Low: l, Close: cl, Volume: v})
				case err, ok := <-errsCh:
					if !ok {
						errsCh = nil
						continue
					}
					errs = append(errs, err)
				}
			}

			assert.ElementsMatch(t, c.errs, errs)
			assert.ElementsMatch(t, c.bars, bars)
		})
	}
}

func TestOpen(t *testing.T) {
	tbl := []struct {
		time        time.Time
		qty         float64
		filledQty   float64
		filledPrice float64
	}{
		{qty: 3, filledQty: 2, time: time.Unix(1, 0), filledPrice: 123},
		{qty: 0, filledQty: 0, time: time.Unix(2, 0), filledPrice: 100},
		{qty: 100, filledQty: 100, time: time.Unix(3, 0), filledPrice: 1000},
		{qty: 1000, filledQty: 2000, time: time.Unix(4, 0), filledPrice: 1000000},
		{qty: 10000000000, filledQty: 10000000000, time: time.Unix(5, 0), filledPrice: 10000000000},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			asset := market.NewAsset("BTC", 1024)
			asset.Receive(market.Bar{Close: decimal.NewFromFloat(1)})

			price := decimal.NewFromFloat(c.filledPrice)
			o := &alpaca.Order{
				Symbol:         "BTC",
				FilledQty:      decimal.NewFromFloat(c.filledQty),
				FilledAvgPrice: &price,
				FilledAt:       &c.time,
			}
			a := AlpacaPlatform{
				log: slog.New(slog.DiscardHandler),
				api: &mockAlpacaApi{
					placeOrder: func(req alpaca.PlaceOrderRequest) (*alpaca.Order, error) {
						if req.Symbol == asset.Symbol && req.Side == alpaca.Buy {
							o.Qty = req.Qty
							return o, nil
						}

						return nil, errors.New("unknown symbol")
					},
					getOrder: func(orderId string) (*alpaca.Order, error) {
						return o, nil
					},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			p, err := a.Open(ctx, asset, decimal.NewFromFloat(c.qty))
			require.NoError(t, err)

			assert.True(t, decimal.NewFromFloat(c.qty).Equal(*o.Qty))
			assert.True(t, decimal.NewFromFloat(c.filledQty).Equal(p.Qty))
			assert.True(t, decimal.NewFromFloat(c.filledPrice).Equal(p.EntryPrice))
			assert.True(t, decimal.NewFromFloat(c.filledQty*c.filledPrice).Equal(p.Price))
			assert.Equal(t, c.time, p.OpenTime)
		})
	}
}

func TestClose(t *testing.T) {
	tbl := []struct {
		symbol    string
		filledAt  time.Time
		fillPrice float64
		fillQty   float64
		buyTime   time.Time
		buyPrice  float64
		spend     float64
	}{
		{symbol: "BTC", filledAt: time.Unix(1, 0), fillPrice: 123, fillQty: 5, buyTime: time.Unix(2, 0), buyPrice: 100, spend: 1000},
		{symbol: "ETH", filledAt: time.Unix(2, 0), fillPrice: 0, fillQty: 1, buyTime: time.Unix(10, 0), buyPrice: 100, spend: 123},
		{symbol: "USD", filledAt: time.Unix(3, 0), fillPrice: 10000000000, fillQty: 10000000000, buyTime: time.Unix(100000, 0), buyPrice: 10000000000, spend: 10000000000},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			asset := market.NewAsset(c.symbol, 1024)
			asset.Receive(market.Bar{Close: decimal.NewFromFloat(1)})

			fillPrice := decimal.NewFromFloat(c.fillPrice)
			o := &alpaca.Order{
				FilledAt:       &c.filledAt,
				FilledAvgPrice: &fillPrice,
				FilledQty:      decimal.NewFromFloat(c.fillQty),
			}
			a := AlpacaPlatform{
				log: slog.New(slog.DiscardHandler),
				api: &mockAlpacaApi{
					closePosition: func(symbol string, req alpaca.ClosePositionRequest) (*alpaca.Order, error) {
						if symbol != c.symbol || !req.Percentage.Equal(decimal.NewFromInt(100)) {
							return nil, errors.New("unknown symbol")
						}

						o.Symbol = symbol
						return o, nil
					},
					getOrder: func(orderId string) (*alpaca.Order, error) {
						return o, nil
					},
				},
			}

			p := market.Position{
				Asset:      asset,
				OpenTime:   c.buyTime,
				EntryPrice: decimal.NewFromFloat(c.buyPrice),
				Price:      decimal.NewFromFloat(c.spend),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			d, err := a.Close(ctx, &p)
			require.NoError(t, err)

			assert.Equal(t, c.symbol, d.Symbol)
			assert.Equal(t, c.filledAt, d.SellTime)
			assert.Equal(t, c.buyTime, d.BuyTime)
			assert.True(t, decimal.NewFromFloat(c.buyPrice).Equal(d.BuyPrice))
			assert.True(t, decimal.NewFromFloat(c.spend).Equal(d.Spend))
			assert.True(t, decimal.NewFromFloat(c.fillPrice).Equal(d.SellPrice))
			assert.True(t, decimal.NewFromFloat(c.fillQty).Equal(d.Qty))
		})
	}
}

func TestGetBalance(t *testing.T) {
	tbl := []struct {
		balance float64
		err     error
	}{
		{balance: 1000},
		{err: errors.New("some error")},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			a := AlpacaPlatform{
				log: slog.New(slog.DiscardHandler),
				api: &mockAlpacaApi{
					getAccount: func() (*alpaca.Account, error) {
						if c.err != nil {
							return nil, c.err
						}

						return &alpaca.Account{BuyingPower: decimal.NewFromFloat(c.balance)}, nil
					},
				},
			}

			balance, err := a.GetBalance()
			require.ErrorIs(t, err, c.err)
			assert.True(t, decimal.NewFromFloat(c.balance).Equal(balance))
		})
	}
}

func TestNewAlpacaPlatform_closesAllPositions(t *testing.T) {
	called := false
	_, err := newAlpacaPlatformWithApi(slog.New(slog.DiscardHandler), config.Alpaca{}, &mockAlpacaApi{
		closeAllPositions: func(req alpaca.CloseAllPositionsRequest) ([]alpaca.Order, error) {
			if req.CancelOrders {
				called = true
			}
			return []alpaca.Order{}, nil
		},
	})

	require.NoError(t, err)
	assert.True(t, called)
}
