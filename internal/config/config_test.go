package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead_Strategy(t *testing.T) {
	cfg, err := Read(strings.NewReader(`
strategies:
    BTC:
        budget: 1000
        buy_confidence: 0.8
        sell_confidence: 0.7
        position_scale: 1
        market_buffer: 1024
        indicator:
            macd:
                fast: 8	
                slow: 12
                signal: 10
                buy_threshold: 10.1
                buy_cap: 100.9
                sell_threshold: -5.5
                sell_cap: -200.4
                cross_lookback: 3
`))

	require.NoError(t, err)

	btc, ok := cfg.Strategies["BTC"]
	require.True(t, ok)

	assert.Equal(t, int64(1000), btc.Budget)
	assert.Equal(t, 0.8, btc.BuyConfidence)
	assert.Equal(t, 0.7, btc.SellConfidence)
	assert.Equal(t, 1.0, btc.PositionScale)
	assert.Equal(t, 1024, btc.MarketBuffer)

	macd, ok := btc.IndRef.Indicator.(MACD)
	require.True(t, ok)

	assert.Equal(t, 8, macd.Fast)
	assert.Equal(t, 12, macd.Slow)
	assert.Equal(t, 10, macd.Signal)
	assert.Equal(t, 10.1, macd.BuyThreshold)
	assert.Equal(t, -5.5, macd.SellThreshold)
	assert.Equal(t, 100.9, macd.BuyCap)
	assert.Equal(t, -200.4, macd.SellCap)
	assert.Equal(t, 3, macd.CrossLookback)
}

func TestRead_Emulator(t *testing.T) {
	cfg, err := Read(strings.NewReader(`
platform:
    emulator:
        data:
            BTC: /var/data/btc.txt
            ETH: /var/data/eth.txt
        start: 2014-09-12T11:45:26.000Z
        end: 2020-12-31T08:30:12.000Z
        buy_comission: 0.002
        sell_comission: 0.0015
`))

	require.NoError(t, err)

	emu, ok := cfg.PlatformRef.Platform.(Emulator)
	require.True(t, ok)

	start, err := time.Parse("2006-01-02T15:04:05.000Z", "2014-09-12T11:45:26.000Z")
	require.NoError(t, err)
	end, err := time.Parse("2006-01-02T15:04:05.000Z", "2020-12-31T08:30:12.000Z")
	require.NoError(t, err)

	assert.Equal(t, "/var/data/btc.txt", emu.Data["BTC"])
	assert.Equal(t, "/var/data/eth.txt", emu.Data["ETH"])
	assert.Equal(t, start, emu.Start)
	assert.Equal(t, end, emu.End)
	assert.Equal(t, 0.002, emu.BuyComission)
	assert.Equal(t, 0.0015, emu.SellComission)
}
