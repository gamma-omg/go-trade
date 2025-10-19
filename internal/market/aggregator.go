package market

import (
	"time"

	"github.com/shopspring/decimal"
)

type IdentityAggregator struct {
}

func (a *IdentityAggregator) Aggregate(bars <-chan Bar) <-chan Bar {
	return bars
}

type IntervalAggregator struct {
	Interval time.Duration
	cur      *Bar
}

func (a *IntervalAggregator) Aggregate(bars <-chan Bar) <-chan Bar {
	res := make(chan Bar)
	go func() {
		defer close(res)
		for b := range bars {
			if a.cur == nil {
				a.cur = &Bar{
					Time: b.Time,
					Open: b.Open,
					High: b.High,
					Low:  b.Low,
				}
			}

			a.cur.Close = b.Close
			a.cur.High = decimal.Max(a.cur.High, b.High)
			a.cur.Low = decimal.Min(a.cur.Low, b.Low)
			a.cur.Volume = a.cur.Volume.Add(b.Volume)

			if b.Time.Add(1*time.Minute).Sub(a.cur.Time) >= a.Interval {
				res <- *a.cur
				a.cur = nil
			}
		}
	}()

	return res
}
