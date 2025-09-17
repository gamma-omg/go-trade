package config

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Strategies map[string]Strategy `yaml:"strategies"`
}

type Strategy struct {
	Budget         int64   `yaml:"budget"`
	BuyConfidence  float64 `yaml:"buy_threshold"`
	SellConfidence float64 `yaml:"sell_threshold"`
	PositionScale  float64 `yaml:"position_scale"`
	MarketBuffer   int     `yaml:"market_buffer"`
	IndRef         IndicatorReference
}

type MACD struct {
	Fast          int     `yaml:"fast"`
	Slow          int     `yaml:"slow"`
	Signal        int     `yaml:"signal"`
	BuyThreshold  float64 `yaml:"buy_threshold"`
	BuyCap        float64 `yaml:"buy_cap"`
	SellThreshold float64 `yaml:"sell_threshold"`
	SellCap       float64 `yaml:"sell_cap"`
	CrossLookback int     `yaml:"cross_lookback"`
}

type Ensemble struct {
	Indicators []struct {
		Weight float64
		IndRef IndicatorReference
	}
}

type Indicator interface{}

type IndicatorReference struct {
	Indicator Indicator
}

func (w *IndicatorReference) UnmarshallYAML(value *yaml.Node) error {
	if len(value.Content) == 0 {
		return nil
	}

	if value.Kind != yaml.MappingNode || len(value.Content) != 2 {
		return errors.New("invalid indicator yaml format")
	}

	key := value.Content[0].Value
	switch key {
	case "macd":
		var macd MACD
		if err := value.Content[1].Decode(&macd); err != nil {
			return fmt.Errorf("failed parsing macd indicator config: %w", err)
		}
		w.Indicator = macd
	case "ensemble":
		var ensemble Ensemble
		if err := value.Content[1].Decode(&ensemble); err != nil {
			return fmt.Errorf("failed parsing ensemble indicator config: %w", err)
		}
		w.Indicator = ensemble
	default:
		return fmt.Errorf("unknown indicator type: %s", key)
	}

	return nil
}
