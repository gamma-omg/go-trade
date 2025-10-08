package indicator

import (
	"fmt"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
)

type MACDIndicator struct {
	cfg  config.MACD
	bars barsProvider
}

func NewMACD(cfg config.MACD, bars barsProvider) *MACDIndicator {
	return &MACDIndicator{
		cfg:  cfg,
		bars: bars,
	}
}

func (i *MACDIndicator) GetSignal() (s Signal, err error) {
	s = Signal{
		Act:        ACT_HOLD,
		Confidence: 1.0,
	}

	count := max(i.cfg.Fast, i.cfg.Slow, i.cfg.Signal)
	if !i.bars.HasBars(count) {
		return
	}

	bars, err := i.bars.GetBars(count)
	if err != nil {
		err = fmt.Errorf("failed to get data for macd indicator: %w", err)
		return
	}

	macd := calcMACD(bars, i.cfg.Fast, i.cfg.Slow, i.cfg.Signal)
	last := macd[count-1]

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

	return
}

func calcMACD(bars []market.Bar, fast, slow, signal int) []float64 {
	n := len(bars)
	prices := make([]float64, n)
	for i, b := range bars {
		prices[i], _ = b.Close.Float64()
	}

	fastEma := ema(prices, fast)
	slowEma := ema(prices, slow)
	diff := make([]float64, n)
	for i := range n {
		diff[i] = fastEma[i] - slowEma[i]
	}

	signalEma := ema(diff, signal)
	macd := make([]float64, n)
	for i := 0; i < n; i++ {
		macd[i] = diff[i] - signalEma[i]
	}

	return macd
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
