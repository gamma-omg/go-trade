package agent

import (
	"fmt"
	"log/slog"

	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/indicator"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type tradingIndicator interface {
	GetSignal() (indicator.Signal, error)
}

type positionManager interface {
	Open(symbol string, qty decimal.Decimal) (*market.Position, error)
	Close(position *market.Position) error
}

type positionScaler interface {
	GetSize(budget decimal.Decimal, confidence float64) decimal.Decimal
}

type marketProvider interface {
	GetQuantity(size decimal.Decimal, symbol string) (decimal.Decimal, error)
}

type account interface {
	GetBalance() decimal.Decimal
}

type TradingStrategy struct {
	log       *slog.Logger
	symbol    string
	cfg       config.Strategy
	indicator tradingIndicator
	posMan    positionManager
	posScaler positionScaler
	market    marketProvider
	acc       account
	position  *market.Position
}

func (ts *TradingStrategy) Run() error {
	s, err := ts.indicator.GetSignal()
	if err != nil {
		return fmt.Errorf("failed to get signal from indicator: %w", err)
	}

	if s.Act == indicator.ACT_HOLD {
		return nil
	}

	ts.log.Info("signal detected", "action", s.Act, "confidence", s.Confidence)

	if s.Act == indicator.ACT_BUY && s.Confidence >= ts.cfg.BuyConfidence {
		if err = ts.buy(s.Confidence); err != nil {
			return fmt.Errorf("failed to process buy signal: %w", err)
		}
	}

	if s.Act == indicator.ACT_SELL && s.Confidence >= ts.cfg.SellConfidence {
		if err = ts.sell(s.Confidence); err != nil {
			return fmt.Errorf("failed to process sell signal: %w", err)
		}
	}

	return nil
}

func (ts *TradingStrategy) buy(confidence float64) error {
	funds := ts.getAvailableFunds()
	size := ts.posScaler.GetSize(funds, confidence)
	qty, err := ts.market.GetQuantity(size, ts.symbol)
	if err != nil {
		return fmt.Errorf("failed to get position quantity: %w", err)
	}

	p, err := ts.posMan.Open(ts.symbol, qty)
	if err != nil {
		return fmt.Errorf("failed to open position: %w", err)
	}

	ts.position = p
	return nil
}

func (ts *TradingStrategy) sell(_ float64) error {
	if ts.position == nil {
		return nil
	}

	if err := ts.posMan.Close(ts.position); err != nil {
		return fmt.Errorf("failed to sell position: %w", err)
	}

	return nil
}

func (ts *TradingStrategy) getAvailableFunds() decimal.Decimal {
	available := decimal.NewFromInt(ts.cfg.Budget)
	if ts.position != nil {
		available = decimal.Max(decimal.NewFromInt(0), available.Sub(ts.position.EntryPrice))
	}

	return decimal.Min(ts.acc.GetBalance(), available)
}
