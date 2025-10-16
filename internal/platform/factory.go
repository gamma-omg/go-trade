package platform

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/gamma-omg/trading-bot/internal/platform/alpaca"
	"github.com/gamma-omg/trading-bot/internal/platform/emulator"
	"github.com/shopspring/decimal"
)

type tradingPlatform interface {
	GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error)
	Open(ctx context.Context, symbol string, size decimal.Decimal) (*market.Position, error)
	Close(ctx context.Context, p *market.Position) (market.Deal, error)
	GetBalance() (decimal.Decimal, error)
}

func Create(log *slog.Logger, cfg config.Config) (tradingPlatform, error) {
	alpacaCfg, ok := cfg.PlatformRef.Platform.(config.Alpaca)
	if ok {
		return alpaca.NewAlpacaPlatform(alpacaCfg)
	}

	emulatorCfg, ok := cfg.PlatformRef.Platform.(config.Emulator)
	if ok {
		return emulator.NewTradingEmulator(log, emulatorCfg)
	}

	return nil, errors.New("unknown trading platform")
}
