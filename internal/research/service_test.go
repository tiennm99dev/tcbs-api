package research

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeHosts routes every provider to its own httptest server and counts the
// requests each one receives.
type fakeHosts struct {
	vndChart, vndFinfo, vciChart, vciIQ, kbs http.HandlerFunc

	mu    sync.Mutex
	calls map[string]int
}

func (f *fakeHosts) count(host string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[host]
}

func (f *fakeHosts) service(t *testing.T, cache Cache) *Service {
	t.Helper()
	f.calls = map[string]int{}
	url := func(name string, h *http.HandlerFunc) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f.mu.Lock()
			f.calls[name]++
			f.mu.Unlock()
			if *h == nil {
				http.Error(w, "unexpected", http.StatusTeapot)
				t.Errorf("unexpected %s request %s", name, r.URL)
				return
			}
			(*h)(w, r)
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	s := NewService(ServiceOptions{
		RatePerHost:      1000,
		Backoff:          []time.Duration{time.Millisecond, time.Millisecond},
		Cache:            cache,
		Now:              func() time.Time { return testNow },
		VNDirectChartURL: url("vndChart", &f.vndChart),
		VNDirectFinfoURL: url("vndFinfo", &f.vndFinfo),
		VietcapChartURL:  url("vciChart", &f.vciChart),
		VietcapIQURL:     url("vciIQ", &f.vciIQ),
		KBSURL:           url("kbs", &f.kbs),
	})
	t.Cleanup(s.Close)
	return s
}

func status(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { http.Error(w, http.StatusText(code), code) }
}

func TestServiceDailyFallbackChain(t *testing.T) {
	f := &fakeHosts{
		vndChart: status(http.StatusInternalServerError),
		vciChart: status(http.StatusBadRequest),
		kbs:      serveFixture(t, "kbs_dataday_fpt.json"),
	}
	s := f.service(t, Cache{})

	got, err := s.Daily(context.Background(), "FPT", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "kbs" || got.Adjusted || len(got.Bars) != 4 {
		t.Errorf("series = %+v", got)
	}
	// A 500 is retried twice; a 400 is not retried.
	if f.count("vndChart") != 3 || f.count("vciChart") != 1 || f.count("kbs") != 1 {
		t.Errorf("calls = %v", f.calls)
	}
	if w := s.Warnings(); len(w) != 1 || !strings.Contains(w[0], "FPT: daily bars from kbs") {
		t.Errorf("warnings = %q", w)
	}
}

func TestServiceDailyAllFail(t *testing.T) {
	f := &fakeHosts{
		vndChart: func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"s":"no_data"}`)) },
		vciChart: func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`[]`)) },
		kbs: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"symbol":"ZZZZ","data_day":[]}`))
		},
	}
	s := f.service(t, Cache{})
	if _, err := s.Daily(context.Background(), "ZZZZ", day(2026, 9, 1), testNow); err == nil {
		t.Fatal("want an error")
	}
	if w := s.Warnings(); len(w) != 1 || !strings.Contains(w[0], "ZZZZ: no daily bars") {
		t.Errorf("warnings = %q", w)
	}
}

func TestServiceDailyCached(t *testing.T) {
	f := &fakeHosts{vndChart: serveFixture(t, "vnd_dchart_fpt.json")}
	s := f.service(t, Cache{Dir: t.TempDir()})

	first, err := s.Daily(context.Background(), "FPT", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Daily(context.Background(), "FPT", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if f.count("vndChart") != 1 {
		t.Errorf("dchart calls = %d, want 1 (second read from cache)", f.count("vndChart"))
	}
	if len(second.Bars) != len(first.Bars) || !second.Bars[0].Date.Equal(first.Bars[0].Date) || second.Source != "vndirect" {
		t.Errorf("cached series differs: %+v", second)
	}

	batch := s.DailyBatch(context.Background(), []string{"fpt", "FPT"}, day(2026, 9, 1), testNow)
	if len(batch) != 1 || f.count("vndChart") != 1 {
		t.Errorf("batch = %d series, calls = %d", len(batch), f.count("vndChart"))
	}
}

func TestServiceFundamentalsBatching(t *testing.T) {
	var batches atomic.Int32
	f := &fakeHosts{vndFinfo: func(w http.ResponseWriter, r *http.Request) {
		codes := strings.Split(strings.TrimPrefix(r.URL.Query().Get("where"), "code:"), ",")
		batches.Add(1)
		if len(codes) > 40 {
			t.Errorf("batch of %d symbols, want at most 40", len(codes))
		}
		var rows []string
		for _, c := range codes {
			rows = append(rows, fmt.Sprintf(`{"code":%q,"ratioCode":"PRICE_TO_BOOK","value":1.5,"reportDate":"2026-06-30"}`, c))
		}
		_, _ = fmt.Fprintf(w, `{"data":[%s]}`, strings.Join(rows, ","))
	}}
	s := f.service(t, Cache{})

	var symbols []string
	for i := 0; i < 45; i++ {
		symbols = append(symbols, fmt.Sprintf("S%02d", i))
	}
	got, err := s.Fundamentals(context.Background(), symbols)
	if err != nil {
		t.Fatal(err)
	}
	if batches.Load() != 2 || len(got) != 45 {
		t.Errorf("batches = %d, results = %d", batches.Load(), len(got))
	}
	wantFloat(t, "S44 PB", got["S44"].PB, 1.5)
}

func TestServiceFundamentalsRetryPerSymbol(t *testing.T) {
	f := &fakeHosts{vndFinfo: func(w http.ResponseWriter, r *http.Request) {
		where := r.URL.Query().Get("where")
		switch where {
		case "code:FPT":
			_, _ = w.Write(fixture(t, "vnd_ratios_latest.json"))
		case "code:VCB":
			_, _ = w.Write(fixture(t, "vnd_ratios_latest.json"))
		default: // the batch and BAD fail
			http.Error(w, "boom", http.StatusBadGateway)
		}
	}}
	s := f.service(t, Cache{})

	got, err := s.Fundamentals(context.Background(), []string{"FPT", "VCB", "BAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantFloat(t, "FPT PE", got["FPT"].PE, 10.76823468030994)
	if got["VCB"].DE != nil {
		t.Error("bank DE should stay nil")
	}
	bad, ok := got["BAD"]
	if !ok || bad.Symbol != "BAD" || bad.PE != nil || bad.ROE != nil {
		t.Errorf("BAD = %+v, %v; want an entry with nil fields", bad, ok)
	}
	if w := s.Warnings(); len(w) != 1 || !strings.Contains(w[0], "BAD: ratios unavailable") {
		t.Errorf("warnings = %q", w)
	}
}

func TestServiceBankExtras(t *testing.T) {
	f := &fakeHosts{vciIQ: serveFixture(t, "vci_statfin_vcb.json")}
	s := f.service(t, Cache{})

	in := Fundamentals{Symbol: "VCB", Source: "vndirect", PE: ptr(11.4), AsOf: day(2026, 10, 12)}
	got, err := s.BankExtras(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	wantFloat(t, "NPL", got.NPL, 0.0061066523)
	wantFloat(t, "CASA", got.CASA, 0.3392956361)
	wantFloat(t, "PE kept from VNDirect", got.PE, 11.4)
	wantFloat(t, "PB filled", got.PB, 1.9072314564)
	wantFloat(t, "DE filled", got.DE, 9.6943344266)
	if !got.AsOf.Equal(day(2026, 6, 30)) {
		t.Errorf("AsOf = %v, want the older Vietcap quarter end", got.AsOf)
	}
}

func TestServiceNetProfitYears(t *testing.T) {
	f := &fakeHosts{kbs: serveFixture(t, "kbs_kqkd_fpt.json")}
	s := f.service(t, Cache{Dir: t.TempDir()})
	for i := 0; i < 2; i++ {
		got, err := s.NetProfitYears(context.Background(), "FPT")
		if err != nil || len(got) != 4 || got[3] != 9376127630 {
			t.Fatalf("got %v, %v", got, err)
		}
	}
	if f.count("kbs") != 1 {
		t.Errorf("kbs calls = %d, want 1", f.count("kbs"))
	}
}

func TestServiceDividends(t *testing.T) {
	t.Run("vndirect", func(t *testing.T) {
		f := &fakeHosts{vndFinfo: serveFixture(t, "vnd_events_fpt.json")}
		s := f.service(t, Cache{})
		got, err := s.Dividends(context.Background(), "FPT", day(2024, 10, 13))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 6 || !got[5].ExDate.Equal(day(2024, 12, 2)) {
			t.Errorf("got %+v, want the 6 events since 2024-10-13 (two KINDDIV)", got)
		}
	})
	t.Run("vietcap fallback", func(t *testing.T) {
		f := &fakeHosts{vndFinfo: status(http.StatusServiceUnavailable), vciIQ: serveFixture(t, "vci_events_fpt.json")}
		s := f.service(t, Cache{})
		got, err := s.Dividends(context.Background(), "FPT", day(2025, 10, 13))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 5 || got[0] != (Dividend{ExDate: day(2026, 9, 21), StockRatio: 0.1}) {
			t.Errorf("got %+v", got)
		}
		if w := s.Warnings(); len(w) != 1 || !strings.Contains(w[0], "dividends from vietcap") {
			t.Errorf("warnings = %q", w)
		}
	})
}

func TestServiceProfiles(t *testing.T) {
	f := &fakeHosts{vndFinfo: func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v4/stocks":
			_, _ = w.Write(fixture(t, "vnd_stocks.json"))
		case r.URL.Query().Get("q") == "codeList:FPT":
			_, _ = w.Write(fixture(t, "vnd_industry_fpt.json"))
		case r.URL.Query().Get("q") == "codeList:VCB":
			_, _ = w.Write(fixture(t, "vnd_industry_vcb.json"))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}}
	s := f.service(t, Cache{Dir: t.TempDir()})

	got, err := s.Profiles(context.Background(), []string{"FPT", "VCB", "E1VFVN30", "ZZZZ"})
	if err != nil {
		t.Fatal(err)
	}
	vcb := got["VCB"]
	if vcb.ICB1 != "FINANCIALS" || vcb.ICB2 != "Banks" || vcb.ICB4 != "Banks" || vcb.Exchange != "HOSE" || len(vcb.Peers) == 0 {
		t.Errorf("VCB = %+v", vcb)
	}
	if got["FPT"].ICB4 != "Computer Services" {
		t.Errorf("FPT = %+v", got["FPT"])
	}
	if p, ok := got["E1VFVN30"]; !ok || p.Type != "ETF" || p.ICB1 != "" {
		t.Errorf("ETF = %+v, %v; want a listing without industry", p, ok)
	}
	if _, ok := got["ZZZZ"]; ok {
		t.Error("unknown symbol should be absent")
	}

	// FPT and VCB are cached; the ETF without industry data is not.
	before := f.count("vndFinfo")
	if _, err := s.Profiles(context.Background(), []string{"FPT", "VCB"}); err != nil {
		t.Fatal(err)
	}
	if f.count("vndFinfo") != before {
		t.Errorf("cached profiles refetched")
	}
}

func TestServiceUniverse(t *testing.T) {
	t.Run("VN100 from KBS", func(t *testing.T) {
		f := &fakeHosts{kbs: serveFixture(t, "kbs_index_100.json")}
		s := f.service(t, Cache{})
		got, err := s.Universe(context.Background(), "vn100")
		if err != nil || len(got) != 100 {
			t.Fatalf("got %d, %v", len(got), err)
		}
	})
	t.Run("VN100 falls back to VN30", func(t *testing.T) {
		f := &fakeHosts{kbs: status(http.StatusInternalServerError), vndFinfo: serveFixture(t, "vnd_stocks_vn30.json")}
		s := f.service(t, Cache{})
		got, err := s.Universe(context.Background(), "VN100")
		if err != nil || len(got) != 30 {
			t.Fatalf("got %d, %v", len(got), err)
		}
		if w := s.Warnings(); len(w) != 1 || !strings.Contains(w[0], "using VN30 instead") {
			t.Errorf("warnings = %q", w)
		}
	})
	t.Run("VN30 falls back to KBS", func(t *testing.T) {
		f := &fakeHosts{
			vndFinfo: status(http.StatusInternalServerError),
			kbs: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/investment/index/30/stocks" {
					t.Errorf("path = %s", r.URL.Path)
				}
				_, _ = w.Write(fixture(t, "kbs_index_30.json"))
			},
		}
		s := f.service(t, Cache{})
		got, err := s.Universe(context.Background(), "VN30")
		if err != nil || len(got) != 30 {
			t.Fatalf("got %d, %v", len(got), err)
		}
	})
	t.Run("unknown group", func(t *testing.T) {
		s := (&fakeHosts{}).service(t, Cache{})
		if _, err := s.Universe(context.Background(), "VN999"); err == nil {
			t.Error("want an error")
		}
	})
}

func TestServiceForEachBoundsWorkers(t *testing.T) {
	s := NewService(ServiceOptions{Workers: 4})
	defer s.Close()
	var inFlight, peak atomic.Int32
	symbols := make([]string, 20)
	for i := range symbols {
		symbols[i] = fmt.Sprint(i)
	}
	s.ForEach(context.Background(), symbols, func(context.Context, string) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
	})
	if peak.Load() > 4 {
		t.Errorf("peak workers = %d, want at most 4", peak.Load())
	}
}
