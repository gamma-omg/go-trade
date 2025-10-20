package market

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetGetBars(t *testing.T) {
	tbl := []struct {
		bars  []float64
		head  int
		count int
		out   []float64
		err   bool
	}{
		{
			bars:  []float64{1, 2, 3, 4, 5, 6},
			head:  5,
			count: 1,
			out:   []float64{6},
			err:   false,
		},
		{
			bars:  []float64{-1, -2, -3, -4, -5, -6},
			head:  5,
			count: 3,
			out:   []float64{-4, -5, -6},
			err:   false,
		},
		{
			bars:  []float64{10, -10, 20, -20, 30, -30, 40, -40},
			head:  7,
			count: 8,
			out:   []float64{10, -10, 20, -20, 30, -30, 40, -40},
			err:   false,
		},
		{
			bars:  []float64{1, 2, 3},
			head:  2,
			count: 4,
			out:   []float64{},
			err:   true,
		},
		{
			bars:  []float64{1, 2, 3, 4, 5, 6},
			head:  5,
			count: 4,
			out:   []float64{3, 4, 5, 6},
			err:   false,
		},
		{
			bars:  []float64{1, 2, 3, 4, 5, 6},
			head:  2,
			count: 3,
			out:   []float64{1, 2, 3},
			err:   false,
		},
		{
			bars:  []float64{1, 2, 3, 4, 5, 6},
			head:  7,
			count: 3,
			out:   []float64{6, 1, 2},
			err:   false,
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
				Symbol: fmt.Sprintf("s%d", i),
				bars:   in,
				head:   c.head,
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

func TestGetBars_invalidArgs(t *testing.T) {
	a := Asset{}

	_, err := a.GetBars(-10)
	require.Error(t, err)

	_, err = a.GetBars(0)
	require.Error(t, err)
}

func TestGetLastBar(t *testing.T) {
	tbl := []struct {
		bars []float64
		head int
		ans  float64
		err  bool
	}{
		{
			bars: []float64{},
			head: -1,
			ans:  0,
			err:  true,
		},
		{
			bars: []float64{1},
			head: 0,
			ans:  1,
			err:  false,
		},
		{
			bars: []float64{1, 2, 3},
			head: 1,
			ans:  2,
			err:  false,
		},
		{
			bars: []float64{1, 2, 3},
			head: 3,
			ans:  1,
			err:  false,
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			bars := make([]Bar, len(c.bars))
			for i, v := range c.bars {
				bars[i] = Bar{Close: decimal.NewFromFloat(v)}
			}

			a := Asset{
				bars: bars,
				head: c.head,
				size: len(bars),
			}

			b, err := a.GetLastBar()
			assert.Equal(t, c.err, err != nil)
			assert.True(t, decimal.NewFromFloat(c.ans).Equal(b.Close))
		})
	}
}

func TestAssetHasBars(t *testing.T) {
	tbl := []struct {
		bars  []float64
		head  int
		count int
		out   bool
	}{
		{bars: []float64{1, 2, 3, 4}, head: 3, count: 2, out: true},
		{bars: []float64{1, 2, 3, 4}, head: 3, count: 0, out: true},
		{bars: []float64{1, 2, 3, 4}, head: 3, count: 4, out: true},
		{bars: []float64{1, 2, 3, 4}, head: 3, count: 5, out: false},
		{bars: []float64{1, 2, 3, 4}, head: 3, count: 10, out: false},
		{bars: []float64{1, 2, 3, 4}, head: 10, count: 10, out: true},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			in := make([]Bar, len(c.bars))
			for i, v := range c.bars {
				in[i] = Bar{Close: decimal.NewFromFloat(v)}
			}

			a := &Asset{
				Symbol: fmt.Sprintf("s%d", i),
				bars:   in,
				head:   c.head,
				size:   len(in),
			}

			assert.Equal(t, c.out, a.HasBars(c.count))
		})
	}
}

func TestAssetReceive(t *testing.T) {
	a := NewAsset("a", 3)

	b1 := Bar{}
	b2 := Bar{}
	b3 := Bar{}
	a.Receive(b1)
	a.Receive(b2)
	a.Receive(b3)
	assert.Equal(t, a.bars[:3], []Bar{b1, b2, b3})

	b4 := Bar{}
	b5 := Bar{}
	a.Receive(b4)
	a.Receive(b5)
	assert.Equal(t, a.bars[:3], []Bar{b4, b5, b3})
}
