package agent

import (
	"bytes"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDump(t *testing.T) {
	var buff bytes.Buffer
	d := NewCsvBarsDump(&buff)
	err := d.Dump(market.Bar{
		Time:   time.Unix(1588223760, 0),
		Open:   decimal.NewFromInt(100),
		High:   decimal.NewFromInt(200),
		Low:    decimal.NewFromInt(300),
		Close:  decimal.NewFromInt(400),
		Volume: decimal.NewFromInt(500),
	})

	require.NoError(t, err)
	assert.Equal(t, buff.String(), `timestamp,open,high,low,close,volume
1588223760,100,200,300,400,500
`)
}
