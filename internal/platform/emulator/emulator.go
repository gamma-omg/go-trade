package emulator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type TradingEmulator struct {
	cfg     config.Emulator
	readers map[string]*barReader
	bars    map[string]chan market.Bar
	Acc     *defaultAccount
	PosMan  positionManager
}

func NewTradingEmulator(log *slog.Logger, cfg config.Emulator) (*TradingEmulator, error) {
	comission := newFixedRateComission(cfg.BuyComission, cfg.SellComission)
	acc := &defaultAccount{balance: decimal.NewFromInt(int64(cfg.Balance))}

	emu := &TradingEmulator{
		cfg:     cfg,
		readers: make(map[string]*barReader),
		bars:    make(map[string]chan market.Bar),
		Acc:     acc,
		PosMan:  *newPositionManager(log, comission, acc),
	}

	for symbol, path := range cfg.Data {
		rdr, err := newBarReaderWithFilter(path, func(b market.Bar) bool {
			return b.Time.After(cfg.Start) && b.Time.Before(cfg.End)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bars reader: %w", err)
		}

		emu.bars[symbol] = make(chan market.Bar)
		emu.readers[symbol] = rdr
	}

	return emu, nil
}

func (e *TradingEmulator) GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error) {
	bars := make(chan market.Bar, 64)
	errs := make(chan error, 1)
	go func() {
		defer close(bars)
		defer close(errs)

		rdr, ok := e.readers[symbol]
		if !ok {
			errs <- fmt.Errorf("failed to find reader for symbol: %s", symbol)
			return
		}

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
