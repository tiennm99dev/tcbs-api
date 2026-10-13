# Phase 2: CLI skeleton, config and `.env`, portfolio pull

## Files

Create:

- `cmd/stock-advisor/main.go`: `main` calls `run`, then `os.Exit(code)`
- `cmd/stock-advisor/run.go`: `func run(ctx context.Context, args []string, env Env, stdin io.Reader, stdout, stderr io.Writer) int`
- `internal/config/config.go`, `internal/config/dotenv.go`,
  `internal/config/otp.go`, plus tests
- `internal/portfolio/model.go`, `internal/portfolio/tcbs.go`,
  `internal/portfolio/offline.go`, `internal/portfolio/aggregate.go`, plus tests
- `.env.example`: lists the keys with empty values. It is committed.
- `testdata/holdings.json`: a synthetic offline snapshot with fake account
  numbers. It is committed.

`.gitignore`: `.env` is already ignored. Add `/.cache/`, `/out/` and
`/raw/` (the default `--dump-raw` dir).

## Configuration

Precedence: flag, then process env, then the `.env` file, then the default.

| Key / flag | Meaning | Default |
|---|---|---|
| `TCBS_API_KEY` (process env or `.env`; no flag) | API key | required unless `--offline-holdings` |
| `TCBS_CUSTODY_CODE` / `--custody` | custody code (105C...) | required unless offline |
| `--env-file` | `.env` path | `.env` (it is fine if missing) |
| `--accounts` | comma list of sub-account numbers to include | all active |
| `--candidates` | number of candidates to show | 10 |
| `--universe` | `VN30`, `VN100`, or a comma list of tickers | `VN100` (falls back to `VN30` if the VN100 list cannot be fetched) |
| `--out` | report path; `-` means stdout | `-` |
| `--offline-holdings` | read a portfolio snapshot JSON instead of TCBS | - |
| `--save-holdings` | write the pulled snapshot as JSON (mode 0600) | - |
| `--dump-raw` | dir for the raw TCBS JSON responses (0600, debug) | - |
| `--cache-dir` | public-data cache | `.cache/stock-advisor` |
| `--refresh` | ignore cache freshness and refetch | false |
| `--fee-rate` | broker fee per side | 0.001 |
| `--include-bank-cash` | count IA linked-bank balance in NAV | false |
| `--rules` | JSON file overriding `advisor.Config` thresholds | - |
| `--timeout` | per-request HTTP timeout | 20s |
| `-v` | progress on stderr | false |

`internal/config`:

```go
type Config struct {
    APIKey, CustodyCode string // APIKey is never printed
    Accounts []string
    Candidates int
    Universe []string // or a group name
    Out, OfflineHoldings, SaveHoldings, DumpRaw, CacheDir, RulesFile string
    Refresh, IncludeBankCash, Verbose bool
    FeeRate float64
    Timeout time.Duration
}
func Parse(args []string, getenv func(string) string) (Config, error)
// LoadDotEnv parses KEY=VALUE lines. It accepts a leading `export`, # comments,
// and single or double quotes. It never overrides keys already set in the
// process env. Errors name the line number, never the value.
func LoadDotEnv(path string) (map[string]string, error)
func (c Config) String() string // redacts APIKey as "***"
// PromptOTP writes the prompt to stderr. When stdin is a terminal it reads
// with term.ReadPassword (no echo); otherwise it reads one line from in. It
// trims the input and requires 6 digits.
func PromptOTP(in io.Reader, fd int, out io.Writer) (string, error)
```

Use `golang.org/x/term` at its latest version (`go get golang.org/x/term@latest`).
There is no OTP env var or flag, so the OTP is never stored.

## Portfolio

`internal/portfolio/model.go`. All money is in VND as int64; quantities are
int64.

```go
type Snapshot struct {
    AsOf     time.Time `json:"asOf"`
    Custody  string    `json:"custody"` // masked on save: 105C***1234
    Accounts []Account `json:"accounts"`
}
type Account struct {
    AccountNo, Type, Status string
    Cash      Cash
    Debt      Debt // zero for non-margin accounts
    Holdings  []Holding
}
type Holding struct {
    Symbol string
    TotalQty, SellableQty, ReceivingQty, RestrictedQty, PendingStockDiv int64
    PendingCashDiv int64 // VND; informational, never added to NAV (sCashDividend already covers it)
    AvgCost        int64 // VND per share
    RefPrice       int64 // VND, TCBS currentPrice (reference)
    SecType        string // from secType; ETFs and covered warrants are valued but not scored
}
type Cash struct { Balance, Withdrawable, Receivable, IncomingDividends, BankLinked, CustodyFeeDebt int64 }
type Debt struct { Outstanding, AccruedInterest, FeeDebt int64; RTT float64; RiskCode string }

type Position struct { // aggregated by symbol across accounts
    Symbol string
    Qty, Sellable, Receiving, PendingStockDiv int64
    AvgCost, Price int64 // AvgCost is the qty-weighted mean
    PriceSource string    // "tcbs-match", "ref", or the public provider name
    MarketValue, CostValue, UnrealizedPnL int64
    PnLPct, Weight float64 // Weight = MV / NAV
}
type Summary struct {
    Positions []Position
    StockValue, Cash, ReceivableCash, Debt, NAV int64
    CashRatio, MarginRatio, HHI, EffectiveN float64
    Notes []string
}
```

`internal/portfolio/tcbs.go`:

```go
type Source interface { Snapshot(ctx context.Context) (Snapshot, error) }
type TCBSSource struct { Client *tcbs.Client; Custody string; Accounts []string; RawDir string }
func (s TCBSSource) Snapshot(ctx context.Context) (Snapshot, error)
```

Flow:

1. `ListAccounts` and `ActiveStockAccounts`, then the `--accounts` filter.
2. For each account: `GetStockAssets` and `GetCashBalance` (`data[0]`). For
   accounts with `accountType=="margin"`, also call `GetMarginAccountInfo`.
   For non-margin accounts, a 4xx on the margin call is ignored.
3. Map fields:
   - `ReceivingQty = t0+t1+t2`
   - `RestrictedQty = blocked+mortgaged+onHold`
   - `Cash.Balance = cashBalance`
   - `Cash.Receivable = avlAdvanceAmount` (unsettled sale proceeds; outside
     NAV until the live dump shows whether `cashBalance` already includes it)
   - `IncomingDividends = sCashDividend+bCashDividend`
   - `BankLinked = bankAvlBalance`
   - `CustodyFeeDebt = depoFee`
   - `Debt` from `outstanding`, `accruedInterest`, `totalFeeDebt`, `rtt`,
     `riskStatus.code`
4. Unit normalisation, in one function:
   `func toVND(p float64, ref float64) int64`. If both `p` and `ref` are
   non-zero and below 1000, multiply by 1000. Round to an integer. It applies
   to `costPrice`, `currentPrice` and the `GetStockPrices` prices, since the
   spec documents none of their units. A zero `costPrice` (gifted or
   transferred shares) stays 0 and P&L for that line is reported as unknown.
5. With `RawDir` set, write each response body to `<RawDir>/<endpoint>-<acct>.json`
   (files 0600, dir created 0700). `doRequest` does not expose raw bodies, so
   wrap the SDK's `http.Client` transport with a body-tee `RoundTripper` that
   copies the response only, never the request, and skips the token endpoint
   (`/gaia/v1/oauth2/openapi/token`) entirely, because its response body is
   the JWT. The raw `/se` body carries `fullName` and the custody code; the
   dir is gitignored and the README says to delete it after inspection.

`offline.go`: `type FileSource struct{ Path string }` decodes the `Snapshot`
JSON. `func Save(path string, s Snapshot) error` writes it with mode 0600.

`aggregate.go`:

```go
// Prices maps a symbol to its current price in VND and the source of that price.
func Aggregate(s Snapshot, prices map[string]Quote, includeBankCash bool) Summary
type Quote struct { Price int64; Source string; Intraday bool }
```

- NAV = Σ MV + Σ cash balance + incoming dividends − (outstanding + accrued
  + fee debt + custody fee debt).
  - `BankLinked` is added only when `includeBankCash` is set; otherwise it is
    reported as its own line.
  - Pending stock-dividend shares × price and `Receivable` are reported on
    their own lines, not added to NAV, until the live dump resolves the open
    questions in `plan.md`. Each rule lives in one place in `Aggregate` so the
    switch is a one-line change plus a test update.
  - `CashRatio = Σ cash balance / NAV` (receivable excluded, matching the
    "deployable" meaning used by the ADD rule).
- Weight = MV / NAV.
- HHI uses stock-only weights; `EffectiveN = 1/HHI`.
- If a symbol has no quote, fall back to `RefPrice` and add a note.
- Odd lots (`Qty % 100`) are recorded as notes.

The current price is fetched in phase 5 wiring, in this order:

1. `tcbs.Client.GetStockPrices` (`/tartarus/v1/tickerCommons`), using
   `matchPrice`, or `refPrice` when `matchPrice` is 0, both through `toVND`.
   This endpoint was not live-verified (no credentials); a 401/403 here falls
   through to step 2 with a warning instead of failing the run.
2. If that fails, the research provider's last close.

## Tests

- `config`: dotenv parsing (quotes, `export`, comments, a bad line), flag
  precedence, and `String()` redaction.
- `PromptOTP` with a non-TTY reader. A non-6-digit input is an error.
- `portfolio`:
  - an `httptest` TCBS server with fixtures for accounts, `/se`, cash and
    risk. Include string-typed numbers, a closed account, and a 404 on risk
    for a normal account.
  - `Aggregate` table tests: two accounts holding the same symbol; margin
    debt; IA bank cash with the flag on and off; thousand-VND prices; a
    pending stock dividend and a receivable amount that stay out of NAV; a
    zero `costPrice`; an ETF line that is valued but flagged not scored.
  - the `--dump-raw` writer never contains the `Authorization` header or the
    token, and writes no file for the token endpoint.
