package config

type Config struct {
	Strategies []Strategy `yaml:"strategies"`
}

type Strategy struct {
	Budget         int64   `yaml:"budget"`
	BuyConfidence  float32 `yaml:"buy_threshold"`
	SellConfidence float32 `yaml:"sell_threshold"`
	PositionScale  float32 `yaml:"position_scale"`
}
