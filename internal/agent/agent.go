package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"golang.org/x/sync/errgroup"
)

type barsSource interface {
	Prefetch(symbol string, count int) (<-chan market.Bar, error)
	GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error)
}

type barsAggregator interface {
	Aggregate(bars <-chan market.Bar) <-chan market.Bar
}

type tradingPlatform interface {
	barsSource
	positionManager
	account
}

type tradingStrategy interface {
	Init() error
	Run(ctx context.Context) error
}

type tradingStrategyFactory func(cfg config.Strategy, asset *market.Asset) (tradingStrategy, error)

type TradingAgent struct {
	log             *slog.Logger
	cfg             config.Config
	bars            barsSource
	strategyFactory tradingStrategyFactory
	report          reportBuilder
}

func NewTradingAgent(log *slog.Logger, cfg config.Config, report reportBuilder) (*TradingAgent, error) {
	platform, err := createPlatform(log, cfg.PlatformRef)
	if err != nil {
		return nil, fmt.Errorf("failed to create trading platform: %w", err)
	}

	a := &TradingAgent{
		log:    log,
		cfg:    cfg,
		bars:   platform,
		report: report,
		strategyFactory: func(cfg config.Strategy, asset *market.Asset) (tradingStrategy, error) {
			ind, err := createIndicator(cfg.IndRef, asset)
			if err != nil {
				return nil, fmt.Errorf("failed to create trading strategy for symbol %s: %w", asset.Symbol, err)
			}

			validator := &defaultPositionValidator{
				takeProfit: cfg.TakeProfit,
				stopLoss:   cfg.StopLoss,
			}
			return newTradingStrategy(asset, cfg, ind, validator, platform, platform, report, log), nil
		},
	}
	return a, nil
}

func (a *TradingAgent) Run(ctx context.Context) error {
	a.log.Info("starting agent")

	grp, ctx := errgroup.WithContext(ctx)
	for symbol, cfg := range a.cfg.Strategies {
		symbol, cfg := symbol, cfg

		grp.Go(func() (err error) {
			asset := market.NewAsset(symbol, cfg.MarketBuffer)
			s, err := a.strategyFactory(cfg, asset)
			if err != nil {
				return fmt.Errorf("failed to create strategy for symbol %s: %w", symbol, err)
			}

			if err := s.Init(); err != nil {
				return fmt.Errorf("failed to initialize trading strategy for %s: %w", symbol, err)
			}

			barsDump, closer, err := createBarsDump(cfg.DataDump)
			if err != nil {
				return fmt.Errorf("failed to create bars dump for symbol %s: %w", symbol, err)
			}
			if closer != nil {
				defer func() {
					if cerr := closer.Close(); cerr != nil {
						err = errors.Join(err, fmt.Errorf("failed to close bars dump for symbol %s: %w", symbol, err))
					}
				}()
			}

			agg := createBarsAggregator(cfg.AggregateBars)
			prefetchBars(ctx, a.bars, agg, asset, cfg.Prefetch)

			bars, errs := a.bars.GetBars(ctx, symbol)
			bars = agg.Aggregate(bars)

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case err, ok := <-errs:
					if ok {
						return fmt.Errorf("error reading bars for %s: %w", symbol, err)
					}
				case bar, ok := <-bars:
					if !ok {
						return nil
					}

					if err := barsDump.Dump(bar); err != nil {
						return fmt.Errorf("failed to dump bar for symbol %s: %w", symbol, err)
					}

					asset.Receive(bar)

					if err := s.Run(ctx); err != nil {
						return fmt.Errorf("failed to run strategy for %s: %w", symbol, err)
					}
				}
			}
		})
	}

	if err := grp.Wait(); err != nil {
		return err
	}

	if err := a.saveReport(); err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	return nil
}

func createBarsDump(path string) (*csvBarsDump, io.Closer, error) {
	if path == "" {
		return newCsvBarsDump(io.Discard), nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return nil, nil, fmt.Errorf("failed to create dump output directory %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open bars dump file: %w", err)
	}

	return newCsvBarsDump(f), f, nil
}

func createBarsAggregator(n int) barsAggregator {
	if n > 1 {
		return &market.IntervalAggregator{
			BarDuration: 1 * time.Minute,
			Interval:    time.Duration(n) * time.Minute,
		}
	}

	return &market.IdentityAggregator{}
}

func prefetchBars(ctx context.Context, bars barsSource, agg barsAggregator, asset *market.Asset, n int) error {
	if n < 1 {
		return nil
	}

	barsCh, err := bars.Prefetch(asset.Symbol, n)
	if err != nil {
		return fmt.Errorf("failed to prefetch last %d bars for symbol %s: %w", n, asset.Symbol, err)
	}

	for b := range agg.Aggregate(barsCh) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			asset.Receive(b)
		}
	}

	return nil
}

func (a *TradingAgent) saveReport() (err error) {
	f, err := os.Create(a.cfg.Report)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close report file: %w", err))
		}
	}()

	if err := a.report.Write(f); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}
