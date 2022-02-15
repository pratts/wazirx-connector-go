package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	jsonData, _ := ioutil.ReadFile("./api_mapper.json")
	var data map[string]APIDetails
	err := json.Unmarshal(jsonData, &data)

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
	if len(params) == 0 {
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
	return make(map[string]interface{}), nil
}

func (client Client) delete(detail APIDetails, params map[string]interface{}) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

func main() {
	var client = New("test", "test")
	params := make(map[string]interface{})
	params["symbol"] = "btcinr"

	data, err := client.call("system_status", params)
	fmt.Println("Error: ", err)
	fmt.Println("Data: ", data)
}
