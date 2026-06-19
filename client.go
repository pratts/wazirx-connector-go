package wazirxconnectorgo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type APIDetails struct {
	Client   string
	Action   string
	Endpoint string
	Url      string
}

var BASE_URL = "https://api.wazirx.com"
var GET = "get"
var POST = "post"
var DELETE = "delete"

var API_MAP = `{
	"ping":                        {"client":"public", "action":"get",       "endpoint":"ping",                        "url":"/sapi/v1/ping"},
	"time":                        {"client":"public", "action":"get",       "endpoint":"time",                        "url":"/sapi/v1/time"},
	"system_status":               {"client":"public", "action":"get",       "endpoint":"system_status",               "url":"/sapi/v1/systemStatus"},
	"exchange_info":               {"client":"public", "action":"get",       "endpoint":"exchange_info",               "url":"/sapi/v1/exchangeInfo"},
	"tickers":                     {"client":"public", "action":"get",       "endpoint":"tickers",                     "url":"/sapi/v1/tickers/24hr"},
	"ticker":                      {"client":"public", "action":"get",       "endpoint":"ticker",                      "url":"/sapi/v1/ticker/24hr"},
	"depth":                       {"client":"public", "action":"get",       "endpoint":"depth",                       "url":"/sapi/v1/depth"},
	"trades":                      {"client":"public", "action":"get",       "endpoint":"trades",                      "url":"/sapi/v1/trades"},
	"kline":                       {"client":"public", "action":"get",       "endpoint":"kline",                       "url":"/sapi/v1/klines"},
	"historical_trades":           {"client":"signed", "action":"get",       "endpoint":"historical_trades",           "url":"/sapi/v1/historicalTrades"},
	"my_trades":                   {"client":"signed", "action":"get",       "endpoint":"my_trades",                   "url":"/sapi/v1/myTrades"},
	"create_order":                {"client":"signed", "action":"post",      "endpoint":"order",                       "url":"/sapi/v1/order"},
	"create_test_order":           {"client":"signed", "action":"post",      "endpoint":"test_order",                  "url":"/sapi/v1/order/test"},
	"query_order":                 {"client":"signed", "action":"get",       "endpoint":"order",                       "url":"/sapi/v1/order"},
	"cancel_order":                {"client":"signed", "action":"delete",    "endpoint":"order",                       "url":"/sapi/v1/order"},
	"open_orders":                 {"client":"signed", "action":"get",       "endpoint":"open_orders",                 "url":"/sapi/v1/openOrders"},
	"cancel_open_orders":          {"client":"signed", "action":"delete",    "endpoint":"open_orders",                 "url":"/sapi/v1/openOrders"},
	"all_orders":                  {"client":"signed", "action":"get",       "endpoint":"all_orders",                  "url":"/sapi/v1/allOrders"},
	"account_info":                {"client":"signed", "action":"get",       "endpoint":"account",                     "url":"/sapi/v1/account"},
	"funds_info":                  {"client":"signed", "action":"get",       "endpoint":"funds",                       "url":"/sapi/v1/funds"},
	"create_auth_token":           {"client":"signed", "action":"post",      "endpoint":"create_auth_token",           "url":"/sapi/v1/create_auth_token"},
	"coin_info":                   {"client":"signed", "action":"get",       "endpoint":"coin_info",                   "url":"/sapi/v1/coins"},
	"withdraw_history":            {"client":"signed", "action":"get",       "endpoint":"withdraw_history",            "url":"/sapi/v1/crypto/withdraws"},
	"deposit_address":             {"client":"signed", "action":"get",       "endpoint":"deposit_address",             "url":"/sapi/v1/crypto/deposits/address"},
	"withdraw":                    {"client":"signed", "action":"postquery", "endpoint":"withdraw",                    "url":"/sapi/v1/crypto/withdraws"},
	"sub_account_transfer_history":{"client":"signed", "action":"get",       "endpoint":"sub_account_transfer_history","url":"/sapi/v1/sub_account/fund_transfer/history"},
	"sub_account_accounts":        {"client":"signed", "action":"get",       "endpoint":"sub_account_accounts",        "url":"/sapi/v1/sub_account/accounts"},
	"sub_account_fund_transfer":   {"client":"signed", "action":"post",      "endpoint":"sub_account_fund_transfer",   "url":"/sapi/v1/sub_account/fund_transfer"}
}`

type Client struct {
	apiKey     string
	secretKey  string
	apiDetails map[string]APIDetails
}

func New(apiKey string, secretKey string) *Client {
	return &Client{apiKey: apiKey, secretKey: secretKey, apiDetails: readMapperJson()}
}

func readMapperJson() map[string]APIDetails {
	data := make(map[string]APIDetails)
	if err := json.Unmarshal([]byte(API_MAP), &data); err != nil {
		return make(map[string]APIDetails)
	}
	return data
}

func (client Client) getAPIDetailForName(name string) (APIDetails, bool) {
	detail, isFound := client.apiDetails[name]
	return detail, isFound
}

func (client Client) getHeaders(clientType string) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/x-www-form-urlencoded")
	if clientType == "signed" {
		headers.Add("X-API-Key", client.apiKey)
	}
	return headers
}

func (client Client) getEncodedParams(params map[string]any) string {
	encoded := url.Values{}
	for key, value := range params {
		encoded.Set(key, fmt.Sprintf("%v", value))
	}
	return encoded.Encode()
}

func (client Client) generateSignature(params map[string]any) string {
	hash := hmac.New(sha256.New, []byte(client.secretKey))
	hash.Write([]byte(client.getEncodedParams(params)))
	return hex.EncodeToString(hash.Sum(nil))
}

func (client Client) call(name string, params map[string]any) (any, error) {
	detail, isFound := client.getAPIDetailForName(name)
	if !isFound {
		return nil, fmt.Errorf("invalid api type: %s", name)
	}
	if params == nil {
		params = make(map[string]any)
	}

	if detail.Client == "signed" {
		params["recvWindow"] = 10000
		params["timestamp"] = time.Now().UnixMilli()
		params["signature"] = client.generateSignature(params)
	}

	switch detail.Action {
	case GET:
		return client.get(detail, params)
	case POST:
		return client.post(detail, params)
	case DELETE:
		return client.delete(detail, params)
	case "postquery":
		return client.postWithQuery(detail, params)
	default:
		return nil, fmt.Errorf("invalid action type: %s", detail.Action)
	}
}

func parseResponse(body io.ReadCloser) (any, error) {
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	var res any
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (client Client) get(detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequest("GET", BASE_URL+detail.Url+"?"+client.getEncodedParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating GET request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp.Body)
}

func (client Client) post(detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequest("POST", BASE_URL+detail.Url, strings.NewReader(client.getEncodedParams(params)))
	if err != nil {
		return nil, fmt.Errorf("error creating POST request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp.Body)
}

// postWithQuery sends a POST with params in the URL query string rather than the body.
// Used by the withdraw endpoint which follows this pattern.
func (client Client) postWithQuery(detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequest("POST", BASE_URL+detail.Url+"?"+client.getEncodedParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating POST request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp.Body)
}

func (client Client) delete(detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequest("DELETE", BASE_URL+detail.Url+"?"+client.getEncodedParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating DELETE request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp.Body)
}

// --- General ---

func (client Client) Ping() (any, error) {
	return client.call("ping", nil)
}

func (client Client) Time() (any, error) {
	return client.call("time", nil)
}

func (client Client) SystemStatus() (any, error) {
	return client.call("system_status", nil)
}

func (client Client) ExchangeInfo() (any, error) {
	return client.call("exchange_info", nil)
}

// --- Market Data ---

func (client Client) Tickers() (any, error) {
	return client.call("tickers", nil)
}

func (client Client) Ticker(symbol string) (any, error) {
	return client.call("ticker", map[string]any{"symbol": symbol})
}

func (client Client) Depth(symbol string, limit int) (any, error) {
	return client.call("depth", map[string]any{"symbol": symbol, "limit": limit})
}

func (client Client) Trades(symbol string, limit int) (any, error) {
	return client.call("trades", map[string]any{"symbol": symbol, "limit": limit})
}

// Kline returns candlestick data. interval must be one of: 1m 5m 15m 30m 1h 2h 4h 6h 12h 1d 1w.
// Pass 0 for limit, startTime, or endTime to omit them and use API defaults.
func (client Client) Kline(symbol, interval string, limit int, startTime, endTime int64) (any, error) {
	params := map[string]any{
		"symbol":   symbol,
		"interval": interval,
	}
	if limit > 0 {
		params["limit"] = limit
	}
	if startTime > 0 {
		params["startTime"] = startTime
	}
	if endTime > 0 {
		params["endTime"] = endTime
	}
	return client.call("kline", params)
}

func (client Client) HistoricalTrades(symbol string, limit int) (any, error) {
	return client.call("historical_trades", map[string]any{
		"symbol": symbol,
		"limit":  limit,
	})
}

// MyTrades returns trades for the account. Pass orderId=0 to fetch by symbol/time instead.
func (client Client) MyTrades(orderId int64) (any, error) {
	return client.call("my_trades", map[string]any{"orderId": orderId})
}

// --- Orders ---

// CreateOrder places a new spot order. orderType is "limit" or "stop_limit".
func (client Client) CreateOrder(symbol, side, orderType, price, quantity string) (any, error) {
	return client.call("create_order", map[string]any{
		"symbol":   symbol,
		"side":     side,
		"type":     orderType,
		"price":    price,
		"quantity": quantity,
	})
}

// CreateTestOrder validates an order without sending it to the matching engine.
func (client Client) CreateTestOrder(symbol, side, orderType, price, quantity string) (any, error) {
	return client.call("create_test_order", map[string]any{
		"symbol":   symbol,
		"side":     side,
		"type":     orderType,
		"price":    price,
		"quantity": quantity,
	})
}

func (client Client) QueryOrder(orderId int64) (any, error) {
	return client.call("query_order", map[string]any{"orderId": orderId})
}

func (client Client) CancelOrder(symbol string, orderId int64) (any, error) {
	return client.call("cancel_order", map[string]any{
		"symbol":  symbol,
		"orderId": orderId,
	})
}

func (client Client) OpenOrders(symbol string) (any, error) {
	return client.call("open_orders", map[string]any{"symbol": symbol})
}

func (client Client) CancelOpenOrders(symbol string) (any, error) {
	return client.call("cancel_open_orders", map[string]any{"symbol": symbol})
}

func (client Client) AllOrders(symbol string) (any, error) {
	return client.call("all_orders", map[string]any{"symbol": symbol})
}

// --- Account ---

func (client Client) AccountInfo() (any, error) {
	return client.call("account_info", nil)
}

func (client Client) FundsInfo() (any, error) {
	return client.call("funds_info", nil)
}

func (client Client) CreateAuthToken() (any, error) {
	return client.call("create_auth_token", nil)
}

// --- Crypto SAPIs ---

func (client Client) CoinInfo() (any, error) {
	return client.call("coin_info", nil)
}

// WithdrawHistory returns withdrawal records. transferType: 0=external, 1=internal.
func (client Client) WithdrawHistory(transferType, limit int) (any, error) {
	return client.call("withdraw_history", map[string]any{
		"transferType": transferType,
		"limit":        limit,
	})
}

func (client Client) DepositAddress(coin, network string) (any, error) {
	return client.call("deposit_address", map[string]any{
		"coin":    coin,
		"network": network,
	})
}

// Withdraw initiates a crypto withdrawal. withdrawConsent must be the exact consent string required by the API.
func (client Client) Withdraw(coin, address, amount, network, withdrawConsent string) (any, error) {
	return client.call("withdraw", map[string]any{
		"coin":            coin,
		"address":         address,
		"amount":          amount,
		"network":         network,
		"withdrawConsent": withdrawConsent,
	})
}

// --- Sub-Accounts ---

func (client Client) SubAccountTransferHistory() (any, error) {
	return client.call("sub_account_transfer_history", nil)
}

func (client Client) SubAccountAccounts() (any, error) {
	return client.call("sub_account_accounts", nil)
}

func (client Client) SubAccountFundTransfer(fromEmail, toEmail, currency string, amount float64) (any, error) {
	return client.call("sub_account_fund_transfer", map[string]any{
		"fromEmail": fromEmail,
		"toEmail":   toEmail,
		"currency":  currency,
		"amount":    amount,
	})
}
