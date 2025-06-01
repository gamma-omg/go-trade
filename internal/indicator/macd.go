package indicator

import (
	"fmt"

	"github.com/gamma-omg/trading-bot/internal/config"
)

type MACDIndicator struct {
	cfg  config.MACD
	bars barsProvider
}

func (i *MACDIndicator) GetSignal() (s Signal, err error) {
	macd, err := i.calcMACD()
	if err != nil {
		err = fmt.Errorf("failed to calculate macd: %w", err)
	}

	n := len(macd)
	last := macd[n-1]

	if last > i.cfg.BuyThreshold && hasCrossOver(macd, i.cfg.CrossLookback) {
		s = Signal{
			Act:        ACT_BUY,
			Confidence: min(1, (last-i.cfg.BuyThreshold)/(i.cfg.BuyCap-i.cfg.BuyThreshold)),
		}
		return
	}

	if last < i.cfg.SellThreshold && hasCrossOver(macd, i.cfg.CrossLookback) {
		s = Signal{
			Act:        ACT_SELL,
			Confidence: min(1, (last-i.cfg.SellThreshold)/(i.cfg.SellCap-i.cfg.SellThreshold)),
		}
		return
	}

	s = Signal{
		Act:        ACT_HOLD,
		Confidence: 1.0,
	}
	return
}

func (i *MACDIndicator) calcMACD() (macd []float64, err error) {
	count := max(i.cfg.Fast, i.cfg.Slow, i.cfg.Signal)
	if !i.bars.HasBars(count) {
		err = fmt.Errorf("insufficient data: requires at least %d bars", count)
		return
	}

	bars, err := i.bars.GetBars(count)
	if err != nil {
		err = fmt.Errorf("failed to get bars data: %w", err)
		return
	}

	prices := make([]float64, count)
	for i, b := range bars {
		prices[i], _ = b.Close.Float64()
	}

	fast := ema(prices, i.cfg.Fast)
	slow := ema(prices, i.cfg.Slow)
	diff := make([]float64, count)
	for i := range count {
		diff[i] = fast[i] - slow[i]
	}

	signal := ema(diff, i.cfg.Signal)
	macd = make([]float64, count)
	for i := 0; i < count; i++ {
		macd[i] = diff[i] - signal[i]
	}

	return
}

func hasCrossOver(macd []float64, lookback int) bool {
	l := len(macd)
	if l < 2 {
		return false
	}

	n := min(lookback, l)
	for i := 1; i <= n; i++ {
		next := macd[l-i]
		prev := macd[l-i-1]
		if prev < 0 && next > 0 || prev > 0 && next < 0 {
			return true
		}
	}

	return false
}
