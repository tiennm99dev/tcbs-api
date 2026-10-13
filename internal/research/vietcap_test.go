package research

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func newTestVietcap(t *testing.T, h http.HandlerFunc) *Vietcap {
	srv := newServer(t, h)
	return &Vietcap{HTTP: newTestHTTP(t), ChartURL: srv.URL, IQURL: srv.URL}
}

func TestVietcapDaily(t *testing.T) {
	v := newTestVietcap(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/chart/OHLCChart/gap-chart" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Origin") != vietcapOrigin || r.Header.Get("Referer") != vietcapOrigin {
			t.Errorf("Origin/Referer = %q / %q", r.Header.Get("Origin"), r.Header.Get("Referer"))
		}
		var body struct {
			TimeFrame string   `json:"timeFrame"`
			Symbols   []string `json:"symbols"`
			To        int64    `json:"to"`
			CountBack int      `json:"countBack"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body.TimeFrame != "ONE_DAY" || !reflect.DeepEqual(body.Symbols, []string{"FPT"}) || body.To != testNow.Unix() || body.CountBack != 520 {
			t.Errorf("body = %+v", body)
		}
		_, _ = w.Write(fixture(t, "vci_gapchart_fpt.json"))
	})

	s, err := v.Daily(context.Background(), "FPT", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if s.Source != "vietcap" || !s.Adjusted || len(s.Bars) != 5 {
		t.Fatalf("series = %+v", s)
	}
	// Vietcap quotes VND and sends t as strings.
	want := Bar{Date: day(2026, 10, 13), Open: 57100, High: 57400, Low: 56100, Close: 56300, Volume: 6775400}
	if s.Bars[4] != want {
		t.Errorf("last bar = %+v, want %+v", s.Bars[4], want)
	}
	if !s.Bars[0].Date.Equal(day(2026, 10, 7)) {
		t.Errorf("first date = %v", s.Bars[0].Date)
	}
}

func TestVietcapDailyEmpty(t *testing.T) {
	v := newTestVietcap(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`[]`)) })
	if _, err := v.Daily(context.Background(), "ZZZZ", day(2026, 9, 1), testNow); err == nil {
		t.Error("want an error for an empty answer")
	}
}

func TestVietcapBankRatios(t *testing.T) {
	v := newTestVietcap(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/iq-insight-service/v1/company/VCB/statistics-financial" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write(fixture(t, "vci_statfin_vcb.json"))
	})
	got, err := v.BankRatios(context.Background(), "VCB")
	if err != nil {
		t.Fatal(err)
	}
	// The latest RATIO_TTM row is 2026 Q2; the RATIO_YEAR row is ignored.
	if !got.AsOf.Equal(day(2026, 6, 30)) {
		t.Errorf("AsOf = %v, want 2026-06-30", got.AsOf)
	}
	wantFloat(t, "NPL", got.NPL, 0.0061066523)
	wantFloat(t, "CASA", got.CASA, 0.3392956361)
	wantFloat(t, "PE", got.PE, 11.3753921051)
	wantFloat(t, "PB", got.PB, 1.9072314564)
	wantFloat(t, "ROE", got.ROE, 0.1791575023)
	wantFloat(t, "DE", got.DE, 9.6943344266)
}

func TestVietcapDividends(t *testing.T) {
	v := newTestVietcap(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("ticker") != "FPT" || q.Get("fromDate") != "20231013" || q.Get("toDate") != "20271013" || q.Get("eventCode") != "DIV,ISS" {
			t.Errorf("query = %v", q)
		}
		_, _ = w.Write(fixture(t, "vci_events_fpt.json"))
	})
	got, err := v.Dividends(context.Background(), "FPT", day(2023, 10, 13), testNow.AddDate(1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 7 {
		t.Fatalf("events = %d, want 7", len(got))
	}
	if got[0] != (Dividend{ExDate: day(2026, 9, 21), StockRatio: 0.1}) {
		t.Errorf("newest = %+v", got[0])
	}
	var cash float64
	for _, d := range got {
		cash += d.CashPerShare
	}
	if cash != 3000 {
		t.Errorf("cash total = %v, want 3000", cash)
	}
}

func TestQuarterEnd(t *testing.T) {
	for _, tc := range []struct {
		y, q int
		want string
	}{{2026, 1, "2026-03-31"}, {2026, 2, "2026-06-30"}, {2025, 4, "2025-12-31"}, {2025, 5, "0001-01-01"}} {
		if got := quarterEnd(tc.y, tc.q).Format("2006-01-02"); got != tc.want {
			t.Errorf("quarterEnd(%d, %d) = %s, want %s", tc.y, tc.q, got, tc.want)
		}
	}
}
