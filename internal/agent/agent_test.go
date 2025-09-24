package agent

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/stretchr/testify/assert"
)

type mockBarsSource struct {
	bars chan market.Bar
}

func (m *mockBarsSource) GetBars(symbol string) <-chan market.Bar {
	return m.bars
}

type mockTradingStrategy struct {
	runCalls int
}

func (m *mockTradingStrategy) Run(ctx context.Context) error {
	m.runCalls++
	return nil
}

func TestCreateIndicator_MACD(t *testing.T) {
	ind, err := createIndicator(config.IndicatorReference{
		Indicator: config.MACD{
			Fast:          10,
			Slow:          20,
			Signal:        10,
			BuyThreshold:  100,
			BuyCap:        1000,
			SellThreshold: 100,
			SellCap:       1000,
			CrossLookback: 3,
		},
	}, market.NewAsset("BTC", 1))

	assert.NoError(t, err)
	assert.IsType(t, &indicator.MACDIndicator{}, ind)
}

func TestCreateIndicator_Ensemble(t *testing.T) {
	ind, err := createIndicator(config.IndicatorReference{
		Indicator: config.Ensemble{
			Indicators: []struct {
				Weight float64
				IndRef config.IndicatorReference
			}{
				{
					Weight: 1.0,
					IndRef: config.IndicatorReference{
						Indicator: config.MACD{},
					},
				},
				{
					Weight: 2.0,
					IndRef: config.IndicatorReference{
						Indicator: config.MACD{},
					},
				},
			},
		},
	}, market.NewAsset("BTC", 1))

	assert.NoError(t, err)
	assert.IsType(t, &indicator.EnsembleIndicator{}, ind)

	e := ind.(*indicator.EnsembleIndicator)
	assert.Equal(t, 2, len(e.Children))
	assert.Equal(t, 1.0, e.Children[0].Weight)
	assert.Equal(t, 2.0, e.Children[1].Weight)
	assert.IsType(t, &indicator.MACDIndicator{}, e.Children[0].Indicator)
	assert.IsType(t, &indicator.MACDIndicator{}, e.Children[1].Indicator)
}

func TestCreateIndicator_InvalidType(t *testing.T) {
	ind, err := createIndicator(config.IndicatorReference{
		Indicator: "invalid",
	}, market.NewAsset("BTC", 1))

	assert.Error(t, err)
	assert.Nil(t, ind)
}

func TestCreateIndicator_EmptyEnsemble(t *testing.T) {
	ind, err := createIndicator(config.IndicatorReference{
		Indicator: config.Ensemble{},
	}, market.NewAsset("BTC", 1))

	assert.NoError(t, err)
	assert.IsType(t, &indicator.EnsembleIndicator{}, ind)

	e := ind.(*indicator.EnsembleIndicator)
	assert.Equal(t, 0, len(e.Children))
}

func TestAgentRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	src := mockBarsSource{bars: make(chan market.Bar, 3)}
	str := mockTradingStrategy{}
	a := TradingAgent{
		log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		bars: &src,
		factory: func(symbol string, cfg config.Strategy, asset *market.Asset) (tradingStrategy, error) {
			return &str, nil
		},
		cfg: config.Config{
			Strategies: map[string]config.Strategy{
				"BTC": {MarketBuffer: 1},
			},
		},
	}

	done := make(chan struct{})
	go func() {
		a.Run(ctx)
		close(done)
	}()

	src.bars <- market.Bar{}
	src.bars <- market.Bar{}
	src.bars <- market.Bar{}
	close(src.bars)

	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("test failed due to run timeout: %v", ctx.Err())
	}

	assert.Equal(t, 3, str.runCalls)
}
