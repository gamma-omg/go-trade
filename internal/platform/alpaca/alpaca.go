package alpaca

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
	"github.com/gamma-omg/trading-bot/internal/config"
	"github.com/gamma-omg/trading-bot/internal/market"
	"github.com/shopspring/decimal"
)

type AlpacaPlatform struct {
	cfg    config.Alpaca
	client *alpaca.Client
}

func NewAlpacaPlatform(cfg config.Alpaca) (*AlpacaPlatform, error) {
	c := alpaca.NewClient(alpaca.ClientOpts{
		BaseURL:   cfg.BaseUrl,
		APIKey:    cfg.ApiKey,
		APISecret: cfg.Secret,
	})
	_, err := c.CloseAllPositions(alpaca.CloseAllPositionsRequest{CancelOrders: true})
	if err != nil {
		return nil, fmt.Errorf("failed to close active positions: %w", err)
	}

	return &AlpacaPlatform{
		cfg:    cfg,
		client: c,
	}, nil
}

func (ap *AlpacaPlatform) GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error) {
	bars := make(chan market.Bar)
	errs := make(chan error)

	go func() {
		defer close(bars)
		defer close(errs)

		c := stream.NewCryptoClient(marketdata.US,
			stream.WithCredentials(ap.cfg.ApiKey, ap.cfg.Secret),
			stream.WithLogger(stream.DefaultLogger()),
			stream.WithCryptoBars(func(cb stream.CryptoBar) {
				b := market.Bar{
					Time:   cb.Timestamp,
					Open:   decimal.NewFromFloat(cb.Open),
					Close:  decimal.NewFromFloat(cb.Close),
					High:   decimal.NewFromFloat(cb.High),
					Low:    decimal.NewFromFloat(cb.Low),
					Volume: decimal.NewFromFloat(cb.Volume),
				}

				bars <- b
			}, symbol))

		if err := c.Connect(ctx); err != nil {
			errs <- err
			return
		}

		select {
		case <-ctx.Done():
			errs <- ctx.Err()
		case err := <-c.Terminated():
			errs <- err
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

	qty := bar.Close.Div(size)
	ord, err := ap.client.PlaceOrder(alpaca.PlaceOrderRequest{
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

	ord, err := ap.client.ClosePosition(sym, r)
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

	before := p.Price
	after := ord.FilledAvgPrice.Mul(ord.FilledQty)
	d = market.Deal{
		Symbol:    p.Asset.Symbol,
		SellTime:  *ord.FilledAt,
		SellPrice: *ord.FilledAvgPrice,
		Qty:       ord.FilledQty,
		BuyTime:   p.OpenTime,
		BuyPrice:  p.EntryPrice,
		Spend:     p.Price,
		Gain:      after.Sub(before),
	}

	return
}

func (ap *AlpacaPlatform) GetBalance() (b decimal.Decimal, err error) {
	acc, err := ap.client.GetAccount()
	if err != nil {
		err = fmt.Errorf("failed to get alpaca account: %w", err)
		return
	}

	b = acc.BuyingPower
	return
}

func (ap *AlpacaPlatform) waitFillOrder(ctx context.Context, o *alpaca.Order) (*alpaca.Order, error) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			order, err := ap.client.GetOrder(o.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to update order state: %w", err)
			}

			if order.FilledAt != nil {
				return order, nil
			}
		}
	}
}
