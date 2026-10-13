---
title: stock-advisor CLI on top of the TCBS SDK
status: pending
created: 2026-10-13
---

# stock-advisor CLI

## Outcome

`go run ./cmd/stock-advisor` signs in to TCBS with the user's API key and an
iOTP prompt. It pulls holdings, cash and margin debt for every active
sub-account under the custody code, then fetches public market and fundamental
data. It scores each held stock and a candidate universe with deterministic,
rule-based stability scores. The output is a Markdown report with portfolio
health, a HOLD/TRIM/ADD/REVIEW action for each holding, and the top stable
candidates the user does not hold yet.

## Constraints

- Pure Go, `CGO_ENABLED=0`, standard library first. The only new dependency is
  `golang.org/x/term`, for the no-echo OTP prompt.
- The SDK moves into the `tcbs/` subpackage (`github.com/miti99dev/stock-advisor/tcbs`).
  All existing SDK tests must keep passing after the move.
- Secrets (`TCBS_API_KEY`, the OTP, the JWT) never reach logs, reports, the
  cache or error output. `.env`, the cache dir and the report output dir are
  gitignored.
- Unit tests make no live calls: they use `httptest` servers and `testdata/`
  fixtures.
- Public market data comes from VNDirect finfo/dchart as the primary source,
  with Vietcap (VCI) and KBS as fallbacks. The old TCBS public research hosts
  are dead.
- Code comments carry no plan IDs, phase numbers or local paths.

## Non-goals

- No order placement, money transfer or any other write call from the CLI.
  The SDK keeps those methods, but the CLI never calls them.
- No web UI, server, daemon or scheduler.
- No derivatives, bonds or funds in the advice. Listed stocks are valued and
  scored. ETFs and covered warrants found in `/se` are valued in NAV but not
  scored; they get HOLD with the note "not scored (fund/warrant)".
- No SDK work beyond what the CLI needs. Spec gaps the advisor never touches
  (loan `addons`/`insurance`, `TPlusPackage` extras, the extra `Description`
  fields, int-vs-number fields on loan and statement models) stay as they are.
- No price prediction or ML. Only the rule-based scoring in phase 4.

## Acceptance criteria

1. `go vet ./...`, `gofmt -l .` (empty), `go test ./...` and
   `CGO_ENABLED=0 go build ./...` all pass.
2. The SDK lives in `tcbs/`. Every pre-existing SDK test passes there, and the
   model fixes in phase 1 have tests.
3. `stock-advisor --offline-holdings testdata/holdings.json`, run against
   fixture servers, produces a report that matches the golden file
   (end-to-end test in `cmd/stock-advisor`).
4. A live run asks for the OTP once, reads every active sub-account and writes
   a report with NAV, weights, P&L, per-holding actions with rule citations,
   a research block for each held stock (name, exchange, sector, key ratios,
   recent cash dividends, 52-week range), a VN-Index market-context line,
   up to N candidates, data timestamps and the disclaimer.
5. When a public source fails, the run falls back to the next provider or drops
   metrics with a printed coverage note. It never exits because one ticker
   lacks data.
6. Grepping the report, the cache dir and stderr for the API key and the token
   finds nothing (covered by a test).
7. The README documents install, `.env`, flags, data sources, scoring summary,
   disclaimer and the new SDK import path.

## Phases

| # | Phase | Depends on | File |
|---|---|---|---|
| 1 | Move the SDK to `tcbs/`, fix model mismatches | - | [phase-01-sdk-move-and-fixes.md](phase-01-sdk-move-and-fixes.md) |
| 2 | CLI skeleton, config/.env, portfolio pull | 1 | [phase-02-cli-config-portfolio.md](phase-02-cli-config-portfolio.md) |
| 3 | Public research providers, cache, metrics | 1 | [phase-03-research-data.md](phase-03-research-data.md) |
| 4 | Stability scoring and recommendation rules | 2, 3 | [phase-04-advisor-scoring.md](phase-04-advisor-scoring.md) |
| 5 | Markdown report, wiring, e2e test, README | 2, 3, 4 | [phase-05-report-wiring-docs.md](phase-05-report-wiring-docs.md) |

Phases 2 and 3 can run in parallel because they touch disjoint packages.

## Open questions (resolve with the first live run, `--dump-raw`)

- Whether TCBS `/se` numbers come back as JSON numbers or as strings. Phase 1
  adds a tolerant number type for both cases.
- Whether `totalQtty` already includes `t0/t1/t2` and `stockDividend`. The
  default assumes it includes `t0/t1/t2` and does NOT include `stockDividend`.
  Pending stock-dividend shares are shown on their own line and are NOT added
  to NAV until the live dump proves how TCBS counts them, so neither case can
  double count.
- Whether `cashBalance` already nets out today's matched buys (`dsecured`,
  `buyingAmount`) and already contains unsettled sale proceeds
  (`avlAdvanceAmount`). Phase 2 uses `cashBalance` and reports
  `avlAdvanceAmount` as receivable cash on its own line, outside NAV, until the
  live dump settles it.
- Whether TCBS `costPrice`/`currentPrice` are in VND or thousand VND. Phase 2
  normalises them with a heuristic.
- The root `openapi.json` duplicates `references/openapi.json`. Both are left
  in place unless the user wants one removed.
