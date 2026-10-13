package research

import (
	"math"
	"testing"
	"time"
)

// weekdays returns n consecutive weekday dates ending on 2026-10-13.
func weekdays(n int) []time.Time {
	out := make([]time.Time, n)
	d := day(2026, 10, 13)
	for i := n - 1; i >= 0; i-- {
		for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			d = d.AddDate(0, 0, -1)
		}
		out[i] = d
		d = d.AddDate(0, 0, -1)
	}
	return out
}

func seriesOf(closes ...float64) Series {
	dates := weekdays(len(closes))
	s := Series{Symbol: "T", Bars: make([]Bar, len(closes))}
	for i, c := range closes {
		s.Bars[i] = Bar{Date: dates[i], Open: c, High: c, Low: c, Close: c, Volume: 1000}
	}
	return s
}

// indexAndDouble builds an index and a stock whose log return is exactly
// twice the index's every day.
func indexAndDouble(n int) (index, stock Series) {
	dates := weekdays(n)
	index.Bars = make([]Bar, n)
	stock.Bars = make([]Bar, n)
	cum := 0.0
	for i := 0; i < n; i++ {
		if i > 0 {
			cum += 0.01*math.Sin(float64(i)*1.3) + 0.004*math.Cos(float64(i)*0.7)
		}
		index.Bars[i] = Bar{Date: dates[i], Close: 1000 * math.Exp(cum)}
		stock.Bars[i] = Bar{Date: dates[i], Close: 50000 * math.Exp(2*cum)}
	}
	return index, stock
}

func TestLogReturns(t *testing.T) {
	s := seriesOf(100, 110, 0, 121)
	r := LogReturns(s)
	if len(r) != 1 || !near(r[s.Bars[1].Date], math.Log(1.1)) {
		t.Errorf("got %v", r)
	}
}

func TestVolatility(t *testing.T) {
	// Closes alternate between 100 and 100·e^0.02, so returns are ±0.02 with
	// mean 0 over an even count n: sample stdev = 0.02·sqrt(n/(n−1)).
	closes := make([]float64, 131)
	for i := range closes {
		closes[i] = 100
		if i%2 == 1 {
			closes[i] = 100 * math.Exp(0.02)
		}
	}
	got, ok := Volatility(seriesOf(closes...), 250)
	want := 0.02 * math.Sqrt(130.0/129.0) * math.Sqrt(250)
	if !ok || !near(got, want) {
		t.Errorf("Volatility = %v, %v; want %v", got, ok, want)
	}

	// Only the last 120 returns are used when sessions = 120.
	got, ok = Volatility(seriesOf(closes...), 120)
	want = 0.02 * math.Sqrt(120.0/119.0) * math.Sqrt(250)
	if !ok || !near(got, want) {
		t.Errorf("Volatility(120) = %v, %v; want %v", got, ok, want)
	}

	if _, ok := Volatility(seriesOf(closes[:120]...), 250); ok {
		t.Error("119 returns must not be enough")
	}
}

func TestBetaPerfectlyCorrelatedDouble(t *testing.T) {
	index, stock := indexAndDouble(300)
	beta, r2, ok := Beta(stock, index, 250)
	if !ok || math.Abs(beta-2) > 1e-9 || math.Abs(r2-1) > 1e-9 {
		t.Errorf("Beta = %v, R² = %v, ok = %v; want 2, 1", beta, r2, ok)
	}

	// A missing stock bar must not misalign the returns.
	stock.Bars = append(stock.Bars[:100:100], stock.Bars[101:]...)
	beta, _, ok = Beta(stock, index, 250)
	if !ok || math.Abs(beta-2) > 1e-9 {
		t.Errorf("Beta with a gap = %v, %v; want 2", beta, ok)
	}

	if _, _, ok := Beta(Series{Bars: stock.Bars[:50]}, index, 250); ok {
		t.Error("50 bars must not be enough")
	}
}

func TestWeeklyBeta(t *testing.T) {
	index, stock := indexAndDouble(520)
	beta, ok := WeeklyBeta(stock, index, 104)
	if !ok || math.Abs(beta-2) > 1e-9 {
		t.Errorf("WeeklyBeta = %v, %v; want 2", beta, ok)
	}
	if _, ok := WeeklyBeta(Series{Bars: stock.Bars[:100]}, index, 104); ok {
		t.Error("20 weeks must not be enough")
	}
}

func TestWeeklyClosesUseLastDayOfWeek(t *testing.T) {
	s := Series{Bars: []Bar{
		{Date: day(2026, 10, 5), Close: 1},  // Monday
		{Date: day(2026, 10, 8), Close: 2},  // Thursday (Friday is a holiday)
		{Date: day(2026, 10, 12), Close: 3}, // next Monday
		{Date: day(2026, 10, 16), Close: 4}, // Friday
	}}
	got := weeklyCloses(s)
	if len(got) != 2 || got[0].close != 2 || got[1].close != 4 || !got[1].key.Equal(day(2026, 10, 12)) {
		t.Errorf("got %+v", got)
	}
}

func TestMaxDrawdown(t *testing.T) {
	got, ok := MaxDrawdown(seriesOf(100, 120, 90, 110, 60, 80), 6)
	if !ok || !near(got, 0.5) {
		t.Errorf("MDD = %v, %v; want 0.5", got, ok)
	}
	// Over the last 3 bars only (110, 60, 80): 1 − 60/110.
	got, ok = MaxDrawdown(seriesOf(100, 120, 90, 110, 60, 80), 3)
	if !ok || !near(got, 1-60.0/110) {
		t.Errorf("MDD(3) = %v, %v", got, ok)
	}
	if _, ok := MaxDrawdown(seriesOf(100, 90), 250); ok {
		t.Error("2 bars must not be enough for a 1-year drawdown")
	}
}

func TestADTV(t *testing.T) {
	s := Series{Bars: []Bar{
		{Close: 999, Volume: 999},
		{Close: 10000, Volume: 100},
		{Close: 20000, Volume: 300},
	}}
	if got := ADTV(s, 2); got != (1e6+6e6)/2 {
		t.Errorf("ADTV = %v", got)
	}
	if got := ADTV(Series{}, 60); got != 0 {
		t.Errorf("empty ADTV = %v", got)
	}
}

func TestTrailingCashYield(t *testing.T) {
	now := testNow
	divs := []Dividend{
		{ExDate: day(2026, 11, 20), CashPerShare: 9999}, // not yet ex
		{ExDate: day(2026, 9, 1), StockRatio: 0.15},     // stock dividend, never counted
		{ExDate: day(2026, 8, 1), CashPerShare: 1000},
		{ExDate: day(2026, 5, 1), CashPerShare: 500},
		{ExDate: day(2025, 8, 1), CashPerShare: 800},
		{ExDate: day(2024, 8, 1), CashPerShare: 700},
	}
	y, years := TrailingCashYield(divs, 30000, now)
	if !near(y, 1500.0/30000) || years != 3 {
		t.Errorf("yield = %v, years = %d; want 0.05, 3", y, years)
	}

	// No payment in the middle year.
	y, years = TrailingCashYield([]Dividend{divs[2], divs[5]}, 20000, now)
	if !near(y, 0.05) || years != 2 {
		t.Errorf("yield = %v, years = %d; want 0.05, 2", y, years)
	}

	if y, years := TrailingCashYield(divs, 0, now); y != 0 || years != 3 {
		t.Errorf("zero price: %v, %d", y, years)
	}
}

func TestAdjustJumpsSplit(t *testing.T) {
	// A 2:1 split between the 3rd and 4th bars halves the price.
	s := seriesOf(100, 102, 104, 52, 53)
	got, adjusted := AdjustJumps(s, 0.07)
	if !adjusted {
		t.Fatal("split not detected")
	}
	want := []float64{50, 51, 52, 52, 53}
	for i, b := range got.Bars {
		if !near(b.Close, want[i]) {
			t.Errorf("bar %d close = %v, want %v", i, b.Close, want[i])
		}
	}
	if got.Bars[0].Volume != 2000 || got.Bars[4].Volume != 1000 {
		t.Errorf("volumes = %v, %v; want earlier volumes doubled", got.Bars[0].Volume, got.Bars[4].Volume)
	}
	if s.Bars[0].Close != 100 {
		t.Error("input series was modified")
	}

	// A limit-down move (−7%) stays within band + 1pp.
	if _, adjusted := AdjustJumps(seriesOf(100, 93, 94), 0.07); adjusted {
		t.Error("a move within the band is not a jump")
	}
	// An already adjusted series is returned as is.
	s.Adjusted = true
	if _, adjusted := AdjustJumps(s, 0.07); adjusted {
		t.Error("adjusted series must not be touched")
	}
}

func TestPriceBand(t *testing.T) {
	for ex, want := range map[string]float64{"HOSE": 0.07, "hnx": 0.10, "UPCOM": 0.15, "": 0.15} {
		if got := PriceBand(ex); got != want {
			t.Errorf("PriceBand(%q) = %v, want %v", ex, got, want)
		}
	}
}

func TestGapAndStaleSessions(t *testing.T) {
	index := seriesOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	stock := Series{}
	for i, b := range index.Bars {
		if i == 0 || i == 4 || i >= 8 {
			continue // not listed yet, one gap, then stale for two sessions
		}
		stock.Bars = append(stock.Bars, b)
	}
	if got := GapSessions(stock, index, 60); got != 1 {
		t.Errorf("GapSessions = %d, want 1", got)
	}
	if got := GapSessions(stock, index, 3); got != 0 {
		t.Errorf("GapSessions(last 3) = %d, want 0", got)
	}
	if got := StaleSessions(stock, index); got != 2 {
		t.Errorf("StaleSessions = %d, want 2", got)
	}
	if got := StaleSessions(Series{}, index); got != 10 {
		t.Errorf("StaleSessions(empty) = %d, want 10", got)
	}
}

func TestRangeAndReturn(t *testing.T) {
	s := seriesOf(100, 90, 130, 120)
	s.Bars[2].High = 135
	s.Bars[1].Low = 0 // missing low falls back to the close
	if lo, hi := Range(s, 250); lo != 90 || hi != 135 {
		t.Errorf("Range = %v, %v", lo, hi)
	}
	if lo, hi := Range(s, 1); lo != 120 || hi != 120 {
		t.Errorf("Range(1) = %v, %v", lo, hi)
	}
	if r, ok := Return(s, 3); !ok || !near(r, 0.2) {
		t.Errorf("Return = %v, %v; want 0.2", r, ok)
	}
	if _, ok := Return(s, 4); ok {
		t.Error("Return needs sessions+1 bars")
	}
}
