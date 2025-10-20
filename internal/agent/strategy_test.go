package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAccount struct {
	balance int
}

func (a *mockAccount) GetBalance() (decimal.Decimal, error) {
	return decimal.NewFromInt(int64(a.balance)), nil
}

type mockPositionScaler struct {
	scaleFunc func(budget decimal.Decimal, confidence float64) decimal.Decimal
}

func (s *mockPositionScaler) GetSize(budget decimal.Decimal, confidence float64) decimal.Decimal {
	return s.scaleFunc(budget, confidence)
}

type mockPositionManager struct {
	positions []*market.Position
	qtyFunc   func(size decimal.Decimal, symbol string) decimal.Decimal
}

func (pm *mockPositionManager) Open(_ context.Context, asset *market.Asset, size decimal.Decimal) (*market.Position, error) {
	pos := &market.Position{
		Asset: asset,
		Qty:   pm.qtyFunc(size, asset.Symbol),
	}
	pm.positions = append(pm.positions, pos)
	return pos, nil
}

func (pm *mockPositionManager) Close(_ context.Context, p *market.Position) (market.Deal, error) {
	pm.positions = slices.DeleteFunc(pm.positions, func(x *market.Position) bool {
		return x.Asset == p.Asset
	})

	return market.Deal{}, nil
}

type mockIndicator struct {
	act        indicator.Action
	confidence float64
}

func (m *mockIndicator) GetSignal() (indicator.Signal, error) {
	return indicator.Signal{
		Act:        m.act,
		Confidence: m.confidence,
	}, nil
}

func (m *mockIndicator) DrawDebug(d *indicator.DebugPlot) error {
	return nil
}

type mockReport struct {
	deals []market.Deal
}

func (m *mockReport) SubmitDeal(d market.Deal) {
	m.deals = append(m.deals, d)
}

func (m *mockReport) Write(w io.Writer) error {
	return nil
}

type mockPositionValidator struct {
	needClose bool
}

func (m *mockPositionValidator) NeedClose(p *market.Position) (bool, error) {
	return m.needClose, nil
}

func TestStrategyRun(t *testing.T) {
	cfg := config.Strategy{
		Budget:         1000,
		BuyConfidence:  0.5,
		SellConfidence: 0.5,
	}
	tbl := []struct {
		act         indicator.Action
		confidence  float64
		position    *market.Position
		initialPos  int
		expectedPos int
	}{
		{act: indicator.ActHold, confidence: 1.0, initialPos: 5, expectedPos: 5},
		{act: indicator.ActBuy, confidence: 0.4, initialPos: 5, expectedPos: 5},
		{act: indicator.ActBuy, confidence: 0.6, initialPos: 5, expectedPos: 6},
		{act: indicator.ActSell, confidence: 0.4, initialPos: 5, expectedPos: 5},
		{act: indicator.ActSell, confidence: 0.6, initialPos: 5, expectedPos: 5},
		{act: indicator.ActSell, confidence: 0.6, initialPos: 5, expectedPos: 4, position: &market.Position{}},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			posMan := mockPositionManager{qtyFunc: func(size decimal.Decimal, symbol string) decimal.Decimal {
				return size
			}}
			if c.position != nil {
				posMan.positions = append(posMan.positions, c.position)
			}

			a := market.NewAsset(fmt.Sprintf("s%d", i), 1)
			for len(posMan.positions) < c.initialPos {
				posMan.positions = append(posMan.positions, &market.Position{Asset: a})
			}

			scaler := mockPositionScaler{
				scaleFunc: func(budget decimal.Decimal, confidence float64) decimal.Decimal {
					return budget
				},
			}

			s := TradingStrategy{
				asset:        a,
				log:          slog.Default(),
				cfg:          cfg,
				posMan:       &posMan,
				posScaler:    &scaler,
				posValidator: &mockPositionValidator{needClose: false},
				position:     c.position,
				acc:          &mockAccount{balance: int(cfg.Budget)},
				report:       &mockReport{},
				indicator: &mockIndicator{
					act:        c.act,
					confidence: c.confidence,
				},
			}

			require.NoError(t, s.Run(context.Background()))
			assert.Len(t, posMan.positions, c.expectedPos)
		})
	}
}

func TestRun_closesInvalidPosition(t *testing.T) {
	cfg := config.Strategy{
		Budget:         1000,
		BuyConfidence:  0.5,
		SellConfidence: 0.5,
	}
	asset := market.NewAsset("sym", 1)
	posMan := mockPositionManager{qtyFunc: func(size decimal.Decimal, symbol string) decimal.Decimal {
		return size
	}}
	posMan.positions = []*market.Position{{Asset: asset}}
	scaler := mockPositionScaler{
		scaleFunc: func(budget decimal.Decimal, confidence float64) decimal.Decimal {
			return budget
		},
	}

	s := TradingStrategy{
		asset:        asset,
		log:          slog.Default(),
		cfg:          cfg,
		posMan:       &posMan,
		posScaler:    &scaler,
		posValidator: &mockPositionValidator{needClose: true},
		position:     posMan.positions[0],
		acc:          &mockAccount{balance: int(cfg.Budget)},
		report:       &mockReport{},
		indicator:    &mockIndicator{},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	require.NoError(t, s.Run(ctx))
	assert.Len(t, posMan.positions, 0)
	assert.Nil(t, s.position)
}

func TestBuy(t *testing.T) {
	scaler := &mockPositionScaler{
		scaleFunc: func(budget decimal.Decimal, confidence float64) decimal.Decimal {
			return budget.Mul(decimal.NewFromFloat(float64(confidence)))
		},
	}
	posMan := &mockPositionManager{qtyFunc: func(size decimal.Decimal, symbol string) decimal.Decimal {
		return size
	}}

	s := TradingStrategy{
		asset:     market.NewAsset("BTC", 1),
		posScaler: scaler,
		posMan:    posMan,
		acc:       &mockAccount{balance: 1000},
		cfg: config.Strategy{
			Budget: 1000,
		},
	}

	require.NoError(t, s.buy(context.Background(), 0.6))
	assert.Len(t, posMan.positions, 1)

	p := posMan.positions[0]
	assert.True(t, p.Qty.Round(0).Equal(decimal.NewFromInt(600)))
}

func TestSell(t *testing.T) {
	p := &market.Position{Asset: market.NewAsset("p", 1)}
	o := &market.Position{Asset: market.NewAsset("o", 1)}
	posMan := &mockPositionManager{
		positions: []*market.Position{p, o},
	}
	r := &mockReport{}
	s := TradingStrategy{
		posMan:   posMan,
		position: p,
		report:   r,
	}

	require.NoError(t, s.sell(context.Background(), 0.6))

	assert.ElementsMatch(t, []*market.Position{o}, posMan.positions)
	assert.Len(t, r.deals, 1)
	assert.Nil(t, s.position)
}

func TestGetAvailableFunds(t *testing.T) {
	tbl := []struct {
		budget    int64
		balance   int64
		spent     int64
		available int64
	}{
		{budget: 1000, balance: 10000, spent: 100, available: 900},
		{budget: 1000, balance: 10000, spent: 0, available: 1000},
		{budget: 1000, balance: 500, spent: 200, available: 500},
		{budget: 1000, balance: 10000, spent: 2000, available: 0},
		{budget: 1000, balance: 10000, spent: 2000, available: 0},
		{budget: 10000, balance: 0, spent: 2000, available: 0},
		{budget: 10000, balance: 0, spent: 0, available: 0},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			var p *market.Position
			if c.spent > 0 {
				p = &market.Position{
					EntryPrice: decimal.NewFromInt(c.spent),
				}
			}

			acc := mockAccount{balance: int(c.balance)}
			s := TradingStrategy{
				acc: &acc,
				cfg: config.Strategy{
					Budget: c.budget,
				},
				position: p,
			}

			available, err := s.getAvailableFunds()
			require.NoError(t, err)
			assert.Equal(t, decimal.NewFromInt(c.available), available)
		})
	}
}
