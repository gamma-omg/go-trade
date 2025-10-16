package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
)

type barsSource interface {
	GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error)
}

type tradingPlatform interface {
	barsSource
	positionManager
	account
}

type tradingStrategy interface {
	Run(ctx context.Context) error
}

type tradingStrategyFactory func(symbol string, cfg config.Strategy, asset *market.Asset) (tradingStrategy, error)

type TradingAgent struct {
	log     *slog.Logger
	cfg     config.Config
	bars    barsSource
	factory tradingStrategyFactory
	report  reportBuilder
}

func NewTradingAgent(log *slog.Logger, cfg config.Config, platform tradingPlatform, report reportBuilder) *TradingAgent {
	return &TradingAgent{
		log:    log,
		cfg:    cfg,
		bars:   platform,
		report: report,
		factory: func(symbol string, cfg config.Strategy, asset *market.Asset) (tradingStrategy, error) {
			ind, err := createIndicator(cfg.IndRef, asset)
			if err != nil {
				return nil, fmt.Errorf("failed to creat trading strategy for symbol %s: %w", symbol, err)
			}

			validator := &defaultPositionValidator{
				takeProfit: cfg.TakeProfit,
				stopLoss:   cfg.StopLoss,
			}
			return newTradingStrategy(asset, cfg, ind, validator, platform, platform, report, log), nil
		},
	}
}

func createIndicator(ref config.IndicatorReference, asset *market.Asset) (tradingIndicator, error) {
	macd, ok := ref.Indicator.(config.MACD)
	if ok {
		return indicator.NewMACD(macd, asset), nil
	}

	ensemble, ok := ref.Indicator.(config.Ensemble)
	if ok {
		children := make([]indicator.WeightedIndicator, len(ensemble.Indicators))
		for i, c := range ensemble.Indicators {
			child, err := createIndicator(c.IndRef, asset)
			if err != nil {
				return nil, fmt.Errorf("failed to create child indicator: %w", err)
			}

			children[i] = indicator.WeightedIndicator{
				Weight:    c.Weight,
				Indicator: child,
			}
		}

		return &indicator.EnsembleIndicator{Children: children}, nil
	}

	return nil, fmt.Errorf("unknown indicator: %v", ref)
}

func (a *TradingAgent) Run(ctx context.Context) error {
	a.log.Info("starting agent")

	var wg sync.WaitGroup
	for symbol, cfg := range a.cfg.Strategies {
		wg.Add(1)
		go func(symbol string, cfg config.Strategy) {
			defer wg.Done()

			asset := market.NewAsset(symbol, cfg.MarketBuffer)
			s, err := a.factory(symbol, cfg, asset)
			if err != nil {
				a.log.Error("failed to run strategy for symbol", slog.Any("error", err), slog.String("symbol", symbol))
				return
			}

			bars, errs := a.bars.GetBars(ctx, symbol)
			for {
				select {
				case <-ctx.Done():
					return
				case err, ok := <-errs:
					if ok {
						a.log.Error("error reading bars", slog.Any("error", err), slog.String("symbol", symbol))
					}
				case bar, ok := <-bars:
					if !ok {
						return
					}

					asset.Receive(bar)
					if err := s.Run(ctx); err != nil {
						a.log.Error(err.Error(), slog.String("symbol", symbol))
					}
				}
			}
		}(symbol, cfg)
	}

	wg.Wait()

	if err := a.saveReport(); err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	return nil
}

func (a *TradingAgent) saveReport() error {
	f, err := os.Create(a.cfg.Report)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); err != nil {
			err = cerr
		}
	}()

	if err := a.report.Write(f); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}
