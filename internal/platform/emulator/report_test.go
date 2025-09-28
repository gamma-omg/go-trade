package emulator

import (
	"bytes"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	r := newJsonReportBuilder()
	r.SubmitDeal(Deal{
		Symbol: "BTC",
		Spend:  decimal.NewFromInt(100),
		Gain:   decimal.NewFromInt(120),
	})
	r.SubmitDeal(Deal{
		Symbol: "ETH",
		Spend:  decimal.NewFromInt(1000),
		Gain:   decimal.NewFromInt(1200),
	})

	var buff bytes.Buffer
	err := r.Write(&buff)
	require.NoError(t, err)

	assert.JSONEq(t, `
{
	"total_spend": "1100",
	"total_gain": "1320",
	"total_gain_pct": 1.2,
	"deals": {
		"BTC": [{
			"spend": "100",
			"gain": "120",
			"gain_pct": 1.2
		}],
		"ETH": [{
			"spend": "1000",
			"gain": "1200",
			"gain_pct": 1.2
		}]
	}
}`, buff.String())
}
