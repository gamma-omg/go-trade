package alpaca

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type alpacaApiWrapper interface {
	GetCryptoBars(symbol string, req marketdata.GetCryptoBarsRequest) ([]marketdata.CryptoBar, error)
	GetCryptoBarsStream(ctx context.Context, symbol string) (<-chan stream.CryptoBar, <-chan error)
	PlaceOrder(req alpaca.PlaceOrderRequest) (*alpaca.Order, error)
	ClosePosition(symbol string, req alpaca.ClosePositionRequest) (*alpaca.Order, error)
	GetOrder(orderID string) (*alpaca.Order, error)
	GetAccount() (*alpaca.Account, error)
	CloseAllPositions(req alpaca.CloseAllPositionsRequest) ([]alpaca.Order, error)
}

type AlpacaPlatform struct {
	cfg config.Alpaca
	log *slog.Logger
	api alpacaApiWrapper
}

func newAlpacaPlatformWithApi(log *slog.Logger, cfg config.Alpaca, api alpacaApiWrapper) (*AlpacaPlatform, error) {
	_, err := api.CloseAllPositions(alpaca.CloseAllPositionsRequest{CancelOrders: true})
	if err != nil {
		return nil, fmt.Errorf("failed to close active positions: %w", err)
	}

	return &AlpacaPlatform{
		cfg: cfg,
		log: log,
		api: api,
	}, nil
}

func NewAlpacaPlatform(log *slog.Logger, cfg config.Alpaca) (*AlpacaPlatform, error) {
	api := newAlpacaApi(cfg.ApiKey, cfg.Secret, cfg.BaseUrl)
	return newAlpacaPlatformWithApi(log, cfg, api)
}

func (ap *AlpacaPlatform) Prefetch(symbol string, count int) (<-chan market.Bar, error) {
	bars, err := ap.api.GetCryptoBars(symbol, marketdata.GetCryptoBarsRequest{
		CryptoFeed: marketdata.US,
		TimeFrame:  marketdata.NewTimeFrame(1, marketdata.Min),
		Start:      time.Now().Add(time.Duration(-count-1) * time.Minute),
		TotalLimit: count + 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical data for %s: %w", symbol, err)
	}

	n := len(bars)
	if n < count {
		return nil, fmt.Errorf("failed to fetch required bars count for %s: %w", symbol, err)
	}

	res := make(chan market.Bar, count)
	defer close(res)

	for _, b := range bars[n-count:] {
		res <- market.Bar{
			Time:   b.Timestamp,
			Open:   decimal.NewFromFloat(b.Open),
			High:   decimal.NewFromFloat(b.High),
			Low:    decimal.NewFromFloat(b.Low),
			Close:  decimal.NewFromFloat(b.Close),
			Volume: decimal.NewFromFloat(b.Volume),
		}
	}

	return res, nil
}

func (ap *AlpacaPlatform) GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error) {
	bars := make(chan market.Bar)

	cBars, errs := ap.api.GetCryptoBarsStream(ctx, symbol)
	go func() {
		defer close(bars)
		for cb := range cBars {
			bars <- market.Bar{
				Time:   cb.Timestamp,
				Open:   decimal.NewFromFloat(cb.Open),
				Close:  decimal.NewFromFloat(cb.Close),
				High:   decimal.NewFromFloat(cb.High),
				Low:    decimal.NewFromFloat(cb.Low),
				Volume: decimal.NewFromFloat(cb.Volume),
			}
		}
	}()

	return bars, errs
}

func (ap *AlpacaPlatform) Open(ctx context.Context, asset *market.Asset, size decimal.Decimal) (p *market.Position, err error) {
	bar, err := asset.GetLastBar()
	if err != nil {
		err = fmt.Errorf("failed to get symbold price: %w", err)
		return
	}

	qty := size.Div(bar.Close)
	ap.log.Info("open alpaca position", slog.String("symbol", asset.Symbol), slog.String("qty", qty.String()), slog.String("size", size.String()))

	ord, err := ap.api.PlaceOrder(alpaca.PlaceOrderRequest{
		Side:        alpaca.Buy,
		Symbol:      asset.Symbol,
		Qty:         &qty,
		Type:        alpaca.Market,
		TimeInForce: alpaca.IOC,
	})
	if err != nil {
		err = fmt.Errorf("failed to place order: %w", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ord, err = ap.waitFillOrder(ctx, ord)
	if err != nil {
		err = fmt.Errorf("failed to fill order: %w", err)
		return
	}

	p = &market.Position{
		Asset:      asset,
		EntryPrice: *ord.FilledAvgPrice,
		OpenTime:   *ord.FilledAt,
		Qty:        ord.FilledQty,
		Price:      ord.FilledQty.Mul(*ord.FilledAvgPrice),
	}

	return
}

func (ap *AlpacaPlatform) Close(ctx context.Context, p *market.Position) (d market.Deal, err error) {
	r := alpaca.ClosePositionRequest{Percentage: decimal.NewFromInt(100)}

	// for some reason in Alpaca we buy BTC/USD but sell BTCUSD symbol
	sym := strings.Replace(p.Asset.Symbol, "/", "", -1)

	ord, err := ap.api.ClosePosition(sym, r)
	if err != nil {
		err = fmt.Errorf("failed to close position: %w", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ord, err = ap.waitFillOrder(ctx, ord)
	if err != nil {
		err = fmt.Errorf("failed to fill order: %w", err)
		return
	}

	d = market.Deal{
		Symbol:    p.Asset.Symbol,
		SellTime:  *ord.FilledAt,
		SellPrice: *ord.FilledAvgPrice,
		Qty:       ord.FilledQty,
		BuyTime:   p.OpenTime,
		BuyPrice:  p.EntryPrice,
		Spend:     p.Price,
	}

	return
}

func (ap *AlpacaPlatform) GetBalance() (b decimal.Decimal, err error) {
	acc, err := ap.api.GetAccount()
	if err != nil {
		err = fmt.Errorf("failed to get alpaca account: %w", err)
		return
	}

	b = acc.BuyingPower
	return
}

func (ap *AlpacaPlatform) waitFillOrder(ctx context.Context, o *alpaca.Order) (*alpaca.Order, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			order, err := ap.api.GetOrder(o.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to update order state: %w", err)
			}

			if order.FilledAt != nil {
				return order, nil
			}
		}
	}
}
