package alpaca

import (
	"context"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata/stream"
)

type alpacaApi struct {
	apiKey string
	secret string
	client *alpaca.Client
}

func newAlpacaApi(apiKey string, secret string, baseUrl string) *alpacaApi {
	return &alpacaApi{
		apiKey: apiKey,
		secret: secret,
		client: alpaca.NewClient(alpaca.ClientOpts{
			BaseURL:   baseUrl,
			APIKey:    apiKey,
			APISecret: secret,
		}),
	}
}

func (a *alpacaApi) GetCryptoBars(symbol string, req marketdata.GetCryptoBarsRequest) ([]marketdata.CryptoBar, error) {
	return marketdata.GetCryptoBars(symbol, req)
}

func (a *alpacaApi) GetCryptoBarsStream(ctx context.Context, symbol string) (<-chan stream.CryptoBar, <-chan error) {
	errs := make(chan error)
	bars := make(chan stream.CryptoBar)

	go func() {
		c := stream.NewCryptoClient(marketdata.US,
			stream.WithCredentials(a.apiKey, a.secret),
			stream.WithLogger(stream.DefaultLogger()),
			stream.WithCryptoBars(func(cb stream.CryptoBar) {
				bars <- cb
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

func (a *alpacaApi) PlaceOrder(req alpaca.PlaceOrderRequest) (*alpaca.Order, error) {
	return a.client.PlaceOrder(req)
}

func (a *alpacaApi) ClosePosition(symbol string, req alpaca.ClosePositionRequest) (*alpaca.Order, error) {
	return a.client.ClosePosition(symbol, req)
}

func (a *alpacaApi) GetOrder(orderID string) (*alpaca.Order, error) {
	return a.client.GetOrder(orderID)
}

func (a *alpacaApi) GetAccount() (*alpaca.Account, error) {
	return a.client.GetAccount()
}

func (a *alpacaApi) CloseAllPositions(req alpaca.CloseAllPositionsRequest) ([]alpaca.Order, error) {
	return a.client.CloseAllPositions(req)
}
