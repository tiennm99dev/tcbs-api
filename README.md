# stock-advisor

A portfolio stability advisor for [TCBS](https://tcinvest.tcbs.com.vn/)
accounts. It signs in to the TCBS OpenAPI, reads the holdings, cash and margin
debt of every active sub-account, adds public market and fundamental data, and
writes a Markdown report: portfolio health, a HOLD/TRIM/ADD/REVIEW action for
each holding with the rule that fired, and the most stable stocks you do not
hold yet.

**It is read-only.** The CLI only calls read endpoints. It places no orders and
moves no money.

The repo also contains the Go SDK for the TCBS OpenAPI that the CLI is built on
(see [SDK](#sdk)).

> **Disclaimer.** Informational, rule-based output; not financial advice.
> Verify prices and rules with your broker before trading. The scores are
> deterministic rules over public data; they predict nothing.

## Requirements

- Go (the version in [`go.mod`](go.mod)). The build is pure Go
  (`CGO_ENABLED=0` works) and embeds the timezone database.
- A TCBS OpenAPI key, issued in TCInvest, and the iOTP from the TCBS app. The
  CLI asks for the 6-digit OTP once per run, without echoing it.

## Setup

```bash
cp .env.example .env
```

Fill in the two keys. `.env` is gitignored; never commit it.

| Key | Meaning |
|---|---|
| `TCBS_API_KEY` | Your OpenAPI key from TCInvest |
| `TCBS_CUSTODY_CODE` | Your custody code, `105C` followed by digits, e.g. `105C000000` |

The process environment wins over `.env`, and `--custody` wins over both.

## Usage

```bash
go run ./cmd/stock-advisor --out out/report.md
```

The CLI prompts for the OTP on stderr, pulls the portfolio, fetches public
data and writes the report (mode 0600). `out/` is gitignored. Add `-v` to see
progress on stderr.

Build a binary instead:

```bash
CGO_ENABLED=0 go build ./cmd/stock-advisor
./stock-advisor --out out/report.md
```

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `--custody` | `TCBS_CUSTODY_CODE` | Custody code (`105C...`) |
| `--env-file` | `.env` | Path of the `.env` file; a missing default is fine |
| `--accounts` | all active | Comma list of sub-account numbers to include |
| `--universe` | `VN100` | Candidate universe: `VN30`, `VN100`, or a comma list of tickers |
| `--candidates` | `10` | Number of candidates to show |
| `--out` | `-` | Report path; `-` means stdout |
| `--offline-holdings` | | Read a portfolio snapshot JSON instead of TCBS (no key or OTP needed) |
| `--save-holdings` | | Write the pulled snapshot as JSON (mode 0600, custody code masked) |
| `--dump-raw` | | Directory for the raw TCBS JSON responses, for debugging. They hold your name and custody code; delete them after inspection |
| `--cache-dir` | `.cache/stock-advisor` | Public-data cache; `""` disables it |
| `--refresh` | `false` | Ignore cache freshness and refetch everything |
| `--fee-rate` | `0.001` | Broker fee per side |
| `--include-bank-cash` | `false` | Count the IA linked-bank balance in NAV |
| `--rules` | | JSON file overriding advisor thresholds |
| `--timeout` | `20s` | Per-request HTTP timeout |
| `-v` | `false` | Progress on stderr |

Exit codes: `0` success, `1` fatal error (sign-in, no portfolio, unwritable
output), `2` usage error. Missing public data is never fatal: it becomes a
warning in the report.

### Offline runs

Save a snapshot once, then re-run the analysis without TCBS:

```bash
go run ./cmd/stock-advisor --save-holdings out/holdings.json --out out/report.md
go run ./cmd/stock-advisor --offline-holdings out/holdings.json --out out/report.md
```

[`testdata/holdings.json`](testdata/holdings.json) shows the snapshot format
with fake accounts.

## What it fetches

From TCBS (with your token): sub-accounts, holdings (`/se`), cash, margin
risk and, for held symbols, current quotes (`tickerCommons`). Nothing from
TCBS is cached, and the API key, OTP and token never reach the report, the
cache, logs or error output.

Public data needs no login:

| Data | Primary | Fallbacks |
|---|---|---|
| Daily bars (2 years, VN-Index included) | VNDirect `dchart-api.vndirect.com.vn` | Vietcap `trading.vietcap.com.vn`, then KBS `kbbuddywts.kbsec.com.vn` |
| Ratios (P/E, P/B, ROE, D/E, EPS, BVPS) | VNDirect `api-finfo.vndirect.com.vn` | Vietcap fills gaps for banks |
| Bank NPL and CASA | Vietcap `iq.vietcap.com.vn` | none |
| Dividends (cash and stock) | VNDirect finfo | Vietcap |
| Annual net profit (earnings consistency) | KBS | none |
| Listing, ICB industry and peers | VNDirect finfo | none |
| VN30 / VN100 constituents | VNDirect (VN30), KBS (VN30, VN100) | VN100 falls back to VN30 |

These public endpoints are undocumented and can change without notice. When a
source fails, the run falls back to the next one or drops the metric, and the
report lists every fallback and gap under **Warnings**. The exchange
warning/control/restricted lists are not available from any of them, so that
check is not covered.

The cache keeps only normalised public data, under `--cache-dir` (mode 0700):
ratios and dividend events for 24 hours, daily bars for 6 hours during market
hours and 18 hours otherwise, profiles for 30 days and index constituents for
7 days. `--refresh` refetches everything. A cold VN100 run takes a minute or
two because each host is throttled to 3 requests per second.

## Scoring and actions

Each listed stock gets a 0-100 stability score from three pillars:

| Pillar | Weight | Metrics (general profile) |
|---|---|---|
| Price | 40 | 1-year volatility, beta, max drawdown, liquidity (ADTV) |
| Fundamentals | 35 | ROE, earnings consistency, debt/equity |
| Valuation | 25 | P/E (against the industry median when 5+ peers have one), P/B, dividend yield |

Banks, securities firms, insurers, real estate and utilities use their own
profiles (for example NPL, CASA and P/B against fair P/B for banks). A missing
metric is never scored as zero: its weight moves to the other metrics, and the
report shows coverage and confidence. Scores of 75+ are Stable, 60-74
Moderate, 45-59 Weak/Volatile and below 45 Speculative. ETFs and covered
warrants are valued but not scored.

Per holding, the first matching rule wins:

1. **REVIEW**: a red flag (not listed, negative equity, repeated losses,
   ADTV below 1bn VND, price gaps or stale bars, stale fundamentals), an
   invalid score, a score below 45, or a deep loss with a weak score.
2. **TRIM**: weight above 25%, sector above 40%, a weak score after a 30%+
   gain, or a weak score at more than 10% weight. Sizes are whole board lots
   capped at the sellable quantity.
3. **ADD**: score 70+, weight under 15%, room in the sector, cash above 5%,
   margin at most 20% and a valuation pillar of 50+.
4. **HOLD** otherwise.

Candidates are universe stocks you do not hold with a valid score of 70+,
coverage 80+, volatility at most 40%, no red flags, 2+ years listed and three
profitable years, at most 2 per sector.

Every threshold lives in
[`internal/advisor/config.go`](internal/advisor/config.go). Override any of
them with `--rules`, a JSON file that names only the fields to change:

```json
{"SingleSoftCap": 0.15, "Scoring": {"StableScore": 80}}
```

## SDK

The TCBS OpenAPI SDK lives in the `tcbs` package:

```bash
go get github.com/miti99dev/stock-advisor/tcbs
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/miti99dev/stock-advisor/tcbs"
)

func main() {
    client := tcbs.NewClient()
    ctx := context.Background()

    // Exchange the API key and iOTP for a JWT; the client keeps the token.
    if _, err := client.GetToken(ctx, "your-api-key", "123456"); err != nil {
        log.Fatal(err)
    }

    prices, err := client.GetStockPrices(ctx, []string{"FPT", "VNM"})
    if err != nil {
        log.Fatal(err)
    }
    for _, p := range prices {
        fmt.Printf("%s: %.0f\n", p.Symbol, p.MatchPrice.Float())
    }
}
```

A longer walk-through is in [`tcbs/example`](tcbs/example/main.go).

### Configuration

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

### API coverage

#### Authentication
| Method | Description |
|--------|-------------|
| `GetToken` | Exchange API Key + OTP for JWT token |

#### Account
| Method | Description |
|--------|-------------|
| `GetSubAccountInfo` | Get sub-account profile information |
| `ListAccounts` | List the sub-accounts under a custody code |

#### Stock Orders
| Method | Description |
|--------|-------------|
| `PlaceOrder` | Place a stock order |
| `UpdateOrder` | Modify an existing order |
| `CancelOrder` | Cancel existing orders |

#### Stock Queries
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

#### Market Data
| Method | Description |
|--------|-------------|
| `GetStockPrices` | Get stock ticker pricing |
| `GetForeignRoom` | Get foreign investor room info |
| `GetPutThroughInfo` | Get put-through match info |
| `GetIntradayHistory` | Get intraday price history |
| `GetSupplyDemand` | Get 15-min supply/demand data |
| `GetSupplyDemandExt` | Get extended supply/demand data |
| `GetSupplyDemandMonth` | Get monthly supply/demand data |

#### Money Management
| Method | Description |
|--------|-------------|
| `TransferMoney` | Transfer between sub-accounts |
| `DepositMargin` | Deposit margin for derivatives |
| `WithdrawMargin` | Withdraw margin for derivatives |

#### Derivatives
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

#### WebSocket Streams
| Method | Description |
|--------|-------------|
| `ConnectStockMatch` | Stock match information stream |
| `ConnectDerivativeMatch` | Derivative match information stream |
| `ConnectCenter` | General WebSocket center |
| `ConnectStockPrice` | Normal stock price stream |
| `ConnectDerivativePrice` | Derivative price stream |

### WebSocket usage

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

## Development

```bash
go test ./...
CGO_ENABLED=0 go build ./cmd/stock-advisor
```

Tests make no network calls: the TCBS and public-data clients run against
`httptest` fixture servers. `UPDATE_GOLDEN=1 go test ./cmd/... ./internal/report/`
rewrites the golden reports after an intended change.

### Generated client

`tcbsgen/` is a second TCBS client, generated by
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) from the official
spec published on the [TCBS developer portal](https://developers.tcbs.com.vn).
The portal spec cannot be generated from as published: it writes nested fields
as flat names (`assets[].symbol`), declares path parameters some paths lack,
and has no operation IDs. `tools/specfix` repairs those three defects and names
each operation after its docs page (`stock/asset` becomes `StockAsset`), using
the map in `references/tcbs-operation-ids.json`.

To pick up a new portal spec:

```bash
curl -sSfo references/tcbs-openapi-v1.0.0.json https://developers.tcbs.com.vn/specs/openapi-v1.0.0.json
go generate ./tcbsgen
```

`specfix` fails if an endpoint is added or removed without a matching entry in
the operation-ID map. The hand-written `tcbs/` package stays the one the CLI
uses: the portal spec documents fewer holdings and cash fields than
`references/openapi.json`, and which shape the live API returns is still
unconfirmed.

## License

See [LICENSE](LICENSE) for details.
