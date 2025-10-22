package agent

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	r := NewJsonReportBuilder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.SubmitDeal(market.Deal{
		Symbol:    "BTC",
		Qty:       decimal.NewFromInt(10),
		SellPrice: decimal.NewFromInt(12),
		Spend:     decimal.NewFromInt(100),
	})
	r.SubmitDeal(market.Deal{
		Symbol:    "ETH",
		Qty:       decimal.NewFromInt(10),
		SellPrice: decimal.NewFromInt(120),
		Spend:     decimal.NewFromInt(1000),
	})

	var buff bytes.Buffer
	err := r.Write(&buff)
	require.NoError(t, err)

	assert.JSONEq(t, `
{
	"total_gain": "220",
	"total_gain_pct": 0.2,
	"deals": {
		"BTC": [{
			"spend": "100",
			"gain": "20",
			"gain_pct": 0.2
		}],
		"ETH": [{
			"spend": "1000",
			"gain": "200",
			"gain_pct": 0.2
		}]
	}
}`, buff.String())
}

func TestWrite_emptyReport(t *testing.T) {
	r := NewJsonReportBuilder(slog.New(slog.NewTextHandler(io.Discard, nil)))

	var buff bytes.Buffer
	err := r.Write(&buff)
	require.NoError(t, err)

	assert.JSONEq(t, "{}", buff.String())
}

func TestSubmitDeal_divideByZero(t *testing.T) {
	r := NewJsonReportBuilder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	r.SubmitDeal(market.Deal{
		Symbol:    "BTC",
		Qty:       decimal.NewFromInt(1),
		SellPrice: decimal.NewFromInt(100),
		Spend:     decimal.NewFromInt(0),
	})

	var buff bytes.Buffer
	err := r.Write(&buff)
	require.NoError(t, err)

	assert.JSONEq(t, `
{
	"total_gain": "100",
	"deals": {
		"BTC": [{
			"spend": "0",
			"gain": "100"
		}]
	}
}`, buff.String())
}
