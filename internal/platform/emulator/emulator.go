package emulator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type TradingEmulator struct {
	cfg     config.Emulator
	readers map[string]*barReader
	bars    map[string]chan market.Bar
	prices  *defaultPriceProvider
	report  *jsonReportBuilder
	Acc     *defaultAccount
	PosMan  positionManager
}

func NewTradingEmulator(log *slog.Logger, cfg config.Emulator) (*TradingEmulator, error) {
	prices := newDefaultPriceProvider()
	comission := newFixedRateComission(cfg.BuyComission, cfg.SellComission)
	report := newJsonReportBuilder(log)
	acc := &defaultAccount{balance: decimal.NewFromInt(int64(cfg.Balance))}

	emu := &TradingEmulator{
		cfg:     cfg,
		readers: make(map[string]*barReader),
		bars:    make(map[string]chan market.Bar),
		prices:  prices,
		report:  report,
		Acc:     acc,
		PosMan:  *newPositionManager(log, comission, prices, report, acc),
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

func (e *TradingEmulator) GetBars(symbol string) (<-chan market.Bar, error) {
	bars, ok := e.bars[symbol]
	if !ok {
		return nil, fmt.Errorf("unknown symbol: %s", symbol)
	}

	return bars, nil
}

func (e *TradingEmulator) Run(ctx context.Context) error {
	errCh := make(chan error, len(e.readers))

	var wg sync.WaitGroup
	for symbol, rdr := range e.readers {
		wg.Add(1)
		dst := e.bars[symbol]

		go func(symbol string, rdr *barReader, dst chan<- market.Bar) {
			defer wg.Done()
			defer close(dst)

			for b := range rdr.Read(ctx) {
				if b.err != nil {
					errCh <- b.err
					break
				}

				e.prices.UpdatePrice(symbol, b.bar)
				dst <- b.bar
			}
		}(symbol, rdr, dst)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if err := e.report.WriteToFile(e.cfg.Report); err != nil {
		return fmt.Errorf("failed to create trading report: %w", err)
	}

	return nil
}
