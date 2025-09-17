package market

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func Test_ConstScaler(t *testing.T) {
	tbl := []struct {
		budget     float64
		size       float64
		result     float64
		confidence float64
	}{
		{budget: 1000, size: 100, result: 100, confidence: 0.1},
		{budget: 100, size: 1000, result: 100, confidence: 0.5},
		{budget: 100, size: 100, result: 100, confidence: 1.0},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			s := ConstScaler{Size: decimal.NewFromFloat(c.size)}
			res := s.GetSize(decimal.NewFromFloat(c.budget), c.confidence)
			assert.Equal(t, decimal.NewFromFloat(c.result), res)
		})
	}
}

func Test_LinearScaler(t *testing.T) {
	tbl := []struct {
		budget     float64
		scale      float64
		confidence float64
		result     float64
	}{
		{budget: 1000, scale: 0.5, confidence: 0.1, result: 50},
		{budget: 1000, scale: 1, confidence: 0, result: 0},
		{budget: 1000, scale: 1, confidence: 1, result: 1000},
		{budget: 1000, scale: 1, confidence: 0.5, result: 500},
	}

	for i, c := range tbl {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			s := LinearScaler{MaxScale: c.scale}
			res := s.GetSize(decimal.NewFromFloat(c.budget), c.confidence)
			assert.Equal(t, c.result, res.InexactFloat64())
		})
	}
}
