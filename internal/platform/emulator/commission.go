package emulator

import "github.com/shopspring/decimal"

type fixedRateCommission struct {
	buyFactor  decimal.Decimal
	sellFactor decimal.Decimal
}

func newFixedRateCommission(buyPct, sellPct float64) *fixedRateCommission {
	return &fixedRateCommission{
		buyFactor:  decimal.NewFromFloat(1 - buyPct),
		sellFactor: decimal.NewFromFloat(1 - sellPct),
	}
}

func (c *fixedRateCommission) ApplyOnBuy(sum decimal.Decimal) decimal.Decimal {
	return sum.Mul(c.buyFactor)
}

func (c *fixedRateCommission) ApplyOnSell(sum decimal.Decimal) decimal.Decimal {
	return sum.Mul(c.sellFactor)
}

type noCommission struct{}

func (c *noCommission) ApplyOnBuy(sum decimal.Decimal) decimal.Decimal {
	return sum
}

func (c *noCommission) ApplyOnSell(sum decimal.Decimal) decimal.Decimal {
	return sum
}
