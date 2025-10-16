package agent

import (
	"context"
	"fmt"
	"io"
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
	Open(ctx context.Context, a *market.Asset, size decimal.Decimal) (*market.Position, error)
	Close(ctx context.Context, p *market.Position) (market.Deal, error)
}

type positionScaler interface {
	GetSize(budget decimal.Decimal, confidence float64) decimal.Decimal
}

type account interface {
	GetBalance() (decimal.Decimal, error)
}

type reportBuilder interface {
	SubmitDeal(d market.Deal)
	Write(w io.Writer) error
}

type positionValidator interface {
	NeedClose(p *market.Position) (bool, error)
}

type TradingStrategy struct {
	log          *slog.Logger
	asset        *market.Asset
	cfg          config.Strategy
	indicator    tradingIndicator
	posMan       positionManager
	posScaler    positionScaler
	posValidator positionValidator
	acc          account
	report       reportBuilder
	position     *market.Position
}

func newTradingStrategy(asset *market.Asset, cfg config.Strategy, indicator tradingIndicator, validator positionValidator, positionManager positionManager, acc account, report reportBuilder, log *slog.Logger) *TradingStrategy {
	return &TradingStrategy{
		log:          log,
		asset:        asset,
		cfg:          cfg,
		indicator:    indicator,
		posValidator: validator,
		posScaler:    &market.LinearScaler{MaxScale: cfg.PositionScale},
		posMan:       positionManager,
		acc:          acc,
		report:       report,
		position:     nil,
	}
}

func (ts *TradingStrategy) Run(ctx context.Context) error {
	if ts.position != nil {
		close, err := ts.posValidator.NeedClose(ts.position)
		if err != nil {
			return fmt.Errorf("failed to validate position: %w", err)
		}

		if close {
			if err := ts.sell(ctx, 1.0); err != nil {
				return fmt.Errorf("failed to sell position: %w", err)
			}
		}
	}

	s, err := ts.indicator.GetSignal()
	if err != nil {
		return fmt.Errorf("failed to get signal from indicator: %w", err)
	}

	if s.Act == indicator.ACT_HOLD {
		return nil
	}

	if ts.position == nil && s.Act == indicator.ACT_BUY && s.Confidence >= ts.cfg.BuyConfidence {
		if err = ts.buy(ctx, s.Confidence); err != nil {
			return fmt.Errorf("failed to process buy signal: %w", err)
		}
	}

	if ts.position != nil && s.Act == indicator.ACT_SELL && s.Confidence >= ts.cfg.SellConfidence {
		if err = ts.sell(ctx, s.Confidence); err != nil {
			return fmt.Errorf("failed to process sell signal: %w", err)
		}
	}

	return nil
}

func (ts *TradingStrategy) buy(ctx context.Context, confidence float64) error {
	funds, err := ts.getAvailableFunds()
	if err != nil {
		return fmt.Errorf("failed to get available funds: %w", err)
	}

	size := ts.posScaler.GetSize(funds, confidence)
	p, err := ts.posMan.Open(ctx, ts.asset, size)
	if err != nil {
		return fmt.Errorf("failed to open position: %w", err)
	}

	ts.position = p
	return nil
}

func (ts *TradingStrategy) sell(ctx context.Context, _ float64) error {
	d, err := ts.posMan.Close(ctx, ts.position)
	if err != nil {
		return fmt.Errorf("failed to sell position: %w", err)
	}

	ts.report.SubmitDeal(d)
	ts.position = nil
	return nil
}

func (ts *TradingStrategy) getAvailableFunds() (decimal.Decimal, error) {
	available := decimal.NewFromInt(ts.cfg.Budget)
	if ts.position != nil {
		available = decimal.Max(decimal.NewFromInt(0), available.Sub(ts.position.EntryPrice))
	}

	balance, err := ts.acc.GetBalance()
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("failed to get current balance: %w", err)
	}

	return decimal.Min(balance, available), nil
}
