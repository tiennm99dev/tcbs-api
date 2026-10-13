# Phase 1: move the SDK to `tcbs/` and fix model mismatches

## Context

The SDK (`package tcbs`) currently sits at the repo root. The root becomes the
module home for the CLI and its internal packages. The spec check against
`references/openapi.json` found that every needed endpoint exists and all JSON
tags match. The live responses may send numbers as strings, and the
custody-to-sub-account call has a misleading name. This phase fixes only what
the CLI reads; other spec gaps (loan `addons`/`insurance`, `TPlusPackage`
extras, int-vs-number fields on loan and statement models, the extra
`Description` fields) are out of scope because nothing in the CLI uses them.

## Files

Move these with `git mv`, so history stays intact. The package name stays `tcbs`.

- `account.go`, `auth.go`, `client.go`, `derivative.go`, `money.go`,
  `stock_market.go`, `stock_order.go`, `stock_query.go`, `websocket.go`
- `models_account.go`, `models_asset.go`, `models_derivative.go`,
  `models_market.go`, `models_money.go`, `models_order.go`
- all `*_test.go`
- `example/main.go` moves to `tcbs/example/main.go`, with its import changed
  to `github.com/miti99dev/stock-advisor/tcbs`

These files stay at the root: `openapi.json`, `references/`, `LICENSE`,
`README.md`, `go.mod`, `go.sum`.

Create:

- `tcbs/number.go`: the tolerant number type
- `tcbs/number_test.go`
- `tcbs/accounts.go`: `ListAccounts`, `ActiveStockAccounts`

`go.mod`: keep the module path. Run `go mod tidy`; `nhooyr.io/websocket` is
used directly, so it becomes a direct requirement. Raise the `go` directive
only if a later phase needs a newer stdlib API.

## SDK changes

1. **Tolerant numbers.** Add the `Number` type:
   ```go
   // Number decodes a JSON number, a numeric string ("1000", "0E-10") or
   // null/"" (as 0).
   type Number float64
   func (n *Number) UnmarshalJSON(b []byte) error
   func (n Number) Float() float64
   ```
   Use it only for the float fields the CLI reads: `StockHoldingInfo`,
   `CashInvestment`, `IAInfo`, `MarginAccountInfoResponse`, `RiskPolicy`,
   `MarginQuotaResponse.MarginLimit`, and the price fields of
   `MarketStockInfo` (`GetStockPrices`). It is a named float64, so untyped
   constants in existing tests still compile. Where a test compares against a
   float64 variable, it converts.
2. **Token errors.** Add `Code string` and `Message string` to `APIError`.
   `doRequest` fills them by best-effort decoding of `{code,message}` from the
   body. `Error()` prints status, code and message, and falls back to the
   body, truncated to 512 bytes. Request headers are never included.
3. **Account listing.**
   ```go
   // ListAccounts returns the sub-accounts under a custody code (105C...).
   func (c *Client) ListAccounts(ctx context.Context, custodyCode string) ([]MarginQuotaResponse, error) // calls GetMarginQuota
   // ActiveStockAccounts drops closed accounts (accountStatus "C") and the
   // internal CW/SPW/SPT types.
   func ActiveStockAccounts(in []MarginQuotaResponse) []MarginQuotaResponse
   ```
   `GetMarginQuota` stays, so the public API is unchanged. Fix its doc comment
   to say it returns the sub-accounts.
4. Run `gofmt` on all moved files; `models_account.go` is currently not aligned.

## Tests

- Move all existing tests and keep them passing unchanged, apart from the
  float64 variable conversions noted above.
- `number_test.go`: table tests for `1000`, `"1000"`, `"0E-10"`, `null`, `""`,
  `1.5e3` and invalid `"abc"` (an error).
- `stock_query_test.go`: a `/se` fixture where `availableTrading` and
  `totalQtty` are strings decodes correctly, and a cash fixture with string
  amounts decodes too.
- `client_test.go`: a 400 response with `{"code":"203074","message":"The API Key is invalid"}`
  populates `APIError.Code` and `APIError.Message`.
- `accounts_test.go`: `ActiveStockAccounts` filtering.

## Validation

`go vet ./... && go test ./tcbs/... && CGO_ENABLED=0 go build ./...`

## Risk and rollback

Importers of the old root path break. The module is private and the only
importer is `example/`, which moves too. The README documents the new import
path. To roll back, revert the commit, which is a single `git mv` plus the
edits.
