package research

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestVNDirect(t *testing.T, h http.HandlerFunc) *VNDirect {
	srv := newServer(t, h)
	return &VNDirect{HTTP: newTestHTTP(t), ChartURL: srv.URL, FinfoURL: srv.URL, Now: func() time.Time { return testNow }}
}

func TestVNDirectDaily(t *testing.T) {
	b := fixture(t, "vnd_dchart_fpt.json")
	v := newTestVNDirect(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/dchart/history" || q.Get("symbol") != "FPT" || q.Get("resolution") != "D" || q.Get("from") == "" || q.Get("to") == "" {
			t.Errorf("unexpected request %s", r.URL)
		}
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		if !strings.Contains(r.Header.Get("Accept"), "text/plain") {
			t.Errorf("Accept = %q, dchart needs text/plain", r.Header.Get("Accept"))
		}
		_, _ = w.Write(b)
	})

	s, err := v.Daily(context.Background(), "fpt", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if s.Symbol != "FPT" || s.Source != "vndirect" || !s.Adjusted {
		t.Errorf("series header = %+v", s)
	}
	if len(s.Bars) != 15 {
		t.Fatalf("bars = %d, want 15", len(s.Bars))
	}
	if !s.Bars[0].Date.Equal(day(2026, 9, 23)) || !s.Bars[14].Date.Equal(day(2026, 10, 13)) {
		t.Errorf("dates = %v .. %v", s.Bars[0].Date, s.Bars[14].Date)
	}
	// dchart quotes thousand VND: 62.7 is 62,700 VND.
	if got := s.Bars[6]; got.Close != 62700 || got.Open != 63000 {
		t.Errorf("bar 6 = %+v, want close 62700 open 63000", got)
	}
	if got := s.Bars[14]; got.Close != 56300 || got.Volume != 6775400 {
		t.Errorf("last bar = %+v", got)
	}
}

func TestVNDirectDailyIndexInPoints(t *testing.T) {
	v := newTestVNDirect(t, serveFixture(t, "vnd_dchart_vnindex.json"))
	s, err := v.Daily(context.Background(), "VNINDEX", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Bars[len(s.Bars)-1].Close; got != 1719.42 {
		t.Errorf("VNINDEX close = %v, want 1719.42 points", got)
	}
}

func TestVNDirectDailyNoData(t *testing.T) {
	for name, body := range map[string]string{
		"status":     `{"s":"no_data","t":[],"c":[],"o":[],"h":[],"l":[],"v":[]}`,
		"empty body": ``,
	} {
		t.Run(name, func(t *testing.T) {
			v := newTestVNDirect(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) })
			_, err := v.Daily(context.Background(), "ZZZZ", day(2026, 9, 1), testNow)
			if !errors.Is(err, ErrNoData) {
				t.Errorf("err = %v, want ErrNoData", err)
			}
		})
	}
}

func TestVNDirectRatios(t *testing.T) {
	v := newTestVNDirect(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/v4/ratios/latest" || q.Get("where") != "code:FPT,VCB" || q.Get("order") != "reportDate" ||
			!strings.HasPrefix(q.Get("filter"), "ratioCode:") || !strings.Contains(q.Get("filter"), "ROAE_TR_AVG5Q") {
			t.Errorf("unexpected request %s", r.URL)
		}
		_, _ = w.Write(fixture(t, "vnd_ratios_latest.json"))
	})

	got, err := v.Ratios(context.Background(), []string{"FPT", "vcb", "FPT"})
	if err != nil {
		t.Fatal(err)
	}
	fpt := got["FPT"]
	wantFloat(t, "FPT PE", fpt.PE, 10.76823468030994)
	wantFloat(t, "FPT PB", fpt.PB, 2.701909520423043)
	wantFloat(t, "FPT ROE", fpt.ROE, 0.2408673527274518)
	wantFloat(t, "FPT ROA", fpt.ROA, 0.12679982923465996)
	wantFloat(t, "FPT DivYield", fpt.DivYield, 0.03167268423417636)
	wantFloat(t, "FPT Beta", fpt.Beta, 0.7445541138179093)
	wantFloat(t, "FPT DE", fpt.DE, 0.45003588508124137)
	wantFloat(t, "FPT EPS", fpt.EPS, 5869.844706322774)
	wantFloat(t, "FPT BVPS", fpt.BVPS, 21133.20211813005)
	wantFloat(t, "FPT MarketCap", fpt.MarketCap, 107675048700800)
	if fpt.Source != "vndirect" || !fpt.AsOf.Equal(day(2026, 6, 30)) {
		t.Errorf("FPT source/asOf = %q %v, want vndirect 2026-06-30", fpt.Source, fpt.AsOf)
	}

	vcb := got["VCB"]
	if vcb.DE != nil {
		t.Errorf("bank DE = %v, want nil", *vcb.DE)
	}
	wantFloat(t, "VCB ROE", vcb.ROE, 0.18208992214774047)
	if vcb.NPL != nil || vcb.CASA != nil || len(vcb.NetProfitYears) != 0 {
		t.Errorf("VNDirect must not set bank extras: %+v", vcb)
	}
}

func TestParseRatiosIgnoresUnknownCodes(t *testing.T) {
	got := parseRatios([]finfoRatio{
		{Code: "ABC", RatioCode: "PRICE_TO_BOOK", Value: flexNum{1.5, true}, ReportDate: "2026-10-12"},
		{Code: "ABC", RatioCode: "SOMETHING_NEW", Value: flexNum{99, true}, ReportDate: "2020-01-01"},
		{Code: "ABC", RatioCode: "PRICE_TO_EARNINGS", ReportDate: "2019-01-01"}, // null value
	})
	f := got["ABC"]
	wantFloat(t, "PB", f.PB, 1.5)
	if f.PE != nil {
		t.Errorf("PE = %v, want nil for a null value", *f.PE)
	}
	if !f.AsOf.Equal(day(2026, 10, 12)) {
		t.Errorf("AsOf = %v; unknown codes and null values must not move it", f.AsOf)
	}
}

func TestVNDirectDividends(t *testing.T) {
	v := newTestVNDirect(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("q"); q != "code:FPT~type:DIVIDEND,STOCKDIV,KINDDIV~locale:EN_GB" {
			t.Errorf("q = %q", q)
		}
		_, _ = w.Write(fixture(t, "vnd_events_fpt.json"))
	})
	got, err := v.Dividends(context.Background(), "FPT")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("dividends = %d, want 10", len(got))
	}
	if got[0] != (Dividend{ExDate: day(2026, 9, 21), StockRatio: 0.10}) {
		t.Errorf("newest = %+v, want the KINDDIV bonus issue at ratio 0.10", got[0])
	}
	if got[1] != (Dividend{ExDate: day(2026, 5, 28), CashPerShare: 1000}) {
		t.Errorf("newest cash = %+v", got[1])
	}
	if got[9] != (Dividend{ExDate: day(2023, 7, 5), StockRatio: 0.15}) {
		t.Errorf("stock dividend = %+v, want ratio 0.15", got[9])
	}
	stock := 0
	for _, d := range got {
		if d.StockRatio > 0 && d.ExDate.After(day(2023, 10, 13)) {
			stock++
		}
	}
	if stock != 3 {
		t.Errorf("stock payouts in the last 3 years = %d, want 3 (KINDDIV)", stock)
	}
}

func TestParseFinfoEvents(t *testing.T) {
	got := parseFinfoEvents([]finfoEvent{
		{Type: "DIVIDEND", Ratio: flexNum{12, true}, EffectiveDate: "2026-07-01"},                              // ratio only
		{Type: "DIVIDEND", Dividend: flexNum{500, true}, EffectiveDate: "2027-12-31"},                          // placeholder date
		{Type: "DIVIDEND", Dividend: flexNum{700, true}, Ratio: flexNum{7, true}, EffectiveDate: "2027-01-05"}, // announced
	}, testNow)
	want := []Dividend{
		{ExDate: day(2027, 1, 5), CashPerShare: 700},
		{ExDate: day(2026, 7, 1), CashPerShare: 1200},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestVNDirectListings(t *testing.T) {
	v := newTestVNDirect(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("q"); q != "code:FPT,VCB,E1VFVN30" {
			t.Errorf("q = %q", q)
		}
		_, _ = w.Write(fixture(t, "vnd_stocks.json"))
	})
	got, err := v.Listings(context.Background(), []string{"FPT", "VCB", "E1VFVN30"})
	if err != nil {
		t.Fatal(err)
	}
	fpt := got["FPT"]
	want := Profile{Symbol: "FPT", Name: "FPT Corporation", Exchange: "HOSE", Type: "STOCK", Status: "listed", ListedDate: day(2006, 12, 13)}
	if !reflect.DeepEqual(fpt, want) {
		t.Errorf("FPT = %+v, want %+v", fpt, want)
	}
	if got["E1VFVN30"].Type != "ETF" {
		t.Errorf("ETF type = %q", got["E1VFVN30"].Type)
	}
}

func TestVNDirectIndustry(t *testing.T) {
	v := newTestVNDirect(t, serveFixture(t, "vnd_industry_fpt.json"))
	icb1, icb2, icb4, peers, err := v.Industry(context.Background(), "FPT")
	if err != nil {
		t.Fatal(err)
	}
	if icb1 != "TECHNOLOGY" || icb2 != "Technology" || icb4 != "Computer Services" {
		t.Errorf("ICB = %q / %q / %q", icb1, icb2, icb4)
	}
	if want := []string{"SBD", "CMT", "PAI", "CMG"}; !reflect.DeepEqual(peers, want) {
		t.Errorf("peers = %v, want %v", peers, want)
	}
}

func TestVNDirectIndexMembers(t *testing.T) {
	v := newTestVNDirect(t, func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("q"); q != "indexCode:VN30" {
			t.Errorf("q = %q", q)
		}
		_, _ = w.Write(fixture(t, "vnd_stocks_vn30.json"))
	})
	got, err := v.IndexMembers(context.Background(), "vn30")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 30 || got[0] != "ACB" {
		t.Errorf("members = %v", got)
	}
}
