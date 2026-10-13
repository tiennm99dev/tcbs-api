package advisor

import (
	"testing"
	"time"
)

func TestMarketPhase(t *testing.T) {
	vn := func(y int, m time.Month, d, h, min int) time.Time {
		return time.Date(y, m, d, h, min, 0, 0, vnTZ).UTC()
	}
	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"before open", vn(2026, 10, 13, 8, 59), PhaseLastClose},
		{"open", vn(2026, 10, 13, 9, 0), PhaseIntraday},
		{"midday", vn(2026, 10, 13, 11, 45), PhaseIntraday},
		{"last minute", vn(2026, 10, 13, 14, 59), PhaseIntraday},
		{"close", vn(2026, 10, 13, 15, 0), PhaseLastClose},
		{"evening", vn(2026, 10, 13, 20, 0), PhaseLastClose},
		{"saturday", vn(2026, 10, 17, 10, 0), PhaseLastClose},
		{"sunday", vn(2026, 10, 18, 10, 0), PhaseLastClose},
		{"utc date differs", time.Date(2026, 10, 12, 2, 30, 0, 0, time.UTC), PhaseIntraday},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MarketPhase(tc.now); got != tc.want {
				t.Errorf("MarketPhase(%s) = %q, want %q", tc.now, got, tc.want)
			}
		})
	}
}

func TestLots(t *testing.T) {
	cases := []struct{ qty, floor, odd int64 }{
		{-5, 0, 0}, {0, 0, 0}, {1, 0, 1}, {99, 0, 99}, {100, 100, 0}, {150, 100, 50}, {1299, 1200, 99},
	}
	for _, tc := range cases {
		if got := FloorLot(tc.qty); got != tc.floor {
			t.Errorf("FloorLot(%d) = %d, want %d", tc.qty, got, tc.floor)
		}
		if got := OddLot(tc.qty); got != tc.odd {
			t.Errorf("OddLot(%d) = %d, want %d", tc.qty, got, tc.odd)
		}
	}
}

func TestCostsAndBreakeven(t *testing.T) {
	if got := BuyCost(100, 50000, 0.001); got != 5005000 {
		t.Errorf("BuyCost = %d", got)
	}
	if got := SellProceeds(100, 50000, 0.001, 0.001); got != 4990000 {
		t.Errorf("SellProceeds = %d", got)
	}
	cases := []struct {
		avg       int64
		fee, tax  float64
		breakeven int64
	}{
		{0, 0.001, 0.001, 0},
		{-1, 0.001, 0.001, 0},
		{10000, 0, 0, 10000},
		{10000, 0.001, 0.001, 10031},    // 10000 × 1.001 / 0.998 = 10030.06
		{100000, 0.0015, 0.001, 100402}, // 100401.0025 rounds up
	}
	for _, tc := range cases {
		if got := Breakeven(tc.avg, tc.fee, tc.tax); got != tc.breakeven {
			t.Errorf("Breakeven(%d, %v, %v) = %d, want %d", tc.avg, tc.fee, tc.tax, got, tc.breakeven)
		}
	}
}
