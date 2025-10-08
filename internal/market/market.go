package market

import (
	"fmt"
)

type Asset struct {
	symbol string
	bars   []Bar
	head   int
	size   int
}

func NewAsset(symbol string, bufSize int) *Asset {
	return &Asset{
		symbol: symbol,
		bars:   make([]Bar, 2*bufSize),
		head:   0,
		size:   2 * bufSize,
	}
}

func (a *Asset) GetBars(count int) ([]Bar, error) {
	n := a.head % a.size
	if n < count-1 {
		return nil, fmt.Errorf("insufficient data")
	}

	return a.bars[n-count+1 : n+1], nil
}

func (a *Asset) HasBars(count int) bool {
	return a.head%a.size >= count-1
}

func (a *Asset) Receive(bar Bar) {
	a.head = (a.head + 1) % a.size
	a.bars[a.head] = bar
}
