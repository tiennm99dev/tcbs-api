package portfolio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/miti99dev/stock-advisor/tcbs"
)

// Source produces a portfolio snapshot.
type Source interface {
	Snapshot(ctx context.Context) (Snapshot, error)
}

// TCBSSource reads the snapshot from the TCBS OpenAPI. Client must already
// hold a token. It only calls read endpoints.
type TCBSSource struct {
	Client   *tcbs.Client
	Custody  string
	Accounts []string // optional filter of sub-account numbers; empty means all active
	// RawDir is the --dump-raw dir. The bodies themselves are written by the
	// RawDumpTransport installed in Client's http.Client (see NewTCBSClient);
	// Snapshot creates the dir (0700) up front so a bad path fails before any
	// account call.
	RawDir string
}

const accountTypeMargin = "margin"

// Snapshot pulls holdings, cash and margin debt for every selected active
// sub-account.
func (s TCBSSource) Snapshot(ctx context.Context) (Snapshot, error) {
	if s.Client == nil {
		return Snapshot{}, errors.New("portfolio: nil TCBS client")
	}
	if s.RawDir != "" {
		if err := ensureRawDir(s.RawDir); err != nil {
			return Snapshot{}, err
		}
	}
	all, err := s.Client.ListAccounts(ctx, s.Custody)
	if err != nil {
		return Snapshot{}, fmt.Errorf("portfolio: list accounts: %w", err)
	}
	selected, err := filterAccounts(tcbs.ActiveStockAccounts(all), s.Accounts)
	if err != nil {
		return Snapshot{}, err
	}

	snap := Snapshot{AsOf: time.Now(), Custody: s.Custody}
	for _, a := range selected {
		acct, err := s.account(ctx, a)
		if err != nil {
			return Snapshot{}, err
		}
		snap.Accounts = append(snap.Accounts, acct)
	}
	return snap, nil
}

// filterAccounts keeps the accounts named in want, in TCBS order. Every named
// account must be active.
func filterAccounts(active []tcbs.MarginQuotaResponse, want []string) ([]tcbs.MarginQuotaResponse, error) {
	if len(want) == 0 {
		if len(active) == 0 {
			return nil, errors.New("portfolio: no active stock sub-accounts under the custody code")
		}
		return active, nil
	}
	byNo := make(map[string]bool, len(active))
	for _, a := range active {
		byNo[strings.ToUpper(a.AccountNo)] = true
	}
	keep := make(map[string]bool, len(want))
	for _, w := range want {
		w = strings.ToUpper(strings.TrimSpace(w))
		if !byNo[w] {
			return nil, fmt.Errorf("portfolio: --accounts names %s, which is not an active stock sub-account", w)
		}
		keep[w] = true
	}
	var out []tcbs.MarginQuotaResponse
	for _, a := range active {
		if keep[strings.ToUpper(a.AccountNo)] {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s TCBSSource) account(ctx context.Context, a tcbs.MarginQuotaResponse) (Account, error) {
	acct := Account{AccountNo: a.AccountNo, Type: a.AccountType, Status: a.AccountStatus}

	se, err := s.Client.GetStockAssets(ctx, a.AccountNo)
	if err != nil {
		return Account{}, fmt.Errorf("portfolio: stock assets for %s: %w", a.AccountNo, err)
	}
	prices := make([]float64, 0, 2*len(se.Stock))
	for _, st := range se.Stock {
		prices = append(prices, st.CostPrice.Float(), st.CurrentPrice.Float())
	}
	scale := priceScale(prices)
	for _, st := range se.Stock {
		if h, ok := mapHolding(st, scale); ok {
			acct.Holdings = append(acct.Holdings, h)
		}
	}

	cash, err := s.Client.GetCashBalance(ctx, a.AccountNo)
	if err != nil {
		return Account{}, fmt.Errorf("portfolio: cash for %s: %w", a.AccountNo, err)
	}
	if len(cash.Data) > 0 {
		acct.Cash = mapCash(cash.Data[0])
	}

	risk, err := s.Client.GetMarginAccountInfo(ctx, a.AccountNo)
	switch {
	case err == nil:
		acct.Debt = mapDebt(risk, a.AccountNo)
	case !strings.EqualFold(a.AccountType, accountTypeMargin) && isClientError(err):
		// Non-margin accounts have no risk record; TCBS answers 4xx.
	default:
		return Account{}, fmt.Errorf("portfolio: margin risk for %s: %w", a.AccountNo, err)
	}
	return acct, nil
}

func isClientError(err error) bool {
	var apiErr *tcbs.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
}

// mapHolding maps one /se stock row; scale converts its prices to VND (see
// priceScale).
func mapHolding(st tcbs.StockHoldingInfo, scale float64) (Holding, bool) {
	h := Holding{
		Symbol:          strings.ToUpper(strings.TrimSpace(st.Symbol)),
		TotalQty:        toInt(st.TotalQtty),
		SellableQty:     toInt(st.AvailableTrading),
		ReceivingQty:    toInt(st.T0) + toInt(st.T1) + toInt(st.T2),
		RestrictedQty:   toInt(st.Blocked) + toInt(st.Mortgaged) + toInt(st.OnHold),
		PendingStockDiv: toInt(st.StockDividend),
		PendingCashDiv:  toInt(st.CashDividend),
		AvgCost:         toVND(st.CostPrice.Float(), scale),
		RefPrice:        toVND(st.CurrentPrice.Float(), scale),
		SecType:         strings.TrimSpace(st.SecType),
	}
	if h.Symbol == "" {
		return Holding{}, false
	}
	if h.TotalQty == 0 && h.ReceivingQty == 0 && h.PendingStockDiv == 0 && h.PendingCashDiv == 0 {
		return Holding{}, false
	}
	return h, true
}

func mapCash(c tcbs.CashInvestment) Cash {
	return Cash{
		Balance:           toInt(c.CashBalance),
		Withdrawable:      toInt(c.AvlWithdraw),
		Receivable:        toInt(c.AvlAdvanceAmount),
		IncomingDividends: toInt(c.SCashDividend) + toInt(c.BCashDividend),
		BankLinked:        toInt(c.BankAvlBalance),
		CustodyFeeDebt:    toInt(c.DepoFee),
	}
}

// mapDebt picks the risk record for accountNo, or the first one.
func mapDebt(items []tcbs.MarginAccountInfoResponse, accountNo string) Debt {
	if len(items) == 0 {
		return Debt{}
	}
	r := items[0]
	for _, it := range items {
		if it.AccountNo == accountNo {
			r = it
			break
		}
	}
	d := Debt{
		Outstanding:     toInt(r.Outstanding),
		AccruedInterest: toInt(r.AccruedInterest),
		FeeDebt:         toInt(r.TotalFeeDebt),
		RTT:             r.RTT.Float(),
	}
	if r.RiskStatus != nil {
		d.RiskCode = r.RiskStatus.Code
	}
	return d
}

// priceScale picks the unit of one TCBS response from all of its prices:
// 1000 (thousand VND) only when every non-zero price is below 1000, and 1
// (VND) otherwise. The spec's examples are in VND, and one listed share
// priced at 1000 VND or more proves the whole response is in VND, so a
// sub-1000 VND penny stock next to it is never scaled. The unit is decided per
// response, never per row.
func priceScale(prices []float64) float64 {
	seen := false
	for _, p := range prices {
		switch {
		case p == 0:
		case math.Abs(p) >= 1000:
			return 1
		default:
			seen = true
		}
	}
	if seen {
		return 1000
	}
	return 1
}

// toVND converts a TCBS price to VND with the response's scale.
func toVND(p, scale float64) int64 {
	return int64(math.Round(p * scale))
}

func toInt(n tcbs.Number) int64 {
	return int64(math.Round(n.Float()))
}

// TCBSQuotes maps tickerCommons rows (tcbs.Client.GetStockPrices) to quotes:
// matchPrice when it is non-zero (intraday), otherwise refPrice. Rows with
// neither are skipped.
func TCBSQuotes(rows []tcbs.MarketStockInfo) map[string]Quote {
	prices := make([]float64, 0, 2*len(rows))
	for _, r := range rows {
		prices = append(prices, r.MatchPrice.Float(), r.RefPrice.Float())
	}
	scale := priceScale(prices)
	out := make(map[string]Quote, len(rows))
	for _, r := range rows {
		sym := strings.ToUpper(strings.TrimSpace(r.Symbol))
		match, ref := r.MatchPrice.Float(), r.RefPrice.Float()
		switch {
		case sym == "":
		case match != 0:
			out[sym] = Quote{Price: toVND(match, scale), Source: SourceTCBSMatch, Intraday: true}
		case ref != 0:
			out[sym] = Quote{Price: toVND(ref, scale), Source: SourceRef}
		}
	}
	return out
}

// NewTCBSClient builds an SDK client on base (http.DefaultTransport when nil)
// with the per-request timeout. When rawDir is set, every response body
// except the token endpoint's is also written to rawDir.
func NewTCBSClient(baseURL string, base http.RoundTripper, timeout time.Duration, rawDir string) *tcbs.Client {
	rt := base
	if rawDir != "" {
		rt = &RawDumpTransport{Dir: rawDir, Base: base}
	}
	opts := []tcbs.Option{tcbs.WithHTTPClient(&http.Client{Transport: rt, Timeout: timeout})}
	if baseURL != "" {
		opts = append(opts, tcbs.WithBaseURL(baseURL))
	}
	return tcbs.NewClient(opts...)
}

// tokenPath is the TCBS token endpoint. Its response body is the JWT, so the
// raw dump never touches it.
const tokenPath = "/gaia/v1/oauth2/openapi/token"

// RawDumpTransport tees TCBS response bodies to Dir as
// <endpoint>-<key>.json (files 0600, dir 0700). It copies responses only,
// never requests, so the Authorization header and request bodies are never
// written, and it skips the token endpoint entirely.
type RawDumpTransport struct {
	Dir  string
	Base http.RoundTripper // http.DefaultTransport when nil
}

// RoundTrip implements http.RoundTripper.
func (t *RawDumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err != nil || strings.TrimRight(req.URL.Path, "/") == tokenPath {
		return resp, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("portfolio: raw dump: read body: %w", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if err := ensureRawDir(t.Dir); err != nil {
		return nil, err
	}
	path := filepath.Join(t.Dir, rawDumpName(req.URL.Path))
	if err := writeFile0600(path, body); err != nil {
		return nil, fmt.Errorf("portfolio: raw dump: %w", err)
	}
	return resp, nil
}

// rawDumpName turns /aion/v1/accounts/0001N/se into se-0001N.json and
// /tartarus/v1/tickerCommons into tickerCommons.json.
func rawDumpName(urlPath string) string {
	var segs []string
	for _, s := range strings.Split(urlPath, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	name := "response"
	if n := len(segs); n > 0 {
		name = segs[n-1]
		if n >= 4 {
			name += "-" + segs[n-2]
		}
	}
	return sanitizeName(name) + ".json"
}

func sanitizeName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		}
		return '_'
	}, s)
}

func ensureRawDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("portfolio: raw dump dir: %w", err)
	}
	return nil
}
