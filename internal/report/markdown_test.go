package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/miti99dev/stock-advisor/internal/advisor"
	"github.com/miti99dev/stock-advisor/internal/portfolio"
	"github.com/miti99dev/stock-advisor/internal/research"
)

func f(v float64) *float64 { return &v }

func day0(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// sampleData covers every section: a scored holding with a trim, an
// invalid score, a fund, a candidate, warnings and sources.
func sampleData() Data {
	cfg := advisor.Default()
	fpt := advisor.Score{
		Symbol: "FPT", Sector: advisor.SectorGeneral, Value: 81, Valid: true,
		Tier: advisor.TierStable, Confidence: advisor.ConfidenceHigh, Coverage: 90,
		Pillars: map[string]float64{advisor.PillarPrice: 70.4, advisor.PillarFundamentals: 93.6, advisor.PillarValuation: 81.3},
		Metrics: []advisor.MetricScore{
			{Name: advisor.MetricVOL, Pillar: advisor.PillarPrice, Raw: 0.319, Score: 70.3, Weight: 15, Contribution: 10.5},
			{Name: advisor.MetricLIQ, Pillar: advisor.PillarPrice, Raw: 3.8e11, Score: 100, Weight: 5, Contribution: 5},
			{Name: advisor.MetricEarnCons, Pillar: advisor.PillarFundamentals, Dropped: true, Note: "EARN_CONS unavailable"},
			{Name: advisor.MetricDY, Pillar: advisor.PillarValuation, Raw: 0.036, Score: 69.2, Weight: 10, Contribution: 6.9, Note: "+10: cash paid in each of the last 3 years"},
		},
		Notes: []string{"EARN_CONS unavailable: weight redistributed within fundamentals"},
	}
	hpg := advisor.Score{
		Symbol: "HPG", Sector: advisor.SectorGeneral, Tier: advisor.TierInsufficient,
		Confidence: advisor.ConfidenceLow, Coverage: 40, Pillars: map[string]float64{},
		RedFlags: []string{"ADTV 0.5bn VND < 1bn VND (hard to exit)"},
	}
	sum := portfolio.Summary{
		Positions: []portfolio.Position{
			{Symbol: "FPT", Qty: 1537, Sellable: 1437, Receiving: 100, PendingStockDiv: 225, AvgCost: 100000, Price: 120300,
				PriceSource: "vndirect", MarketValue: 184901100, CostValue: 153700000, PnLPct: 0.203, Weight: 0.52, Scored: true},
			{Symbol: "HPG", Qty: 1000, Sellable: 1000, Price: 26950, PriceSource: portfolio.SourceRef, MarketValue: 26950000, Weight: 0.076, Scored: true},
			{Symbol: "E1VFVN30", Qty: 2000, Sellable: 2000, AvgCost: 24100, Price: 26400, PriceSource: "tcbs-match",
				MarketValue: 52800000, PnLPct: 0.095, Weight: 0.148},
		},
		StockValue: 264651100, Cash: 90000000, ReceivableCash: 12000000, Debt: 0, NAV: 354651100,
		IncomingDividends: 450000, BankCash: 10000000, PendingStockDivValue: 27067500,
		CashRatio: 0.254, MarginRatio: 0, HHI: 0.3, EffectiveN: 3.33,
		Notes: []string{"HPG: average cost unknown (gifted or transferred shares)"},
	}
	return Data{
		GeneratedAt: time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC),
		Phase:       advisor.PhaseLastClose,
		Accounts:    []Account{{No: "0000001N", Type: "normal"}, {No: "0000001G", Type: "margin"}},
		Summary:     sum,
		Advice: advisor.PortfolioAdvice{
			Messages: []string{"Portfolio concentrated: effectiveN 3.3 < 4."},
			Holdings: []advisor.Advice{
				{Symbol: "FPT", Action: advisor.ActionTrim, Rule: "TRIM: weight 52.0% > hard cap 25%; trim to 20%",
					Reason: "sell 9 lots (900 shares)", Secondary: []string{"ADD blocked by the trim"},
					Lots: 9, Qty: 900, EstCash: 108053820, Breakeven: 100301, OddLot: 37},
				{Symbol: "HPG", Action: advisor.ActionReview, Rule: "REVIEW: red flag | ADTV 0.5bn VND < 1bn VND (hard to exit)"},
				{Symbol: "E1VFVN30", Action: advisor.ActionHold, Rule: "HOLD: not scored (fund/warrant)", Breakeven: 24173},
			},
			SectorWeights: map[advisor.Sector]float64{advisor.SectorGeneral: 0.596, advisor.SectorFund: 0.148},
		},
		Scores: map[string]advisor.Score{"FPT": fpt, "HPG": hpg},
		Candidates: []advisor.Candidate{{
			Score: advisor.Score{Symbol: "VNM", Sector: advisor.SectorGeneral, Value: 91, Valid: true, Tier: advisor.TierStable,
				Metrics: []advisor.MetricScore{{Name: advisor.MetricBeta, Raw: 0.49}}},
			Inputs:  advisor.Inputs{Symbol: "VNM", Price: 58000, VOL: f(0.291), Beta: f(0.6), DY: f(0.081), ROE: f(0.304), PE: f(11.13), PB: f(3.83)},
			Reasons: []string{"DE 0.24 (score 100)", "ROE 30.4% (score 100)"},
			LotCost: 5805800, MaxAlloc: 13738003,
		}},
		CandidateNote: advisor.NoteWatchlistOnly,
		Universe:      "VN30 (30 stocks)",
		Sources: []SourceStamp{
			{Name: "offline holdings file", Kind: "holdings, cash, margin", AsOf: time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)},
			{Name: "vndirect", Kind: "daily bars (latest bar)", AsOf: day0(2026, 10, 13)},
		},
		Config:   cfg,
		Warnings: []string{"ZZZ: no listing data", "ABC: no daily bars from any source", "ZZZ: no listing data"},
		Market:   MarketContext{IndexClose: 1719.42, Return1Y: 0.002, Vol1Y: 0.205, AsOf: day0(2026, 10, 13), HasReturn: true, HasVol: true},
		Research: map[string]HoldingResearch{
			"FPT": {Name: "FPT Corporation", Exchange: "HOSE", Sector: advisor.SectorGeneral, ICB4: "Computer Services",
				Low52W: 95000, High52W: 130500.4, Dividends: []research.Dividend{
					{ExDate: day0(2025, 6, 12), CashPerShare: 1000},
					{ExDate: day0(2026, 5, 28), StockRatio: 0.15},
				}},
			"HPG": {Name: "Hoa Phat Group", Exchange: "HOSE", DividendsMissing: true},
		},
	}
}

func TestRender_Golden(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleData()); err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "report.golden.md")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with UPDATE_GOLDEN=1 to create it)", err)
	}
	if got := buf.String(); got != string(want) {
		t.Errorf("report differs from %s (UPDATE_GOLDEN=1 rewrites it):\n%s", golden, got)
	}
}

func TestRender_Deterministic(t *testing.T) {
	var a, b bytes.Buffer
	if err := Render(&a, sampleData()); err != nil {
		t.Fatal(err)
	}
	if err := Render(&b, sampleData()); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Error("two renders of the same data differ")
	}
}

func TestRender_Privacy(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleData()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, leak := range []string{"0000001N", "0000001G", "105C"} {
		if strings.Contains(out, leak) {
			t.Errorf("report contains %q", leak)
		}
	}
	for _, want := range []string{"****001N (normal)", Disclaimer, `red flag \| ADTV`} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	if !strings.HasSuffix(out, Disclaimer+"\n") {
		t.Error("report does not end with the disclaimer")
	}
}

func TestRender_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Data{Phase: advisor.PhaseIntraday, Config: advisor.Default()}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"No holdings.", "VN-Index bars unavailable", "No stock in the universe passed", "## Warnings\n\nNone."} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("empty report lacks %q:\n%s", want, buf.String())
		}
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }

func TestRender_WriteError(t *testing.T) {
	if err := Render(failWriter{}, sampleData()); err == nil {
		t.Error("want the write error")
	}
}

func TestFormats(t *testing.T) {
	vnd := map[int64]string{0: "0", 999: "999", 1000: "1,000", 1234500: "1,234,500", -1234500: "-1,234,500"}
	for in, want := range vnd {
		if got := VND(in); got != want {
			t.Errorf("VND(%d) = %q, want %q", in, got, want)
		}
	}
	if got := VNDf(1234.5); got != "1,235" {
		t.Errorf("VNDf = %q", got)
	}
	if got := Pct(0.1234); got != "12.3%" {
		t.Errorf("Pct = %q", got)
	}
	if got := Ratio(1.005); got != "1.00" && got != "1.01" {
		t.Errorf("Ratio = %q", got)
	}
	masks := map[string]string{"0000001N": "****001N", "123": "****", " 105C000001 ": "****0001"}
	for in, want := range masks {
		if got := MaskAccount(in); got != want {
			t.Errorf("MaskAccount(%q) = %q, want %q", in, got, want)
		}
	}
}
