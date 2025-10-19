package indicator

import (
	"fmt"
	"math"
	"testing"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMACD_GetSignal(t *testing.T) {
	tbl := []struct {
		prices        []float64
		fast          int
		slow          int
		signal        int
		lookback      int
		buyThreshold  float64
		buyCap        float64
		sellThreshold float64
		sellCap       float64
		out           Signal
	}{
		{
			prices:        []float64{1, 1, 1, 1, 1},
			fast:          3,
			slow:          5,
			signal:        4,
			lookback:      1,
			buyThreshold:  0.3,
			buyCap:        0.7,
			sellThreshold: -0.3,
			sellCap:       -0.7,
			out: Signal{
				Act:        ACT_HOLD,
				Confidence: 1.0,
			},
		},
		{
			prices:        []float64{13, 6, 14, 5, 14},
			fast:          3,
			slow:          5,
			signal:        4,
			lookback:      1,
			buyThreshold:  0.3,
			buyCap:        0.7,
			sellThreshold: -0.3,
			sellCap:       -0.7,
			out: Signal{
				Act:        ACT_BUY,
				Confidence: 0.4608425925925947,
			},
		},
		{
			prices:        []float64{107, 110, 108, 111, 115, 101},
			fast:          4,
			slow:          6,
			signal:        3,
			lookback:      1,
			buyThreshold:  0.3,
			buyCap:        0.7,
			sellThreshold: -0.3,
			sellCap:       -0.7,
			out: Signal{
				Act:        ACT_SELL,
				Confidence: 0.7042627893139927,
			},
		},
		{
			prices:        []float64{5, 4, 6, 3, 30},
			fast:          3,
			slow:          5,
			signal:        4,
			lookback:      1,
			buyThreshold:  0.3,
			buyCap:        0.7,
			sellThreshold: -0.3,
			sellCap:       -0.7,
			out: Signal{
				Act:        ACT_BUY,
				Confidence: 1.0, // should NOT exceed +1
			},
		},
		{
			prices:        []float64{100, 102, 105, 103, 125, 30},
			fast:          4,
			slow:          6,
			signal:        3,
			lookback:      1,
			buyThreshold:  0.3,
			buyCap:        0.7,
			sellThreshold: -0.3,
			sellCap:       -0.7,
			out: Signal{
				Act:        ACT_SELL,
				Confidence: 1.0, // should NOT exceed +1
			},
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			ind := MACDIndicator{
				cfg: config.MACD{
					Fast:          c.fast,
					Slow:          c.slow,
					Signal:        c.signal,
					BuyThreshold:  c.buyThreshold,
					BuyCap:        c.buyCap,
					SellThreshold: c.sellThreshold,
					SellCap:       c.sellCap,
					CrossLookback: c.lookback,
					EmaWarmup:     1,
				},
				bars: &mockBarsProvider{closePrices: c.prices},
			}

			s, err := ind.GetSignal()
			require.NoError(t, err)
			assert.Equal(t, c.out, s)
		})
	}
}

func TestMACD_calcMACD(t *testing.T) {
	tbl := []struct {
		prices  []float64
		fast    int
		slow    int
		signal  int
		macd    []float64
		epsilon float64
	}{
		{
			prices: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			fast:   3,
			slow:   5,
			signal: 4,
			macd: []float64{
				0.0,
				0.1000000000000000,
				0.1766666666666666,
				0.2087777777777778,
				0.2062851851851852,
			},
			epsilon: 1e-6,
		},
		{
			prices: []float64{10, 9, 11, 8, 12, 7, 13, 6, 14, 5},
			fast:   3,
			slow:   5,
			signal: 4,
			macd: []float64{
				0.0,
				0.6,
				-0.24,
				0.456,
				-0.5264,
			},
			epsilon: 1e-6,
		},
		{
			prices: []float64{100, 102, 105, 103, 107, 110, 108, 111, 115, 117},
			fast:   4,
			slow:   6,
			signal: 3,
			macd: []float64{
				0.0,
				0.17142857142857082,
				0.02530612244898478,
				0.14550437317784493,
				0.3303888379841746,
				0.3325805985601282,
			},
			epsilon: 1e-6,
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			count := max(c.fast, c.slow, c.signal)
			barsProvider := mockBarsProvider{c.prices}
			bars, err := barsProvider.GetBars(count)
			require.NoError(t, err)

			macd := calcMACD(bars, c.fast, c.slow, c.signal)
			require.Len(t, macd, count)

			for j, v := range macd {
				if math.Abs(v-c.macd[j]) > c.epsilon {
					t.Errorf("invalid macd element at position %d: expected %f, got %f", j, c.macd[j], v)
				}
			}
		})
	}
}

func TestMACD_hasCrossOver(t *testing.T) {
	tbl := []struct {
		data      []float64
		lookback  int
		crossover bool
	}{
		{data: []float64{}, lookback: 0, crossover: false},
		{data: []float64{}, lookback: 100, crossover: false},
		{data: []float64{-1, 1}, lookback: 1, crossover: true},
		{data: []float64{1, -1}, lookback: 1, crossover: true},
		{data: []float64{1, -1, 2, 3}, lookback: 1, crossover: false},
		{data: []float64{1, -1, 2, 3}, lookback: 2, crossover: true},
		{data: []float64{1, 10, -2, -3}, lookback: 1, crossover: false},
		{data: []float64{1, 10, -2, -3}, lookback: 2, crossover: true},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			assert.Equal(t, c.crossover, hasCrossOver(c.data, c.lookback))
		})
	}
}

func TestMACD_holdWhenNotEnoughData(t *testing.T) {
	barsProvider := mockBarsProvider{closePrices: []float64{1, 2, 3, 4}}
	ind := MACDIndicator{
		cfg: config.MACD{
			Fast:      8,
			Slow:      12,
			Signal:    10,
			EmaWarmup: 1,
		},
		bars: &barsProvider,
	}

	s, err := ind.GetSignal()
	require.NoError(t, err)

	assert.Equal(t, Signal{ACT_HOLD, 1.0}, s)
}
