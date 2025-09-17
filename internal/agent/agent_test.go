package agent

import (
	"testing"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/stretchr/testify/assert"
)

func Test_createIndicator_MACD(t *testing.T) {
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

func Test_createIndicator_Ensemble(t *testing.T) {
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

func Test_createIndicator_InvalidType(t *testing.T) {
	ind, err := createIndicator(config.IndicatorReference{
		Indicator: "invalid",
	}, market.NewAsset("BTC", 1))

	assert.Error(t, err)
	assert.Nil(t, ind)
}

func Test_createIndicator_EmptyEnsemble(t *testing.T) {
	ind, err := createIndicator(config.IndicatorReference{
		Indicator: config.Ensemble{},
	}, market.NewAsset("BTC", 1))

	assert.NoError(t, err)
	assert.IsType(t, &indicator.EnsembleIndicator{}, ind)

	e := ind.(*indicator.EnsembleIndicator)
	assert.Equal(t, 0, len(e.Children))
}
