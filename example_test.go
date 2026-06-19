package wazirxconnectorgo

import (
	"fmt"
	"log"
)

func ExampleNew() {
	// Public client — no credentials needed for public endpoints.
	publicClient := New("", "")
	_ = publicClient

	// Authenticated client — required for trading, account, and crypto endpoints.
	client := New("your-api-key", "your-secret-key")
	_ = client
}

func ExampleClient_Ping() {
	client := New("", "")
	data, err := client.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}

func ExampleClient_Tickers() {
	client := New("", "")
	data, err := client.Tickers()
	if err != nil {
		log.Fatal(err)
	}
	// Response is []any; each element is a map[string]any for one symbol.
	tickers := data.([]any)
	for _, t := range tickers {
		ticker := t.(map[string]any)
		fmt.Printf("%s: %s\n", ticker["symbol"], ticker["lastPrice"])
	}
}

func ExampleClient_Ticker() {
	client := New("", "")
	data, err := client.Ticker("btcinr")
	if err != nil {
		log.Fatal(err)
	}
	ticker := data.(map[string]any)
	fmt.Println("Last price:", ticker["lastPrice"])
}

func ExampleClient_Depth() {
	client := New("", "")
	// Valid limits: 1 5 10 20 50 100 500 1000
	data, err := client.Depth("btcinr", 10)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}

func ExampleClient_Kline() {
	client := New("", "")
	// Pass 0 for limit/startTime/endTime to use API defaults.
	data, err := client.Kline("btcinr", "1h", 5, 1647822960, 1647823020)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}

func ExampleClient_CreateOrder() {
	client := New("your-api-key", "your-secret-key")
	data, err := client.CreateOrder("btcinr", "buy", "limit", "3000000", "0.001")
	if err != nil {
		log.Fatal(err)
	}
	order := data.(map[string]any)
	fmt.Println("Order ID:", order["orderId"])
}

func ExampleClient_QueryOrder() {
	client := New("your-api-key", "your-secret-key")
	data, err := client.QueryOrder(23223196)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}

func ExampleClient_CancelOrder() {
	client := New("your-api-key", "your-secret-key")
	data, err := client.CancelOrder("btcinr", 23223196)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}

func ExampleClient_AccountInfo() {
	client := New("your-api-key", "your-secret-key")
	data, err := client.AccountInfo()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}

func ExampleClient_Withdraw() {
	client := New("your-api-key", "your-secret-key")
	consent := "I hereby confirm that I am withdrawing these crypto assets."
	data, err := client.Withdraw("eth", "0xYourAddress", "0.05", "eth", consent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(data)
}
