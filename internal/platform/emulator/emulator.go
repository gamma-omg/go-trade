package emulator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type TradingEmulator struct {
	cfg    config.Emulator
	Acc    *defaultAccount
	PosMan positionManager
}

func NewTradingEmulator(log *slog.Logger, cfg config.Emulator) (*TradingEmulator, error) {
	comission := newFixedRateComission(cfg.BuyComission, cfg.SellComission)
	acc := &defaultAccount{balance: decimal.NewFromInt(int64(cfg.Balance))}

	emu := &TradingEmulator{
		cfg:    cfg,
		Acc:    acc,
		PosMan: *newPositionManager(log, comission, acc),
	}

	return emu, nil
}

func (e *TradingEmulator) Prefetch(symbol string, count int) (<-chan market.Bar, error) {
	return nil, errors.New("operation not supported")
}

func (e *TradingEmulator) GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error) {
	bars := make(chan market.Bar, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(bars)
		defer close(errs)

		path, ok := e.cfg.Data[symbol]
		if !ok {
			errs <- fmt.Errorf("no data file for symbol %s", symbol)
			return
		}

		rdr, closer, err := newBarReaderWithFilter(path, func(b market.Bar) bool {
			return b.Time.After(e.cfg.Start) && b.Time.Before(e.cfg.End)
		})
		if err != nil {
			errs <- fmt.Errorf("failed to create bars reader: %w", err)
			return
		}
		defer closer.Close()

		for r := range rdr.Read(ctx) {
			if r.err != nil {
				errs <- r.err
				continue
			}

			bars <- r.bar
		}
	}()

	return bars, errs
}

func (e *TradingEmulator) Open(ctx context.Context, asset *market.Asset, size decimal.Decimal) (*market.Position, error) {
	return e.PosMan.Open(ctx, asset, size)
}

func (e *TradingEmulator) Close(ctx context.Context, p *market.Position) (market.Deal, error) {
	return e.PosMan.Close(ctx, p)
}

func (e *TradingEmulator) GetBalance() (decimal.Decimal, error) {
	return e.Acc.GetBalance(), nil
}
