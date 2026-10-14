# Authentication

Every iFlash Open API call except the three under `/gaia/v1/oauth2/openapi/`
needs a JWT in the `Authorization: Bearer <token>` header. (Those three carry
their credentials in the request body; the portal spec does not exempt them
from its global bearer scheme, but the older spec marks the token call with
`security: []`.) The JWT comes from
an API key plus a one-time Smart OTP (iOTP).

Sources: the `/gaia/v1/oauth2/openapi/*` operations and the getting-started
text in the TCBS specs (the current portal spec and the earlier edition), and
the [TCBS developer portal](https://developers.tcbs.com.vn).

## What you need

| Item | Where it comes from | Lifetime |
|---|---|---|
| API key | Registered in TCInvest: Utilities (Tiện ích), iFlash Open API, at <https://tcinvest.tcbs.com.vn/iflash-open-api>. TCInvest also shows and revokes it. | Has an expiry date; check and extend it with the calls below |
| iOTP | The 6-digit Smart OTP in the TCInvest app | One use |
| JWT | `POST /gaia/v1/oauth2/openapi/token` | At most 8 hours from the iOTP confirmation |
| Custody code | Your TCBS custody code, `105C` followed by digits (e.g. `105C000000`) | Fixed |
| Sub-account number | Returned by `GetMarginQuota` / `ListAccounts` for a custody code | Fixed |

The older spec also says partners (as opposed to individual customers)
register with TCBS for a Client ID and Client Secret; contact TCBS for that
route. Neither client in this module uses them.

## Endpoints

| Call | Body | Response | `tcbs` | `tcbsgen` |
|---|---|---|---|---|
| `POST /gaia/v1/oauth2/openapi/token` | `{"apiKey","otp"}` | `{"token"}` | `GetToken` | `AuthTokenWithResponse` |
| `POST /gaia/v1/oauth2/openapi/apikey/info` | `{"apiKey"}` | `{"expireDate"}` | none | `AuthApikeyInfoWithResponse` |
| `POST /gaia/v1/oauth2/openapi/apikey/extend` | `{"apiKey","otp"}` | `{"expireDate"}` (the new date) | none | `AuthApikeyExtendWithResponse` |

- `expireDate` is ISO 8601 / RFC 3339 in Vietnam time (`+07:00`).
- `apikey/info` needs only the key, so it is a cheap way to check that a key
  is valid and when it lapses, without spending an iOTP.
- A failed token exchange returns `{"code","message"}`; the older spec's
  example is code `203074`, message `The API Key is invalid`. The `tcbs`
  client surfaces any non-2xx response as `*tcbs.APIError` with `Code` and
  `Message` filled from such a body.
- When the JWT expires, ask the user for a fresh iOTP and call the token
  endpoint again. Neither spec has a refresh-token call.

## Using the token

`tcbs`: `GetToken` stores the token on the client. Every later REST call
sends it as `Authorization: Bearer <token>`. The `Connect*` WebSocket calls
send it only as that header when dialing; the portal does not document the
header for WebSocket, and each stream also needs an in-band auth frame
built from the token (`d|a|||<base64(JWT)>` or
`authenticate|<base64({"jwt":"<JWT>"})>`, depending on the stream) before it
delivers data. See the README's WebSocket section. If you obtained the token elsewhere, pass it with
`tcbs.WithToken(token)` or `client.SetToken(token)`.

`tcbsgen`: the generated client has no built-in auth. Add the header with a
request editor:

```go
c, err := tcbsgen.NewClientWithResponses("https://openapi.tcbs.com.vn",
    tcbsgen.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
        req.Header.Set("Authorization", "Bearer "+token)
        return nil
    }))
```

## Environments

| Environment | Base URL | Constant |
|---|---|---|
| Production | `https://openapi.tcbs.com.vn` | `tcbs.ProductionBaseURL` (default) |
| SIT | `https://openapisit.tcbs.com.vn` | `tcbs.SITBaseURL` |

The SIT URL comes from the older spec; the portal spec lists production only.

## Supplying credentials safely

The API key, iOTP and JWT give full trading and money-transfer access to the
account. Treat them like a password.

- Read them at run time from environment variables or a secret store, never
  from source code. The example in [`tcbs/example`](../tcbs/example/main.go)
  uses `TCBS_API_KEY`, `TCBS_OTP`, `TCBS_CUSTODY_CODE` and `TCBS_ACCOUNT_NO`.
- If you keep them in a local `.env` file, make sure it is gitignored (this
  repository ignores `.env` and `.env.*`) and load it with your shell or tool
  of choice; the library does not read `.env` files.
- Prompt for the iOTP when it is needed and do not store it.
- Do not log the key, iOTP or token, and do not put them in error messages.
  `tcbs.APIError` carries only the response status, code, message and body.
- Account responses contain personal data (name, custody code, balances).
  Keep raw dumps out of version control and delete them after use.
- Use only obvious placeholders in examples, tests and issues: `105C000000`,
  `0001000000`, `<app>.example.com`.
