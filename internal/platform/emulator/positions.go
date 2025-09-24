package emulator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type comissionCharger interface {
	ApplyOnBuy(decimal.Decimal) decimal.Decimal
	ApplyOnSell(decimal.Decimal) decimal.Decimal
}

type priceProvider interface {
	GetLastBar(symbol string) (market.Bar, error)
}

type reportBuilder interface {
	SubmitDeal(d Deal)
}

type Deal struct {
	Symbol    string
	BuyTime   time.Time
	SellTime  time.Time
	BuyPrice  decimal.Decimal
	SellPrice decimal.Decimal
	Gain      decimal.Decimal
	GainPct   float64
}

type positionManager struct {
	positions map[string]market.Position
	report    reportBuilder
	prices    priceProvider
	comission comissionCharger
	mu        sync.Mutex
}

func newPositionManager(comission comissionCharger, prices priceProvider, report reportBuilder) *positionManager {
	return &positionManager{
		comission: comission,
		prices:    prices,
		report:    report,
		positions: make(map[string]market.Position),
	}
}

func (pm *positionManager) Open(ctx context.Context, symbol string, size decimal.Decimal) (p market.Position, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, ok := pm.positions[symbol]; ok {
		err = fmt.Errorf("position for %s is already open", symbol)
		return
	}

	bar, err := pm.prices.GetLastBar(symbol)
	if err != nil {
		err = fmt.Errorf("cannot find buy price for %s: %w", symbol, err)
		return
	}

	size = pm.comission.ApplyOnBuy(size)
	p = market.Position{
		Symbol:     symbol,
		EntryPrice: bar.Close,
		OpenTime:   bar.Time,
		Qty:        size.Div(bar.Close),
	}
	pm.positions[symbol] = p
	return p, nil
}

func (pm *positionManager) Close(ctx context.Context, symbol string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	p, ok := pm.positions[symbol]
	if !ok {
		return fmt.Errorf("no open positions for symbol %s", symbol)
	}

	bar, err := pm.prices.GetLastBar(symbol)
	if err != nil {
		return fmt.Errorf("cannot find sell price for %s: %w", symbol, err)
	}

	delete(pm.positions, symbol)

	before := p.Qty.Mul(p.EntryPrice)
	after := pm.comission.ApplyOnSell(bar.Close)
	pct, _ := after.Div(before).Float64()
	d := Deal{
		Symbol:    symbol,
		SellTime:  bar.Time,
		SellPrice: bar.Close,
		BuyTime:   p.OpenTime,
		BuyPrice:  p.EntryPrice,
		Gain:      after.Sub(before),
		GainPct:   pct,
	}
	pm.report.SubmitDeal(d)
	return nil
}
