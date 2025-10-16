package emulator

import (
	"context"
	"fmt"
	"log/slog"

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

type account interface {
	Deposit(amount decimal.Decimal) error
	Withdraw(amount decimal.Decimal) error
}

type positionManager struct {
	log       *slog.Logger
	prices    priceProvider
	comission comissionCharger
	acc       account
}

func newPositionManager(log *slog.Logger, comission comissionCharger, prices priceProvider, acc account) *positionManager {
	return &positionManager{
		log:       log,
		comission: comission,
		prices:    prices,
		acc:       acc,
	}
}

func (pm *positionManager) Open(ctx context.Context, symbol string, size decimal.Decimal) (p *market.Position, err error) {
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
	p = &market.Position{
		Symbol:     symbol,
		EntryPrice: bar.Close,
		OpenTime:   bar.Time,
		Qty:        size.Div(bar.Close),
		Price:      price,
	}

	return p, nil
}

func (pm *positionManager) Close(ctx context.Context, p *market.Position) (d market.Deal, err error) {
	bar, err := pm.prices.GetLastBar(p.Symbol)
	if err != nil {
		err = fmt.Errorf("cannot find sell price for %s: %w", p.Symbol, err)
		return
	}

	before := p.Qty.Mul(p.EntryPrice)
	after := pm.comission.ApplyOnSell(p.Qty.Mul(bar.Close))
	if err = pm.acc.Deposit(after); err != nil {
		err = fmt.Errorf("failed to deposit funds: %w", err)
		return
	}

	d = market.Deal{
		Symbol:    p.Symbol,
		SellTime:  bar.Time,
		SellPrice: bar.Close,
		BuyTime:   p.OpenTime,
		BuyPrice:  p.EntryPrice,
		Qty:       p.Qty,
		Spend:     p.Price,
		Gain:      after.Sub(before),
	}
	return
}
