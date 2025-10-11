package emulator

import (
	"context"
	"fmt"
	"log/slog"
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

type account interface {
	Deposit(amount decimal.Decimal) error
	Withdraw(amount decimal.Decimal) error
}

type Deal struct {
	Symbol    string
	BuyTime   time.Time
	SellTime  time.Time
	BuyPrice  decimal.Decimal
	SellPrice decimal.Decimal
	Qty       decimal.Decimal
	Spend     decimal.Decimal
	Gain      decimal.Decimal
}

type positionManager struct {
	log       *slog.Logger
	positions map[string]market.Position
	report    reportBuilder
	prices    priceProvider
	comission comissionCharger
	acc       account
	mu        sync.Mutex
}

func newPositionManager(log *slog.Logger, comission comissionCharger, prices priceProvider, report reportBuilder, acc account) *positionManager {
	return &positionManager{
		log:       log,
		comission: comission,
		prices:    prices,
		report:    report,
		acc:       acc,
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

	if err = pm.acc.Withdraw(size); err != nil {
		err = fmt.Errorf("failed to withdraw funds: %w", err)
		return
	}

	price := size
	size = pm.comission.ApplyOnBuy(size)
	p = market.Position{
		Symbol:     symbol,
		EntryPrice: bar.Close,
		OpenTime:   bar.Time,
		Qty:        size.Div(bar.Close),
		Price:      price,
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

	before := p.Qty.Mul(p.EntryPrice)
	after := pm.comission.ApplyOnSell(p.Qty.Mul(bar.Close))
	if err = pm.acc.Deposit(after); err != nil {
		return fmt.Errorf("failed to deposit funds: %w", err)
	}

	delete(pm.positions, symbol)

	d := Deal{
		Symbol:    symbol,
		SellTime:  bar.Time,
		SellPrice: bar.Close,
		BuyTime:   p.OpenTime,
		BuyPrice:  p.EntryPrice,
		Qty:       p.Qty,
		Spend:     p.Price,
		Gain:      after.Sub(before),
	}
	pm.report.SubmitDeal(d)

	return nil
}
