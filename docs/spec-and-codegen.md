# Specs and code generation

This module builds two clients from two editions of the TCBS iFlash Open API
spec. This page records where each spec comes from, what is wrong with the
current one, how `tools/specfix` repairs it, and what is still unknown about
live responses.

## The two specs

| File | Edition | Operations | Used by |
|---|---|---|---|
| [`references/openapi.json`](../references/openapi.json) | Older spec, title "iFlash Open API", Vietnamese summaries, production and SIT servers | 46 (41 REST, 5 WebSocket) | the hand-written [`tcbs`](../tcbs/) package |
| [`references/tcbs-openapi-v1.0.0.json`](../references/tcbs-openapi-v1.0.0.json) | Current portal spec, title "TCBS iFlash Open API", English, production server only | 48 REST | [`tools/specfix`](../tools/specfix/), then [`tcbsgen`](../tcbsgen/) |
| [`tcbsgen/openapi.json`](../tcbsgen/openapi.json) | The portal spec after `specfix` | 48 | oapi-codegen input |

The portal spec is published on the [TCBS developer
portal](https://developers.tcbs.com.vn) at
`https://developers.tcbs.com.vn/specs/openapi-v1.0.0.json`.

What differs between the editions:

- Only in the portal spec: `GET /ananke/v1/securities`,
  `GET /api/v1/derivatives/contracts`,
  `GET /hydros/v1/account/{accountNo}/pricing-policy`, `GET /nyx/v2/stock/bars`,
  `GET /nyx/v2/stock/bars-long-term`, and the API-key calls
  `POST /gaia/v1/oauth2/openapi/apikey/info` and `.../apikey/extend`. Use
  `tcbsgen` for these; `tcbs` does not wrap them.
- WebSocket streams. The five streams (`/ws/aither`, `/ws/nesoi`,
  `/ws/ouranos/v1/stream`, `/ws/thesis/v1/stream/normal`,
  `/ws/thesis/v1/stream/derivative`) are missing from the portal's JSON spec.
  The older spec lists them, but each description only points to a PDF and
  does not describe the protocol. The portal documents all five on its HTML
  pages under `https://developers.tcbs.com.vn/docs/v1.0.0/` (`stock/ws-price-board`,
  `derivative/ws-price-board`, `stock/ws-history`, `stock/ws-order-matching`,
  `derivative/ws-order-matching`); those pages are the authoritative protocol
  reference. `tcbs` only dials the sockets (`ConnectStockPrice`,
  `ConnectDerivativePrice`, `ConnectCenter`, `ConnectStockMatch`,
  `ConnectDerivativeMatch`) and passes frames through; it does not implement
  the auth, subscribe or heartbeat framing. The README's WebSocket section
  summarises the frames per stream. `tcbsgen` has no WebSocket support.
- Response shapes. The portal documents fewer fields. Holdings
  (`GET /aion/v1/accounts/{accountNo}/se`) is the clearest case: the portal
  returns `assets[]` with 4 fields (`symbol`, `quantity`, `avgPrice`,
  `marketValue`), while the older spec returns `stock[]` of
  `StockHoldingInfo` with 24 fields.

## Why `tcbs` follows the older spec

The hand-written client predates the portal spec and models the richer
response shapes of `references/openapi.json`. Which shape the live API
returns for holdings and cash is not confirmed, so `tcbs` keeps the older
shapes until a live response settles it. `tcbsgen` follows the portal
exactly, so the two clients can be compared against the same account.

## Defects in the portal spec

`specfix` repairs three defects that stop generators from producing a usable
client (see the package comment in
[`tools/specfix/main.go`](../tools/specfix/main.go)):

1. **Flattened field names.** Nested response fields are written as flat
   property names, such as `assets[].symbol` or `data.items[].price`. `specfix`
   rebuilds the nested objects and arrays and keeps each field's description.
   A response that is itself a top-level array writes its item fields as
   `[].name`; `specfix` turns that into an array schema instead of an object
   with a property named `""`. This affects
   `GET /aion/v1/customers/{custodyId}/accounts` and
   `GET /hydros/v1/account/{accountNo}/risk`.
2. **Stray path parameters.** `GET /aion/v1/accounts/{accountNo}/ppse`
   declares `symbol` and `price` as path parameters its path does not have.
   `specfix` drops them.
3. **No operation IDs.** Without them, generators name methods after paths.
   `specfix` sets each ID from
   [`references/tcbs-operation-ids.json`](../references/tcbs-operation-ids.json),
   which names each operation after its portal docs page (`stock/asset`
   becomes `StockAsset`). Every operation must have an entry and every entry
   must match an operation, so a portal update that adds or renames an
   endpoint fails the generation instead of silently producing a path-named
   method.

On the current spec, `specfix` reports
`288 flattened fields nested, 2 stray path params dropped, 48 operationIds set`.

### Gaps `specfix` does not repair

- **No 200 body.** `StockSecuritiesInfo` (`GET /ananke/v1/securities`),
  `StockPricingPolicy` (`GET /hydros/v1/account/{accountNo}/pricing-policy`)
  and `DerivativeWsPriceBoard` (`GET /api/v1/derivatives/contracts`) document
  no response body. `tcbsgen` returns their raw bytes in `resp.Body`.
- **Property-less objects.** Seven objects have no properties and come out as
  untyped values, for example `basicInfo` in `AccountSubAccount`, `data` in
  `DerivativeAccountStatus`, and the `ordersList` items in
  `StockCancelOrder`.
- **String-encoded numbers.** The spec types amounts and quantities as
  `integer` or `number`, but TCBS can send numbers as strings. The older spec
  itself shows this: `printAmount` and `intAmount` in
  `MarginInfoResponseArray` are typed `number` with the example `"0E-10"`.
  `tcbs.Number` accepts a JSON number, a numeric string (`"1000"`,
  `"0E-10"`), `null` or `""`, but `tcbs` uses it only on the fields known to
  arrive as strings: holdings (`StockHoldingInfo`), cash (`CashInvestment`,
  `IAInfo`), REST stock prices (`MarketStockInfo`) and the margin risk models
  (`MarginAccountInfoResponse`, `MarginQuotaResponse`, `RiskPolicy`). Its
  other numeric fields, including every derivative and WebSocket model, are
  plain `float64`/`int` and fail the same way as `tcbsgen`, whose generated
  models use plain `int64`/`float64` throughout: such a response makes the
  whole call fail to decode. The portal's stock price-board page shows the
  stream sending prices as strings (`"matchPrice":"24550"`).

A possible next step is a small override file applied by `specfix` that
types the affected fields as `oneOf: [string, number]` (or as `tcbs.Number`
through `x-go-type`), adds fields the live API returns but the portal omits,
and documents the three missing response bodies. It is not built yet,
because the affected fields are not confirmed.

## Open questions about live responses

- Holdings and cash: the portal's `assets[]` shape or the older `stock[]`
  shape.
- Which numeric fields TCBS actually sends as strings.
- Holdings quantities: how `totalQtty` relates to the settlement buckets
  (`t0`, `t1`, `t2`) and pending dividends in `StockHoldingInfo`.
- Price units: whether a given price field is in VND or thousands of VND.

## Refreshing the generated client

```bash
curl -sSfo references/tcbs-openapi-v1.0.0.json https://developers.tcbs.com.vn/specs/openapi-v1.0.0.json
go generate ./tcbsgen
git diff --stat references tcbsgen
```

`go generate` runs the two directives in
[`tcbsgen/generate.go`](../tcbsgen/generate.go): `specfix` writes
`tcbsgen/openapi.json`, then oapi-codegen (`@latest`, configured by
`tcbsgen/oapi-codegen.yaml`) writes `tcbsgen/client.gen.go`. If `specfix`
stops on an operation-ID mismatch, add or rename the entry in
`references/tcbs-operation-ids.json` and run it again. Never edit the two
generated files by hand.

## Generator comparison

On 2026-10-14 the repaired spec was fed to the leading generator in four
languages. Each client was tested against a local server whose `StockAsset`
response carried `"quantity":"500"`, a string where the spec says integer.

| | Go: oapi-codegen 2.8.0 | Node: openapi-typescript 7.13 + openapi-fetch 0.17 | Python: openapi-python-client 0.29.1 | Rust: progenitor 0.15.1 |
|---|---|---|---|---|
| Output | 1 file, about 13k lines | 1 `.d.ts`, about 3.8k lines | 165 files, about 19.7k lines | 1 file, about 4.7k lines |
| Runtime deps | `oapi-codegen/runtime` (+2) | `openapi-fetch` | `httpx`, `attrs` | 123 crates |
| Type check | `go vet` clean | `tsc` strict clean | mypy `--strict` and pyright clean | builds, no warnings |
| Response types | anonymous nested structs | anonymous, via `paths[...]` | named classes | named structs |
| `"quantity":"500"` | call fails to decode | passes the string through | passes the string through despite `int` | call fails to decode |
| Bearer auth | request editor | `headers` option | `AuthenticatedClient(token=...)` | not generated |
| No-body 200 responses | raw `Body []byte` kept | untyped | `Any` | body discarded |

Also tried: openapi-generator 7.26.0 (works once the spec is repaired; before
the repair its TypeScript, Python and Rust output did not compile on the
property named `""`), `@hey-api/openapi-ts` 0.99 (crashed on TypeScript 7),
and datamodel-code-generator (models only, but the only one that coerced
`"500"` to `500`).

Conclusions:

- Every generator builds a working client from the repaired spec, and none
  handles string-encoded numbers correctly without help. Tolerant numbers
  have to come from the spec or a hand-written layer in any language.
- Go with oapi-codegen is as good as any alternative tested and fails loudly
  on bad data instead of passing it on, which is why this module ships it.
