package emulator

import (
	"context"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	f := writeCsv(t, "data", `timestamp,open,high,low,close,volume
1390134600.0,800.0,800.0,800.0,800.0,0.0
1437452040.0,279.22,279.22,279.22,279.22,0.0
1460413380.0,421.07,521.07,321.06,121.06,1.192
1553889480.0,4080.0,4080.1,4080.0,4080.1,2.035854
1758127500.0,115510,115510,115482,115493,1.05828858
1758152940.0,116570,116577,116569,116574,1.60268598`)

	emu, err := NewTradingEmulator(config.Emulator{
		Data: map[string]string{
			"BTC": f,
		},
		Start: time.Unix(0, 0),
		End:   time.Unix(0xfffffffffffffff, 0),
	})
	require.NoError(t, err)

	barsCh, err := emu.GetBars("BTC")
	require.NoError(t, err)

	done := make(chan struct{})
	var bars []market.Bar
	go func() {
		defer close(done)
		for b := range barsCh {
			bars = append(bars, b)
		}
	}()

	require.NoError(t, emu.Run(ctx))

	<-done
	assert.Equal(t, 6, len(bars))
}
