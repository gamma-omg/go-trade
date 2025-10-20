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
	BarDuration time.Duration
	Interval    time.Duration
}

func (a *IntervalAggregator) Aggregate(bars <-chan Bar) <-chan Bar {
	res := make(chan Bar)
	go func() {
		defer close(res)

		var cur *Bar
		var end time.Time
		for b := range bars {
			if cur != nil && !b.Time.Before(end) {
				res <- *cur
				cur = nil
			}

			if cur == nil {
				end = b.Time.Truncate(a.Interval).Add(a.Interval)
				cur = &Bar{
					Time: b.Time,
					Open: b.Open,
					High: b.High,
					Low:  b.Low,
				}
			}

			cur.Close = b.Close
			cur.High = decimal.Max(cur.High, b.High)
			cur.Low = decimal.Min(cur.Low, b.Low)
			cur.Volume = cur.Volume.Add(b.Volume)

			bEnd := b.Time.Add(a.BarDuration)
			if !bEnd.Before(end) {
				res <- *cur
				cur = nil
			}
		}

		if cur != nil {
			res <- *cur
		}
	}()

	return res
}
