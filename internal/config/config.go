package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Strategies  map[string]Strategy `yaml:"strategies"`
	Report      string              `yaml:"report"`
	PlatformRef PlatformReference   `yaml:"platform"`
}

func Read(r io.Reader) (*Config, error) {
	var cfg Config
	d := yaml.NewDecoder(r)
	err := d.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse config file: %w", err)
	}

	return &cfg, nil
}

func ReadFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file: %w", err)
	}

	return Read(f)
}

type Strategy struct {
	Budget         int64              `yaml:"budget"`
	BuyConfidence  float64            `yaml:"buy_confidence"`
	SellConfidence float64            `yaml:"sell_confidence"`
	TakeProfit     float64            `yaml:"take_profit"`
	StopLoss       float64            `yaml:"stop_loss"`
	PositionScale  float64            `yaml:"position_scale"`
	MarketBuffer   int                `yaml:"market_buffer"`
	IndRef         IndicatorReference `yaml:"indicator"`
	DataDump       string             `yaml:"data_dump"`
}

type PlatformReference struct {
	Platform Platform
}

type Platform interface{}

// indicator configs

type DebugLevel int

const (
	DebugNone DebugLevel = iota
	DebugBuyOrSell
	DebugAll
)

type MACD struct {
	Fast          int        `yaml:"fast"`
	Slow          int        `yaml:"slow"`
	Signal        int        `yaml:"signal"`
	BuyThreshold  float64    `yaml:"buy_threshold"`
	BuyCap        float64    `yaml:"buy_cap"`
	SellThreshold float64    `yaml:"sell_threshold"`
	SellCap       float64    `yaml:"sell_cap"`
	CrossLookback int        `yaml:"cross_lookback"`
	EmaWarmup     int        `yaml:"ema_warmup"`
	DebugLevel    DebugLevel `yaml:"debug_level"`
	DebugDir      string     `yaml:"debug_dir"`
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

func (w *IndicatorReference) UnmarshalYAML(value *yaml.Node) error {
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

// platform configs

type Emulator struct {
	Data          map[string]string `yaml:"data"`
	Start         time.Time         `yaml:"start"`
	End           time.Time         `yaml:"end"`
	BuyComission  float64           `yaml:"buy_comission"`
	SellComission float64           `yaml:"sell_comission"`
	Balance       float64           `yaml:"balance"`
}

type Alpaca struct {
	BaseUrl string `yaml:"base_url"`
	ApiKey  string `yaml:"api_key"`
	Secret  string `yaml:"secret"`
}

func (w *PlatformReference) UnmarshalYAML(value *yaml.Node) error {
	if len(value.Content) == 0 {
		return nil
	}

	if value.Kind != yaml.MappingNode || len(value.Content) != 2 {
		return errors.New("invalid platform yaml format")
	}

	key := value.Content[0].Value
	switch key {
	case "emulator":
		var emu Emulator
		if err := value.Content[1].Decode(&emu); err != nil {
			return fmt.Errorf("failed parsing emulator platform config: %w", err)
		}
		w.Platform = emu
	case "alpaca":
		var alpaca Alpaca
		if err := value.Content[1].Decode(&alpaca); err != nil {
			return fmt.Errorf("failed parsing Alpaca platform config: %w", err)
		}
		w.Platform = alpaca
	default:
		return fmt.Errorf("unknown platform type: %s", key)
	}

	return nil
}
