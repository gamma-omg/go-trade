package common

import (
	"fmt"
	"sync"

	"github.com/gamma-omg/trading-bot/internal/market"
)

type DefaultPriceProvider struct {
	prices map[string]market.Bar
	mu     sync.RWMutex
}

func NewDefaultPriceProvider() *DefaultPriceProvider {
	return &DefaultPriceProvider{
		prices: make(map[string]market.Bar),
	}
}

func (pp *DefaultPriceProvider) UpdatePrice(symbol string, bar market.Bar) {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	pp.prices[symbol] = bar
}

func (pp *DefaultPriceProvider) GetLastBar(symbol string) (bar market.Bar, err error) {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	bar, ok := pp.prices[symbol]
	if !ok {
		err = fmt.Errorf("unknown symbol %s", symbol)
		return
	}

	return
}
