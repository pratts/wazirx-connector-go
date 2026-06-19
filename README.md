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
    wazirx "github.com/pratts/wazirx-connector-go"
)

// Public endpoints — no credentials required
client := wazirx.New("", "")

// Authenticated endpoints — API key and secret required
client = wazirx.New("your-api-key", "your-secret-key")
```

Generate your API key and secret at <https://wazirx.com/settings/keys>.

## Return Types

All methods return `(any, error)`. Cast the result to the appropriate type:

- **Object responses** → `map[string]any`
- **List responses** → `[]any` (Tickers, Trades, Klines, open orders, etc.)

```go
data, err := client.Ticker("btcinr")
if err != nil {
    log.Fatal(err)
}
ticker := data.(map[string]any)
fmt.Println(ticker["lastPrice"])
```

## Authentication

Signed endpoints automatically inject `timestamp` (milliseconds), `recvWindow`, and an HMAC-SHA256 `signature`. You never need to set these manually.

---

## API Reference

### General

| Method | Description |
|---|---|
| `Ping()` | Test connectivity to the REST API |
| `Time()` | Get current server time |
| `SystemStatus()` | Get system status (normal / maintenance) |
| `ExchangeInfo()` | Get trading rules and symbol metadata |

### Market Data (public)

| Method | Description |
|---|---|
| `Tickers()` | 24 hr price change statistics for all symbols |
| `Ticker(symbol)` | 24 hr price change statistics for one symbol |
| `Depth(symbol, limit)` | Order book. Valid limits: 1 5 10 20 50 100 500 1000 |
| `Trades(symbol, limit)` | Recent trades, newest-first. Max limit: 1000 |
| `Kline(symbol, interval, limit, startTime, endTime)` | OHLCV candlestick data. Pass `0` for limit/startTime/endTime to use API defaults. Valid intervals: `1m 5m 15m 30m 1h 2h 4h 6h 12h 1d 1w` |

### Market Data (signed)

| Method | Description |
|---|---|
| `HistoricalTrades(symbol, limit)` | Older trade history for a symbol |
| `MyTrades(orderId)` | Account trade history. Pass `0` for orderId to fetch most recent trades |

### Orders

| Method | Description |
|---|---|
| `CreateOrder(symbol, side, orderType, price, quantity)` | Place a spot order. `side`: `"buy"` or `"sell"`. `orderType`: `"limit"` or `"stop_limit"` |
| `CreateTestOrder(symbol, side, orderType, price, quantity)` | Validate an order without sending it to the matching engine |
| `QueryOrder(orderId)` | Get status and details of a single order |
| `OpenOrders(symbol)` | Get all open orders for a symbol |
| `AllOrders(symbol)` | Get all orders (open, cancelled, and filled) for a symbol |
| `CancelOrder(symbol, orderId)` | Cancel a single active order |
| `CancelOpenOrders(symbol)` | Cancel all active orders on a symbol |

### Account

| Method | Description |
|---|---|
| `AccountInfo()` | Account balances and permissions |
| `FundsInfo()` | Fund balances for the current account |
| `CreateAuthToken()` | Create a short-lived token for WebSocket stream authentication |

### Crypto (deposits & withdrawals)

| Method | Description |
|---|---|
| `CoinInfo()` | Metadata for all supported coins (networks, deposit/withdraw status) |
| `DepositAddress(coin, network)` | Deposit address for a coin on a given network |
| `WithdrawHistory(transferType, limit)` | Withdrawal history. `transferType`: `0` = external chain, `1` = internal (WazirX-to-WazirX) |
| `Withdraw(coin, address, amount, network, withdrawConsent)` | Submit a withdrawal request. `withdrawConsent` must be exactly: `"I hereby confirm that I am withdrawing these crypto assets."` |

### Sub-Accounts

| Method | Description |
|---|---|
| `SubAccountAccounts()` | List sub-accounts under the master account |
| `SubAccountTransferHistory()` | Fund transfer history across sub-accounts |
| `SubAccountFundTransfer(fromEmail, toEmail, currency, amount)` | Transfer funds between accounts |

---

## Examples

```go
// Get order book
depth, _ := client.Depth("btcinr", 10)

// Get candlestick data (last 5 hourly candles)
klines, _ := client.Kline("btcinr", "1h", 5, 0, 0)

// Place a limit buy order
order, err := client.CreateOrder("btcinr", "buy", "limit", "3000000", "0.001")
if err != nil {
    log.Fatal(err)
}
fmt.Println(order.(map[string]any)["orderId"])

// Cancel an order
client.CancelOrder("btcinr", 23223196)

// Get account balances
info, _ := client.AccountInfo()

// Withdraw crypto
consent := "I hereby confirm that I am withdrawing these crypto assets."
client.Withdraw("eth", "0xYourAddress", "0.05", "eth", consent)
```
