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

type Position struct {
	EntryPrice decimal.Decimal
	Qty        decimal.Decimal
}
