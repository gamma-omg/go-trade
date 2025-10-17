package agent

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/gamma-omg/trading-bot/internal/platform/alpaca"
	"github.com/gamma-omg/trading-bot/internal/platform/emulator"
)

func createIndicator(cfg config.IndicatorReference, asset *market.Asset) (tradingIndicator, error) {
	macd, ok := cfg.Indicator.(config.MACD)
	if ok {
		return indicator.NewMACD(macd, asset), nil
	}

	ensemble, ok := cfg.Indicator.(config.Ensemble)
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

	return nil, fmt.Errorf("unknown indicator: %v", cfg)
}

func createPlatform(log *slog.Logger, cfg config.PlatformReference) (tradingPlatform, error) {
	alpacaCfg, ok := cfg.Platform.(config.Alpaca)
	if ok {
		return alpaca.NewAlpacaPlatform(alpacaCfg)
	}

	emulatorCfg, ok := cfg.Platform.(config.Emulator)
	if ok {
		return emulator.NewTradingEmulator(log, emulatorCfg)
	}

	return nil, errors.New("unknown trading platform")
}
