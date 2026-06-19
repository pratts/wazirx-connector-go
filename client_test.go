package wazirxconnectorgo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// --- Helpers ---

func setupMockServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(handler)
	original := BASE_URL
	BASE_URL = ts.URL
	t.Cleanup(func() {
		ts.Close()
		BASE_URL = original
	})
	return New("test-api-key", "test-secret")
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func assertMethod(t *testing.T, r *http.Request, method string) {
	t.Helper()
	if r.Method != method {
		t.Errorf("method = %q, want %q", r.Method, method)
	}
}

func assertPath(t *testing.T, r *http.Request, path string) {
	t.Helper()
	if r.URL.Path != path {
		t.Errorf("path = %q, want %q", r.URL.Path, path)
	}
}

func assertSignedParams(t *testing.T, params url.Values) {
	t.Helper()
	for _, key := range []string{"timestamp", "recvWindow", "signature"} {
		if params.Get(key) == "" {
			t.Errorf("missing signed param %q", key)
		}
	}
}

func assertAPIKeyHeader(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("X-API-Key") != "test-api-key" {
		t.Errorf("X-API-Key = %q, want %q", r.Header.Get("X-API-Key"), "test-api-key")
	}
}

// --- Unit tests ---

func TestReadMapperJson_AllEntriesPresent(t *testing.T) {
	data := readMapperJson()
	expected := []string{
		"ping", "time", "system_status", "exchange_info",
		"tickers", "ticker", "depth", "trades", "kline",
		"historical_trades", "my_trades",
		"create_order", "create_test_order", "query_order",
		"cancel_order", "open_orders", "cancel_open_orders", "all_orders",
		"account_info", "funds_info", "create_auth_token",
		"coin_info", "withdraw_history", "deposit_address", "withdraw",
		"sub_account_transfer_history", "sub_account_accounts", "sub_account_fund_transfer",
	}
	for _, key := range expected {
		if _, ok := data[key]; !ok {
			t.Errorf("API_MAP missing key %q", key)
		}
	}
}

func TestReadMapperJson_TickerURLCorrect(t *testing.T) {
	// Regression: ticker was previously mapped to /sapi/v1/depth by mistake.
	data := readMapperJson()
	if data["ticker"].Url != "/sapi/v1/ticker/24hr" {
		t.Errorf("ticker URL = %q, want /sapi/v1/ticker/24hr", data["ticker"].Url)
	}
}

func TestGetEncodedParams_SortsKeysAlphabetically(t *testing.T) {
	client := New("", "")
	got := client.getEncodedParams(map[string]any{"b": "2", "a": "1"})
	if got != "a=1&b=2" {
		t.Errorf("getEncodedParams = %q, want %q", got, "a=1&b=2")
	}
}

func TestGenerateSignature(t *testing.T) {
	client := New("", "test-secret")
	params := map[string]any{"symbol": "btcinr", "timestamp": 1000}

	// Independent reference: HMAC-SHA256("symbol=btcinr&timestamp=1000", "test-secret")
	h := hmac.New(sha256.New, []byte("test-secret"))
	h.Write([]byte("symbol=btcinr&timestamp=1000"))
	expected := hex.EncodeToString(h.Sum(nil))

	if got := client.generateSignature(params); got != expected {
		t.Errorf("generateSignature = %q, want %q", got, expected)
	}
}

func TestGetHeaders_PublicOmitsAPIKey(t *testing.T) {
	client := New("my-api-key", "")
	h := client.getHeaders("public")
	if h.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Error("Content-Type header missing or wrong")
	}
	if h.Get("X-API-Key") != "" {
		t.Error("X-API-Key must not be set for public endpoints")
	}
}

func TestGetHeaders_SignedIncludesAPIKey(t *testing.T) {
	client := New("my-api-key", "")
	h := client.getHeaders("signed")
	if h.Get("X-API-Key") != "my-api-key" {
		t.Errorf("X-API-Key = %q, want %q", h.Get("X-API-Key"), "my-api-key")
	}
}

func TestCall_UnknownNameReturnsError(t *testing.T) {
	client := New("key", "secret")
	_, err := client.call("does_not_exist", nil)
	if err == nil {
		t.Error("expected error for unknown API name, got nil")
	}
}

// --- General ---

func TestPing(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/ping")
		writeJSON(w, map[string]any{})
	})
	if _, err := client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestTime(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/time")
		writeJSON(w, map[string]any{"serverTime": 1234567890000})
	})
	if _, err := client.Time(); err != nil {
		t.Fatalf("Time: %v", err)
	}
}

func TestSystemStatus(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/systemStatus")
		writeJSON(w, map[string]any{"status": "normal"})
	})
	if _, err := client.SystemStatus(); err != nil {
		t.Fatalf("SystemStatus: %v", err)
	}
}

func TestExchangeInfo(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/exchangeInfo")
		writeJSON(w, map[string]any{"timezone": "UTC"})
	})
	if _, err := client.ExchangeInfo(); err != nil {
		t.Fatalf("ExchangeInfo: %v", err)
	}
}

// --- Market Data ---

func TestTickers_ReturnsSlice(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/tickers/24hr")
		writeJSON(w, []any{map[string]any{"symbol": "btcinr"}})
	})
	data, err := client.Tickers()
	if err != nil {
		t.Fatalf("Tickers: %v", err)
	}
	if _, ok := data.([]any); !ok {
		t.Errorf("Tickers: expected []any, got %T", data)
	}
}

func TestTicker(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/ticker/24hr")
		if r.URL.Query().Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", r.URL.Query().Get("symbol"))
		}
		writeJSON(w, map[string]any{"symbol": "btcinr"})
	})
	if _, err := client.Ticker("btcinr"); err != nil {
		t.Fatalf("Ticker: %v", err)
	}
}

func TestDepth(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/depth")
		q := r.URL.Query()
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q, want 10", q.Get("limit"))
		}
		writeJSON(w, map[string]any{"bids": []any{}})
	})
	if _, err := client.Depth("btcinr", 10); err != nil {
		t.Fatalf("Depth: %v", err)
	}
}

func TestTrades(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/trades")
		q := r.URL.Query()
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		if q.Get("limit") != "20" {
			t.Errorf("limit = %q, want 20", q.Get("limit"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.Trades("btcinr", 20); err != nil {
		t.Fatalf("Trades: %v", err)
	}
}

func TestKline_AllParamsSent(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/klines")
		q := r.URL.Query()
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		if q.Get("interval") != "1h" {
			t.Errorf("interval = %q, want 1h", q.Get("interval"))
		}
		if q.Get("limit") != "5" {
			t.Errorf("limit = %q, want 5", q.Get("limit"))
		}
		if q.Get("startTime") != "1647822960" {
			t.Errorf("startTime = %q, want 1647822960", q.Get("startTime"))
		}
		if q.Get("endTime") != "1647823020" {
			t.Errorf("endTime = %q, want 1647823020", q.Get("endTime"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.Kline("btcinr", "1h", 5, 1647822960, 1647823020); err != nil {
		t.Fatalf("Kline: %v", err)
	}
}

func TestKline_ZeroValuesOmitted(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "" {
			t.Errorf("limit should be omitted when 0, got %q", q.Get("limit"))
		}
		if q.Get("startTime") != "" {
			t.Errorf("startTime should be omitted when 0, got %q", q.Get("startTime"))
		}
		if q.Get("endTime") != "" {
			t.Errorf("endTime should be omitted when 0, got %q", q.Get("endTime"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.Kline("btcinr", "1d", 0, 0, 0); err != nil {
		t.Fatalf("Kline: %v", err)
	}
}

// --- Signed GET endpoints ---

func TestHistoricalTrades(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/historicalTrades")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.HistoricalTrades("btcinr", 10); err != nil {
		t.Fatalf("HistoricalTrades: %v", err)
	}
}

func TestMyTrades(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/myTrades")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("orderId") != "40014554366" {
			t.Errorf("orderId = %q, want 40014554366", q.Get("orderId"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.MyTrades(40014554366); err != nil {
		t.Fatalf("MyTrades: %v", err)
	}
}

func TestQueryOrder(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/order")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("orderId") != "23223196" {
			t.Errorf("orderId = %q, want 23223196", q.Get("orderId"))
		}
		writeJSON(w, map[string]any{"orderId": 23223196})
	})
	if _, err := client.QueryOrder(23223196); err != nil {
		t.Fatalf("QueryOrder: %v", err)
	}
}

func TestOpenOrders(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/openOrders")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.OpenOrders("btcinr"); err != nil {
		t.Fatalf("OpenOrders: %v", err)
	}
}

func TestAllOrders(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/allOrders")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.AllOrders("btcinr"); err != nil {
		t.Fatalf("AllOrders: %v", err)
	}
}

func TestAccountInfo(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/account")
		assertSignedParams(t, r.URL.Query())
		assertAPIKeyHeader(t, r)
		writeJSON(w, map[string]any{"canTrade": true})
	})
	if _, err := client.AccountInfo(); err != nil {
		t.Fatalf("AccountInfo: %v", err)
	}
}

func TestFundsInfo(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/funds")
		assertSignedParams(t, r.URL.Query())
		assertAPIKeyHeader(t, r)
		writeJSON(w, []any{})
	})
	if _, err := client.FundsInfo(); err != nil {
		t.Fatalf("FundsInfo: %v", err)
	}
}

func TestCoinInfo(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/coins")
		assertSignedParams(t, r.URL.Query())
		assertAPIKeyHeader(t, r)
		writeJSON(w, []any{})
	})
	if _, err := client.CoinInfo(); err != nil {
		t.Fatalf("CoinInfo: %v", err)
	}
}

func TestWithdrawHistory(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/crypto/withdraws")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("transferType") != "2" {
			t.Errorf("transferType = %q, want 2", q.Get("transferType"))
		}
		if q.Get("limit") != "5" {
			t.Errorf("limit = %q, want 5", q.Get("limit"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.WithdrawHistory(2, 5); err != nil {
		t.Fatalf("WithdrawHistory: %v", err)
	}
}

func TestDepositAddress(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/crypto/deposits/address")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("coin") != "btc" {
			t.Errorf("coin = %q, want btc", q.Get("coin"))
		}
		if q.Get("network") != "btc" {
			t.Errorf("network = %q, want btc", q.Get("network"))
		}
		writeJSON(w, map[string]any{"address": "bc1q..."})
	})
	if _, err := client.DepositAddress("btc", "btc"); err != nil {
		t.Fatalf("DepositAddress: %v", err)
	}
}

func TestSubAccountTransferHistory(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/sub_account/fund_transfer/history")
		assertSignedParams(t, r.URL.Query())
		assertAPIKeyHeader(t, r)
		writeJSON(w, []any{})
	})
	if _, err := client.SubAccountTransferHistory(); err != nil {
		t.Fatalf("SubAccountTransferHistory: %v", err)
	}
}

func TestSubAccountAccounts(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "GET")
		assertPath(t, r, "/sapi/v1/sub_account/accounts")
		assertSignedParams(t, r.URL.Query())
		assertAPIKeyHeader(t, r)
		writeJSON(w, []any{})
	})
	if _, err := client.SubAccountAccounts(); err != nil {
		t.Fatalf("SubAccountAccounts: %v", err)
	}
}

// --- Signed DELETE endpoints ---

func TestCancelOrder(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "DELETE")
		assertPath(t, r, "/sapi/v1/order")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		if q.Get("orderId") != "23223196" {
			t.Errorf("orderId = %q, want 23223196", q.Get("orderId"))
		}
		writeJSON(w, map[string]any{"status": "CANCELLED"})
	})
	if _, err := client.CancelOrder("btcinr", 23223196); err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
}

func TestCancelOpenOrders(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "DELETE")
		assertPath(t, r, "/sapi/v1/openOrders")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", q.Get("symbol"))
		}
		writeJSON(w, []any{})
	})
	if _, err := client.CancelOpenOrders("btcinr"); err != nil {
		t.Fatalf("CancelOpenOrders: %v", err)
	}
}

// --- Signed POST (body params) endpoints ---

func TestCreateOrder(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/sapi/v1/order")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		b := r.PostForm
		assertSignedParams(t, b)
		assertAPIKeyHeader(t, r)
		if b.Get("symbol") != "btcinr" {
			t.Errorf("symbol = %q, want btcinr", b.Get("symbol"))
		}
		if b.Get("side") != "buy" {
			t.Errorf("side = %q, want buy", b.Get("side"))
		}
		if b.Get("type") != "limit" {
			t.Errorf("type = %q, want limit", b.Get("type"))
		}
		if b.Get("price") != "3000000" {
			t.Errorf("price = %q, want 3000000", b.Get("price"))
		}
		if b.Get("quantity") != "0.001" {
			t.Errorf("quantity = %q, want 0.001", b.Get("quantity"))
		}
		writeJSON(w, map[string]any{"orderId": 12345})
	})
	if _, err := client.CreateOrder("btcinr", "buy", "limit", "3000000", "0.001"); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
}

func TestCreateTestOrder(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/sapi/v1/order/test")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		assertSignedParams(t, r.PostForm)
		assertAPIKeyHeader(t, r)
		writeJSON(w, map[string]any{})
	})
	if _, err := client.CreateTestOrder("btcinr", "buy", "limit", "3000000", "0.001"); err != nil {
		t.Fatalf("CreateTestOrder: %v", err)
	}
}

func TestCreateAuthToken(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/sapi/v1/create_auth_token")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		assertSignedParams(t, r.PostForm)
		assertAPIKeyHeader(t, r)
		writeJSON(w, map[string]any{"token": "abc123"})
	})
	if _, err := client.CreateAuthToken(); err != nil {
		t.Fatalf("CreateAuthToken: %v", err)
	}
}

func TestSubAccountFundTransfer(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/sapi/v1/sub_account/fund_transfer")
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		b := r.PostForm
		assertSignedParams(t, b)
		assertAPIKeyHeader(t, r)
		if b.Get("fromEmail") != "from@example.com" {
			t.Errorf("fromEmail = %q", b.Get("fromEmail"))
		}
		if b.Get("toEmail") != "to@example.com" {
			t.Errorf("toEmail = %q", b.Get("toEmail"))
		}
		if b.Get("currency") != "btc" {
			t.Errorf("currency = %q", b.Get("currency"))
		}
		if b.Get("amount") != "0.5" {
			t.Errorf("amount = %q, want 0.5", b.Get("amount"))
		}
		writeJSON(w, map[string]any{"txnId": "abc"})
	})
	if _, err := client.SubAccountFundTransfer("from@example.com", "to@example.com", "btc", 0.5); err != nil {
		t.Fatalf("SubAccountFundTransfer: %v", err)
	}
}

// --- Signed POST (query-string params) endpoints ---

func TestWithdraw(t *testing.T) {
	client := setupMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertMethod(t, r, "POST")
		assertPath(t, r, "/sapi/v1/crypto/withdraws")
		q := r.URL.Query()
		assertSignedParams(t, q)
		assertAPIKeyHeader(t, r)
		if q.Get("coin") != "eth" {
			t.Errorf("coin = %q, want eth", q.Get("coin"))
		}
		if q.Get("address") != "0xAddress123" {
			t.Errorf("address = %q, want 0xAddress123", q.Get("address"))
		}
		if q.Get("amount") != "0.1" {
			t.Errorf("amount = %q, want 0.1", q.Get("amount"))
		}
		if q.Get("network") != "eth" {
			t.Errorf("network = %q, want eth", q.Get("network"))
		}
		writeJSON(w, map[string]any{"id": "w123"})
	})
	consent := "I hereby confirm that I am withdrawing these crypto assets."
	if _, err := client.Withdraw("eth", "0xAddress123", "0.1", "eth", consent); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
}
