package agent

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"testing"

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

func (a *mockAccount) GetBalance() decimal.Decimal {
	return decimal.NewFromInt(int64(a.balance))
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

func (pm *mockPositionManager) Open(_ context.Context, symbol string, size decimal.Decimal) (*market.Position, error) {
	pos := &market.Position{Qty: pm.qtyFunc(size, symbol)}
	pm.positions = append(pm.positions, pos)
	return pos, nil
}

func (pm *mockPositionManager) Close(_ context.Context, pos *market.Position) error {
	pm.positions = slices.DeleteFunc(pm.positions, func(p *market.Position) bool {
		return p == pos
	})

	return nil
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
		{act: indicator.ACT_HOLD, confidence: 1.0, initialPos: 5, expectedPos: 5},
		{act: indicator.ACT_BUY, confidence: 0.4, initialPos: 5, expectedPos: 5},
		{act: indicator.ACT_BUY, confidence: 0.6, initialPos: 5, expectedPos: 6},
		{act: indicator.ACT_SELL, confidence: 0.4, initialPos: 5, expectedPos: 5},
		{act: indicator.ACT_SELL, confidence: 0.6, initialPos: 5, expectedPos: 5},
		{act: indicator.ACT_SELL, confidence: 0.6, initialPos: 5, expectedPos: 4, position: &market.Position{}},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			posMan := mockPositionManager{qtyFunc: func(size decimal.Decimal, symbol string) decimal.Decimal {
				return size
			}}
			if c.position != nil {
				posMan.positions = append(posMan.positions, c.position)
			}
			for len(posMan.positions) < c.initialPos {
				posMan.positions = append(posMan.positions, &market.Position{})
			}

			scaler := mockPositionScaler{
				scaleFunc: func(budget decimal.Decimal, confidence float64) decimal.Decimal {
					return budget
				},
			}

			s := TradingStrategy{
				log:       slog.Default(),
				cfg:       cfg,
				posMan:    &posMan,
				posScaler: &scaler,
				position:  c.position,
				acc:       &mockAccount{balance: int(cfg.Budget)},
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
	p := &market.Position{}
	o := &market.Position{}
	posMan := mockPositionManager{
		positions: []*market.Position{p, o},
	}
	s := TradingStrategy{
		posMan:   &posMan,
		position: p,
	}

	require.NoError(t, s.sell(context.Background(), 0.6))

	assert.ElementsMatch(t, []*market.Position{o}, posMan.positions)
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

			available := s.getAvailableFunds()
			assert.Equal(t, decimal.NewFromInt(c.available), available)
		})
	}
}
