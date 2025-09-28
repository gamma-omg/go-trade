package emulator

import (
	"fmt"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/market"
)

type defaultPriceProvider struct {
	prices map[string]market.Bar
	mu     sync.RWMutex
}

func newDefaultPriceProvider() *defaultPriceProvider {
	return &defaultPriceProvider{
		prices: make(map[string]market.Bar),
	}
}

func (pp *defaultPriceProvider) UpdatePrice(symbol string, bar market.Bar) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	pp.prices[symbol] = bar
}

func (pp *defaultPriceProvider) GetLastBar(symbol string) (bar market.Bar, err error) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	bar, ok := pp.prices[symbol]
	if !ok {
		err = fmt.Errorf("unknown symbol %s", symbol)
		return
	}

	return
}
