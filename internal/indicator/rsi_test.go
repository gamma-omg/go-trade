package indicator

import (
	"fmt"
	"testing"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSI_GetSignal(t *testing.T) {
	tbl := []struct {
		bars       []float64
		period     int
		overbought float64
		signal     Signal
	}{
		{
			bars:       []float64{10, 10.5, 10.2},
			period:     5,
			overbought: 0.70,
			signal:     Signal{ActHold, 1.0},
		},
		{
			bars:       []float64{5, 5, 5, 5},
			period:     4,
			overbought: 0.70,
			signal:     Signal{ActHold, 1.0},
		},
		{
			bars:       []float64{1, 2, 3, 4},
			period:     4,
			overbought: 0.80,
			signal:     Signal{ActSell, 1.0},
		},
		{
			bars:       []float64{4, 3, 2, 1},
			period:     4,
			overbought: 0.80,
			signal:     Signal{ActBuy, 1.0},
		},
		{
			bars:       []float64{1, 2, 3, 2, 1, 2},
			period:     6,
			overbought: 179.0 / 324.1,
			signal:     Signal{ActSell, 179.0 / 324.0},
		},
		{
			bars:       []float64{1, 2, 1, 2, 1},
			period:     5,
			overbought: 0.60,
			signal:     Signal{ActHold, 1.0},
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			bars := &mockBarsProvider{c.bars}
			rsi := NewRSI(config.RSI{
				Period:     c.period,
				Overbought: c.overbought,
			}, bars)

			s, err := rsi.GetSignal()
			require.NoError(t, err)
			assert.Equal(t, c.signal.Act, s.Act)
			assert.InDelta(t, c.signal.Confidence, s.Confidence, 1e-3)
		})
	}
}
