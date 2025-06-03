package indicator

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEma(t *testing.T) {
	tbl := []struct {
		data    []float64
		ema     []float64
		period  int
		epsilon float64
	}{
		{
			data:    []float64{2, 4, 6, 8, 12, 14, 16, 18, 20},
			ema:     []float64{2, 3.333, 5.111, 7.037, 10.346, 12.782, 14.927, 16.976, 18.992},
			period:  2,
			epsilon: 0.001,
		},
		{
			data:    []float64{6, 7, 11, 4, 5, 6, 10, 12, 7, 13},
			ema:     []float64{6, 6.5, 8.75, 6.375, 5.688, 5.844, 7.922, 9.961, 8.48, 10.74},
			period:  3,
			epsilon: 0.001,
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			actual := ema(c.data, c.period)
			require.Len(t, actual, len(c.ema))

			for i, v := range actual {
				if math.Abs(v-c.ema[i]) > c.epsilon {
					t.Errorf("invalid ema component at %d: expected: %f got: %f ", i, c.ema[i], v)
				}
			}
		})
	}
}
