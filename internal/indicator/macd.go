package indicator

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/pplcc/plotext"
	"github.com/pplcc/plotext/custplotter"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

type MACDIndicator struct {
	cfg  config.MACD
	bars barsProvider
}

func NewMACD(cfg config.MACD, bars barsProvider) *MACDIndicator {
	return &MACDIndicator{
		cfg:  cfg,
		bars: bars,
	}
}

func (i *MACDIndicator) GetSignal() (s Signal, err error) {
	s = Signal{
		Act:        ACT_HOLD,
		Confidence: 1.0,
	}

	count := i.getRequiredDataPoints()
	if !i.bars.HasBars(count) {
		return
	}

	bars, err := i.bars.GetBars(count)
	if err != nil {
		err = fmt.Errorf("failed to get data for macd indicator: %w", err)
		return
	}

	macd := calcMACD(bars, i.cfg.Fast, i.cfg.Slow, i.cfg.Signal)
	last := macd[count-1]

	if last > i.cfg.BuyThreshold && hasCrossOver(macd, i.cfg.CrossLookback) {
		s = Signal{
			Act:        ACT_BUY,
			Confidence: min(1, (last-i.cfg.BuyThreshold)/(i.cfg.BuyCap-i.cfg.BuyThreshold)),
		}

		if i.cfg.DebugLevel >= config.DebugBuyOrSell {
			if err = i.drawDebug(bars, macd, s); err != nil {
				err = fmt.Errorf("failed to debug macd indicator on buy: %w", err)
				return
			}
		}
		return
	}

	if last < i.cfg.SellThreshold && hasCrossOver(macd, i.cfg.CrossLookback) {
		s = Signal{
			Act:        ACT_SELL,
			Confidence: min(1, (last-i.cfg.SellThreshold)/(i.cfg.SellCap-i.cfg.SellThreshold)),
		}

		if i.cfg.DebugLevel >= config.DebugBuyOrSell {
			if err = i.drawDebug(bars, macd, s); err != nil {
				err = fmt.Errorf("failed to debug macd indicator on sell: %w", err)
				return
			}
		}
		return
	}

	if i.cfg.DebugLevel >= config.DebugAll {
		if err = i.drawDebug(bars, macd, s); err != nil {
			err = fmt.Errorf("failed to debug macd indicator: %w", err)
			return
		}
	}

	return
}

func (i *MACDIndicator) getRequiredDataPoints() int {
	return i.cfg.EmaWarmup * max(i.cfg.Fast, i.cfg.Slow, i.cfg.Signal)
}

func (i *MACDIndicator) drawDebug(bars []market.Bar, macd []float64, s Signal) error {
	count := len(bars)
	last := bars[count-1]
	pricePoints := make(custplotter.TOHLCVs, count)
	for i, b := range bars {
		o, _ := b.Open.Float64()
		c, _ := b.Close.Float64()
		h, _ := b.High.Float64()
		l, _ := b.Low.Float64()
		pricePoints[i].T = float64(b.Time.Unix())
		pricePoints[i].O = o
		pricePoints[i].C = c
		pricePoints[i].H = h
		pricePoints[i].L = l
	}

	p1 := plot.New()
	p1.Title.Text = "Price"
	p1.Y.Label.Text = "Price"
	p1.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	plotPrice, err := custplotter.NewCandlesticks(pricePoints)
	if err != nil {
		return fmt.Errorf("failed to create price graph: %w", err)
	}
	p1.Add(plotPrice)

	p2 := plot.New()
	p2.Title.Text = "MACD"
	p2.Y.Label.Text = "Signal"
	p2.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	count = len(macd)
	macdPoints := make(custplotter.TOHLCVs, count)
	for i, v := range macd {
		macdPoints[i].T = pricePoints[i].T
		macdPoints[i].O = 0
		macdPoints[i].C = v
		macdPoints[i].V = math.Abs(v)
	}

	plotMacd, err := custplotter.NewVBars(macdPoints)
	if err != nil {
		return fmt.Errorf("failed to create macd graph: %w", err)
	}

	p2.Add(plotMacd)

	plotext.UniteAxisRanges([]*plot.Axis{&p1.X, &p2.X})
	tbl := plotext.Table{
		RowHeights: []float64{2, 1},
		ColWidths:  []float64{1},
	}

	plots := [][]*plot.Plot{{p1}, {p2}}
	img := vgimg.New(450, 300)
	dc := draw.New(img)

	canvases := tbl.Align(plots, dc)
	plots[0][0].Draw(canvases[0][0])
	plots[1][0].Draw(canvases[1][0])

	plotFile := filepath.Join(i.cfg.DebugDir, fmt.Sprintf("%s_%s_%.2f.png", last.Time, s.Act.String(), s.Confidence))
	w, err := os.Create(plotFile)
	if err != nil {
		return fmt.Errorf("failed to create plot file: %w", err)
	}

	png := vgimg.PngCanvas{Canvas: img}
	if _, err := png.WriteTo(w); err != nil {
		return fmt.Errorf("failed to write plot file %w", err)
	}

	return nil
}

func calcMACD(bars []market.Bar, fast, slow, signal int) []float64 {
	n := len(bars)
	prices := make([]float64, n)
	for i, b := range bars {
		prices[i], _ = b.Close.Float64()
	}

	fastEma := ema(prices, fast)
	slowEma := ema(prices, slow)
	diff := make([]float64, n)
	for i := range n {
		diff[i] = fastEma[i] - slowEma[i]
	}

	signalEma := ema(diff, signal)
	macd := make([]float64, n)
	for i := 0; i < n; i++ {
		macd[i] = diff[i] - signalEma[i]
	}

	return macd
}

func hasCrossOver(macd []float64, lookback int) bool {
	l := len(macd)
	if l < 2 {
		return false
	}

	n := min(lookback, l)
	for i := 1; i <= n; i++ {
		next := macd[l-i]
		prev := macd[l-i-1]
		if prev < 0 && next > 0 || prev > 0 && next < 0 {
			return true
		}
	}

	return false
}
