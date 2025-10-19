package market

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type testBar struct {
	time time.Time
	o    float64
	h    float64
	l    float64
	c    float64
	v    float64
}

func newTestBar(b Bar) testBar {
	o, _ := b.Open.Float64()
	h, _ := b.High.Float64()
	l, _ := b.Low.Float64()
	c, _ := b.Close.Float64()
	v, _ := b.Volume.Float64()
	return testBar{b.Time, o, h, l, c, v}
}

func (b *testBar) ToBar() Bar {
	return Bar{
		Time:   b.time,
		Open:   decimal.NewFromFloat(b.o),
		High:   decimal.NewFromFloat(b.h),
		Low:    decimal.NewFromFloat(b.l),
		Close:  decimal.NewFromFloat(b.c),
		Volume: decimal.NewFromFloat(b.v),
	}
}

func TestAggregate(t *testing.T) {
	tbl := []struct {
		interval time.Duration
		in       []testBar
		out      []testBar
	}{
		{
			interval: 3 * time.Minute,
			in: []testBar{
				{time: time.Unix(1, 0), o: 1, h: 3, l: 1, c: 2, v: 1},
				{time: time.Unix(2, 0), o: 3, h: 5, l: 3, c: 4, v: 2},
				{time: time.Unix(3, 0), o: 4, h: 4, l: 2, c: 3, v: 3},
				{time: time.Unix(4, 0), o: 9, h: 9, l: 9, c: 9, v: 9},
			},
			out: []testBar{
				{time: time.Unix(1, 0), o: 1, h: 5, l: 1, c: 3, v: 6},
			},
		},
		{
			interval: 3 * time.Minute,
			in: []testBar{
				{time: time.Unix(1, 0), o: 10, h: 12, l: 9, c: 11, v: 100},
				{time: time.Unix(2, 0), o: 11, h: 13, l: 10, c: 12, v: 200},
				{time: time.Unix(3, 0), o: 12, h: 12.5, l: 11, c: 11.5, v: 150},
				{time: time.Unix(4, 0), o: 20, h: 21, l: 19, c: 20.5, v: 300},
				{time: time.Unix(5, 0), o: 20.5, h: 22, l: 20, c: 21, v: 100},
				{time: time.Unix(6, 0), o: 21, h: 21.5, l: 20.5, c: 21.2, v: 50},
			},
			out: []testBar{
				{time: time.Unix(1, 0), o: 10, h: 13, l: 9, c: 11.5, v: 450},
				{time: time.Unix(4, 0), o: 20, h: 22, l: 19, c: 21.2, v: 450},
			},
		},
		{
			interval: 3 * time.Minute,
			in: []testBar{
				{time: time.Unix(1, 0), o: 5, h: 6, l: 5, c: 5.5, v: 10},
				{time: time.Unix(2, 0), o: 5.5, h: 7, l: 5.5, c: 6.5, v: 20},
				{time: time.Unix(7, 0), o: 8, h: 9, l: 7.5, c: 8.5, v: 30},
				{time: time.Unix(8, 0), o: 8.5, h: 9.5, l: 8, c: 9, v: 40},
				{time: time.Unix(9, 0), o: 9, h: 10, l: 8.8, c: 9.2, v: 50},
			},
			out: []testBar{
				{time: time.Unix(1, 0), o: 5, h: 7, l: 5, c: 6.5, v: 30},
				{time: time.Unix(7, 0), o: 8, h: 10, l: 7.5, c: 9.2, v: 120},
			},
		},
		{
			interval: 3 * time.Minute,
			in: []testBar{
				{time: time.Unix(0, 0), o: 1, h: 2, l: 1, c: 2, v: 1},
				{time: time.Unix(3, 0), o: 2, h: 3, l: 2, c: 3, v: 2},
				{time: time.Unix(6, 0), o: 3, h: 4, l: 3, c: 3.5, v: 3},
			},
			out: []testBar{
				{time: time.Unix(0, 0), o: 1, h: 2, l: 1, c: 2, v: 1},
				{time: time.Unix(3, 0), o: 2, h: 3, l: 2, c: 3, v: 2},
			},
		},
		{
			interval: 3 * time.Minute,
			in: []testBar{
				{time: time.Unix(100, 0), o: 1, h: 1, l: 1, c: 1, v: 1},
			},
			out: []testBar{},
		},
		{
			interval: 3 * time.Minute,
			in: []testBar{
				{time: time.Unix(10, 0), o: 100, h: 101, l: 99.5, c: 100.5, v: 10},
				{time: time.Unix(11, 0), o: 100.5, h: 102, l: 98, c: 99, v: 20},
				{time: time.Unix(12, 0), o: 99, h: 100, l: 97.5, c: 98, v: 30},
				{time: time.Unix(13, 0), o: 98, h: 98.5, l: 96, c: 97, v: 40},
				{time: time.Unix(14, 0), o: 97, h: 99, l: 95.5, c: 98.5, v: 50},
				{time: time.Unix(15, 0), o: 98.5, h: 100, l: 97, c: 99.5, v: 60},
				{time: time.Unix(16, 0), o: 99.5, h: 101, l: 98.5, c: 100, v: 70},
			},
			out: []testBar{
				{time: time.Unix(10, 0), o: 100, h: 102, l: 97.5, c: 98, v: 60},
				{time: time.Unix(13, 0), o: 98, h: 100, l: 95.5, c: 99.5, v: 150},
			},
		},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			a := IntervalAggregator{Interval: c.interval}
			in := make(chan Bar, len(c.in))

			var out []testBar
			go func() {
				for b := range a.Aggregate(in) {
					out = append(out, newTestBar(b))
				}
			}()

			for _, b := range c.in {
				in <- b.ToBar()
			}
			close(in)

			assert.InDeltaSlice(t, c.out, out, 1e-3)
		})
	}

}

func TestAggregate_continuesAggregation(t *testing.T) {
	agg := IntervalAggregator{Interval: 3 * time.Minute}
	in := make(chan Bar, 2)
	in <- (&testBar{time: time.Unix(1, 0), o: 1, h: 3, l: 1, c: 2, v: 1}).ToBar()
	in <- (&testBar{time: time.Unix(2, 0), o: 3, h: 5, l: 3, c: 4, v: 2}).ToBar()
	close(in)

	var out []Bar
	for b := range agg.Aggregate(in) {
		out = append(out, b)
	}

	assert.Empty(t, out)

	in = make(chan Bar, 1)
	in <- (&testBar{time: time.Unix(3, 0), o: 4, h: 4, l: 2, c: 3, v: 3}).ToBar()
	close(in)

	for b := range agg.Aggregate(in) {
		out = append(out, b)
	}

	assert.InDeltaSlice(t, []testBar{{time: time.Unix(1, 0), o: 1, h: 5, l: 1, c: 3, v: 6}}, out, 1e-3)
}
