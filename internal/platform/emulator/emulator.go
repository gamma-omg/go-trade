package emulator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
)

type TradingEmulator struct {
	readers map[string]*barReader
	bars    map[string]chan market.Bar
}

func NewTradingEmulator(cfg config.Emulator) (*TradingEmulator, error) {
	emu := &TradingEmulator{
		readers: make(map[string]*barReader),
		bars:    make(map[string]chan market.Bar),
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
		src := rdr.Read(ctx)
		dst := e.bars[symbol]

		go func(src <-chan barReadResult, dst chan<- market.Bar) {
			defer wg.Done()
			defer close(dst)

			for b := range src {
				if b.err != nil {
					errCh <- b.err
					break
				}

				dst <- b.bar
				// todo: process bar
			}
		}(src, dst)
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

	return nil
}
