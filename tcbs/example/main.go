// Command example walks through the main calls of the tcbs package.
//
// It reads the credentials and account identifiers from the environment so
// none of them live in source code:
//
//	TCBS_API_KEY       API key issued in TCInvest
//	TCBS_OTP           current 6-digit Smart OTP (iOTP) from the TCInvest app
//	TCBS_CUSTODY_CODE  custody code, e.g. 105C000000
//	TCBS_ACCOUNT_NO    stock sub-account number, e.g. 0001000000
//	TCBS_DERIVATIVE_SUB_ACCOUNT  optional derivative sub-account; step 7 runs only when set
//
// Step 4 places a real order. Run the example against the SIT environment
// (the default here) unless you mean to trade.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tiennm99dev/tcbs-api/tcbs"
)

func mustEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is not set", name)
	}
	return v
}

func main() {
	apiKey := mustEnv("TCBS_API_KEY")
	otp := mustEnv("TCBS_OTP")
	custodyCode := mustEnv("TCBS_CUSTODY_CODE")
	accountNo := mustEnv("TCBS_ACCOUNT_NO")

	// SIT environment. Use tcbs.NewClient() for production.
	client := tcbs.NewClient(tcbs.WithBaseURL(tcbs.SITBaseURL))

	ctx := context.Background()

	// 1. Authenticate - exchange API Key + OTP for JWT token
	token, err := client.GetToken(ctx, apiKey, otp)
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	fmt.Printf("Token obtained (%d chars)\n", len(token.Token))

	// Or set token directly if you already have one:
	// client.SetToken(os.Getenv("TCBS_TOKEN"))

	// 2. Get account info
	account, err := client.GetSubAccountInfo(ctx, custodyCode, "basicInfo,bankSubAccounts")
	if err != nil {
		log.Fatalf("Failed to get account info: %v", err)
	}
	if account.BasicInfo != nil {
		fmt.Printf("Account status: %s\n", account.BasicInfo.Status)
	}

	// 3. Get stock prices
	prices, err := client.GetStockPrices(ctx, []string{"FPT", "VNM", "TCB"})
	if err != nil {
		log.Fatalf("Failed to get prices: %v", err)
	}
	for _, p := range prices {
		fmt.Printf("%s: ref=%.0f match=%.0f\n", p.Symbol, p.RefPrice.Float(), p.MatchPrice.Float())
	}

	// 4. Place a stock order (a real order on production)
	order, err := client.PlaceOrder(ctx, accountNo, &tcbs.PlaceOrderRequest{
		Symbol:    "FPT",
		ExecType:  "NB", // Buy
		Quantity:  100,
		Price:     120000,
		PriceType: "LO", // Limit order
	})
	if err != nil {
		log.Fatalf("Failed to place order: %v", err)
	}
	fmt.Printf("Order placed: %s\n", order.OrderID)

	// 5. Get order book
	orders, err := client.GetOrders(ctx, accountNo)
	if err != nil {
		log.Fatalf("Failed to get orders: %v", err)
	}
	fmt.Printf("Total orders: %d\n", orders.TotalCount)

	// 6. Get purchasing power
	pp, err := client.GetPurchasingPower(ctx, accountNo)
	if err != nil {
		log.Fatalf("Failed to get purchasing power: %v", err)
	}
	fmt.Printf("Purchasing power: %.0f\n", pp.PP0)

	// 7. Derivative - get cash status
	if subAccount := os.Getenv("TCBS_DERIVATIVE_SUB_ACCOUNT"); subAccount != "" {
		cashStatus, err := client.GetDerivativeCashStatus(ctx, custodyCode, subAccount, "0")
		if err != nil {
			log.Fatalf("Failed to get derivative cash status: %v", err)
		}
		if cashStatus.Data != nil {
			fmt.Printf("Derivative NAV: %.0f\n", cashStatus.Data.NAV)
		}
	}

	// 8. Get derivative market info
	derivatives, err := client.GetDerivativeMarketInfo(ctx, []string{"VN30F2503"})
	if err != nil {
		log.Fatalf("Failed to get derivative info: %v", err)
	}
	for _, d := range derivatives {
		fmt.Printf("%s: last=%.1f OI=%.0f\n", d.Ticker, d.LastPrice, d.OpenInterest)
	}

	// 9. Get supply/demand (15-minute)
	sd, err := client.GetSupplyDemand(ctx, "FPT", "all")
	if err != nil {
		log.Fatalf("Failed to get supply/demand: %v", err)
	}
	fmt.Printf("Supply/demand data points: %d\n", len(sd.Data))

	// 10. Get monthly supply/demand
	sdm, err := client.GetSupplyDemandMonth(ctx, "FPT", "all")
	if err != nil {
		log.Fatalf("Failed to get monthly supply/demand: %v", err)
	}
	fmt.Printf("Monthly supply/demand data points: %d\n", len(sdm.Data))
}
