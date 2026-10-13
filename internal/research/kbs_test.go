package research

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

func newTestKBS(t *testing.T, h http.HandlerFunc) *KBS {
	srv := newServer(t, h)
	return &KBS{HTTP: newTestHTTP(t), BaseURL: srv.URL}
}

func TestKBSDaily(t *testing.T) {
	k := newTestKBS(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/investment/stocks/FPT/data_day" || q.Get("sdate") != "01-09-2026" || q.Get("edate") != "13-10-2026" {
			t.Errorf("unexpected request %s", r.URL)
		}
		_, _ = w.Write(fixture(t, "kbs_dataday_fpt.json"))
	})
	s, err := k.Daily(context.Background(), "FPT", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	if s.Source != "kbs" || s.Adjusted || len(s.Bars) != 4 {
		t.Fatalf("series = %+v", s)
	}
	// Rows come newest first; the series is oldest first.
	if !s.Bars[0].Date.Equal(day(2026, 10, 8)) || !s.Bars[3].Date.Equal(day(2026, 10, 13)) {
		t.Errorf("dates = %v .. %v", s.Bars[0].Date, s.Bars[3].Date)
	}
	if s.Bars[3].Close != 56300 || s.Bars[3].Volume != 6775400 {
		t.Errorf("last bar = %+v", s.Bars[3])
	}
}

func TestKBSDailyIndexStringRow(t *testing.T) {
	k := newTestKBS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/investment/index/VNINDEX/data_day" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write(fixture(t, "kbs_dataday_vnindex.json"))
	})
	s, err := k.Daily(context.Background(), "VNINDEX", day(2026, 9, 1), testNow)
	if err != nil {
		t.Fatal(err)
	}
	// The newest row's values are strings in the fixture.
	want := Bar{Date: day(2026, 10, 13), Open: 1718.84, High: 1726.35, Low: 1710.95, Close: 1719.42, Volume: 560876017}
	if got := s.Bars[len(s.Bars)-1]; got != want {
		t.Errorf("last bar = %+v, want %+v", got, want)
	}
}

func TestKBSNetProfitYears(t *testing.T) {
	for _, tc := range []struct {
		symbol, file string
		want         []float64
	}{
		// Non-bank: the parent-company row.
		{"FPT", "kbs_kqkd_fpt.json", []float64{5310108591, 6465190433, 7856767812, 9376127630}},
		// Bank: no parent row, so net profit after tax.
		{"VCB", "kbs_kqkd_vcb.json", []float64{29919054000, 33054448000, 33853117000, 35197940000}},
	} {
		t.Run(tc.symbol, func(t *testing.T) {
			k := newTestKBS(t, func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if r.URL.Path != "/investment/stock/finance-info/"+tc.symbol || q.Get("type") != "KQKD" || q.Get("pageSize") != "4" || q.Get("termtype") != "1" {
					t.Errorf("unexpected request %s", r.URL)
				}
				_, _ = w.Write(fixture(t, tc.file))
			})
			got, err := k.NetProfitYears(context.Background(), tc.symbol)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseNetProfitTooFewYears(t *testing.T) {
	var r kbsKQKD
	raw := `{"Head":[{"YearPeriod":2025},{"YearPeriod":2024},{"YearPeriod":2023}],
		"Content":{"g":[{"NameEn":"18. Net profit after tax","Value1":"10","Value2":-5,"Value3":null}]}}`
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if got := parseNetProfit(r); len(got) != 0 {
		t.Errorf("got %v, want empty for 2 years", got)
	}
	r.Content = map[string][]kbsNorm{"g": {{NameEn: "Revenue"}}}
	if got := parseNetProfit(r); len(got) != 0 {
		t.Errorf("got %v, want empty without a profit row", got)
	}
}

func TestKBSIndexMembers(t *testing.T) {
	k := newTestKBS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/investment/index/100/stocks" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write(fixture(t, "kbs_index_100.json"))
	})
	got, err := k.IndexMembers(context.Background(), "100")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 {
		t.Errorf("members = %d, want 100", len(got))
	}
}
