# TCBS client codegen: Go, Node, Python, Rust

Date: 2026-10-14. Input: `tcbsgen/openapi.json`, which is the portal spec
(`references/tcbs-openapi-v1.0.0.json`, published 2026-10-09) after
`tools/specfix` repairs it. Each client was tested against a local server
returning `{"assets":[{"symbol":"FPT","quantity":"500",...}]}` from
`StockAsset`. The quantity is a string, as TCBS is known to send some numbers.

## Result

Every generator produces a building, type-checked client from the fixed spec.
No generator handles the string quantity correctly: the strict ones reject the
whole response, and the loose ones pass a string through a field typed as a
number. Tolerant numbers have to come from the spec or a hand-written layer,
whatever the language.

## Best generator per language

| | Go: oapi-codegen 2.8.0 | Node: openapi-typescript 7.13 + openapi-fetch 0.17 | Python: openapi-python-client 0.29.1 | Rust: progenitor 0.15.1 |
|---|---|---|---|---|
| Output | 1 file, 12,965 lines (7,185 code) | 1 `.d.ts`, 3,797 lines | 165 files, 19,685 lines | 1 file, 4,679 lines (3,667 code) |
| Runtime deps | `oapi-codegen/runtime` (+2) | `openapi-fetch` | `httpx`, `attrs` | 123 crates (reqwest, tokio, serde) |
| Type check | `go vet` clean | `tsc` strict clean; works from plain JS via JSDoc | mypy `--strict` 0, pyright 0 | builds, 0 warnings |
| `StockAsset` call | `c.StockAssetWithResponse(ctx, accountNo)` | `client.GET('/aion/v1/accounts/{accountNo}/se', {params:{path:{accountNo}}})` | `stock_asset.sync(account_no, client=c)` (+ async) | `client.stock_asset(account_no).await` |
| Response types | anonymous nested structs, pointer fields | anonymous, via `paths[...]` | named attrs classes (`StockAssetResponse200AssetsItem`) | named (`StockAssetResponseAssetsItem`) |
| `"quantity":"500"` | whole call errors (`cannot unmarshal string ... int64`) | succeeds, `quantity` is the string `"500"` | succeeds, `quantity` is the string `'500'` despite `int` type | whole call errors (`expected i64`) |
| Bearer auth | request editor function | `headers` on `createClient` | `AuthenticatedClient(token=...)` | not generated; pass a `reqwest::Client` with default headers |
| No-content 200 responses (3 ops) | raw `Body []byte` kept | untyped | `Any` | body discarded (`()`) |
| Maturity | v2, active | v7 / 0.x runtime | 0.x | 0.x, released 2026-10-09 |

## Other generators tried

- **openapi-generator** (`javascript`, `typescript-fetch`, `python`, `rust`
  7.26.0): works everywhere once the spec is fixed. Before the fix, its
  `typescript-fetch`, `python` and `rust` output did not compile on the
  property named `""`. `javascript` pulls in deprecated `superagent`, 3 high
  audit findings and untyped results. `python` uses strict pydantic models:
  108 pyright errors, and the string quantity raises `ValidationError`.
  `rust` has built-in bearer auth and a default base URL, but on this host it
  needs `--features rustls` because the OpenSSL headers are missing.
- **@hey-api/openapi-ts 0.99** (Node): good named types, no runtime deps, but
  crashes on TypeScript 7 (current npm `latest`) and emits `.ts` only.
- **datamodel-code-generator** (Python): models only, no HTTP client. It is
  the only generator that turned `"500"` into `500`, through pydantic's lax
  mode.

## Spec issues found by the comparison

- Responses that are top-level arrays were written as `[].field`; `specfix`
  turned them into a property named `""`. Fixed in `92a9d86`.
- `StockSecuritiesInfo`, `StockPricingPolicy` and `DerivativeWsPriceBoard`
  document no 200 body, so typed clients cannot read them.
- Seven objects (e.g. `basicInfo`) have no properties and come out as
  untyped maps.
- Numbers TCBS may send as strings are typed as plain `integer`/`number`.

## Recommendation

Stay on Go. oapi-codegen is as good as any of the alternatives here, and only
Go and Rust fail loudly on bad data rather than passing it on. To make
generated clients usable against live TCBS data, extend `specfix` with a small
override file that:

1. types the numeric fields TCBS sends as strings as `oneOf: [string, number]`,
   or tags them with `x-go-type` pointing at `tcbs.Number`;
2. adds the fields the live API returns but the portal omits (to be confirmed
   from the first `--dump-raw` capture);
3. documents the three missing 200 response bodies.

## Unresolved questions

- Which holdings and cash response shape the live API returns: the portal's
  4-field `assets[]`, or the 24-field `stock[]` the `tcbs/` package follows.
- Which numeric fields TCBS actually sends as strings.

Working files and per-language write-ups: the session scratchpad `langs/`
directory (not committed).
