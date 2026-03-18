package main

import (
	"context"
	"fmt"
	"log"

	tcbs "github.com/tiennm99/tcbs-api/src"
)

func main() {
	// Create a client (defaults to production URL)
	client := tcbs.NewClient()

	// Or use SIT environment:
	// client := tcbs.NewClient(tcbs.WithBaseURL(tcbs.SITBaseURL))

	ctx := context.Background()

	// 1. Authenticate - exchange API Key + OTP for JWT token
	token, err := client.GetToken(ctx, "your-api-key", "your-otp")
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	fmt.Printf("Token obtained, expires in %d seconds\n", token.ExpiresIn)

	// Or set token directly if you already have one:
	// client.SetToken("your-jwt-token")

	// 2. Get account info
	account, err := client.GetSubAccountInfo(ctx, "105C334455", "basicInfo,bankSubAccounts")
	if err != nil {
		log.Fatalf("Failed to get account info: %v", err)
	}
	if account.BasicInfo != nil {
		fmt.Printf("Account: %s - %s\n", account.BasicInfo.Code105C, account.BasicInfo.FullName)
	}

	// 3. Get stock prices
	prices, err := client.GetStockPrices(ctx, []string{"FPT", "VNM", "TCB"})
	if err != nil {
		log.Fatalf("Failed to get prices: %v", err)
	}
	for _, p := range prices {
		fmt.Printf("%s: ref=%.0f match=%.0f\n", p.Ticker, p.RefPrice, p.MatchPrice)
	}

	// 4. Place a stock order
	order, err := client.PlaceOrder(ctx, "0001170730", &tcbs.PlaceOrderRequest{
		Symbol:    "FPT",
		ExecType:  "NB", // Buy
		OrderQtty: 100,
		Price:     120000,
		PriceType: "LO", // Limit order
	})
	if err != nil {
		log.Fatalf("Failed to place order: %v", err)
	}
	fmt.Printf("Order placed: %s\n", order.OrderID)

	// 5. Get order book
	orders, err := client.GetOrders(ctx, "0001170730")
	if err != nil {
		log.Fatalf("Failed to get orders: %v", err)
	}
	fmt.Printf("Total orders: %d\n", orders.TotalCount)

	// 6. Get purchasing power
	pp, err := client.GetPurchasingPower(ctx, "0001170730")
	if err != nil {
		log.Fatalf("Failed to get purchasing power: %v", err)
	}
	fmt.Printf("Purchasing power: %.0f\n", pp.PP0)

	// 7. Derivative - get cash status
	cashStatus, err := client.GetDerivativeCashStatus(ctx, "105C031402", "105C031402A", "0")
	if err != nil {
		log.Fatalf("Failed to get derivative cash status: %v", err)
	}
	if cashStatus.Data != nil {
		fmt.Printf("Derivative NAV: %.0f\n", cashStatus.Data.NAV)
	}

	// 8. Get derivative market info
	derivatives, err := client.GetDerivativeMarketInfo(ctx, []string{"VN30F2503"})
	if err != nil {
		log.Fatalf("Failed to get derivative info: %v", err)
	}
	for _, d := range derivatives {
		fmt.Printf("%s: last=%.1f OI=%.0f\n", d.Ticker, d.LastPrice, d.OpenInterest)
	}
}
