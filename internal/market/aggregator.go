package market

import (
	"time"

	"github.com/shopspring/decimal"
)

type BarAggregator func(bars <-chan Bar) <-chan Bar

func IndentityAggregator() BarAggregator {
	return func(bars <-chan Bar) <-chan Bar {
		return bars
	}
}

func IntervalAggregator(barDuration, interval time.Duration) BarAggregator {
	return func(bars <-chan Bar) <-chan Bar {
		out := make(chan Bar)
		go func() {
			defer close(out)

			var cur *Bar
			var end time.Time
			for b := range bars {
				if cur != nil && !b.Time.Before(end) {
					out <- *cur
					cur = nil
				}

				if cur == nil {
					end = b.Time.Truncate(interval).Add(interval)
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

				bEnd := b.Time.Add(barDuration)
				if !bEnd.Before(end) {
					out <- *cur
					cur = nil
				}
			}

			if cur != nil {
				out <- *cur
			}
		}()

		return out
	}
}
