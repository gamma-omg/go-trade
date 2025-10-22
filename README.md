# Trading Bot

A configurable and extensible trading agent framework written in Go that supports multiple trading platforms and technical indicators. The bot can execute automated trading strategies based on configurable signal thresholds and indicator combinations.

## Features

- **Multiple Platform Support**: Trade on different platforms using a unified interface
  - Alpaca (live/paper trading)
  - Emulator (backtesting with historical data)
- **Technical Indicators**: Built-in technical analysis indicators
  - RSI (Relative Strength Index)
  - MACD (Moving Average Convergence Divergence)
  - Ensemble (weighted voting orchestrator)
- **Flexible Configuration**: YAML-based configuration for strategies, platforms, and indicators
- **Position Management**: Automated position opening/closing with take-profit and stop-loss
- **Backtesting**: Test strategies against historical data using the Emulator platform
- **Debug Support**: Visual debugging with plot generation for indicator analysis

### Configuration Structure

#### Top-Level Configuration

```yaml
strategies:
  <SYMBOL_1>: # Strategy configuration (see below)
  <SYMBOL_2>: # Strategy configuration (see below)
  ...
report: report.json # Output file for trading report
platform:
  # Platform configuration (see below)
```

### Strategy Configuration

Configure trading strategies per symbol:

```yaml
strategies:
  BTC/USD:
    budget: 1000                    # Initial trading budget (in USD)
    buy_confidence: 0.8             # Minimum confidence threshold to trigger buy (0.0-1.0)
    sell_confidence: 0.6            # Minimum confidence threshold to trigger sell (0.0-1.0)
    take_profit: 1.02               # Take profit multiplier (1.02 = 2% profit)
    stop_loss: 0.99                 # Stop loss multiplier (0.99 = 1% loss)
    position_scale: 1               # Position sizing multiplier
    prefetch: 50                    # Number of historical bars to prefetch for warmup
    market_buffer: 1024             # Internal market data buffer size
    aggregate_bars: 5               # Number of bars to aggregate (optional)
    data_dump: data/BTC.csv         # Save market data to CSV (optional)
    debug_dir: debug                # Directory for debug plots
    debug_level: 1                  # Debug level: 0=None, 1=BuyOrSell, 2=All
    debug_window: 30                # Number of bars to include in debug plots
    indicator:
      # Indicator configuration (see below)
```

### Indicator Configuration

#### RSI (Relative Strength Index)

```yaml
indicator:
  rsi:
    period: 7                      # RSI calculation period (number of bars)
    overbought: 0.6                # Overbought/oversold threshold (0.0-1.0)
```

#### MACD (Moving Average Convergence Divergence)

```yaml
indicator:
  macd:
    fast: 6                        # Fast EMA period
    slow: 13                       # Slow EMA period
    signal: 5                      # Signal line EMA period
    buy_threshold: 1.0             # Minimum MACD value to consider buying
    buy_cap: 2                     # Maximum MACD value for confidence calculation
    sell_threshold: 1.0            # Minimum MACD value to consider selling
    sell_cap: -2                   # Maximum MACD value for confidence calculation
    cross_lookback: 1              # Number of bars to look back for zero-line crossover
    ema_warmup: 3                  # EMA warmup multiplier
```

#### Ensemble (Weighted Voting Orchestrator)

Combine multiple indicators using weighted voting:

```yaml
indicator:
  ensemble:
    - weight: 1                    # Weight for this indicator's vote
      indicator:
        rsi:
          period: 7
          overbought: 0.6
    - weight: 1                    # Equal weight for MACD
      indicator:
        macd:
          fast: 6
          slow: 13
          signal: 5
          buy_threshold: 1.0
          buy_cap: 2
          sell_threshold: 1.0
          sell_cap: -2
          cross_lookback: 1
          ema_warmup: 3
```

### Platform Configuration

#### Alpaca Platform

Live and paper trading using Alpaca's trading API:

```yaml
platform:
  alpaca:
    base_url: "https://paper-api.alpaca.markets"  # Paper trading (use appropriate URL for live trading)
    api_key: "your_api_key_here"
    secret: "your_secret_here"
```

#### Emulator Platform

Backtesting with historical CSV data:

```yaml
platform:
  emulator:
    data:
      BTC: data/btcusd_1-min_data.csv           # CSV file with historical bars
    start: 2025-01-01T11:45:26.000Z             # Simulation start time
    end: 2026-01-01T08:30:12.000Z               # Simulation end time
    buy_commission: 0.002                       # Buy commission rate (0.2%)
    sell_commission: 0.002                      # Sell commission rate (0.2%)
    balance: 10000                              # Starting account balance
```

## Usage

### Running with Alpaca

1. Copy the example configuration:
   ```bash
   cp config/example/alpaca.yaml config/alpaca.yaml
   ```

2. Edit `config/alpaca.yaml` with your Alpaca API credentials

3. Run the bot:
   ```bash
   CONFIG=config/alpaca.yaml go run cmd/main.go
   ```

### Running Backtests with Emulator

1. Copy the example configuration:
   ```bash
   cp config/example/emulator.yaml config/emulator.yaml
   ```

2. Prepare your historical data CSV file with columns: `timestamp,open,high,low,close,volume`

3. Update `config/emulator.yaml` with your data file path and date range

4. Run the backtest:
   ```bash
   CONFIG=config/emulator.yaml go run cmd/main.go
   ```

5. View results in `report.json` and debug plots in the `debug/` directory

## Development

### Adding New Indicators

1. Create a new file in `internal/indicator/`
2. Implement the `tradingIndicator` interface:
   ```go
   type tradingIndicator interface {
       GetSignal() (Signal, error)
       DrawDebug(*DebugPlot) error
   }
   ```
3. Add configuration struct in `internal/config/config.go`
4. Register in the `IndicatorReference.UnmarshalYAML()` method

### Adding New Platforms

1. Create a new directory in `internal/platform/`
2. Implement the platform interface:
   ```go
   type Platform interface {
       GetBars(ctx context.Context, symbol string) (<-chan market.Bar, <-chan error)
       Prefetch(symbol string, count int) (<-chan market.Bar, error)
       Open(ctx context.Context, asset *market.Asset, size decimal.Decimal) (*market.Position, error)
       Close(ctx context.Context, p *market.Position) (market.Deal, error)
       GetBalance() (decimal.Decimal, error)
   }
   ```
3. Add configuration struct in `internal/config/config.go`
4. Register in the `PlatformReference.UnmarshalYAML()` method

## License

See LICENSE file for details.

## Disclaimer

This trading bot is provided for educational purposes only. Trading cryptocurrencies and other financial instruments involves substantial risk of loss. Use at your own risk. The authors are not responsible for any financial losses incurred through the use of this software.

Always test strategies thoroughly with paper trading or backtesting before risking real capital.
