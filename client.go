// Package wazirxconnectorgo is an unofficial Go client for the WazirX spot exchange REST API.
//
// Create a client with New, then call any of the typed methods. Public endpoints work
// without credentials; signed endpoints require a valid API key and secret.
// Timestamp, recvWindow, and HMAC-SHA256 signature are injected automatically for
// every signed call — callers never need to manage them manually.
package wazirxconnectorgo

import (
	"context"
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

const (
	actionGet       = "get"
	actionPost      = "post"
	actionDelete    = "delete"
	actionPostQuery = "postquery"
)

// APIDetails holds the routing metadata for a single API endpoint.
type APIDetails struct {
	Client string // "public" or "signed"
	Action string // see action* constants
	URL    string // path, e.g. "/sapi/v1/ping"
}

// APIError is returned when the WazirX API responds with a non-2xx HTTP status.
// Callers can inspect StatusCode and Body for details, or use errors.As.
type APIError struct {
	StatusCode int
	Body       any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %v", e.StatusCode, e.Body)
}

var endpointMap = map[string]APIDetails{
	"ping":                         {Client: "public", Action: actionGet,       URL: "/sapi/v1/ping"},
	"time":                         {Client: "public", Action: actionGet,       URL: "/sapi/v1/time"},
	"system_status":                {Client: "public", Action: actionGet,       URL: "/sapi/v1/systemStatus"},
	"exchange_info":                {Client: "public", Action: actionGet,       URL: "/sapi/v1/exchangeInfo"},
	"tickers":                      {Client: "public", Action: actionGet,       URL: "/sapi/v1/tickers/24hr"},
	"ticker":                       {Client: "public", Action: actionGet,       URL: "/sapi/v1/ticker/24hr"},
	"depth":                        {Client: "public", Action: actionGet,       URL: "/sapi/v1/depth"},
	"trades":                       {Client: "public", Action: actionGet,       URL: "/sapi/v1/trades"},
	"kline":                        {Client: "public", Action: actionGet,       URL: "/sapi/v1/klines"},
	"historical_trades":            {Client: "signed", Action: actionGet,       URL: "/sapi/v1/historicalTrades"},
	"my_trades":                    {Client: "signed", Action: actionGet,       URL: "/sapi/v1/myTrades"},
	"create_order":                 {Client: "signed", Action: actionPost,      URL: "/sapi/v1/order"},
	"create_test_order":            {Client: "signed", Action: actionPost,      URL: "/sapi/v1/order/test"},
	"query_order":                  {Client: "signed", Action: actionGet,       URL: "/sapi/v1/order"},
	"cancel_order":                 {Client: "signed", Action: actionDelete,    URL: "/sapi/v1/order"},
	"open_orders":                  {Client: "signed", Action: actionGet,       URL: "/sapi/v1/openOrders"},
	"cancel_open_orders":           {Client: "signed", Action: actionDelete,    URL: "/sapi/v1/openOrders"},
	"all_orders":                   {Client: "signed", Action: actionGet,       URL: "/sapi/v1/allOrders"},
	"account_info":                 {Client: "signed", Action: actionGet,       URL: "/sapi/v1/account"},
	"funds_info":                   {Client: "signed", Action: actionGet,       URL: "/sapi/v1/funds"},
	"create_auth_token":            {Client: "signed", Action: actionPost,      URL: "/sapi/v1/create_auth_token"},
	"coin_info":                    {Client: "signed", Action: actionGet,       URL: "/sapi/v1/coins"},
	"withdraw_history":             {Client: "signed", Action: actionGet,       URL: "/sapi/v1/crypto/withdraws"},
	"deposit_address":              {Client: "signed", Action: actionGet,       URL: "/sapi/v1/crypto/deposits/address"},
	"withdraw":                     {Client: "signed", Action: actionPostQuery, URL: "/sapi/v1/crypto/withdraws"},
	"sub_account_transfer_history": {Client: "signed", Action: actionGet,       URL: "/sapi/v1/sub_account/fund_transfer/history"},
	"sub_account_accounts":         {Client: "signed", Action: actionGet,       URL: "/sapi/v1/sub_account/accounts"},
	"sub_account_fund_transfer":    {Client: "signed", Action: actionPost,      URL: "/sapi/v1/sub_account/fund_transfer"},
}

// Client is the WazirX API client. Use New to create one.
type Client struct {
	apiKey     string
	secretKey  string
	baseURL    string
	recvWindow int
	httpClient *http.Client
	apiDetails map[string]APIDetails
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the default HTTP client.
// Use this to set custom TLS config, proxy settings, or transport.
func WithHTTPClient(c *http.Client) Option {
	return func(client *Client) { client.httpClient = c }
}

// WithBaseURL overrides the default API base URL.
// Primarily useful for testing or staging environments.
func WithBaseURL(u string) Option {
	return func(client *Client) { client.baseURL = u }
}

// WithRecvWindow sets the recvWindow parameter (in milliseconds) for signed requests.
// Must not exceed 60000. Defaults to 10000.
func WithRecvWindow(ms int) Option {
	return func(client *Client) { client.recvWindow = ms }
}

// New creates a WazirX API client. Pass empty strings for apiKey and secretKey
// when only public endpoints are needed.
func New(apiKey, secretKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		secretKey:  secretKey,
		baseURL:    "https://api.wazirx.com",
		recvWindow: 10000,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiDetails: endpointMap,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (client *Client) getHeaders(clientType string) http.Header {
	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	if clientType == "signed" {
		headers.Set("X-API-Key", client.apiKey)
	}
	return headers
}

func (client *Client) encodeParams(params map[string]any) string {
	v := url.Values{}
	for key, value := range params {
		v.Set(key, fmt.Sprintf("%v", value))
	}
	return v.Encode()
}

func (client *Client) generateSignature(params map[string]any) string {
	hash := hmac.New(sha256.New, []byte(client.secretKey))
	hash.Write([]byte(client.encodeParams(params)))
	return hex.EncodeToString(hash.Sum(nil))
}

func (client *Client) call(ctx context.Context, name string, params map[string]any) (any, error) {
	detail, ok := client.apiDetails[name]
	if !ok {
		return nil, fmt.Errorf("unknown endpoint: %s", name)
	}
	if params == nil {
		params = make(map[string]any)
	}

	if detail.Client == "signed" {
		params["recvWindow"] = client.recvWindow
		params["timestamp"] = time.Now().UnixMilli()
		params["signature"] = client.generateSignature(params)
	}

	switch detail.Action {
	case actionGet:
		return client.get(ctx, detail, params)
	case actionPost:
		return client.post(ctx, detail, params)
	case actionDelete:
		return client.delete(ctx, detail, params)
	case actionPostQuery:
		return client.postWithQuery(ctx, detail, params)
	default:
		return nil, fmt.Errorf("unknown action: %s", detail.Action)
	}
}

func parseResponse(resp *http.Response) (any, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res any
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: res}
	}
	return res, nil
}

func (client *Client) get(ctx context.Context, detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+detail.URL+"?"+client.encodeParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating GET request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp)
}

func (client *Client) post(ctx context.Context, detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+detail.URL, strings.NewReader(client.encodeParams(params)))
	if err != nil {
		return nil, fmt.Errorf("error creating POST request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp)
}

// postWithQuery sends a POST with params in the URL query string rather than the body.
// Used by the withdraw endpoint which follows this pattern.
func (client *Client) postWithQuery(ctx context.Context, detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+detail.URL+"?"+client.encodeParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating POST request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp)
}

func (client *Client) delete(ctx context.Context, detail APIDetails, params map[string]any) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, client.baseURL+detail.URL+"?"+client.encodeParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating DELETE request: %w", err)
	}
	req.Header = client.getHeaders(detail.Client)
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return parseResponse(resp)
}

// --- General ---

// Ping tests connectivity to the REST API.
func (client *Client) Ping(ctx context.Context) (any, error) {
	return client.call(ctx, "ping", nil)
}

// Time returns the current server time.
func (client *Client) Time(ctx context.Context) (any, error) {
	return client.call(ctx, "time", nil)
}

// SystemStatus returns the current system status (normal / system maintenance).
func (client *Client) SystemStatus(ctx context.Context) (any, error) {
	return client.call(ctx, "system_status", nil)
}

// ExchangeInfo returns current exchange trading rules and symbol information.
func (client *Client) ExchangeInfo(ctx context.Context) (any, error) {
	return client.call(ctx, "exchange_info", nil)
}

// --- Market Data ---

// Tickers returns 24-hour price change statistics for all symbols.
// The response is a []any, each element being a map[string]any for one symbol.
func (client *Client) Tickers(ctx context.Context) (any, error) {
	return client.call(ctx, "tickers", nil)
}

// Ticker returns 24-hour price change statistics for a single symbol.
func (client *Client) Ticker(ctx context.Context, symbol string) (any, error) {
	return client.call(ctx, "ticker", map[string]any{"symbol": symbol})
}

// Depth returns the order book for a symbol. Valid limits: 1 5 10 20 50 100 500 1000.
func (client *Client) Depth(ctx context.Context, symbol string, limit int) (any, error) {
	return client.call(ctx, "depth", map[string]any{"symbol": symbol, "limit": limit})
}

// Trades returns recent trades for a symbol, sorted newest-first. Max limit: 1000.
func (client *Client) Trades(ctx context.Context, symbol string, limit int) (any, error) {
	return client.call(ctx, "trades", map[string]any{"symbol": symbol, "limit": limit})
}

// Kline returns OHLCV candlestick data. interval must be one of:
// 1m 5m 15m 30m 1h 2h 4h 6h 12h 1d 1w.
// Pass 0 for limit, startTime, or endTime to omit them and use API defaults.
func (client *Client) Kline(ctx context.Context, symbol, interval string, limit int, startTime, endTime int64) (any, error) {
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
	return client.call(ctx, "kline", params)
}

// HistoricalTrades returns older trades for a symbol (signed — requires API key). Max limit: 1000.
func (client *Client) HistoricalTrades(ctx context.Context, symbol string, limit int) (any, error) {
	return client.call(ctx, "historical_trades", map[string]any{
		"symbol": symbol,
		"limit":  limit,
	})
}

// MyTrades returns the account's trade history filtered by orderID.
// Pass orderID=0 to omit the filter and fetch the most recent trades.
func (client *Client) MyTrades(ctx context.Context, orderID int64) (any, error) {
	return client.call(ctx, "my_trades", map[string]any{"orderId": orderID})
}

// --- Orders ---

// CreateOrder places a new spot order. side is "buy" or "sell".
// orderType is "limit" or "stop_limit".
func (client *Client) CreateOrder(ctx context.Context, symbol, side, orderType, price, quantity string) (any, error) {
	return client.call(ctx, "create_order", map[string]any{
		"symbol":   symbol,
		"side":     side,
		"type":     orderType,
		"price":    price,
		"quantity": quantity,
	})
}

// CreateTestOrder validates an order and signature without sending it to the matching engine.
func (client *Client) CreateTestOrder(ctx context.Context, symbol, side, orderType, price, quantity string) (any, error) {
	return client.call(ctx, "create_test_order", map[string]any{
		"symbol":   symbol,
		"side":     side,
		"type":     orderType,
		"price":    price,
		"quantity": quantity,
	})
}

// QueryOrder returns the status and details of a single order.
func (client *Client) QueryOrder(ctx context.Context, orderID int64) (any, error) {
	return client.call(ctx, "query_order", map[string]any{"orderId": orderID})
}

// CancelOrder cancels an active order.
func (client *Client) CancelOrder(ctx context.Context, symbol string, orderID int64) (any, error) {
	return client.call(ctx, "cancel_order", map[string]any{
		"symbol":  symbol,
		"orderId": orderID,
	})
}

// OpenOrders returns all currently open orders for a symbol.
func (client *Client) OpenOrders(ctx context.Context, symbol string) (any, error) {
	return client.call(ctx, "open_orders", map[string]any{"symbol": symbol})
}

// CancelOpenOrders cancels all active orders on a symbol.
func (client *Client) CancelOpenOrders(ctx context.Context, symbol string) (any, error) {
	return client.call(ctx, "cancel_open_orders", map[string]any{"symbol": symbol})
}

// AllOrders returns all orders (active, cancelled, and filled) for a symbol.
func (client *Client) AllOrders(ctx context.Context, symbol string) (any, error) {
	return client.call(ctx, "all_orders", map[string]any{"symbol": symbol})
}

// --- Account ---

// AccountInfo returns current account information including balances and permissions.
func (client *Client) AccountInfo(ctx context.Context) (any, error) {
	return client.call(ctx, "account_info", nil)
}

// FundsInfo returns fund balances for the current account.
func (client *Client) FundsInfo(ctx context.Context) (any, error) {
	return client.call(ctx, "funds_info", nil)
}

// CreateAuthToken creates a short-lived token used to authenticate WebSocket streams.
func (client *Client) CreateAuthToken(ctx context.Context) (any, error) {
	return client.call(ctx, "create_auth_token", nil)
}

// --- Crypto SAPIs ---

// CoinInfo returns metadata for all supported coins (networks, deposit/withdraw status, etc.).
func (client *Client) CoinInfo(ctx context.Context) (any, error) {
	return client.call(ctx, "coin_info", nil)
}

// WithdrawHistory returns the account's withdrawal history.
// transferType: 0 = external chain withdrawal, 1 = internal (WazirX-to-WazirX) transfer.
func (client *Client) WithdrawHistory(ctx context.Context, transferType, limit int) (any, error) {
	return client.call(ctx, "withdraw_history", map[string]any{
		"transferType": transferType,
		"limit":        limit,
	})
}

// DepositAddress returns the deposit address for a coin on a given network.
func (client *Client) DepositAddress(ctx context.Context, coin, network string) (any, error) {
	return client.call(ctx, "deposit_address", map[string]any{
		"coin":    coin,
		"network": network,
	})
}

// Withdraw submits a crypto withdrawal request.
// withdrawConsent must be exactly: "I hereby confirm that I am withdrawing these crypto assets."
func (client *Client) Withdraw(ctx context.Context, coin, address, amount, network, withdrawConsent string) (any, error) {
	return client.call(ctx, "withdraw", map[string]any{
		"coin":            coin,
		"address":         address,
		"amount":          amount,
		"network":         network,
		"withdrawConsent": withdrawConsent,
	})
}

// --- Sub-Accounts ---

// SubAccountTransferHistory returns the fund transfer history across sub-accounts.
func (client *Client) SubAccountTransferHistory(ctx context.Context) (any, error) {
	return client.call(ctx, "sub_account_transfer_history", nil)
}

// SubAccountAccounts returns the list of sub-accounts under the master account.
func (client *Client) SubAccountAccounts(ctx context.Context) (any, error) {
	return client.call(ctx, "sub_account_accounts", nil)
}

// SubAccountFundTransfer transfers funds between a master account and a sub-account.
// amount is a string to preserve decimal precision (e.g. "0.5", "100.00").
func (client *Client) SubAccountFundTransfer(ctx context.Context, fromEmail, toEmail, currency, amount string) (any, error) {
	return client.call(ctx, "sub_account_fund_transfer", map[string]any{
		"fromEmail": fromEmail,
		"toEmail":   toEmail,
		"currency":  currency,
		"amount":    amount,
	})
}
