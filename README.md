# wazirx-connector-go

An unofficial Go client for the [WazirX](https://wazirx.com) spot exchange REST API.

## Requirements

Go 1.18 or later.

## Installation

```
go get github.com/pratts/wazirx-connector-go
```

## Quick Start

```go
import (
    "context"
    wazirx "github.com/pratts/wazirx-connector-go"
)

// Public endpoints — no credentials required
client := wazirx.New("", "")

// Authenticated endpoints — API key and secret required
client = wazirx.New("your-api-key", "your-secret-key")

ctx := context.Background()
data, err := client.Ping(ctx)
```

Generate your API key and secret at <https://wazirx.com/settings/keys>.

## Configuration

`New` accepts optional configuration via functional options:

```go
import (
    "net/http"
    "time"
)

client := wazirx.New("your-api-key", "your-secret-key",
    wazirx.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
    wazirx.WithRecvWindow(5000),
    wazirx.WithBaseURL("https://api.wazirx.com"),
)
```

| Option | Default | Description |
|---|---|---|
| `WithHTTPClient(c)` | 30 s timeout | Replace the underlying `*http.Client` |
| `WithRecvWindow(ms)` | `10000` | Signed-request validity window in milliseconds |
| `WithBaseURL(url)` | `https://api.wazirx.com` | Override the API base URL |

## Return Types

All methods return `(any, error)`. Cast the result to the appropriate type:

- **Object responses** → `map[string]any`
- **List responses** → `[]any` (Tickers, Trades, Klines, open orders, etc.)

```go
data, err := client.Ticker(ctx, "btcinr")
if err != nil {
    log.Fatal(err)
}
ticker := data.(map[string]any)
fmt.Println(ticker["lastPrice"])
```

### Error handling

Non-2xx responses return an `*APIError`:

```go
data, err := client.CreateOrder(ctx, "btcinr", "buy", "limit", "3000000", "0.001")
if err != nil {
    var apiErr *wazirx.APIError
    if errors.As(err, &apiErr) {
        fmt.Println("HTTP", apiErr.StatusCode, apiErr.Body)
    }
    log.Fatal(err)
}
```

## Authentication

Signed endpoints automatically inject `timestamp` (milliseconds), `recvWindow`, and an HMAC-SHA256 `signature`. You never need to set these manually.

---

## API Reference

All methods take `ctx context.Context` as their first argument.

### General

| Method | Description |
|---|---|
| `Ping(ctx)` | Test connectivity to the REST API |
| `Time(ctx)` | Get current server time |
| `SystemStatus(ctx)` | Get system status (normal / maintenance) |
| `ExchangeInfo(ctx)` | Get trading rules and symbol metadata |

### Market Data (public)

| Method | Description |
|---|---|
| `Tickers(ctx)` | 24 hr price change statistics for all symbols |
| `Ticker(ctx, symbol)` | 24 hr price change statistics for one symbol |
| `Depth(ctx, symbol, limit)` | Order book. Valid limits: 1 5 10 20 50 100 500 1000 |
| `Trades(ctx, symbol, limit)` | Recent trades, newest-first. Max limit: 1000 |
| `Kline(ctx, symbol, interval, limit, startTime, endTime)` | OHLCV candlestick data. Pass `0` for limit/startTime/endTime to use API defaults. Valid intervals: `1m 5m 15m 30m 1h 2h 4h 6h 12h 1d 1w` |

### Market Data (signed)

| Method | Description |
|---|---|
| `HistoricalTrades(ctx, symbol, limit)` | Older trade history for a symbol |
| `MyTrades(ctx, orderID)` | Account trade history. Pass `0` for orderID to fetch most recent trades |

### Orders

| Method | Description |
|---|---|
| `CreateOrder(ctx, symbol, side, orderType, price, quantity)` | Place a spot order. `side`: `"buy"` or `"sell"`. `orderType`: `"limit"` or `"stop_limit"` |
| `CreateTestOrder(ctx, symbol, side, orderType, price, quantity)` | Validate an order without sending it to the matching engine |
| `QueryOrder(ctx, orderID)` | Get status and details of a single order |
| `OpenOrders(ctx, symbol)` | Get all open orders for a symbol |
| `AllOrders(ctx, symbol)` | Get all orders (open, cancelled, and filled) for a symbol |
| `CancelOrder(ctx, symbol, orderID)` | Cancel a single active order |
| `CancelOpenOrders(ctx, symbol)` | Cancel all active orders on a symbol |

### Account

| Method | Description |
|---|---|
| `AccountInfo(ctx)` | Account balances and permissions |
| `FundsInfo(ctx)` | Fund balances for the current account |
| `CreateAuthToken(ctx)` | Create a short-lived token for WebSocket stream authentication |

### Crypto (deposits & withdrawals)

| Method | Description |
|---|---|
| `CoinInfo(ctx)` | Metadata for all supported coins (networks, deposit/withdraw status) |
| `DepositAddress(ctx, coin, network)` | Deposit address for a coin on a given network |
| `WithdrawHistory(ctx, transferType, limit)` | Withdrawal history. `transferType`: `0` = external chain, `1` = internal (WazirX-to-WazirX) |
| `Withdraw(ctx, coin, address, amount, network, withdrawConsent)` | Submit a withdrawal request. `withdrawConsent` must be exactly: `"I hereby confirm that I am withdrawing these crypto assets."` |

### Sub-Accounts

| Method | Description |
|---|---|
| `SubAccountAccounts(ctx)` | List sub-accounts under the master account |
| `SubAccountTransferHistory(ctx)` | Fund transfer history across sub-accounts |
| `SubAccountFundTransfer(ctx, fromEmail, toEmail, currency, amount)` | Transfer funds between accounts |

---

## Examples

```go
ctx := context.Background()

// Get order book
depth, _ := client.Depth(ctx, "btcinr", 10)

// Get candlestick data (last 5 hourly candles)
klines, _ := client.Kline(ctx, "btcinr", "1h", 5, 0, 0)

// Place a limit buy order
order, err := client.CreateOrder(ctx, "btcinr", "buy", "limit", "3000000", "0.001")
if err != nil {
    log.Fatal(err)
}
fmt.Println(order.(map[string]any)["orderId"])

// Cancel an order
client.CancelOrder(ctx, "btcinr", 23223196)

// Get account balances
info, _ := client.AccountInfo(ctx)

// Withdraw crypto
consent := "I hereby confirm that I am withdrawing these crypto assets."
client.Withdraw(ctx, "eth", "0xYourAddress", "0.05", "eth", consent)
```
