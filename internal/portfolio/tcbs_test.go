package portfolio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/miti99dev/stock-advisor/tcbs"
)

const (
	testAPIKey = "sk-test-SECRET"
	testToken  = "jwt-SECRET"
	testOTP    = "123456"
)

// fakeTCBS serves the testdata fixtures and records the paths it was asked for.
type fakeTCBS struct {
	t     *testing.T
	mu    sync.Mutex
	paths []string
}

func (f *fakeTCBS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	f.mu.Unlock()

	if r.URL.Path == "/gaia/v1/oauth2/openapi/token" {
		var req tcbs.TokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey != testAPIKey || req.OTP != testOTP {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":"203074","message":"The API Key is invalid"}`))
			return
		}
		_, _ = w.Write([]byte(`{"token":"` + testToken + `"}`))
		return
	}
	if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var fixture string
	switch p := r.URL.Path; {
	case p == "/aion/v1/customers/105C000001/accounts":
		fixture = "accounts.json"
	case strings.HasPrefix(p, "/aion/v1/accounts/") && strings.HasSuffix(p, "/se"):
		fixture = "se-" + strings.Split(p, "/")[4] + ".json"
	case strings.HasPrefix(p, "/aion/v1/accounts/") && strings.HasSuffix(p, "/cashInvestments"):
		fixture = "cash-" + strings.Split(p, "/")[4] + ".json"
	case strings.HasPrefix(p, "/hydros/v1/account/") && strings.HasSuffix(p, "/risk"):
		fixture = "risk-" + strings.Split(p, "/")[4] + ".json"
	}
	b, err := os.ReadFile(filepath.Join("testdata", fixture))
	if fixture == "" || err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"404","message":"not found"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func newFake(t *testing.T) (*fakeTCBS, *httptest.Server) {
	t.Helper()
	f := &fakeTCBS{t: t}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, srv
}

func signedInClient(t *testing.T, baseURL, rawDir string) *tcbs.Client {
	t.Helper()
	c := NewTCBSClient(baseURL, nil, 5*time.Second, rawDir)
	if _, err := c.GetToken(context.Background(), testAPIKey, testOTP); err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	return c
}

func TestTCBSSource_Snapshot(t *testing.T) {
	f, srv := newFake(t)
	src := TCBSSource{Client: signedInClient(t, srv.URL, ""), Custody: "105C000001"}
	snap, err := src.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.AsOf.IsZero() || snap.Custody != "105C000001" {
		t.Errorf("unexpected header: %+v", snap)
	}
	if len(snap.Accounts) != 2 || snap.Accounts[0].AccountNo != "0001N" || snap.Accounts[1].AccountNo != "0001G" {
		t.Fatalf("expected the two active stock accounts, got %+v", snap.Accounts)
	}
	for _, p := range f.paths {
		if strings.Contains(p, "0001X") || strings.Contains(p, "0001W") {
			t.Errorf("closed or internal account was queried: %s", p)
		}
	}

	n := snap.Accounts[0]
	if n.Type != "normal" || n.Status != "A" {
		t.Errorf("0001N type/status: %q/%q", n.Type, n.Status)
	}
	wantN := []Holding{
		{Symbol: "FPT", TotalQty: 1000, SellableQty: 800, ReceivingQty: 200, AvgCost: 95500, RefPrice: 120300, SecType: "001"},
		{Symbol: "E1VFVN30", TotalQty: 250, SellableQty: 250, AvgCost: 25100, RefPrice: 26400, SecType: "008"},
		{Symbol: "VNM", TotalQty: 137, SellableQty: 137, AvgCost: 0, RefPrice: 61200, SecType: "001"},
	}
	if !reflect.DeepEqual(n.Holdings, wantN) {
		t.Errorf("0001N holdings:\n got %+v\nwant %+v", n.Holdings, wantN)
	}
	wantCash := Cash{Balance: 10000000, Withdrawable: 9000000, Receivable: 2000000, IncomingDividends: 300000, BankLinked: 5000000, CustodyFeeDebt: 10000}
	if n.Cash != wantCash {
		t.Errorf("0001N cash: got %+v want %+v", n.Cash, wantCash)
	}
	if n.Debt != (Debt{}) {
		t.Errorf("0001N should have no debt (risk 404 ignored), got %+v", n.Debt)
	}

	g := snap.Accounts[1]
	wantG := []Holding{{
		Symbol: "FPT", TotalQty: 500, SellableQty: 400, RestrictedQty: 100, PendingStockDiv: 50,
		PendingCashDiv: 1000000, AvgCost: 100000, RefPrice: 120300, SecType: "001",
	}}
	if !reflect.DeepEqual(g.Holdings, wantG) {
		t.Errorf("0001G holdings:\n got %+v\nwant %+v", g.Holdings, wantG)
	}
	wantDebt := Debt{Outstanding: 20000000, AccruedInterest: 150000, FeeDebt: 5000, RTT: 185.5, RiskCode: "SAFE"}
	if g.Debt != wantDebt {
		t.Errorf("0001G debt: got %+v want %+v", g.Debt, wantDebt)
	}
}

func TestTCBSSource_AccountFilter(t *testing.T) {
	_, srv := newFake(t)
	client := signedInClient(t, srv.URL, "")

	snap, err := TCBSSource{Client: client, Custody: "105C000001", Accounts: []string{"0001g"}}.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Accounts) != 1 || snap.Accounts[0].AccountNo != "0001G" {
		t.Errorf("filter: got %+v", snap.Accounts)
	}

	if _, err := (TCBSSource{Client: client, Custody: "105C000001", Accounts: []string{"0001X"}}).Snapshot(context.Background()); err == nil {
		t.Error("filtering on a closed account should fail")
	}
}

func TestTCBSSource_MarginRiskErrorIsFatal(t *testing.T) {
	f, _ := newFake(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/hydros/v1/account/0001G/risk" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f.ServeHTTP(w, r)
	}))
	defer srv.Close()

	_, err := TCBSSource{Client: signedInClient(t, srv.URL, ""), Custody: "105C000001"}.Snapshot(context.Background())
	if err == nil || !strings.Contains(err.Error(), "0001G") {
		t.Errorf("a margin account without risk data should fail, got %v", err)
	}
}

func TestPriceScale(t *testing.T) {
	cases := []struct {
		name   string
		prices []float64
		want   float64
	}{
		{"all thousand VND", []float64{95.5, 120.3, 0, 61.2}, 1000},
		{"VND", []float64{95500, 120300}, 1},
		{"VND penny stock beside a normal one", []float64{600, 800, 95500, 120300}, 1},
		{"price of exactly 1000 is VND", []float64{999.9, 1000}, 1},
		{"no prices", []float64{0, 0}, 1},
		{"empty", nil, 1},
	}
	for _, c := range cases {
		if got := priceScale(c.prices); got != c.want {
			t.Errorf("%s: priceScale(%v) = %v, want %v", c.name, c.prices, got, c.want)
		}
	}
	if got := toVND(25.1236, 1000); got != 25124 {
		t.Errorf("toVND(25.1236, 1000) = %d", got)
	}
	if got := toVND(25123.6, 1); got != 25124 {
		t.Errorf("toVND(25123.6, 1) = %d", got)
	}
}

func TestMapHoldingPennyStockInVND(t *testing.T) {
	// A UPCoM share trading below 1,000 VND in a response that is in VND
	// keeps its price; only the response's unit decides the scale.
	rows := []tcbs.StockHoldingInfo{
		{Symbol: "PNY", TotalQtty: 10000, CostPrice: 800, CurrentPrice: 600},
		{Symbol: "FPT", TotalQtty: 100, CostPrice: 95500, CurrentPrice: 120300},
	}
	var prices []float64
	for _, r := range rows {
		prices = append(prices, r.CostPrice.Float(), r.CurrentPrice.Float())
	}
	h, ok := mapHolding(rows[0], priceScale(prices))
	if !ok || h.AvgCost != 800 || h.RefPrice != 600 {
		t.Errorf("penny stock: %+v", h)
	}
}

func TestTCBSQuotes(t *testing.T) {
	got := TCBSQuotes([]tcbs.MarketStockInfo{
		{Symbol: "fpt", MatchPrice: 121.5, RefPrice: 120.3},
		{Symbol: "VNM", MatchPrice: 0, RefPrice: 61.2},
		{Symbol: "NONE"},
	})
	want := map[string]Quote{
		"FPT": {Price: 121500, Source: SourceTCBSMatch, Intraday: true},
		"VNM": {Price: 61200, Source: SourceRef},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("thousand VND: got %+v, want %+v", got, want)
	}

	got = TCBSQuotes([]tcbs.MarketStockInfo{
		{Symbol: "HPG", MatchPrice: 27450, RefPrice: 27000},
		{Symbol: "PNY", MatchPrice: 600, RefPrice: 800},
		{Symbol: "OLD", RefPrice: 700},
	})
	want = map[string]Quote{
		"HPG": {Price: 27450, Source: SourceTCBSMatch, Intraday: true},
		"PNY": {Price: 600, Source: SourceTCBSMatch, Intraday: true},
		"OLD": {Price: 700, Source: SourceRef},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("VND with penny stocks: got %+v, want %+v", got, want)
	}
}

func TestRawDump_NoSecrets(t *testing.T) {
	_, srv := newFake(t)
	dir := filepath.Join(t.TempDir(), "raw")
	src := TCBSSource{Client: signedInClient(t, srv.URL, dir), Custody: "105C000001", RawDir: dir}
	if _, err := src.Snapshot(context.Background()); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("raw dir mode %o, want 700", perm)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
		fi, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("%s mode %o, want 600", e.Name(), perm)
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"SECRET", "Authorization", "Bearer", testOTP} {
			if strings.Contains(string(b), secret) {
				t.Errorf("%s contains %q", e.Name(), secret)
			}
		}
		if strings.Contains(e.Name(), "token") {
			t.Errorf("token endpoint was dumped: %s", e.Name())
		}
	}
	want := []string{
		"accounts-105C000001.json",
		"cashInvestments-0001G.json", "cashInvestments-0001N.json",
		"risk-0001G.json", "risk-0001N.json",
		"se-0001G.json", "se-0001N.json",
	}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("dumped files:\n got %v\nwant %v", names, want)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "se-0001N.json"))
	if !strings.Contains(string(b), `"FPT"`) {
		t.Errorf("se dump does not hold the response body: %s", b)
	}
}

func TestRawDumpName(t *testing.T) {
	cases := map[string]string{
		"/aion/v1/accounts/0001N/se":    "se-0001N.json",
		"/hydros/v1/account/0001G/risk": "risk-0001G.json",
		"/tartarus/v1/tickerCommons":    "tickerCommons.json",
		"/":                             "response.json",
		"/a/b/../x y/z":                 "z-x_y.json",
	}
	for in, want := range cases {
		if got := rawDumpName(in); got != want {
			t.Errorf("rawDumpName(%q) = %q, want %q", in, got, want)
		}
	}
}
