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

type account interface {
	Deposit(amount decimal.Decimal) error
	Withdraw(amount decimal.Decimal) error
}

type positionManager struct {
	log       *slog.Logger
	comission comissionCharger
	acc       account
}

func newPositionManager(log *slog.Logger, comission comissionCharger, acc account) *positionManager {
	return &positionManager{
		log:       log,
		comission: comission,
		acc:       acc,
	}
}

func (pm *positionManager) Open(_ context.Context, asset *market.Asset, size decimal.Decimal) (p *market.Position, err error) {
	bar, err := asset.GetLastBar()
	if err != nil {
		err = fmt.Errorf("cannot find buy price for %s: %w", asset.Symbol, err)
		return
	}

	if err = pm.acc.Withdraw(size); err != nil {
		err = fmt.Errorf("failed to withdraw funds: %w", err)
		return
	}

	price := size
	size = pm.comission.ApplyOnBuy(size)
	p = &market.Position{
		Asset:      asset,
		EntryPrice: bar.Close,
		OpenTime:   bar.Time,
		Qty:        size.Div(bar.Close),
		Price:      price,
	}

	return p, nil
}

func (pm *positionManager) Close(_ context.Context, p *market.Position) (d market.Deal, err error) {
	bar, err := p.Asset.GetLastBar()
	if err != nil {
		err = fmt.Errorf("cannot find sell price for %s: %w", p.Asset.Symbol, err)
		return
	}

	before := p.Qty.Mul(p.EntryPrice)
	after := pm.comission.ApplyOnSell(p.Qty.Mul(bar.Close))
	if err = pm.acc.Deposit(after); err != nil {
		err = fmt.Errorf("failed to deposit funds: %w", err)
		return
	}

	d = market.Deal{
		Symbol:    p.Asset.Symbol,
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
