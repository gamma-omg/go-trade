package indicator

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockTradingIndicator struct {
	signal Signal
	err    error
}

func (m *mockTradingIndicator) GetSignal() (Signal, error) {
	return m.signal, m.err
}

func Test_Ensemble_GetSignal(t *testing.T) {
	tbl := []struct {
		children []WeightedIndicator
		out      Signal
		err      error
	}{
		// case 0
		{
			children: []WeightedIndicator{
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_SELL, Confidence: 1.0},
					},
				},
			},
			out: Signal{Act: ACT_HOLD, Confidence: 1.0},
		},

		// case 1
		{
			children: []WeightedIndicator{
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
			},
			out: Signal{Act: ACT_BUY, Confidence: 1.0},
		},

		// case 2
		{
			children: []WeightedIndicator{
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
				{
					Weight: 0.1, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
			},
			out: Signal{Act: ACT_BUY, Confidence: 1.0},
		},

		// case 3
		{
			children: []WeightedIndicator{
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: .5},
					},
				},
			},
			out: Signal{Act: ACT_BUY, Confidence: 0.75},
		},

		// case 4
		{
			children: []WeightedIndicator{
				{
					Weight: 0.9, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
				{
					Weight: 0.1, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: .5},
					},
				},
			},
			out: Signal{Act: ACT_BUY, Confidence: 0.95},
		},

		// case 5
		{
			children: []WeightedIndicator{
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_SELL, Confidence: 1.0},
					},
				},
				{
					Weight: 1.0, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_SELL, Confidence: 1.0},
					},
				},
			},
			out: Signal{Act: ACT_SELL, Confidence: 1.0},
		},

		// case 6
		{
			children: []WeightedIndicator{
				{
					Weight: 0.1, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_SELL, Confidence: 1.0},
					},
				},
				{
					Weight: 0.9, Indicator: &mockTradingIndicator{
						signal: Signal{Act: ACT_BUY, Confidence: 1.0},
					},
				},
			},
			out: Signal{Act: ACT_BUY, Confidence: 0.8},
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			i := EnsembleIndicator{children: c.children}
			s, err := i.GetSignal()
			assert.Equal(t, err, c.err)
			assert.Equal(t, s.Act, c.out.Act)
			assert.True(t, math.Abs(s.Confidence-c.out.Confidence) < 1e-4)
		})
	}
}
