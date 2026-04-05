# tcbs-api

Go SDK for [TCBS](https://tcinvest.tcbs.com.vn/) OpenAPI trading platform.

## Install

```bash
go get github.com/tiennm99/tcbs-api
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    tcbs "github.com/tiennm99/tcbs-api"
)

func main() {
    client := tcbs.NewClient()
    ctx := context.Background()

    // Authenticate
    token, err := client.GetToken(ctx, "your-api-key", "your-otp")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Authenticated:", token.Token)

    // Get stock prices
    prices, err := client.GetStockPrices(ctx, []string{"FPT", "VNM"})
    if err != nil {
        log.Fatal(err)
    }
    for _, p := range prices {
        fmt.Printf("%s: %.0f\n", p.Ticker, p.MatchPrice)
    }
}
```

## Configuration

```go
// Production (default)
client := tcbs.NewClient()

// SIT environment
client := tcbs.NewClient(tcbs.WithBaseURL(tcbs.SITBaseURL))

// Custom HTTP client
client := tcbs.NewClient(tcbs.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}))

// Pre-set token
client := tcbs.NewClient(tcbs.WithToken("your-jwt-token"))
```

## API Coverage

### Authentication
| Method | Description |
|--------|-------------|
| `GetToken` | Exchange API Key + OTP for JWT token |

### Account
| Method | Description |
|--------|-------------|
| `GetSubAccountInfo` | Get sub-account profile information |

### Stock Orders
| Method | Description |
|--------|-------------|
| `PlaceOrder` | Place a stock order |
| `UpdateOrder` | Modify an existing order |
| `CancelOrder` | Cancel existing orders |

### Stock Queries
| Method | Description |
|--------|-------------|
| `GetOrders` | Get order book |
| `GetOrderByID` | Get specific order by ID |
| `GetMatchingDetails` | Get order matching details |
| `GetPurchasingPower` | Get purchasing power |
| `GetPurchasingPowerBySymbol` | Get purchasing power for symbol |
| `GetPurchasingPowerBySymbolPrice` | Get purchasing power for symbol at price |
| `GetMarginQuota` | Get margin quota |
| `GetMarginAccountInfo` | Get margin account risk info |
| `GetSupplementaryLoanPackages` | Get loan package details |
| `GetLoans` | Get loan list |
| `GetStockAssets` | Get stock holdings |
| `GetCashBalance` | Get cash balance |
| `GetCashStatements` | Get cash statement history |
| `GetMarginInfo` | Get debt inquiry info |

### Market Data
| Method | Description |
|--------|-------------|
| `GetStockPrices` | Get stock ticker pricing |
| `GetForeignRoom` | Get foreign investor room info |
| `GetPutThroughInfo` | Get put-through match info |
| `GetIntradayHistory` | Get intraday price history |
| `GetSupplyDemand` | Get 15-min supply/demand data |
| `GetSupplyDemandExt` | Get extended supply/demand data |
| `GetSupplyDemandMonth` | Get monthly supply/demand data |

### Money Management
| Method | Description |
|--------|-------------|
| `TransferMoney` | Transfer between sub-accounts |
| `DepositMargin` | Deposit margin for derivatives |
| `WithdrawMargin` | Withdraw margin for derivatives |

### Derivatives
| Method | Description |
|--------|-------------|
| `GetDerivativeCashStatus` | Get derivative cash/margin overview |
| `GetDerivativeClosedPositions` | Get closed positions |
| `GetDerivativeOpenPositions` | Get open positions |
| `GetDerivativeNormalOrders` | List normal orders |
| `GetDerivativeConditionOrders` | List conditional orders |
| `PlaceDerivativeNormalOrder` | Place normal order |
| `PlaceDerivativeConditionOrder` | Place conditional order |
| `ChangeDerivativeNormalOrder` | Modify normal order |
| `ChangeDerivativeConditionOrder` | Modify conditional order |
| `CancelDerivativeNormalOrder` | Cancel normal order |
| `CancelDerivativeConditionOrder` | Cancel conditional order |
| `GetDerivativeMarketInfo` | Get derivative contract pricing |

### WebSocket Streams
| Method | Description |
|--------|-------------|
| `ConnectStockMatch` | Stock match information stream |
| `ConnectDerivativeMatch` | Derivative match information stream |
| `ConnectCenter` | General WebSocket center |
| `ConnectStockPrice` | Normal stock price stream |
| `ConnectDerivativePrice` | Derivative price stream |

## WebSocket Usage

```go
ctx := context.Background()
ws, err := client.ConnectStockPrice(ctx, func(msgType websocket.MessageType, data []byte) {
    fmt.Println("Received:", string(data))
})
if err != nil {
    log.Fatal(err)
}
defer ws.Close()

// Send subscription message
ws.SendJSON(ctx, map[string]string{"action": "subscribe", "ticker": "FPT"})
```

## License

See [LICENSE](LICENSE) for details.
