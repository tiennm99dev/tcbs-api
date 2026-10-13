# Phase 5: Markdown report, wiring, end-to-end test, README

## Files

- `internal/report/markdown.go`, `internal/report/markdown_test.go`,
  `internal/report/testdata/report.golden.md`
- `cmd/stock-advisor/run.go`: completes the orchestration
- `cmd/stock-advisor/run_test.go`: end-to-end test with fixture servers
- `README.md`: rewritten
- `.env.example`, `.gitignore`: finalised in phase 2, checked here

## Report API

```go
type Data struct {
    GeneratedAt time.Time; Phase string // intraday / last close
    Summary portfolio.Summary; Advice advisor.PortfolioAdvice
    Scores map[string]advisor.Score; Candidates []advisor.Candidate; CandidateNote string
    Sources []SourceStamp; Config advisor.Config; Warnings []string
    Market MarketContext; Research map[string]HoldingResearch // keyed by held symbol
}
type MarketContext struct { IndexClose, Return1Y, Vol1Y float64; AsOf time.Time } // VN-Index from the fetched bars
type HoldingResearch struct {
    Name, Exchange string; Sector advisor.Sector; ICB4 string
    Low52W, High52W float64; Dividends []research.Dividend // last 3 years, cash and stock
}
type SourceStamp struct { Name, Kind string; AsOf time.Time }
func Render(w io.Writer, d Data) error // deterministic: sorted keys, fixed number formats
```

## Report layout

1. A title, the generation time (Asia/Ho_Chi_Minh), the price phase, and the
   disclaimer at the top.
2. **Portfolio summary**:
   - NAV, stock value, cash, receivable cash, debt, and IA bank cash on its own line
   - the cash ratio and margin ratio
   - HHI and effectiveN
   - pending stock-dividend shares and receivable sale proceeds, each on its
     own line and marked "not in NAV"
   - the portfolio messages
   - a market-context line: VN-Index close, 1-year return and 1-year
     volatility, computed from the bars already fetched
3. **Sector weights** table.
4. **Holdings** table: symbol, quantity (sellable / receiving), average cost,
   price and its source, MV, weight, P&L %, breakeven, score, tier, action and
   the rule.
5. **Per-holding detail**. Each holding opens with its research block: name,
   exchange, sector (ICB level 4), 52-week low/high, and the cash and stock
   dividends of the last 3 years with ex-dates. Then a metric table (raw value, score,
   effective weight, contribution), pillar subtotals, the sector adjustment,
   coverage and confidence, dropped metrics, red flags, the suggested lots
   with the estimated cash impact, and any odd-lot note.
6. **Stable candidates**: rank, symbol, sector, score and tier, VOL, BETA, DY,
   ROE, PE, PB, price, lot cost, max allocation, and the top 3 reasons, or
   the "For watchlist only." note.
7. **Data sources and timestamps**, the config values in use, and any warnings
   (provider fallbacks, stale data).
8. The footer disclaimer: "Informational, rule-based output; not financial
   advice. Verify prices and rules with your broker before trading."

Number formats: VND with thousands separators (`1,234,500`), percentages with
1 decimal place, ratios with 2.

Privacy: account numbers are masked to their last 4 digits. The report has no
customer name and no custody code.

## Orchestration (`run`)

1. Parse the config and load `.env`. Create `--cache-dir` (0700).
2. Get the portfolio snapshot from the `--offline-holdings` file, or from
   TCBS: build the client, call `PromptOTP`, then `GetToken`, then
   `TCBSSource.Snapshot`. If `--save-holdings` is set, save the snapshot.
3. Quotes for held symbols come from `GetStockPrices` when online. Any symbol
   still missing a quote uses the research last close.
4. Build the symbol set: holdings, the universe, then peers capped at 40.
   Fetch profiles and fundamentals (batched) for all of them. Daily bars,
   dividends, KBS profit history and (banks only) VCI bank ratios are fetched
   only for holdings and universe members; peers are used for the sector
   median P/E alone. Fetch VNINDEX bars once.
5. Build `advisor.Inputs` for every symbol, then run `ScoreStock`,
   `portfolio.Aggregate`, `Recommend` and `Candidates`.
6. Render to `--out` (file mode 0600) or stdout. With `-v`, progress goes to
   stderr; no secrets are ever written there.
7. Exit codes: 0 on success, 1 on a fatal error (auth, or no portfolio), 2 on
   a usage error. A partial data failure is not fatal: it becomes a warning
   in the report.

Tests need to point `run` at fake servers without adding user-facing flags.
`run` takes an unexported `deps` struct (TCBS base URL, research base URLs,
clock), set to production defaults in `main`. The tests fill it with
`httptest` URLs.

## Tests

- `report`: the golden test. Set `UPDATE_GOLDEN=1` to rewrite the golden file.
- `cmd/stock-advisor`, end to end:
  - offline holdings plus fixture research servers produce output that matches
    the golden file
  - the online path uses a fake TCBS server and an OTP passed through the
    stdin reader
  - a secret-leak check: with `TCBS_API_KEY=sk-test-SECRET` and the fake token
    `jwt-SECRET`, scan stdout, stderr, the report file, the cache dir, the
    `--dump-raw` dir and the `--save-holdings` file for "SECRET". It must not
    appear. Run it once more with the fake TCBS server returning 400 and 500
    so error paths are covered.
- No test uses the network. Every HTTP client (TCBS SDK and research) is
  built from one `http.RoundTripper` in `deps`, so the test can hand all of
  them a transport that fails any non-loopback dial; a guard in `TestMain`
  also replaces `http.DefaultTransport` the same way.

## README

Rewrite the README:

- What the tool is: a portfolio stability advisor for TCBS accounts.
- A read-only statement: it places no orders.
- The disclaimer.
- Requirements: Go and a TCBS OpenAPI key from TCInvest.
- Setup: `cp .env.example .env` and fill in the two keys.
- Usage: `go run ./cmd/stock-advisor --out out/report.md`, the flags table,
  and an offline example.
- What it fetches, with every public source, a note that those sources are
  undocumented and can change, the caching behaviour, and `--refresh`.
- A summary of scoring and actions, linking to `internal/advisor/config.go`
  for the thresholds and to `--rules` for overrides.
- SDK usage: `import "github.com/miti99dev/stock-advisor/tcbs"`, with a short
  example, the API coverage tables moved under an SDK section, and a pointer
  to `tcbs/example`.
- Development: `go test ./...`, `CGO_ENABLED=0 go build ./cmd/stock-advisor`.

The README examples use generic placeholders such as `105C000000`, never real
account data.

## Validation

- `gofmt -l .` prints nothing.
- `go vet ./...`, `go test ./...` and `CGO_ENABLED=0 go build ./...` all pass.
- One manual live run with the user's key and OTP and `--dump-raw raw/`.
- Compare the raw JSON with the open questions in `plan.md`: string versus
  number types, `totalQtty` and `stockDividend` semantics, whether
  `cashBalance` includes `avlAdvanceAmount` or nets out `dsecured`, price
  units, and whether `tickerCommons` (`GetStockPrices`) answers with this token. Cross-check NAV
  against the total shown in the TCInvest app. If needed, adjust the
  `toVND` heuristic and the quantity mapping and add a fixture captured from
  that live run, with account numbers replaced by fakes.

## Risk

The public endpoints are undocumented and can change without notice. That
risk is contained in four ways: provider fallback, graceful metric drops, the
coverage and confidence shown in the report, and the per-provider fixtures,
which make it quick to see what changed.
