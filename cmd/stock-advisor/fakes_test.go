package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// testNow is the fixed clock of every end-to-end run: 15:00 in Vietnam on a
// Tuesday, after the close.
var testNow = time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)

// TestMain blocks every non-loopback dial, so no test can reach the network
// even through a client built on http.DefaultTransport.
func TestMain(m *testing.M) {
	http.DefaultTransport = loopbackOnly()
	os.Exit(m.Run())
}

// loopbackOnly is a transport that refuses to dial anything but 127.0.0.1
// and ::1.
func loopbackOnly() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	d := &net.Dialer{Timeout: 5 * time.Second}
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if ip := net.ParseIP(host); err != nil || ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("test: dial to %s blocked: tests make no network calls", addr)
		}
		return d.DialContext(ctx, network, addr)
	}
	return t
}

// testDeps points every client at the given servers with no throttling and
// no retries.
func testDeps(tcbsURL, researchURL string) deps {
	return deps{
		tcbsURL:     tcbsURL,
		vndChartURL: researchURL,
		vndFinfoURL: researchURL,
		vciChartURL: researchURL,
		vciIQURL:    researchURL,
		kbsURL:      researchURL,
		transport:   loopbackOnly(),
		ratePerHost: 1000,
		backoff:     []time.Duration{},
		now:         func() time.Time { return testNow },
	}
}

// fakeStock is one symbol served by the research fake.
type fakeStock struct {
	name, typ, icb1, icb2, icb4 string
	peers                       []string // the ICB level-4 code list, the symbol included
	listed                      string
	price                       float64 // last close, VND
	beta, idio                  float64 // daily return = beta × index + idio × noise
	volume                      float64
	ratios                      map[string]float64 // VNDirect ratio code -> value
	cash                        map[string]float64 // ex-date -> VND per share
	stock                       map[string]float64 // ex-date -> percent of held shares
	profits                     []float64          // fiscal 2025, 2024, 2023, 2022
	bank                        map[string]float64 // Vietcap statistics-financial fields
	kbsOnly                     bool               // dchart has no data; bars come from KBS
}

func ratios(pe, pb, roe, de, eps, bvps float64) map[string]float64 {
	m := map[string]float64{"PRICE_TO_EARNINGS": pe, "PRICE_TO_BOOK": pb, "ROAE_TR_AVG5Q": roe, "EPS_TR": eps, "BVPS_CR": bvps}
	if de >= 0 {
		m["DEBT_TO_EQUITY_AQ"] = de
	}
	return m
}

const (
	icbTech = "Computer Services"
	icbBank = "Banks"
)

var softwarePeers = []string{"FPT", "CMG", "ELC", "ITD", "SAM", "CTR"}
var bankPeers = []string{"VCB", "ACB", "BID"}

// fakeStocks is the market the end-to-end tests see. Symbols missing here
// (ZZZ, OLD) are unknown to every source.
var fakeStocks = map[string]fakeStock{
	"FPT": {name: "FPT Corporation", typ: "STOCK", icb1: "Technology", icb2: "Technology", icb4: icbTech, peers: softwarePeers,
		listed: "2006-12-13", price: 120300, beta: 0.9, idio: 0.011, volume: 3e6,
		ratios: ratios(18.5, 4.8, 0.27, 0.45, 6500, 25000), cash: map[string]float64{"2026-06-12": 1000, "2025-12-01": 1000, "2025-06-12": 1000, "2024-06-12": 1000},
		stock: map[string]float64{"2026-05-28": 15}, profits: []float64{9.4e9, 7.8e9, 6.5e9, 5.3e9}},
	"VNM": {name: "Vietnam Dairy Products", typ: "STOCK", icb1: "Consumer Goods", icb2: "Food & Beverage", icb4: "Food Products",
		peers: []string{"VNM", "MCH"}, listed: "2006-01-19", price: 61200, beta: 0.5, idio: 0.009, volume: 4e6,
		ratios: ratios(15.2, 3.9, 0.26, 0.3, 4000, 15700), cash: map[string]float64{"2026-07-01": 1500, "2026-01-05": 2500, "2025-07-01": 1500, "2024-07-01": 1500},
		profits: []float64{9.1e9, 9.4e9, 9.0e9, 8.5e9}},
	"REE": {name: "Refrigeration Electrical Engineering", typ: "STOCK", icb1: "Utilities", icb2: "Utilities", icb4: "Multiutilities",
		peers: []string{"REE"}, listed: "2000-07-28", price: 66500, beta: 0.6, idio: 0.012, volume: 4e5,
		ratios: ratios(11, 1.3, 0.11, 0.4, 6000, 51000), cash: map[string]float64{"2026-02-26": 1000, "2025-02-27": 1000, "2024-04-12": 1000},
		stock: map[string]float64{"2026-05-18": 15}, profits: []float64{2.2e9, 2.0e9, 2.4e9, 2.6e9}},
	"HPG": {name: "Hoa Phat Group", typ: "STOCK", icb1: "Basic Materials", icb2: "Basic Resources", icb4: "Iron & Steel",
		peers: []string{"HPG", "HSG", "NKG"}, listed: "2007-11-15", price: 26950, beta: 1.4, idio: 0.025, volume: 3e7,
		ratios: ratios(16, 1.5, 0.09, 1.1, 1700, 18000), stock: map[string]float64{"2025-06-20": 20},
		profits: []float64{1.2e10, 6.8e9, 8.4e9, 3.4e10}},
	"VCB": {name: "Bank for Foreign Trade of Vietnam", typ: "STOCK", icb1: "Financials", icb2: icbBank, icb4: icbBank, peers: bankPeers,
		listed: "2009-06-30", price: 63400, beta: 0.8, idio: 0.01, volume: 3e6,
		ratios: ratios(14.5, 2.3, 0.17, -1, 4400, 27500), cash: map[string]float64{"2026-07-23": 450, "2025-10-03": 450},
		stock: map[string]float64{"2025-03-12": 49.5}, profits: []float64{3.4e10, 3.3e10, 3.0e10, 2.9e10},
		bank: map[string]float64{"npl": 0.006, "casaRatio": 0.34, "pe": 14.2, "pb": 2.2, "roe": 0.17}},
	"E1VFVN30": {name: "DCVFM VN30 ETF", typ: "ETF", listed: "2014-10-06", price: 26400, beta: 1, idio: 0.002, volume: 2e5},
	"ACB": {name: "Asia Commercial Bank", typ: "STOCK", icb1: "Financials", icb2: icbBank, icb4: icbBank, peers: bankPeers,
		listed: "2006-11-21", price: 25000, beta: 0.8, idio: 0.009, volume: 8e6,
		ratios: ratios(6.5, 1.2, 0.21, -1, 3800, 21000), cash: map[string]float64{"2026-06-10": 1000, "2025-06-10": 1000, "2024-06-10": 1000},
		profits: []float64{1.7e10, 1.6e10, 1.5e10, 1.3e10},
		bank:    map[string]float64{"npl": 0.012, "casaRatio": 0.22, "pe": 6.5, "pb": 1.2, "roe": 0.21}},
	"MWG": {name: "Mobile World Investment", typ: "STOCK", icb1: "Consumer Services", icb2: "Retail", icb4: "Specialty Retailers",
		peers: []string{"MWG", "FRT", "DGW"}, listed: "2014-07-14", price: 60000, beta: 1.2, idio: 0.02, volume: 6e6,
		ratios: ratios(20, 3.1, 0.15, 0.9, 3000, 19000), cash: map[string]float64{"2026-08-01": 500},
		profits: []float64{3.7e9, 2.4e9, 0.17e9, 4.1e9}},
	"GAS": {name: "PetroVietnam Gas", typ: "STOCK", icb1: "Utilities", icb2: "Utilities", icb4: "Gas Distribution",
		peers: []string{"GAS", "PGD"}, listed: "2012-05-21", price: 70000, beta: 0.7, idio: 0.009, volume: 1.5e6,
		ratios: ratios(13, 2.4, 0.18, 0.3, 5400, 29000), cash: map[string]float64{"2026-08-15": 3600, "2025-08-15": 3600, "2024-08-15": 2000},
		profits: []float64{1.1e10, 1.2e10, 1.5e10, 1.4e10}},
	"PNJ": {name: "Phu Nhuan Jewelry", typ: "STOCK", icb1: "Consumer Goods", icb2: "Personal & Household Goods", icb4: "Clothing & Accessories",
		peers: []string{"PNJ"}, listed: "2009-03-23", price: 95000, beta: 0.7, idio: 0.01, volume: 1e6, kbsOnly: true,
		ratios: ratios(14, 2.9, 0.21, 0.35, 6800, 33000), cash: map[string]float64{"2026-06-20": 1400, "2025-06-20": 1400, "2024-06-20": 2000},
		profits: []float64{2.0e9, 1.9e9, 1.8e9, 1.8e9}},
	"DGC": {name: "Duc Giang Chemicals", typ: "STOCK", icb1: "Basic Materials", icb2: "Chemicals", icb4: "Specialty Chemicals",
		peers: []string{"DGC", "CSV"}, listed: "2020-07-28", price: 100000, beta: 1.1, idio: 0.018, volume: 1.2e6,
		ratios: ratios(13, 3.0, 0.24, 0.2, 7700, 33000), cash: map[string]float64{"2026-07-10": 3000, "2025-07-10": 3000, "2024-07-10": 3000},
		profits: []float64{3.0e9, 3.1e9, 5.5e9, 2.4e9}},
}

// peerPE are the P/Es of symbols the fake knows only as ratio rows.
var peerPE = map[string]float64{"CMG": 22, "ELC": 15, "ITD": 12, "SAM": 30, "CTR": 21, "BID": 13, "MCH": 17, "HSG": 14, "NKG": 12, "FRT": 40, "DGW": 18, "PGD": 9, "CSV": 8}

// calendar is the fake's trading days: every weekday from 2024-09-02 to
// testNow.
var calendar = func() []time.Time {
	var out []time.Time
	for d := time.Date(2024, 9, 2, 0, 0, 0, 0, time.UTC); !d.After(testNow); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			out = append(out, d)
		}
	}
	return out
}()

// noise is a deterministic standard-normal sequence seeded by name.
func noise(name string, n int) []float64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	state := h.Sum64() | 1
	uniform := func() float64 {
		state = state*6364136223846793005 + 1442695040888963407
		return (float64(state>>11) + 0.5) / (1 << 53)
	}
	out := make([]float64, n)
	for i := range out {
		u1, u2 := uniform(), uniform()
		out[i] = math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	}
	return out
}

type fakeBar struct {
	day                    time.Time
	open, high, low, close float64
	volume                 float64
}

// fakeBars builds the daily bars of a symbol, ending at its last price.
// The index ends at 1,700 points.
func fakeBars(sym string) []fakeBar {
	idx := noise("VNINDEX", len(calendar))
	last, beta, idio, volume, round := 1700.0, 1.0, 0.0, 1e8, 0.01
	if sym != indexSymbol {
		st := fakeStocks[sym]
		last, beta, idio, volume, round = st.price, st.beta, st.idio, st.volume, 10
	}
	own := noise(sym, len(calendar))
	cum := make([]float64, len(calendar))
	for i := 1; i < len(calendar); i++ {
		cum[i] = cum[i-1] + beta*0.011*idx[i] + idio*own[i]
	}
	rnd := func(x float64) float64 { return math.Round(x/round) * round }
	bars := make([]fakeBar, len(calendar))
	for i, d := range calendar {
		c := rnd(last * math.Exp(cum[i]-cum[len(cum)-1]))
		o := c
		if i > 0 {
			o = bars[i-1].close
		}
		bars[i] = fakeBar{day: d, open: o, high: rnd(math.Max(o, c) * 1.01), low: rnd(math.Min(o, c) * 0.99), close: c,
			volume: math.Round(volume * (1 + 0.3*math.Sin(float64(i)/7)))}
	}
	return bars
}

// newResearchServer serves VNDirect, Vietcap and KBS from one handler,
// generated from fakeStocks. Vietcap's chart answers 400 so PNJ falls
// through to KBS.
func newResearchServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}

	mux.HandleFunc("/dchart/history", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		sym := q.Get("symbol")
		if st, ok := fakeStocks[sym]; (!ok || st.kbsOnly) && sym != indexSymbol {
			writeJSON(w, map[string]any{"s": "no_data"})
			return
		}
		from, _ := strconv.ParseInt(q.Get("from"), 10, 64)
		to, _ := strconv.ParseInt(q.Get("to"), 10, 64)
		scale := 1000.0
		if sym == indexSymbol {
			scale = 1
		}
		resp := map[string]any{"s": "ok"}
		var ts []int64
		var o, h, l, c, v []float64
		for _, b := range fakeBars(sym) {
			if b.day.Unix() < from || b.day.Unix() > to {
				continue
			}
			ts = append(ts, b.day.Add(2*time.Hour).Unix())
			o, h, l, c = append(o, b.open/scale), append(h, b.high/scale), append(l, b.low/scale), append(c, b.close/scale)
			v = append(v, b.volume)
		}
		resp["t"], resp["o"], resp["h"], resp["l"], resp["c"], resp["v"] = ts, o, h, l, c, v
		writeJSON(w, resp)
	})

	mux.HandleFunc("/api/chart/OHLCChart/gap-chart", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
	})

	mux.HandleFunc("/investment/stocks/", func(w http.ResponseWriter, r *http.Request) {
		sym := strings.Split(strings.TrimPrefix(r.URL.Path, "/investment/stocks/"), "/")[0]
		st, ok := fakeStocks[sym]
		if !ok || !st.kbsOnly {
			http.NotFound(w, r)
			return
		}
		bars := fakeBars(sym)
		rows := make([]map[string]any, 0, len(bars))
		for i := len(bars) - 1; i >= 0; i-- { // newest first
			b := bars[i]
			rows = append(rows, map[string]any{"t": b.day.Format("2006-01-02") + " 07:00", "o": b.open, "h": b.high, "l": b.low,
				"c": strconv.FormatFloat(b.close, 'f', -1, 64), "v": b.volume})
		}
		writeJSON(w, map[string]any{"symbol": sym, "data_day": rows})
	})

	mux.HandleFunc("/investment/stock/finance-info/", func(w http.ResponseWriter, r *http.Request) {
		sym := strings.TrimPrefix(r.URL.Path, "/investment/stock/finance-info/")
		st, ok := fakeStocks[sym]
		if !ok || len(st.profits) == 0 {
			http.NotFound(w, r)
			return
		}
		name := "Profit after tax for shareholders of parent company"
		if st.bank != nil {
			name = "Net profit after tax"
		}
		row := map[string]any{"NameEn": name}
		for i, p := range st.profits {
			row["Value"+strconv.Itoa(i+1)] = p
		}
		writeJSON(w, map[string]any{
			"Head":    []map[string]int{{"YearPeriod": 2025}, {"YearPeriod": 2024}, {"YearPeriod": 2023}, {"YearPeriod": 2022}},
			"Content": map[string]any{"Kết quả kinh doanh": []any{row}},
		})
	})

	mux.HandleFunc("/investment/index/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/investment/index/30/stocks", "/investment/index/100/stocks":
			writeJSON(w, map[string]any{"status": 200, "data": []string{"ACB", "DGC", "FPT", "GAS", "MWG", "PNJ", "VCB", "VNM"}})
		default:
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("/api/iq-insight-service/v1/company/", func(w http.ResponseWriter, r *http.Request) {
		sym := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/iq-insight-service/v1/company/"), "/")[0]
		st, ok := fakeStocks[sym]
		if !ok || st.bank == nil {
			http.NotFound(w, r)
			return
		}
		row := map[string]any{"organCode": sym, "ratioType": "RATIO_TTM", "yearReport": 2026, "quarter": 2}
		for k, v := range st.bank {
			row[k] = v
		}
		writeJSON(w, map[string]any{"status": 200, "data": []any{row}})
	})

	mux.HandleFunc("/v4/stocks", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		var rows []map[string]any
		if list, ok := strings.CutPrefix(q, "code:"); ok {
			for _, sym := range strings.Split(list, ",") {
				if st, ok := fakeStocks[sym]; ok {
					rows = append(rows, map[string]any{"code": sym, "type": st.typ, "floor": "HOSE", "status": "listed",
						"companyNameEng": st.name, "listedDate": st.listed})
				}
			}
		}
		writeJSON(w, map[string]any{"data": rows})
	})

	mux.HandleFunc("/v4/industry_classification", func(w http.ResponseWriter, r *http.Request) {
		sym := strings.TrimPrefix(r.URL.Query().Get("q"), "codeList:")
		st, ok := fakeStocks[sym]
		if !ok || st.icb4 == "" {
			writeJSON(w, map[string]any{"data": []any{}})
			return
		}
		writeJSON(w, map[string]any{"data": []map[string]any{
			{"industryLevel": "4", "englishName": st.icb4 + " ", "codeList": strings.Join(st.peers, ",")},
			{"industryLevel": "2", "englishName": st.icb2, "codeList": sym},
			{"industryLevel": "1", "englishName": strings.ToUpper(st.icb1) + " ", "codeList": sym},
		}})
	})

	mux.HandleFunc("/v4/ratios/latest", func(w http.ResponseWriter, r *http.Request) {
		var rows []map[string]any
		for _, sym := range strings.Split(strings.TrimPrefix(r.URL.Query().Get("where"), "code:"), ",") {
			vals := fakeStocks[sym].ratios
			if pe, ok := peerPE[sym]; ok {
				vals = map[string]float64{"PRICE_TO_EARNINGS": pe}
			}
			codes := make([]string, 0, len(vals))
			for code := range vals {
				codes = append(codes, code)
			}
			sort.Strings(codes)
			for _, code := range codes {
				rows = append(rows, map[string]any{"code": sym, "ratioCode": code, "value": vals[code], "reportDate": "2026-06-30"})
			}
		}
		writeJSON(w, map[string]any{"data": rows})
	})

	mux.HandleFunc("/v4/events", func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimPrefix(r.URL.Query().Get("q"), "code:")
		sym, _, _ := strings.Cut(q, "~")
		st := fakeStocks[sym]
		rows := []map[string]any{}
		for d, cash := range st.cash {
			rows = append(rows, map[string]any{"code": sym, "type": "DIVIDEND", "dividend": cash, "effectiveDate": d})
		}
		for d, pctShares := range st.stock {
			rows = append(rows, map[string]any{"code": sym, "type": "STOCKDIV", "ratio": pctShares, "effectiveDate": d})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i]["effectiveDate"].(string) > rows[j]["effectiveDate"].(string) })
		writeJSON(w, map[string]any{"data": rows})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

const (
	testAPIKey  = "sk-test-SECRET"
	testToken   = "jwt-SECRET"
	testOTP     = "123456"
	testCustody = "105C000001"
)

// fakeTCBS serves the portfolio fixtures of internal/portfolio. failToken
// answers the token call with 400 (echoing the key, as a careless server
// might); failAccounts answers every account call with 500.
type fakeTCBS struct {
	failToken, failAccounts bool

	mu    sync.Mutex
	paths []string
}

func (f *fakeTCBS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/gaia/v1/oauth2/openapi/token" {
		var req struct {
			APIKey string `json:"apiKey"`
			OTP    string `json:"otp"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || f.failToken || req.APIKey != testAPIKey || req.OTP != testOTP {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, `{"code":"203074","message":"The API Key %s is invalid"}`, req.APIKey)
			return
		}
		_, _ = w.Write([]byte(`{"token":"` + testToken + `"}`))
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+testToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if f.failAccounts {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":"500","message":"internal error"}`))
		return
	}

	var fixture string
	switch p := r.URL.Path; {
	case p == "/aion/v1/customers/"+testCustody+"/accounts":
		fixture = "accounts.json"
	case strings.HasPrefix(p, "/aion/v1/accounts/") && strings.HasSuffix(p, "/se"):
		fixture = "se-" + strings.Split(p, "/")[4] + ".json"
	case strings.HasPrefix(p, "/aion/v1/accounts/") && strings.HasSuffix(p, "/cashInvestments"):
		fixture = "cash-" + strings.Split(p, "/")[4] + ".json"
	case strings.HasPrefix(p, "/hydros/v1/account/") && strings.HasSuffix(p, "/risk"):
		fixture = "risk-" + strings.Split(p, "/")[4] + ".json"
	case p == "/tartarus/v1/tickerCommons":
		var rows []map[string]any
		for _, sym := range strings.Split(r.URL.Query().Get("tickers"), ",") {
			if st, ok := fakeStocks[sym]; ok {
				rows = append(rows, map[string]any{"symbol": sym, "refPrice": st.price, "matchPrice": 0})
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": rows, "tradingDate": "13/10/2026"})
		return
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "internal", "portfolio", "testdata", fixture))
	if fixture == "" || err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"404","message":"not found"}`))
		return
	}
	_, _ = w.Write(b)
}

func (f *fakeTCBS) called(path string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.paths {
		if p == path {
			return true
		}
	}
	return false
}
