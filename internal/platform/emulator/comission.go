package emulator

import "github.com/shopspring/decimal"

type fixedRateComission struct {
	buyFactor  decimal.Decimal
	sellFactor decimal.Decimal
}

func newFixedRateComission(buyPct, sellPct float64) *fixedRateComission {
	return &fixedRateComission{
		buyFactor:  decimal.NewFromFloat(1 - buyPct),
		sellFactor: decimal.NewFromFloat(1 - sellPct),
	}
}

func (c *fixedRateComission) ApplyOnBuy(sum decimal.Decimal) decimal.Decimal {
	return sum.Mul(c.buyFactor)
}

func (c *fixedRateComission) ApplyOnSell(sum decimal.Decimal) decimal.Decimal {
	return sum.Mul(c.sellFactor)
}

type noComission struct{}

func (c *noComission) ApplyOnBuy(sum decimal.Decimal) decimal.Decimal {
	return sum
}

func (c *noComission) ApplyOnSell(sum decimal.Decimal) decimal.Decimal {
	return sum
}
