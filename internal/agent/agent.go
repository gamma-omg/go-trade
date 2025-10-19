package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
)

type barsSource interface {
	Prefetch(symbol string, count int) ([]market.Bar, error)
	GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error)
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

	errCh := make(chan error, len(a.cfg.Strategies))

	var wg sync.WaitGroup
	for symbol, cfg := range a.cfg.Strategies {
		wg.Add(1)
		go func(symbol string, cfg config.Strategy) {
			defer wg.Done()

			asset := market.NewAsset(symbol, cfg.MarketBuffer)
			s, err := a.strategyFactory(cfg, asset)
			if err != nil {
				errCh <- fmt.Errorf("failed to create strategy for symbol %s: %w", symbol, err)
				return
			}

			barsDump := NewCsvBarsDump(io.Discard)
			if cfg.DataDump != "" {
				f, err := os.Create(cfg.DataDump)
				if err != nil {
					errCh <- fmt.Errorf("failed to open bars dump file for %s: %w", symbol, err)
					return
				}
				barsDump = NewCsvBarsDump(f)

				defer func() {
					err := f.Close()
					if err != nil {
						errCh <- fmt.Errorf("failed to close bars dump file for %s: %w", symbol, err)
					}
				}()
			}

			if err := s.Init(); err != nil {
				errCh <- fmt.Errorf("failed to initialize trading strategy for %s: %w", symbol, err)
				return
			}

			if cfg.Prefetch > 0 {
				initBars, err := a.bars.Prefetch(symbol, cfg.Prefetch)
				if err != nil {
					errCh <- fmt.Errorf("failed to prefetch last %d bars for symbol %s: %w", cfg.Prefetch, symbol, err)
					return
				}

				for _, b := range initBars {
					asset.Receive(b)
				}
			}

			bars, errs := a.bars.GetBars(ctx, symbol)
			for {
				select {
				case <-ctx.Done():
					return
				case err, ok := <-errs:
					if ok {
						errCh <- fmt.Errorf("error reading bars for %s: %w", symbol, err)
						return
					}
				case bar, ok := <-bars:
					if !ok {
						return
					}

					barsDump.Dump(bar)
					asset.Receive(bar)
					if err := s.Run(ctx); err != nil {
						errCh <- fmt.Errorf("failed to run strategy for %s: %w", symbol, err)
						return
					}
				}
			}
		}(symbol, cfg)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

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
