# wazirx-connector-golang
This is an unofficial Golang wrapper for the Wazirx exchange REST for personal usage 

Wazirx Go connector is a set of helper methods to connecto with Wazirx.com platform via APIs. 

## Usage:
1. Generate API KEY from Wazirx website using the https://wazirx.com/settings/keys

2. Get the library using the command
```
go get github.com/pratts/wazirx-connector-go
```

### API usage (All methods return JSON string as return type)
```
// Importing the rest client class
import (
  wazirxconnectorgo "github.com/pratts/wazirx-connector-go"
)

// Initialize the client object
var client = wazirxconnectorgo.New(apiKey, apiSecret)

// Test connectivity by sending ping
client.Ping();

// Get system status
client.SystemStatus();

// Get server time
client.Time();

// Get exchange info
client.ExchangeInfo();

// 24hr tickers price change statistics
client.Tickers();

// 24hr ticker price change statistics for a symbol : here symbol name(example "btcinr" or one of the symbols from exchange info method)
client.Ticker(symbolName);

// Order book : limit value Valid limits:[1, 5, 10, 20, 50, 100, 500, 1000]
client.Depth(symbolName, limit)

// Recent trades list : limit value Default 500; max 1000.
client.Trades(symbolName, limit)

// Old trade lookup (Market Data)
client.HistoricalTrades(symbolName, limit)

```
