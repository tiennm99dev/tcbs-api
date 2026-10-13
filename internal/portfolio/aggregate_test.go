package portfolio

import (
	"math"
	"strings"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func hasNote(notes []string, sub string) bool {
	for _, n := range notes {
		if strings.Contains(n, sub) {
			return true
		}
	}
	return false
}

func position(t *testing.T, s Summary, sym string) Position {
	t.Helper()
	for _, p := range s.Positions {
		if p.Symbol == sym {
			return p
		}
	}
	t.Fatalf("no position for %s in %+v", sym, s.Positions)
	return Position{}
}

func TestAggregate(t *testing.T) {
	cases := []struct {
		name     string
		snap     Snapshot
		prices   map[string]Quote
		bankCash bool
		check    func(t *testing.T, s Summary)
	}{
		{
			name: "same symbol in two accounts",
			snap: Snapshot{Accounts: []Account{
				{AccountNo: "0001N", Holdings: []Holding{{Symbol: "FPT", TotalQty: 1000, SellableQty: 800, ReceivingQty: 200, AvgCost: 95500, RefPrice: 120000, SecType: "001"}}},
				{AccountNo: "0001G", Holdings: []Holding{{Symbol: "fpt", TotalQty: 500, SellableQty: 500, AvgCost: 100000, RefPrice: 120000, SecType: "001"}}},
			}},
			prices: map[string]Quote{"FPT": {Price: 121000, Source: SourceTCBSMatch, Intraday: true}},
			check: func(t *testing.T, s Summary) {
				if len(s.Positions) != 1 {
					t.Fatalf("expected one FPT position, got %+v", s.Positions)
				}
				p := s.Positions[0]
				if p.Qty != 1500 || p.Sellable != 1300 || p.Receiving != 200 {
					t.Errorf("quantities: %+v", p)
				}
				// (1000*95500 + 500*100000) / 1500 = 97000
				if p.AvgCost != 97000 || p.CostValue != 145500000 {
					t.Errorf("avg cost %d cost value %d", p.AvgCost, p.CostValue)
				}
				if p.Price != 121000 || p.PriceSource != SourceTCBSMatch || p.MarketValue != 181500000 {
					t.Errorf("price/MV: %+v", p)
				}
				if p.UnrealizedPnL != 36000000 || !approx(p.PnLPct, 36000000.0/145500000.0) {
					t.Errorf("pnl: %d %v", p.UnrealizedPnL, p.PnLPct)
				}
				if s.NAV != 181500000 || !approx(p.Weight, 1) || !approx(s.HHI, 1) || !approx(s.EffectiveN, 1) {
					t.Errorf("NAV %d weight %v HHI %v N %v", s.NAV, p.Weight, s.HHI, s.EffectiveN)
				}
				if !p.Scored || len(s.Notes) != 0 {
					t.Errorf("scored %v notes %v", p.Scored, s.Notes)
				}
			},
		},
		{
			name: "margin debt",
			snap: Snapshot{Accounts: []Account{{
				AccountNo: "0001G",
				Cash:      Cash{Balance: 10000000, CustodyFeeDebt: 50000},
				Debt:      Debt{Outstanding: 30000000, AccruedInterest: 400000, FeeDebt: 50000, RTT: 180, RiskCode: "SAFE"},
				Holdings:  []Holding{{Symbol: "HPG", TotalQty: 4000, AvgCost: 25000, RefPrice: 27000}},
			}}},
			check: func(t *testing.T, s Summary) {
				if s.Debt != 30500000 {
					t.Errorf("debt %d", s.Debt)
				}
				// 108,000,000 + 10,000,000 - 30,500,000
				if s.NAV != 87500000 {
					t.Errorf("NAV %d", s.NAV)
				}
				if !approx(s.MarginRatio, 30500000.0/87500000.0) || !approx(s.CashRatio, 10000000.0/87500000.0) {
					t.Errorf("ratios cash %v margin %v", s.CashRatio, s.MarginRatio)
				}
				if p := position(t, s, "HPG"); p.PriceSource != SourceRef || p.Price != 27000 {
					t.Errorf("ref fallback: %+v", p)
				}
				if !hasNote(s.Notes, "HPG: no current quote") {
					t.Errorf("missing ref-price note: %v", s.Notes)
				}
			},
		},
		{
			name: "IA bank cash excluded by default",
			snap: Snapshot{Accounts: []Account{{Cash: Cash{Balance: 1000000, BankLinked: 5000000}}}},
			check: func(t *testing.T, s Summary) {
				if s.BankCash != 5000000 || s.BankCashInNAV || s.Cash != 1000000 || s.NAV != 1000000 {
					t.Errorf("bank %d inNAV %v cash %d NAV %d", s.BankCash, s.BankCashInNAV, s.Cash, s.NAV)
				}
				if !approx(s.CashRatio, 1) {
					t.Errorf("cash ratio %v", s.CashRatio)
				}
			},
		},
		{
			name:     "IA bank cash included with the flag",
			snap:     Snapshot{Accounts: []Account{{Cash: Cash{Balance: 1000000, BankLinked: 5000000}}}},
			bankCash: true,
			check: func(t *testing.T, s Summary) {
				if s.BankCash != 5000000 || !s.BankCashInNAV || s.Cash != 6000000 || s.NAV != 6000000 {
					t.Errorf("bank %d inNAV %v cash %d NAV %d", s.BankCash, s.BankCashInNAV, s.Cash, s.NAV)
				}
			},
		},
		{
			name: "thousand-VND prices normalised in the snapshot",
			snap: Snapshot{Accounts: []Account{{Holdings: []Holding{
				{Symbol: "FPT", TotalQty: 100, AvgCost: toVND(95.5, 1000), RefPrice: toVND(120.3, 1000)},
			}}}},
			check: func(t *testing.T, s Summary) {
				p := position(t, s, "FPT")
				if p.AvgCost != 95500 || p.Price != 120300 || p.MarketValue != 12030000 {
					t.Errorf("normalised: %+v", p)
				}
			},
		},
		{
			name: "pending stock dividend and receivable stay out of NAV",
			snap: Snapshot{Accounts: []Account{{
				Cash:     Cash{Balance: 2000000, Receivable: 7000000, IncomingDividends: 300000},
				Holdings: []Holding{{Symbol: "VCB", TotalQty: 100, PendingStockDiv: 30, PendingCashDiv: 999999, AvgCost: 80000, RefPrice: 90000}},
			}}},
			prices: map[string]Quote{"VCB": {Price: 90000, Source: "vndirect"}},
			check: func(t *testing.T, s Summary) {
				if s.ReceivableCash != 7000000 || s.PendingStockDivValue != 2700000 {
					t.Errorf("receivable %d pending %d", s.ReceivableCash, s.PendingStockDivValue)
				}
				// 9,000,000 stock + 2,000,000 cash + 300,000 incoming dividends
				if s.NAV != 11300000 {
					t.Errorf("NAV %d; receivable, pending shares and PendingCashDiv must stay out", s.NAV)
				}
				if p := position(t, s, "VCB"); p.PendingStockDiv != 30 || p.PriceSource != "vndirect" {
					t.Errorf("position: %+v", p)
				}
				if !approx(s.CashRatio, 2000000.0/11300000.0) {
					t.Errorf("cash ratio %v excludes receivable and dividends", s.CashRatio)
				}
			},
		},
		{
			name: "zero cost price leaves P&L unknown",
			snap: Snapshot{Accounts: []Account{
				{Holdings: []Holding{{Symbol: "VNM", TotalQty: 200, AvgCost: 0, RefPrice: 61200}}},
				{Holdings: []Holding{{Symbol: "VNM", TotalQty: 100, AvgCost: 70000, RefPrice: 61200}}},
			}},
			prices: map[string]Quote{"VNM": {Price: 61200, Source: SourceRef}},
			check: func(t *testing.T, s Summary) {
				p := position(t, s, "VNM")
				if p.MarketValue != 18360000 {
					t.Errorf("MV %d", p.MarketValue)
				}
				if p.AvgCost != 0 || p.CostValue != 0 || p.UnrealizedPnL != 0 || p.PnLPct != 0 {
					t.Errorf("P&L should be unknown: %+v", p)
				}
				if !hasNote(s.Notes, "VNM: average cost unknown") {
					t.Errorf("missing unknown-cost note: %v", s.Notes)
				}
			},
		},
		{
			name: "ETF is valued but not scored",
			snap: Snapshot{Accounts: []Account{{Holdings: []Holding{
				{Symbol: "E1VFVN30", TotalQty: 250, AvgCost: 25000, RefPrice: 26000, SecType: "008"},
				{Symbol: "FPT", TotalQty: 300, AvgCost: 100000, RefPrice: 120000, SecType: "001"},
			}}}},
			check: func(t *testing.T, s Summary) {
				etf := position(t, s, "E1VFVN30")
				if etf.Scored || etf.SecType != "008" || etf.MarketValue != 6500000 {
					t.Errorf("ETF: %+v", etf)
				}
				if !position(t, s, "FPT").Scored {
					t.Error("FPT should be scored")
				}
				if s.StockValue != 42500000 || s.NAV != 42500000 {
					t.Errorf("ETF must count in NAV: stock %d NAV %d", s.StockValue, s.NAV)
				}
				if !hasNote(s.Notes, "E1VFVN30: fund/warrant") {
					t.Errorf("notes: %v", s.Notes)
				}
				// Sorted by MV desc.
				if s.Positions[0].Symbol != "FPT" {
					t.Errorf("order: %+v", s.Positions)
				}
				w1, w2 := 36000000.0/42500000.0, 6500000.0/42500000.0
				if !approx(s.HHI, w1*w1+w2*w2) || !approx(s.EffectiveN, 1/(w1*w1+w2*w2)) {
					t.Errorf("HHI %v N %v", s.HHI, s.EffectiveN)
				}
			},
		},
		{
			name: "no price at all",
			snap: Snapshot{Accounts: []Account{{Cash: Cash{Balance: 100}, Holdings: []Holding{{Symbol: "XYZ", TotalQty: 100, AvgCost: 1000}}}}},
			check: func(t *testing.T, s Summary) {
				if p := position(t, s, "XYZ"); p.MarketValue != 0 {
					t.Errorf("MV %d", p.MarketValue)
				}
				if !hasNote(s.Notes, "XYZ: no price available") || s.HHI != 0 || s.EffectiveN != 0 {
					t.Errorf("notes %v HHI %v", s.Notes, s.HHI)
				}
			},
		},
		{
			name: "non-positive NAV",
			snap: Snapshot{Accounts: []Account{{Debt: Debt{Outstanding: 100}}}},
			check: func(t *testing.T, s Summary) {
				if s.NAV != -100 || s.CashRatio != 0 || s.MarginRatio != 0 || !hasNote(s.Notes, "NAV is not positive") {
					t.Errorf("summary: %+v", s)
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.check(t, Aggregate(c.snap, c.prices, c.bankCash))
		})
	}
}
