package agent

import (
	"fmt"

	"github.com/gamma-omg/trading-bot/internal/market"
)

type defaultPositionValidator struct {
	takeProfit float64
	stopLoss   float64
}

func (v *defaultPositionValidator) NeedClose(p *market.Position) (bool, error) {
	bar, err := p.Asset.GetLastBar()
	if err != nil {
		return false, fmt.Errorf("failed to get price for asset %s: %w", p.Asset.Symbol, err)
	}

	pct, _ := bar.Close.Div(p.EntryPrice).Float64()
	return pct >= v.takeProfit || pct <= v.stopLoss, nil
}
