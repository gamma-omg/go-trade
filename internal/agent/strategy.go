package agent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/pplcc/plotext/custplotter"
	"github.com/shopspring/decimal"
	"gonum.org/v1/plot"
)

type tradingIndicator interface {
	GetSignal() (indicator.Signal, error)
	DrawDebug(d *indicator.DebugPlot) error
}

type positionManager interface {
	Open(ctx context.Context, a *market.Asset, size decimal.Decimal) (*market.Position, error)
	Close(ctx context.Context, p *market.Position) (market.Deal, error)
}

type positionScaler interface {
	GetSize(budget decimal.Decimal, confidence float64) decimal.Decimal
}

type account interface {
	GetBalance() (decimal.Decimal, error)
}

type reportBuilder interface {
	SubmitDeal(d market.Deal)
	Write(w io.Writer) error
}

type positionValidator interface {
	NeedClose(p *market.Position) (bool, error)
}

type TradingStrategy struct {
	log          *slog.Logger
	asset        *market.Asset
	cfg          config.Strategy
	indicator    tradingIndicator
	posMan       positionManager
	posScaler    positionScaler
	posValidator positionValidator
	acc          account
	report       reportBuilder
	position     *market.Position
}

func newTradingStrategy(asset *market.Asset, cfg config.Strategy, indicator tradingIndicator, validator positionValidator, positionManager positionManager, acc account, report reportBuilder, log *slog.Logger) *TradingStrategy {
	return &TradingStrategy{
		log:          log,
		asset:        asset,
		cfg:          cfg,
		indicator:    indicator,
		posValidator: validator,
		posScaler:    &market.LinearScaler{MaxScale: cfg.PositionScale},
		posMan:       positionManager,
		acc:          acc,
		report:       report,
		position:     nil,
	}
}

func (ts *TradingStrategy) Init() error {
	if err := os.RemoveAll(ts.cfg.DebugDir); err != nil {
		return fmt.Errorf("failed to clear debug directory: %w", err)
	}

	if err := os.MkdirAll(ts.cfg.DebugDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	return nil
}

func (ts *TradingStrategy) Run(ctx context.Context) error {
	if ts.position != nil {
		clz, err := ts.posValidator.NeedClose(ts.position)
		if err != nil {
			return fmt.Errorf("failed to validate position: %w", err)
		}
		if clz {
			if err := ts.sell(ctx, 1.0); err != nil {
				return fmt.Errorf("failed to sell position: %w", err)
			}
		}
	}

	s, err := ts.indicator.GetSignal()
	if err != nil {
		return fmt.Errorf("failed to get signal from indicator: %w", err)
	}

	if ts.cfg.DebugLevel >= config.DebugAll {
		if err := ts.drawDebug(s); err != nil {
			ts.log.Error("failed to create debug plot", slog.String("symbol", ts.asset.Symbol), slog.Any("error", err))
		}
	}

	if s.Act == indicator.ActHold {
		return nil
	}

	if ts.position == nil && s.Act == indicator.ActBuy && s.Confidence >= ts.cfg.BuyConfidence {
		if err = ts.buy(ctx, s.Confidence); err != nil {
			return fmt.Errorf("failed to process buy signal: %w", err)
		}

		if ts.cfg.DebugLevel >= config.DebugBuyOrSell {
			if err := ts.drawDebug(s); err != nil {
				ts.log.Error("failed to create debug plot", slog.String("symbol", ts.asset.Symbol), slog.Any("error", err))
			}
		}
	}

	if ts.position != nil && s.Act == indicator.ActSell && s.Confidence >= ts.cfg.SellConfidence {
		if err = ts.sell(ctx, s.Confidence); err != nil {
			return fmt.Errorf("failed to process sell signal: %w", err)
		}

		if ts.cfg.DebugLevel >= config.DebugBuyOrSell {
			if err := ts.drawDebug(s); err != nil {
				ts.log.Error("failed to create debug plot", slog.String("symbol", ts.asset.Symbol), slog.Any("error", err))
			}
		}
	}

	return nil
}

func (ts *TradingStrategy) buy(ctx context.Context, confidence float64) error {
	funds, err := ts.getAvailableFunds()
	if err != nil {
		return fmt.Errorf("failed to get available funds: %w", err)
	}

	size := ts.posScaler.GetSize(funds, confidence)
	p, err := ts.posMan.Open(ctx, ts.asset, size)
	if err != nil {
		return fmt.Errorf("failed to open position: %w", err)
	}

	ts.position = p
	return nil
}

func (ts *TradingStrategy) sell(ctx context.Context, _ float64) error {
	d, err := ts.posMan.Close(ctx, ts.position)
	if err != nil {
		return fmt.Errorf("failed to sell position: %w", err)
	}

	ts.report.SubmitDeal(d)
	ts.position = nil
	return nil
}

func (ts *TradingStrategy) getAvailableFunds() (decimal.Decimal, error) {
	available := decimal.NewFromInt(ts.cfg.Budget)
	if ts.position != nil {
		available = decimal.Max(decimal.NewFromInt(0), available.Sub(ts.position.EntryPrice))
	}

	balance, err := ts.acc.GetBalance()
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("failed to get current balance: %w", err)
	}

	return decimal.Min(balance, available), nil
}

func (ts *TradingStrategy) drawDebug(s indicator.Signal) error {
	if !ts.asset.HasBars(ts.cfg.DebugWindow) {
		return nil
	}

	d := indicator.NewDebugPlot(400, 100)
	if err := ts.drawPricePlot(d); err != nil {
		return fmt.Errorf("failed to debug draw price plot: %w", err)
	}

	if err := ts.indicator.DrawDebug(d); err != nil {
		return fmt.Errorf("failed to debug draw indicator: %w", err)
	}

	last, err := ts.asset.GetLastBar()
	if err != nil {
		return fmt.Errorf("failed to get last bar: %w", err)
	}

	file := filepath.Join(ts.cfg.DebugDir, fmt.Sprintf("%s_%s_%.2f.png", last.Time, s.Act.String(), s.Confidence))
	if err := d.Save(file); err != nil {
		return fmt.Errorf("failed to save debug plot: %w", err)
	}

	return nil
}

func (ts *TradingStrategy) drawPricePlot(d *indicator.DebugPlot) error {
	bars, err := ts.asset.GetBars(ts.cfg.DebugWindow)
	if err != nil {
		return fmt.Errorf("failed to get bars data: %w", err)
	}

	pricePoints := make(custplotter.TOHLCVs, len(bars))
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

	p := plot.New()
	p.Title.Text = "Price"
	p.Y.Label.Text = "Price"
	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	plotPrice, err := custplotter.NewCandlesticks(pricePoints)
	if err != nil {
		return fmt.Errorf("failed to create price graph: %w", err)
	}
	p.Add(plotPrice)
	d.Add(p, 2)

	return nil
}
