package emulator

import (
	"context"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readBars(t *testing.T, ctx context.Context, br *barReader) []market.Bar {
	t.Helper()

	var bars []market.Bar
	for b := range br.Read(ctx) {
		require.NoError(t, b.err)
		bars = append(bars, b.bar)
	}

	return bars
}

func TestRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dataFile := writeCsv(t, "data", `timestamp,open,high,low,close,volume
1460413380.0,421.07,521.07,321.06,121.06,1.192`)
	br, err := newBarReader(dataFile)
	require.NoError(t, err)

	bars := readBars(t, ctx, br)
	assert.Equal(t, time.Unix(1460413380, 0), bars[0].Time)
	assert.Equal(t, decimal.NewFromFloat(421.07), bars[0].Open)
	assert.Equal(t, decimal.NewFromFloat(521.07), bars[0].High)
	assert.Equal(t, decimal.NewFromFloat(321.06), bars[0].Low)
	assert.Equal(t, decimal.NewFromFloat(121.06), bars[0].Close)
	assert.Equal(t, decimal.NewFromFloat(1.192), bars[0].Volume)
}

func TestReadFilter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dataFile := writeCsv(t, "data", `timestamp,open,high,low,close,volume
1390134600.0,800.0,800.0,800.0,800.0,0.0
1437452040.0,279.22,279.22,279.22,279.22,0.0
1460413380.0,421.07,521.07,321.06,121.06,1.192
1553889480.0,4080.0,4080.1,4080.0,4080.1,2.035854
1758127500.0,115510,115510,115482,115493,1.05828858
1758152940.0,116570,116577,116569,116574,1.60268598
`)
	br, err := newBarReaderWithFilter(dataFile, func(b market.Bar) bool {
		return b.Time.After(time.Unix(1437452040, 0)) && b.Time.Before(time.Unix(1758127500, 0))
	})
	require.NoError(t, err)

	bars := readBars(t, ctx, br)
	assert.Len(t, bars, 2)
	assert.Equal(t, time.Unix(1460413380, 0), bars[0].Time)
	assert.Equal(t, time.Unix(1553889480, 0), bars[1].Time)
}
