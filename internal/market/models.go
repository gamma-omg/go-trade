package market

import (
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
