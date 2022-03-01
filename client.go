package wazirxconnectorgo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

var BASE_URL = "https://api.wazirx.com/sapi"
var GET = "get"
var POST = "post"
var DELETE = "delete"
var API_MAP = "{\"ping\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"ping\",\"url\":\"/v1/ping\"},\"time\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"time\",\"url\":\"/v1/time\"},\"system_status\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"time\",\"url\":\"/v1/systemStatus\"},\"exchange_info\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"exchange_info\",\"url\":\"/v1/exchangeInfo\"},\"tickers\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"tickers\",\"url\":\"/v1/tickers/24hr\"},\"ticker\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"ticker\",\"url\":\"/v1/depth\"},\"depth\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"depth\",\"url\":\"/v1/depth\"},\"trades\":{\"client\":\"public\",\"action\":\"get\",\"endpoint\":\"trades\",\"url\":\"/v1/trades\"},\"historical_trades\":{\"client\":\"signed\",\"action\":\"get\",\"endpoint\":\"historical_trades\",\"url\":\"/v1/historicalTrades\"},\"create_order\":{\"client\":\"signed\",\"action\":\"post\",\"endpoint\":\"order\",\"url\":\"/v1/order\"},\"create_test_order\":{\"client\":\"signed\",\"action\":\"post\",\"endpoint\":\"test_order\",\"url\":\"/v1/order/test\"},\"query_order\":{\"client\":\"signed\",\"action\":\"get\",\"endpoint\":\"order\",\"url\":\"/v1/order\"},\"cancel_order\":{\"client\":\"signed\",\"action\":\"delete\",\"endpoint\":\"order\",\"url\":\"/v1/order\"},\"open_orders\":{\"client\":\"signed\",\"action\":\"get\",\"endpoint\":\"open_orders\",\"url\":\"/v1/openOrders\"},\"cancel_open_orders\":{\"client\":\"signed\",\"action\":\"delete\",\"endpoint\":\"open_orders\",\"url\":\"/v1/openOrders\"},\"all_orders\":{\"client\":\"signed\",\"action\":\"get\",\"endpoint\":\"all_orders\",\"url\":\"/v1/allOrders\"},\"account_info\":{\"client\":\"signed\",\"action\":\"get\",\"endpoint\":\"account\",\"url\":\"/v1/account\"},\"funds_info\":{\"client\":\"signed\",\"action\":\"get\",\"endpoint\":\"funds\",\"url\":\"/v1/funds\"},\"create_auth_token\":{\"client\":\"signed\",\"action\":\"post\",\"endpoint\":\"create_auth_token\",\"url\":\"/v1/create_auth_token\"}}"

type Client struct {
	apiKey     string
	secretKey  string
	apiDetails map[string]APIDetails
}

func New(apiKey string, secretKey string) *Client {
	apiDetails := readMapperJson()
	var client = Client{apiKey: apiKey, secretKey: secretKey, apiDetails: apiDetails}
	return &client
}

func readMapperJson() map[string]APIDetails {
	data := make(map[string]APIDetails)
	err := json.Unmarshal([]byte(API_MAP), &data)

	if err == nil {
		return data
	}
	return make(map[string]APIDetails)
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

func (client Client) getEncodedParams(params map[string]interface{}) string {
	encoded := url.Values{}
	for key, value := range params {
		encoded.Set(key, fmt.Sprintf("%v", value))
	}
	return encoded.Encode()
}

func (client Client) generateSignature(params map[string]interface{}) string {
	encodedParams := client.getEncodedParams(params)
	hash := hmac.New(sha256.New, []byte(client.secretKey))
	hash.Write([]byte(encodedParams))
	sha := hex.EncodeToString(hash.Sum(nil))
	return sha
}

func (client Client) call(name string, params map[string]interface{}) (map[string]interface{}, error) {
	detail, isFound := client.getAPIDetailForName(name)
	response := make(map[string]interface{})
	var err error
	if !isFound {
		return nil, fmt.Errorf("Invalid api type")
	}
	if params == nil || len(params) == 0 {
		params = make(map[string]interface{})
	}

	if detail.Client == "signed" {
		signature := client.generateSignature(params)
		params["signature"] = signature
	}

	switch detail.Action {
	case GET:
		response, err = client.get(detail, params)
		break
	case POST:
		response, err = client.post(detail, params)
		break
	case DELETE:
		response, err = client.delete(detail, params)
		break
	default:
		err = fmt.Errorf("Invalid action type")
		break
	}
	return response, err
}

func (client Client) get(detail APIDetails, params map[string]interface{}) (map[string]interface{}, error) {
	request := &http.Client{}
	getRequest, err := http.NewRequest("GET", BASE_URL+detail.Url+"?"+client.getEncodedParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("Error while creating get request")
	}
	getRequest.Header = client.getHeaders(detail.Client)
	response, err := request.Do(getRequest)
	if err != nil {
		return nil, err
	}
	data, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return nil, readErr
	}
	defer response.Body.Close()

	res := make(map[string]interface{})
	json.Unmarshal(data, &res)
	return res, nil
}

func (client Client) post(detail APIDetails, params map[string]interface{}) (map[string]interface{}, error) {
	request := &http.Client{}
	getRequest, err := http.NewRequest("POST", BASE_URL+detail.Url, strings.NewReader(client.getEncodedParams(params)))
	getRequest.Header = client.getHeaders(detail.Client)
	response, err := request.Do(getRequest)
	if err != nil {
		return nil, err
	}
	data, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return nil, readErr
	}
	defer response.Body.Close()

	res := make(map[string]interface{})
	json.Unmarshal(data, &res)
	return res, nil
}

func (client Client) delete(detail APIDetails, params map[string]interface{}) (map[string]interface{}, error) {
	request := &http.Client{}
	getRequest, err := http.NewRequest("DELETE", BASE_URL+detail.Url+"?"+client.getEncodedParams(params), nil)
	if err != nil {
		return nil, fmt.Errorf("Error while creating get request")
	}
	getRequest.Header = client.getHeaders(detail.Client)
	response, err := request.Do(getRequest)
	if err != nil {
		return nil, err
	}
	data, readErr := ioutil.ReadAll(response.Body)
	if readErr != nil {
		return nil, readErr
	}
	defer response.Body.Close()

	res := make(map[string]interface{})
	json.Unmarshal(data, &res)
	return res, nil
}

//	ping
func (client Client) Ping() (map[string]interface{}, error) {
	return client.call("ping", nil)
}

//	time
func (client Client) Time() (map[string]interface{}, error) {
	return client.call("time", nil)
}

//	system_status
func (client Client) SystemStatus() (map[string]interface{}, error) {
	return client.call("system_status", nil)
}

//	exchange_info
func (client Client) ExchangeInfo() (map[string]interface{}, error) {
	return client.call("exchange_info", nil)
}

//	tickers
func (client Client) Tickers() (map[string]interface{}, error) {
	return client.call("tickers", nil)
}

//	ticker
func (client Client) Ticker(symbol string) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	params["symbol"] = symbol
	return client.call("ticker", params)
}

//	depth
func (client Client) Depth(symbol string, limit int) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["limit"] = limit
	return client.call("depth", params)
}

//	trades
func (client Client) Trades(symbol string, limit int) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["limit"] = limit
	return client.call("trades", params)
}

//	historical_trades
func (client Client) HistoricalTrades(symbol string, limit int) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	params["symbol"] = symbol
	params["limit"] = limit
	params["recvWindow"] = 10000
	params["timestamp"] = time.Now().Unix()
	return client.call("historical_trades", params)
}

func main() {
	var client = New("test", "test")
	params := make(map[string]interface{})
	params["symbol"] = "btcinr"

	data, err := client.SystemStatus()
	fmt.Println("Error: ", err)
	fmt.Println("Data: ", data)
}
