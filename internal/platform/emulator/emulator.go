package emulator

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
)

type TradingEmulator struct {
	readers map[string]*barReader
}

func NewTradingEmulator(cfg config.Emulator) (*TradingEmulator, error) {
	emu := &TradingEmulator{
		readers: make(map[string]*barReader),
	}

	for symbol, path := range cfg.Data {
		rdr, err := newBarReaderWithFilter(path, func(b market.Bar) bool {
			return b.Time.After(cfg.Start) && b.Time.Before(cfg.End)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bars reader: %w", err)
		}

		emu.readers[symbol] = rdr
	}

	return emu, nil
}

func (e *TradingEmulator) GetBars(symbol string) (<-chan market.Bar, error) {
	rdr, ok := e.readers[symbol]
	if !ok {
		return nil, fmt.Errorf("unknown symbol: %s", symbol)
	}

	return rdr.bars, nil
}

func (e *TradingEmulator) Run() error {
	errCh := make(chan error, len(e.readers))

	var wg sync.WaitGroup
	for _, rdr := range e.readers {
		wg.Add(1)
		go func(rdr *barReader) {
			defer wg.Done()

			if err := rdr.Read(); err != nil {
				errCh <- err
			}

		}(rdr)
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
