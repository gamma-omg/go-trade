package market

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarketGetAsset(t *testing.T) {
	a1 := &Asset{symbol: "a1"}
	a2 := &Asset{symbol: "a2"}
	m := Market{
		assets: map[string]*Asset{
			"a1": a1,
			"a2": a2,
		},
		bufSize: 10,
	}

	assert.Equal(t, a1, m.GetAsset("a1"))
	assert.Equal(t, a2, m.GetAsset("a2"))

	a3 := m.GetAsset("a3")
	assert.NotNil(t, a3)
	assert.Equal(t, "a3", a3.symbol)
	assert.Equal(t, m.bufSize, a3.size)
}

func TestAssetGetBars(t *testing.T) {
	tbl := []struct {
		bars  []float64
		count int
		out   []float64
		err   bool
	}{
		{
			bars:  []float64{1, 2, 3, 4, 5, 6},
			count: 1,
			out:   []float64{6},
			err:   false,
		},
		{
			bars:  []float64{-1, -2, -3, -4, -5, -6},
			count: 3,
			out:   []float64{-4, -5, -6},
			err:   false,
		},
		{
			bars:  []float64{10, -10, 20, -20, 30, -30, 40, -40},
			count: 8,
			out:   []float64{10, -10, 20, -20, 30, -30, 40, -40},
			err:   false,
		},
		{
			bars:  []float64{1, 2, 3},
			count: 4,
			out:   []float64{},
			err:   true,
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			in := make([]Bar, len(c.bars))
			for i, v := range c.bars {
				in[i] = Bar{Close: decimal.NewFromFloat(v)}
			}

			out := make([]Bar, len(c.out))
			for i, v := range c.out {
				out[i] = Bar{Close: decimal.NewFromFloat(v)}
			}

			a := Asset{
				symbol: fmt.Sprintf("s%d", i),
				bars:   in,
				head:   len(in) - 1,
				size:   len(in),
			}

			bars, err := a.GetBars(c.count)
			if c.err {
				require.Error(t, err)
				return
			}

			assert.ElementsMatch(t, bars, out)
		})
	}
}

func TestAssetHasBars(t *testing.T) {
	tbl := []struct {
		bars  []float64
		count int
		out   bool
	}{
		{bars: []float64{1, 2, 3, 4}, count: 2, out: true},
		{bars: []float64{1, 2, 3, 4}, count: 0, out: true},
		{bars: []float64{1, 2, 3, 4}, count: 4, out: true},
		{bars: []float64{1, 2, 3, 4}, count: 5, out: false},
		{bars: []float64{1, 2, 3, 4}, count: 10, out: false},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			in := make([]Bar, len(c.bars))
			for i, v := range c.bars {
				in[i] = Bar{Close: decimal.NewFromFloat(v)}
			}

			a := &Asset{
				symbol: fmt.Sprintf("s%d", i),
				bars:   in,
				head:   len(in) - 1,
				size:   len(in),
			}

			assert.Equal(t, c.out, a.HasBars(c.count))
		})
	}
}

func Test_Asset_Receive(t *testing.T) {
	a := newAsset("a", 3)

	b1 := Bar{}
	b2 := Bar{}
	b3 := Bar{}
	a.Receive(b1)
	a.Receive(b2)
	a.Receive(b3)
	assert.Equal(t, a.bars, []Bar{b1, b2, b3})

	b4 := Bar{}
	b5 := Bar{}
	a.Receive(b4)
	a.Receive(b5)
	assert.Equal(t, a.bars, []Bar{b4, b5, b3})
}
