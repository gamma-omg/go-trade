package emulator

import (
	"os"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "data")
	require.NoError(t, err)

	src := `timestamp,open,high,low,close,volume
1460413380.0,421.07,521.07,321.06,121.06,1.192`
	_, err = f.WriteString(src)
	require.NoError(t, err)
	f.Close()

	br, err := newBarReader(f.Name())
	require.NoError(t, err)

	var bars []market.Bar
	done := make(chan struct{})
	go func() {
		defer close(done)
		for b := range br.bars {
			bars = append(bars, b)
		}
	}()

	err = br.Read()
	require.NoError(t, err)

	<-done
	assert.Equal(t, time.Unix(1460413380, 0), bars[0].Time)
	assert.Equal(t, decimal.NewFromFloat(421.07), bars[0].Open)
	assert.Equal(t, decimal.NewFromFloat(521.07), bars[0].High)
	assert.Equal(t, decimal.NewFromFloat(321.06), bars[0].Low)
	assert.Equal(t, decimal.NewFromFloat(121.06), bars[0].Close)
	assert.Equal(t, decimal.NewFromFloat(1.192), bars[0].Volume)
}

func TestReadFilter(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "data")
	require.NoError(t, err)

	src := `timestamp,open,high,low,close,volume
1390134600.0,800.0,800.0,800.0,800.0,0.0
1437452040.0,279.22,279.22,279.22,279.22,0.0
1460413380.0,421.07,521.07,321.06,121.06,1.192
1553889480.0,4080.0,4080.1,4080.0,4080.1,2.035854
1758127500.0,115510,115510,115482,115493,1.05828858
1758152940.0,116570,116577,116569,116574,1.60268598
`
	_, err = f.WriteString(src)
	require.NoError(t, err)
	f.Close()

	br, err := newBarReaderWithFilter(f.Name(), func(b market.Bar) bool {
		return b.Time.After(time.Unix(1437452040, 0)) && b.Time.Before(time.Unix(1758127500, 0))
	})
	require.NoError(t, err)

	var bars []market.Bar
	done := make(chan struct{})
	go func() {
		defer close(done)
		for b := range br.bars {
			bars = append(bars, b)
		}
	}()

	err = br.Read()
	require.NoError(t, err)

	<-done
	assert.Equal(t, 2, len(bars))
	assert.Equal(t, time.Unix(1460413380, 0), bars[0].Time)
	assert.Equal(t, time.Unix(1553889480, 0), bars[1].Time)
}
