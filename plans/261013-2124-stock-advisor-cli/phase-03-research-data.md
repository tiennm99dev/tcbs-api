# Phase 3: public research providers, cache and metrics

Every source below was checked live on 2026-10-13 and needs no login,
including with the exact User-Agent below (VCI gap-chart, KBS `data_day`,
KBS KQKD, KBS `index/100/stocks`, VCI `statistics-financial` for VCB). The
`apis.tcbs.com.vn` and `apipubaws.tcbs.com.vn` hosts are dead, so this phase
does not use them.

## Files

- `internal/research/http.go`: shared HTTP client with throttle, retry and User-Agent
- `internal/research/cache.go`: file cache with a TTL
- `internal/research/model.go`: normalised types
- `internal/research/vndirect.go`: primary provider (prices, ratios, events, sector, listing)
- `internal/research/vietcap.go`: fallback for prices, plus bank ratios (NPL, CASA)
- `internal/research/kbs.go`: fallback for prices and constituents, and the annual income data
- `internal/research/service.go`: fallback chain and batch fetch
- `internal/research/metrics.go`: pure computations (volatility, beta, drawdown, ADTV, dividend yield)
- `internal/research/testdata/*.json`: trimmed fixtures, one per endpoint, re-captured with curl
- tests for each file

## HTTP behaviour

- One `*http.Client`, with `--timeout` (default 20s).
- Every request sends `User-Agent: Mozilla/5.0 (X11; Linux aarch64) stock-advisor`
  and `Accept: application/json`. VCI and KBS return 400 without a browser-like
  User-Agent.
- Requests are throttled to 3 per second per host using a `time.Ticker`
  token bucket.
- At most 4 workers run in parallel.
- Retry twice, with backoff of 500ms and 1.5s, on 429, 5xx and network errors.
- Base URLs are struct fields, so tests point them at `httptest`.

## Cache

```go
type Cache struct { Dir string; Now func() time.Time; Refresh bool }
func (c Cache) Get(key string, ttl time.Duration, v any) (ok bool)
func (c Cache) Put(key string, v any) error // atomic write via temp file + rename, mode 0600
```

- Key: `<provider>/<kind>/<symbol-or-hash>.json`, under `--cache-dir`, which
  defaults to `.cache/stock-advisor` and is gitignored.
- The cache stores only normalised public data. Nothing from TCBS goes in it.
- TTLs:
  - daily bars: until 15:30 Asia/Ho_Chi_Minh on the next trading day; in
    practice 6h during market hours and 18h otherwise
  - ratios: 24h
  - events: 24h
  - sector/profile: 30 days
  - constituents: 7 days
- Use `time.LoadLocation("Asia/Ho_Chi_Minh")` and import `_ "time/tzdata"` in
  `main`, so `CGO_ENABLED=0` builds work without system tzdata.

## Normalised model

Units: prices in VND as float64 internally, money in VND, ratios as fractions.

```go
type Bar struct { Date time.Time; Open, High, Low, Close float64; Volume float64 }
type Series struct { Symbol, Source string; Adjusted bool; Bars []Bar } // oldest first
type Fundamentals struct {
    Symbol string; AsOf time.Time; Source string
    PE, PB, ROE, ROA, DivYield, Beta, DE, EPS, BVPS, MarketCap, NPL, CASA *float64 // nil = missing
    NetProfitYears []float64 // most recent last; empty = missing
}
type Dividend struct { ExDate time.Time; CashPerShare float64; StockRatio float64 }
type Profile struct { Symbol, Name, Exchange, Type, Status, ICB1, ICB2, ICB4 string; Peers []string; ListedDate time.Time } // Type: STOCK, ETF, ...; Status: "listed" for normal stocks
type Provider interface {
    Name() string
    Daily(ctx context.Context, symbol string, from, to time.Time) (Series, error)
}
type Service struct { /* providers in fallback order, VNDirect clients, cache */ }
func NewService(opts ServiceOptions) *Service
func (s *Service) Daily(ctx context.Context, symbol string, from, to time.Time) (Series, error) // tries each provider in order
func (s *Service) Fundamentals(ctx context.Context, symbols []string) (map[string]Fundamentals, error) // batched
func (s *Service) Dividends(ctx context.Context, symbol string, since time.Time) ([]Dividend, error)
func (s *Service) Profiles(ctx context.Context, symbols []string) (map[string]Profile, error)
func (s *Service) Universe(ctx context.Context, group string) ([]string, error) // VN30, VN100
```

## Sources

### 1. Daily bars

Primary: VNDirect dchart.

`GET https://dchart-api.vndirect.com.vn/dchart/history?resolution=D&symbol={T}&from={unix}&to={unix}`

- Returns `{s:"ok", t[], o[], h[], l[], c[], v[]}`.
- Prices are in thousand VND and adjusted, so multiply by 1000.
  `Adjusted=true`.
- Index symbols are `VNINDEX` and `VN30`, in points.
- If `s` is not `"ok"`, treat the series as empty and try the next provider.
- Fetch 2 years.

Fallback 1: Vietcap.

`POST https://trading.vietcap.com.vn/api/chart/OHLCChart/gap-chart`

- Body: `{"timeFrame":"ONE_DAY","symbols":["T"],"to":unix,"countBack":520}`.
- Send one symbol per call. The response `t` values are strings.
- Prices are in VND and adjusted.
- Headers: `Origin` and `Referer` set to `https://trading.vietcap.com.vn/`.

Fallback 2: KBS.

`GET https://kbbuddywts.kbsec.com.vn/iis-server/investment/stocks/{T}/data_day?sdate=DD-MM-YYYY&edate=DD-MM-YYYY`

- Index path: `/investment/index/VNINDEX/data_day`.
- Rows come newest first.
- Values can be numbers or strings, so decode them with a flexible number type.
- Prices are in VND and probably unadjusted, so set `Adjusted=false`. The
  metrics then apply the jump-adjust heuristic below.

### 2. Ratios

Use VNDirect for every ratio: one batched call covers holdings, candidates and
peers. Chunk the symbols in groups of 40.

`GET https://api-finfo.vndirect.com.vn/v4/ratios/latest?filter=ratioCode:{codes}&where=code:{T1,T2}&order=reportDate&fields=code,ratioCode,value,reportDate`

The `filter`, `where` and `order` parameters are all required.

| Codes | Maps to |
|---|---|
| `PRICE_TO_EARNINGS` | PE |
| `PRICE_TO_BOOK` | PB |
| `ROAE_TR_AVG5Q` | ROE (fraction) |
| `ROAA_TR_AVG5Q` | ROA |
| `DIVIDEND_YIELD` | DivYield (cross-check only) |
| `BETA` | Beta (cross-check only) |
| `DEBT_TO_EQUITY_AQ` | DE (fraction) |
| `EPS_TR` | EPS (VND) |
| `BVPS_CR` | BVPS (VND) |
| `MARKETCAP` | MarketCap (VND) |

- Banks return no `DEBT_TO_EQUITY_AQ`. That is expected; leave the field nil.
- The fundamentals' `reportDate` is the quarter end. Store the oldest one as
  `AsOf`.
- If one batched call fails, retry the batch per symbol; if that also fails,
  leave the fields nil.

Bank extras come from Vietcap.

`GET https://iq.vietcap.com.vn/api/iq-insight-service/v1/company/{T}/statistics-financial`

- Called only for `sector == bank`.
- Pick the latest `ratioType=="RATIO_TTM"` row and read `npl` and `casaRatio`
  (fractions).
- The `dividendYield` field reads 0 even for dividend payers, so ignore it.
- Its `pe`, `pb`, `roe` and `debtToEquity` fill gaps when VNDirect has none.

Earnings consistency comes from KBS.

`GET https://kbbuddywts.kbsec.com.vn/iis-server/investment/stock/finance-info/{T}?page=1&pageSize=5&type=KQKD&unit=1000&termtype=1&languageid=1`

- Verified live for FPT and VCB. The endpoint returns at most 4 fiscal years
  (`Value1..Value4`, newest first; `Value5` is null and `page=2` is empty), so
  request `pageSize=4`. `Head[i].YearPeriod` gives the year of `Value{i+1}`.
- Prefer `NameEn` "Profit after tax for shareholders of parent company"
  (ReportNormID 2214 for non-banks). Banks have no parent row; fall back to
  `NameEn` containing "Net profit after tax" (ReportNormID 4378 for VCB).
  Match by `NameEn`, not by the Vietnamese group keys.
- Values are in thousand VND (`unit=1000`); only signs and the coefficient of
  variation are used, so the unit does not matter.
- If the shape differs or fewer than 3 years come back, leave
  `NetProfitYears` empty and EARN_CONS is dropped through redistribution.

### 3. Dividends

VNDirect:

`GET https://api-finfo.vndirect.com.vn/v4/events?q=code:{T}~type:DIVIDEND,STOCKDIV~locale:EN_GB&sort=effectiveDate:desc&size=50`

- Use `effectiveDate` as the ex-date.
- For `DIVIDEND`, `dividend` is VND per share. If it is 0, use `ratio` (% of
  10,000 VND par) × 100.
- `STOCKDIV` sets `StockRatio` and is never counted as yield.
- Drop events with ex-dates more than 1 year in the future (placeholder dates).

Fallback: Vietcap.

`GET https://iq.vietcap.com.vn/api/iq-insight-service/v1/events?ticker={T}&fromDate=YYYYMMDD&toDate=YYYYMMDD&eventCode=DIV,ISS&page=0&size=50`

- Uses `exrightDate` and `valuePerShare`.

### 4. Profile, sector and status

From VNDirect:

- `GET https://api-finfo.vndirect.com.vn/v4/stocks?q=code:{T1},{T2}&size=100`
  gives the exchange (`floor`), `type` (STOCK vs ETF and others), `status`,
  `listedDate` and the company name. The only status seen live is `listed`;
  this source does not expose the exchange's warning/control/restricted
  lists, so the red-flag check can only catch a status other than `listed`
  (suspended, delisted) plus price-history gaps.
- `GET https://api-finfo.vndirect.com.vn/v4/industry_classification?q=codeList:{T}`
  gives one row per ICB level. Trim `englishName`. `codeList` at the matching
  level gives the peers, used for the sector median P/E.

### 5. Universe

VNDirect `GET https://api-finfo.vndirect.com.vn/v4/stocks?q=indexCode:VN30&size=50`

- Fallback: `GET https://kbbuddywts.kbsec.com.vn/iis-server/investment/index/30/stocks`.
  KBS uses group codes `30` and `100` for VN30 and VN100.
- VN100 (the default universe) comes from KBS group `100` only, verified live
  on 2026-10-13 (`{status:200,data:[...]}`). VNDirect's `indexCode` holds one
  index per stock, so it cannot list VN100. If KBS fails, fall back to VN30
  and add a warning.
- For VN100 the run fetches about 100 daily series; at 3 req/s per host that
  is under a minute cold and is served from cache afterwards.
- An explicit ticker list skips the fetch.

## Metrics (pure, in `metrics.go`)

```go
func LogReturns(s Series) map[time.Time]float64
func Volatility(s Series, sessions int) (float64, bool)   // stdev × sqrt(250); needs ≥120 returns
func Beta(stock, index Series, sessions int) (beta, r2 float64, ok bool) // date intersection
func WeeklyBeta(stock, index Series, weeks int) (float64, bool) // Friday closes, 2 years
func MaxDrawdown(s Series, sessions int) (float64, bool)
func ADTV(s Series, sessions int) float64                   // mean(close×volume), 60 sessions
func TrailingCashYield(divs []Dividend, price float64, now time.Time) (yield float64, paidYears int)
func AdjustJumps(s Series, band float64) (Series, bool)     // for unadjusted series only
func GapSessions(s Series, index Series, last int) int      // index days missing in the stock series
func StaleSessions(s Series, index Series) int             // index sessions after the stock's last bar
func Range(s Series, sessions int) (low, high float64)      // 52-week range for the report
func Return(s Series, sessions int) (float64, bool)         // 1y return, used for the VN-Index context line
```

Staleness is measured against the VN-Index series, not the calendar, so Tet
and other multi-day holidays do not flag every stock.

`AdjustJumps` treats a daily return whose absolute value exceeds band + 1pp as
an adjustment event. It back-scales all earlier bars by the ratio and reports
that it adjusted the series.

## Tests

- Each provider decodes its fixture into the normalised model, with units
  checked. For example, FPT dchart 62.937 becomes 62,937 VND, and the KBS
  string row parses.
- The fallback chain: dchart returns 500, so the service uses VCI; VCI returns
  400, so it uses KBS.
- The ratios batch: a bank has nil DE; unknown codes are ignored.
- The cache: TTL hit and miss, `Refresh`, and atomic writes in a `t.TempDir()`
  cache dir.
- Metrics: hand-computed table tests for volatility, beta (a perfectly
  correlated 2× series gives beta 2), drawdown, ADTV, trailing yield (stock
  dividends excluded) and a synthetic 2:1 split for `AdjustJumps`.
