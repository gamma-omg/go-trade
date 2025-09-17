package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
)

type barsSource interface {
	GetBars(symbol string) <-chan market.Bar
}

type TradingAgent struct {
	log  *slog.Logger
	cfg  config.Config
	bars barsSource
}

// func newTradingAgent(log *slog.Logger, cfg config.Config, bars barsSource) *TradingAgent {
// 	return &TradingAgent{
// 		log:  log,
// 		cfg:  cfg,
// 		bars: bars,
// 	}
// }

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

func (a *TradingAgent) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for symbol, cfg := range a.cfg.Strategies {
		wg.Add(1)
		go func(symbol string, cfg config.Strategy) {
			defer wg.Done()

			asset := market.NewAsset(symbol, cfg.MarketBuffer)
			ind, err := createIndicator(cfg.IndRef, asset)
			if err != nil {
				a.log.Error("failed to run strategy for symbol", "symbol", symbol, "error", err)
				return
			}

			s := newTradingStrategy(symbol, cfg, ind, a.log)
			bars := a.bars.GetBars(symbol)

			for {
				select {
				case <-ctx.Done():
					return
				case bar, ok := <-bars:
					if !ok {
						return
					}

					asset.Receive(bar)
					if err := s.Run(); err != nil {
						a.log.Error(err.Error(), "symbol", symbol)
					}
				}
			}
		}(symbol, cfg)
	}

	wg.Wait()
}
