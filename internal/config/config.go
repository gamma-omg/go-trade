package config

type Config struct {
	Strategies []Strategy `yaml:"strategies"`
}

type Strategy struct {
	Budget         int64   `yaml:"budget"`
	BuyConfidence  float64 `yaml:"buy_threshold"`
	SellConfidence float64 `yaml:"sell_threshold"`
	PositionScale  float64 `yaml:"position_scale"`
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
