package market

import "github.com/shopspring/decimal"

type ConstScaler struct {
	Size decimal.Decimal
}

func (s *ConstScaler) GetSize(budget decimal.Decimal, confidence float64) decimal.Decimal {
	return decimal.Min(budget, s.Size)
}

type LinearScaler struct {
	MaxScale float64
}

func (s *LinearScaler) GetSize(budget decimal.Decimal, confidence float64) decimal.Decimal {
	return budget.Mul(decimal.NewFromFloat(confidence * s.MaxScale))
}
