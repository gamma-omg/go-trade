package indicator

import (
	"fmt"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

type RSIIndictor struct {
	cfg   config.RSI
	bars  barsProvider
	debug rsiDebugData
}

type rsiDebugData struct {
	bars []market.Bar
	rsi  []float64
}

func NewRSI(cfg config.RSI, bars barsProvider) *RSIIndictor {
	return &RSIIndictor{
		cfg:  cfg,
		bars: bars,
	}
}

func (i *RSIIndictor) GetSignal() (s Signal, err error) {
	s = Signal{ACT_HOLD, 1.0}

	if !i.bars.HasBars(i.cfg.Period) {
		return
	}

	bars, err := i.bars.GetBars(i.cfg.Period)
	if err != nil {
		err = fmt.Errorf("failed to get data for rsi indicator: %w", err)
		return
	}

	res := rs(bars)
	rsi := make([]float64, len(res))
	for i, r := range res {
		if r >= 0 {
			rsi[i] = 1 - 1/(1+r)
		} else {
			rsi[i] = 1
		}
	}

	last := rsi[len(rsi)-1]
	if last >= i.cfg.Overbought {
		s = Signal{ACT_SELL, last}
	}
	if last <= 1-i.cfg.Overbought {
		s = Signal{ACT_BUY, 1 - last}
	}

	i.debug = rsiDebugData{
		bars: bars,
		rsi:  rsi,
	}

	return
}

func (i *RSIIndictor) DrawDebug(d *DebugPlot) error {
	p := plot.New()
	p.Title.Text = "RSI"
	p.Y.Label.Text = "Signal"
	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	pts := make(plotter.XYs, len(i.debug.rsi))
	for x, v := range i.debug.rsi {
		pts[x] = plotter.XY{X: float64(i.debug.bars[x].Time.Unix()), Y: v}
	}
	plotRsi, err := plotter.NewLine(pts)
	if err != nil {
		return fmt.Errorf("failed to create rsi graph: %w", err)
	}

	p.Add(plotRsi)
	d.Add(p, 1)

	return nil
}
