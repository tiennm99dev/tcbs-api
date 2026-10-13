package advisor

import (
	"math"
	"strings"
	"testing"
	"time"
)

// testNow is a Monday 20:00 in Vietnam, outside trading hours.
var testNow = time.Date(2026, 10, 12, 13, 0, 0, 0, time.UTC)

func f(v float64) *float64 { return &v }

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func testConfig() Config {
	c := Default()
	c.Now = testNow
	return c
}

func hasText(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// perfectInputs returns a general stock whose every metric scores 100.
func perfectInputs(sym string) Inputs {
	return Inputs{
		Symbol: sym, Sector: SectorGeneral, Exchange: "HOSE", Price: 50000,
		VOL: f(0.15), Beta: f(0.8), BetaR2: f(0.5), MDD: f(0.10), ADTV: f(5e10),
		ROE: f(0.25), DE: f(0.2), PE: f(10), PB: f(1.2), DY: f(0.07),
		DivPaidYears: 3, Status: "listed", Type: "STOCK", HistoryReturns: 400,
		ProfitYears: []float64{10, 10, 10, 10},
		Listed:      time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// perfectBank returns a bank whose every available metric scores 100.
func perfectBank(sym string) Inputs {
	in := perfectInputs(sym)
	in.Sector = SectorBank
	in.DE = nil
	in.ROE = f(0.26) // fairPB 2.0
	in.PB = f(1.5)   // PB/fairPB 0.75
	in.PE = f(8)
	in.NPL = f(0.008)
	in.CASA = f(0.40)
	return in
}

func metric(t *testing.T, s Score, name string) MetricScore {
	t.Helper()
	for _, m := range s.Metrics {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("metric %s not in %+v", name, s.Metrics)
	return MetricScore{}
}
