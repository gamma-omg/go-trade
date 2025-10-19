package market

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type Bar struct {
	Time   time.Time
	Open   decimal.Decimal
	High   decimal.Decimal
	Low    decimal.Decimal
	Close  decimal.Decimal
	Volume decimal.Decimal
}

type Deal struct {
	Symbol    string
	BuyTime   time.Time
	SellTime  time.Time
	BuyPrice  decimal.Decimal
	SellPrice decimal.Decimal
	Qty       decimal.Decimal
	Spend     decimal.Decimal
	Gain      decimal.Decimal
}

type Position struct {
	Asset      *Asset
	EntryPrice decimal.Decimal
	Qty        decimal.Decimal
	Price      decimal.Decimal
	OpenTime   time.Time
}

type Asset struct {
	Symbol string
	bars   []Bar
	head   int
	size   int
}

func NewAsset(symbol string, bufSize int) *Asset {
	return &Asset{
		Symbol: symbol,
		bars:   make([]Bar, bufSize),
		head:   -1,
		size:   bufSize,
	}
}

func NewAssetWithBars(symbol string, bars []Bar) *Asset {
	return &Asset{
		Symbol: symbol,
		bars:   bars,
		head:   len(bars) - 1,
		size:   len(bars),
	}
}

func (a *Asset) GetBars(count int) ([]Bar, error) {
	if count > a.size {
		return nil, errors.New("requested bars count is greater than asset buffer capacity")
	}

	if count <= 0 {
		return nil, fmt.Errorf("invalid argument: %d", count)
	}

	if a.head < count-1 {
		return nil, errors.New("insufficient data")
	}

	e := a.head%a.size + 1
	s := (a.head-count)%a.size + 1
	if e >= s {
		return a.bars[s:e], nil
	}

	return append(a.bars[s:], a.bars[0:e]...), nil
}

func (a *Asset) GetLastBar() (Bar, error) {
	if a.head < 0 {
		return Bar{}, errors.New("insufficient data")
	}

	n := a.head % a.size
	return a.bars[n], nil
}

func (a *Asset) HasBars(count int) bool {
	return a.head%a.size >= count-1
}

func (a *Asset) Receive(bar Bar) {
	a.head++
	a.bars[a.head%a.size] = bar
}
