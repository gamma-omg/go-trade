package indicator

import (
	"github.com/gamma-omg/trading-bot/internal/market"
)

type barsProvider interface {
	GetBars(count int) ([]market.Bar, error)
	HasBars(count int) bool
}

func ema(data []float64, period int) []float64 {
	if len(data) < period {
		panic("not enough data to compute ema")
	}

	ema := make([]float64, len(data))
	ema[0] = data[0]

	a := 2.0 / (float64(period) + 1)
	for i, val := range data[1:] {
		ema[i+1] = val*a + ema[i]*(1-a)
	}

	return ema
}

func rs(bars []market.Bar) []float64 {
	n := len(bars)
	if n < 2 {
		return []float64{}
	}

	g := make([]float64, n-1)
	l := make([]float64, n-1)
	avgG := make([]float64, n-1)
	avgL := make([]float64, n-1)

	prev := bars[0]
	for i, cur := range bars[1:] {
		diff, _ := cur.Close.Sub(prev.Close).Float64()
		if diff > 0 {
			g[i] = diff
			l[i] = 0
			avgG[0] += diff
		} else {
			g[i] = 0
			l[i] = -diff
			avgL[0] += -diff
		}

		prev = cur
	}

	avgG[0] /= float64(len(g))
	avgL[0] /= float64(len(l))

	floatLen := float64(n)
	for i, v := range g[1:] {
		avgG[i+1] = (avgG[i]*(floatLen-1) + v) / floatLen
	}
	for i, v := range l[1:] {
		avgL[i+1] = (avgL[i]*(floatLen-1) + v) / floatLen
	}

	rs := make([]float64, n-1)
	for i := 0; i < n-1; i++ {
		if avgG[i] == 0 && avgL[i] == 0 {
			rs[i] = 1
		} else if avgL[i] == 0 {
			rs[i] = -1
		} else {
			rs[i] = avgG[i] / avgL[i]
		}
	}

	return rs
}
