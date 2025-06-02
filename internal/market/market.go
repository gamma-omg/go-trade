package market

import (
	"fmt"
)

type Market struct {
	assets  map[string]*Asset
	bufSize int
}

func NewMarket(bufSize int) *Market {
	return &Market{
		assets:  make(map[string]*Asset),
		bufSize: bufSize,
	}
}

func (m *Market) GetAsset(symbol string) *Asset {
	a, ok := m.assets[symbol]
	if !ok {
		a = newAsset(symbol, m.bufSize)
		m.assets[symbol] = a
	}

	return a
}

type Asset struct {
	symbol string
	bars   []Bar
	head   int
	size   int
}

func newAsset(symbol string, bufSize int) *Asset {
	return &Asset{
		symbol: symbol,
		bars:   make([]Bar, bufSize),
		head:   0,
		size:   bufSize,
	}
}

func (a *Asset) GetBars(count int) ([]Bar, error) {
	n := a.head % a.size
	if n < count-1 {
		return nil, fmt.Errorf("insufficient data")
	}

	return a.bars[n-count+1:], nil
}

func (a *Asset) HasBars(count int) bool {
	return a.head%a.size >= count-1
}

func (a *Asset) Receive(bar Bar) {
	a.head = (a.head + 1) % a.size
	a.bars[a.head] = bar
}
