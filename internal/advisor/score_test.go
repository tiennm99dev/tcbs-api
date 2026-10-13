package advisor

import (
	"math"
	"testing"
	"time"

	"github.com/miti99dev/stock-advisor/internal/research"
)

func TestMetricBands(t *testing.T) {
	c := testConfig()
	cases := []struct {
		name   string
		metric string
		sector Sector
		in     Inputs
		score  float64
		raw    float64
		noRaw  bool // skip the raw check
	}{
		// VOL: best 0.20, worst 0.60
		{"vol past best", MetricVOL, SectorGeneral, Inputs{VOL: f(0.10)}, 100, 0.10, false},
		{"vol best", MetricVOL, SectorGeneral, Inputs{VOL: f(0.20)}, 100, 0.20, false},
		{"vol mid", MetricVOL, SectorGeneral, Inputs{VOL: f(0.40)}, 50, 0.40, false},
		{"vol worst", MetricVOL, SectorGeneral, Inputs{VOL: f(0.60)}, 0, 0.60, false},
		{"vol past worst", MetricVOL, SectorGeneral, Inputs{VOL: f(0.90)}, 0, 0.90, false},
		{"vol winsorised", MetricVOL, SectorGeneral, Inputs{VOL: f(3.5)}, 0, 2.0, false},
		{"vol negative winsorised", MetricVOL, SectorGeneral, Inputs{VOL: f(-1)}, 100, 0, false},
		// MDD: best 0.15, worst 0.50
		{"mdd past best", MetricMDD, SectorGeneral, Inputs{MDD: f(0.05)}, 100, 0.05, false},
		{"mdd best", MetricMDD, SectorGeneral, Inputs{MDD: f(0.15)}, 100, 0.15, false},
		{"mdd mid", MetricMDD, SectorGeneral, Inputs{MDD: f(0.325)}, 50, 0.325, false},
		{"mdd worst", MetricMDD, SectorGeneral, Inputs{MDD: f(0.50)}, 0, 0.50, false},
		{"mdd past worst", MetricMDD, SectorGeneral, Inputs{MDD: f(0.70)}, 0, 0.70, false},
		// LIQ: log10 scale, best 2e10, worst 1e9
		{"liq past best", MetricLIQ, SectorGeneral, Inputs{ADTV: f(1e11)}, 100, 1e11, false},
		{"liq best", MetricLIQ, SectorGeneral, Inputs{ADTV: f(2e10)}, 100, 2e10, false},
		{"liq log mid", MetricLIQ, SectorGeneral, Inputs{ADTV: f(math.Sqrt(2e10 * 1e9))}, 50, math.Sqrt(2e10 * 1e9), false},
		{"liq worst", MetricLIQ, SectorGeneral, Inputs{ADTV: f(1e9)}, 0, 1e9, false},
		{"liq past worst", MetricLIQ, SectorGeneral, Inputs{ADTV: f(1e8)}, 0, 1e8, false},
		{"liq zero", MetricLIQ, SectorGeneral, Inputs{ADTV: f(0)}, 0, 0, false},
		// ROE general: best 0.20, worst 0.05
		{"roe past best", MetricROE, SectorGeneral, Inputs{ROE: f(0.30)}, 100, 0.30, false},
		{"roe best", MetricROE, SectorGeneral, Inputs{ROE: f(0.20)}, 100, 0.20, false},
		{"roe mid", MetricROE, SectorGeneral, Inputs{ROE: f(0.125)}, 50, 0.125, false},
		{"roe worst", MetricROE, SectorGeneral, Inputs{ROE: f(0.05)}, 0, 0.05, false},
		{"roe past worst", MetricROE, SectorGeneral, Inputs{ROE: f(-0.10)}, 0, -0.10, false},
		// ROE bank: best 0.20, worst 0.08
		{"bank roe best", MetricROE, SectorBank, Inputs{ROE: f(0.20)}, 100, 0.20, false},
		{"bank roe mid", MetricROE, SectorBank, Inputs{ROE: f(0.14)}, 50, 0.14, false},
		{"bank roe worst", MetricROE, SectorBank, Inputs{ROE: f(0.08)}, 0, 0.08, false},
		{"bank roe past worst", MetricROE, SectorBank, Inputs{ROE: f(0.05)}, 0, 0.05, false},
		// DE general: best 0.3, worst 1.5
		{"de past best", MetricDE, SectorGeneral, Inputs{DE: f(0.1)}, 100, 0.1, false},
		{"de best", MetricDE, SectorGeneral, Inputs{DE: f(0.3)}, 100, 0.3, false},
		{"de mid", MetricDE, SectorGeneral, Inputs{DE: f(0.9)}, 50, 0.9, false},
		{"de worst", MetricDE, SectorGeneral, Inputs{DE: f(1.5)}, 0, 1.5, false},
		{"de winsorised", MetricDE, SectorGeneral, Inputs{DE: f(25)}, 0, 10, false},
		// DE real estate: best 0.5, worst 2.0
		{"re de best", MetricDE, SectorRealEstate, Inputs{DE: f(0.5)}, 100, 0.5, false},
		{"re de mid", MetricDE, SectorRealEstate, Inputs{DE: f(1.25)}, 50, 1.25, false},
		{"re de worst", MetricDE, SectorRealEstate, Inputs{DE: f(2.0)}, 0, 2.0, false},
		{"re de past worst", MetricDE, SectorRealEstate, Inputs{DE: f(3.0)}, 0, 3.0, false},
		// DE utilities: best 0.8, worst 2.5
		{"util de past best", MetricDE, SectorUtilities, Inputs{DE: f(0.2)}, 100, 0.2, false},
		{"util de best", MetricDE, SectorUtilities, Inputs{DE: f(0.8)}, 100, 0.8, false},
		{"util de mid", MetricDE, SectorUtilities, Inputs{DE: f(1.65)}, 50, 1.65, false},
		{"util de worst", MetricDE, SectorUtilities, Inputs{DE: f(2.5)}, 0, 2.5, false},
		// NPL: best 0.010, worst 0.035
		{"npl past best", MetricNPL, SectorBank, Inputs{NPL: f(0.005)}, 100, 0.005, false},
		{"npl best", MetricNPL, SectorBank, Inputs{NPL: f(0.010)}, 100, 0.010, false},
		{"npl mid", MetricNPL, SectorBank, Inputs{NPL: f(0.0225)}, 50, 0.0225, false},
		{"npl worst", MetricNPL, SectorBank, Inputs{NPL: f(0.035)}, 0, 0.035, false},
		{"npl past worst", MetricNPL, SectorBank, Inputs{NPL: f(0.05)}, 0, 0.05, false},
		// CASA: best 0.35, worst 0.10
		{"casa past best", MetricCASA, SectorBank, Inputs{CASA: f(0.50)}, 100, 0.50, false},
		{"casa best", MetricCASA, SectorBank, Inputs{CASA: f(0.35)}, 100, 0.35, false},
		{"casa mid", MetricCASA, SectorBank, Inputs{CASA: f(0.225)}, 50, 0.225, false},
		{"casa worst", MetricCASA, SectorBank, Inputs{CASA: f(0.10)}, 0, 0.10, false},
		{"casa past worst", MetricCASA, SectorBank, Inputs{CASA: f(0.05)}, 0, 0.05, false},
		// DY: best 0.06, worst 0; +10 for 3 paid years; >10% capped at 8%
		{"dy best", MetricDY, SectorGeneral, Inputs{DY: f(0.06)}, 100, 0.06, false},
		{"dy mid", MetricDY, SectorGeneral, Inputs{DY: f(0.03)}, 50, 0.03, false},
		{"dy worst", MetricDY, SectorGeneral, Inputs{DY: f(0)}, 0, 0, false},
		{"dy past best", MetricDY, SectorGeneral, Inputs{DY: f(0.09)}, 100, 0.09, false},
		{"dy bonus", MetricDY, SectorGeneral, Inputs{DY: f(0.03), DivPaidYears: 3}, 60, 0.03, false},
		{"dy bonus capped", MetricDY, SectorGeneral, Inputs{DY: f(0.06), DivPaidYears: 3}, 100, 0.06, false},
		{"dy two years no bonus", MetricDY, SectorGeneral, Inputs{DY: f(0.03), DivPaidYears: 2}, 50, 0.03, false},
		{"dy at special threshold", MetricDY, SectorGeneral, Inputs{DY: f(0.10)}, 100, 0.10, false},
		{"dy special capped", MetricDY, SectorGeneral, Inputs{DY: f(0.15)}, 100, 0.08, false},
		// BETA: best 0.9, worst 1.5
		{"beta past best", MetricBeta, SectorGeneral, Inputs{Beta: f(0.5), ADTV: f(5e10)}, 100, 0.5, false},
		{"beta best", MetricBeta, SectorGeneral, Inputs{Beta: f(0.9)}, 100, 0.9, false},
		{"beta mid", MetricBeta, SectorGeneral, Inputs{Beta: f(1.2)}, 50, 1.2, false},
		{"beta worst", MetricBeta, SectorGeneral, Inputs{Beta: f(1.5)}, 0, 1.5, false},
		{"beta winsorised", MetricBeta, SectorGeneral, Inputs{Beta: f(4)}, 0, 3, false},
		{"low beta illiquid capped", MetricBeta, SectorGeneral, Inputs{Beta: f(0.2), ADTV: f(3e9)}, 70, 0.2, false},
		{"low beta liquid not capped", MetricBeta, SectorGeneral, Inputs{Beta: f(0.2), ADTV: f(6e9)}, 100, 0.2, false},
		{"beta 0.3 illiquid not capped", MetricBeta, SectorGeneral, Inputs{Beta: f(0.3), ADTV: f(3e9)}, 100, 0.3, false},
		{"poor fit uses weekly", MetricBeta, SectorGeneral, Inputs{Beta: f(0.5), BetaR2: f(0.05), BetaWeekly: f(1.2), ADTV: f(5e10)}, 50, 1.2, false},
		{"good fit keeps daily", MetricBeta, SectorGeneral, Inputs{Beta: f(0.5), BetaR2: f(0.1), BetaWeekly: f(1.2), ADTV: f(5e10)}, 100, 0.5, false},
		{"illiquid uses weekly", MetricBeta, SectorGeneral, Inputs{Beta: f(0.5), BetaR2: f(0.5), BetaWeekly: f(1.5), ADTV: f(4e9)}, 0, 1.5, false},
		{"weekly fills missing daily", MetricBeta, SectorGeneral, Inputs{BetaWeekly: f(1.2)}, 50, 1.2, false},
		// PE absolute
		{"pe negative", MetricPE, SectorGeneral, Inputs{PE: f(-5)}, 0, -5, false},
		{"pe zero", MetricPE, SectorGeneral, Inputs{PE: f(0)}, 0, 0, false},
		{"pe low step", MetricPE, SectorGeneral, Inputs{PE: f(2)}, 50, 2, false},
		{"pe 4", MetricPE, SectorGeneral, Inputs{PE: f(4)}, 100, 4, false},
		{"pe 12", MetricPE, SectorGeneral, Inputs{PE: f(12)}, 100, 12, false},
		{"pe 15", MetricPE, SectorGeneral, Inputs{PE: f(15)}, 80, 15, false},
		{"pe 18", MetricPE, SectorGeneral, Inputs{PE: f(18)}, 60, 18, false},
		{"pe 24", MetricPE, SectorGeneral, Inputs{PE: f(24)}, 30, 24, false},
		{"pe 30", MetricPE, SectorGeneral, Inputs{PE: f(30)}, 0, 30, false},
		{"pe 50", MetricPE, SectorGeneral, Inputs{PE: f(50)}, 0, 50, false},
		{"pe winsorised", MetricPE, SectorGeneral, Inputs{PE: f(500)}, 0, 200, false},
		{"pe winsorised low", MetricPE, SectorGeneral, Inputs{PE: f(-500)}, 0, -100, false},
		// PE relative to the sector median (10)
		{"pe rel 0.5", MetricPE, SectorGeneral, Inputs{PE: f(5), SectorMedianPE: f(10)}, 100, 5, false},
		{"pe rel 0.8", MetricPE, SectorGeneral, Inputs{PE: f(8), SectorMedianPE: f(10)}, 100, 8, false},
		{"pe rel 0.9", MetricPE, SectorGeneral, Inputs{PE: f(9), SectorMedianPE: f(10)}, 85, 9, false},
		{"pe rel 1.0", MetricPE, SectorGeneral, Inputs{PE: f(10), SectorMedianPE: f(10)}, 70, 10, false},
		{"pe rel 1.15", MetricPE, SectorGeneral, Inputs{PE: f(11.5), SectorMedianPE: f(10)}, 55, 11.5, false},
		{"pe rel 1.3", MetricPE, SectorGeneral, Inputs{PE: f(13), SectorMedianPE: f(10)}, 40, 13, false},
		{"pe rel 1.45", MetricPE, SectorGeneral, Inputs{PE: f(14.5), SectorMedianPE: f(10)}, 20, 14.5, false},
		{"pe rel 1.6", MetricPE, SectorGeneral, Inputs{PE: f(16), SectorMedianPE: f(10)}, 0, 16, false},
		{"pe rel 2.0", MetricPE, SectorGeneral, Inputs{PE: f(20), SectorMedianPE: f(10)}, 0, 20, false},
		{"pe rel loss", MetricPE, SectorGeneral, Inputs{PE: f(-3), SectorMedianPE: f(10)}, 0, -3, false},
		// PE bank
		{"bank pe low step", MetricPE, SectorBank, Inputs{PE: f(3)}, 50, 3, false},
		{"bank pe 4", MetricPE, SectorBank, Inputs{PE: f(4)}, 100, 4, false},
		{"bank pe 10", MetricPE, SectorBank, Inputs{PE: f(10)}, 100, 10, false},
		{"bank pe 12.5", MetricPE, SectorBank, Inputs{PE: f(12.5)}, 75, 12.5, false},
		{"bank pe 15", MetricPE, SectorBank, Inputs{PE: f(15)}, 50, 15, false},
		{"bank pe 17.5", MetricPE, SectorBank, Inputs{PE: f(17.5)}, 25, 17.5, false},
		{"bank pe 20", MetricPE, SectorBank, Inputs{PE: f(20)}, 0, 20, false},
		{"bank pe 25", MetricPE, SectorBank, Inputs{PE: f(25)}, 0, 25, false},
		{"bank pe ignores median", MetricPE, SectorBank, Inputs{PE: f(10), SectorMedianPE: f(5)}, 100, 10, false},
		// PB general
		{"pb 1.0", MetricPB, SectorGeneral, Inputs{PB: f(1.0)}, 100, 1.0, false},
		{"pb 1.5", MetricPB, SectorGeneral, Inputs{PB: f(1.5)}, 100, 1.5, false},
		{"pb 2.25", MetricPB, SectorGeneral, Inputs{PB: f(2.25)}, 80, 2.25, false},
		{"pb 3", MetricPB, SectorGeneral, Inputs{PB: f(3)}, 60, 3, false},
		{"pb 4.5", MetricPB, SectorGeneral, Inputs{PB: f(4.5)}, 30, 4.5, false},
		{"pb 6", MetricPB, SectorGeneral, Inputs{PB: f(6)}, 0, 6, false},
		{"pb 8", MetricPB, SectorGeneral, Inputs{PB: f(8)}, 0, 8, false},
		{"pb winsorised", MetricPB, SectorGeneral, Inputs{PB: f(50)}, 0, 20, false},
		{"pb negative", MetricPB, SectorGeneral, Inputs{PB: f(-1)}, 0, 0, false},
		{"pb value trap capped", MetricPB, SectorGeneral, Inputs{PB: f(0.5), ROE: f(0.05)}, 50, 0.5, false},
		{"pb low with good roe", MetricPB, SectorGeneral, Inputs{PB: f(0.5), ROE: f(0.08)}, 100, 0.5, false},
		{"pb 0.7 not a trap", MetricPB, SectorGeneral, Inputs{PB: f(0.7), ROE: f(0.05)}, 100, 0.7, false},
		// PB vs fair PB (ROE 0.13 / CoE 0.13 = fairPB 1.0)
		{"pbfair 0.6", MetricPBFair, SectorBank, Inputs{PB: f(0.6), ROE: f(0.13)}, 100, 0.6, false},
		{"pbfair 0.8", MetricPBFair, SectorBank, Inputs{PB: f(0.8), ROE: f(0.13)}, 100, 0.8, false},
		{"pbfair 0.9", MetricPBFair, SectorBank, Inputs{PB: f(0.9), ROE: f(0.13)}, 87.5, 0.9, false},
		{"pbfair 1.0", MetricPBFair, SectorBank, Inputs{PB: f(1.0), ROE: f(0.13)}, 75, 1.0, false},
		{"pbfair 1.25", MetricPBFair, SectorBank, Inputs{PB: f(1.25), ROE: f(0.13)}, 40, 1.25, false},
		{"pbfair 1.5", MetricPBFair, SectorBank, Inputs{PB: f(1.5), ROE: f(0.13)}, 0, 1.5, false},
		{"pbfair 2.0", MetricPBFair, SectorBank, Inputs{PB: f(2.0), ROE: f(0.13)}, 0, 2.0, false},
		{"pbfair high roe", MetricPBFair, SectorSecurities, Inputs{PB: f(2.0), ROE: f(0.26)}, 75, 2.0, false},
		{"pbfair negative roe", MetricPBFair, SectorBank, Inputs{PB: f(1.0), ROE: f(-0.02)}, 0, 1.0, false},
		// EARN_CONS
		{"earn given", MetricEarnCons, SectorGeneral, Inputs{EarnCons: f(64.25)}, 64.3, 64.25, false},
		{"earn steady", MetricEarnCons, SectorGeneral, Inputs{ProfitYears: []float64{10, 10, 10}}, 100, 100, false},
		{"earn growing", MetricEarnCons, SectorGeneral, Inputs{ProfitYears: []float64{30, 20, 10}}, 87.8, 87.8, false},
		{"earn one loss", MetricEarnCons, SectorGeneral, Inputs{ProfitYears: []float64{10, -10, 10, 10}}, 52.5, 52.5, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := scoreMetric(tc.metric, tc.in, tc.sector, c)
			if !r.ok {
				t.Fatalf("metric dropped: %+v", r)
			}
			if !approx(r.score, tc.score) {
				t.Errorf("score = %v, want %v (note %q)", r.score, tc.score, r.note)
			}
			if !tc.noRaw && !approx(r.raw, tc.raw) {
				t.Errorf("raw = %v, want %v", r.raw, tc.raw)
			}
		})
	}
}

func TestMetricNotes(t *testing.T) {
	c := testConfig()
	cases := []struct {
		name, metric string
		sector       Sector
		in           Inputs
		note         string
	}{
		{"special dividend", MetricDY, SectorGeneral, Inputs{DY: f(0.12)}, "high trailing yield, likely special dividend"},
		{"weekly beta", MetricBeta, SectorGeneral, Inputs{Beta: f(0.5), BetaR2: f(0.05), BetaWeekly: f(1)}, "weekly 2y beta used"},
		{"weekly missing", MetricBeta, SectorGeneral, Inputs{Beta: f(0.5), BetaR2: f(0.05)}, "weekly beta unavailable"},
		{"low beta cap", MetricBeta, SectorGeneral, Inputs{Beta: f(0.1), ADTV: f(1e9)}, "low beta on an illiquid stock"},
		{"value trap", MetricPB, SectorGeneral, Inputs{PB: f(0.5), ROE: f(0.05)}, "value trap"},
		{"loss", MetricPE, SectorGeneral, Inputs{PE: f(-1)}, "loss-making"},
		{"relative", MetricPE, SectorGeneral, Inputs{PE: f(9), SectorMedianPE: f(10)}, "PE/sector median 0.90"},
		{"fair pb", MetricPBFair, SectorBank, Inputs{PB: f(1), ROE: f(0.13)}, "fairPB 1.00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := scoreMetric(tc.metric, tc.in, tc.sector, c)
			if !hasText([]string{r.note}, tc.note) {
				t.Errorf("note %q, want it to contain %q", r.note, tc.note)
			}
		})
	}
}

func TestMetricMissing(t *testing.T) {
	c := testConfig()
	cases := []struct {
		name, metric string
		in           Inputs
	}{
		{"vol nil", MetricVOL, Inputs{}},
		{"vol NaN", MetricVOL, Inputs{VOL: f(math.NaN())}},
		{"beta nil", MetricBeta, Inputs{}},
		{"liq nil", MetricLIQ, Inputs{}},
		{"pe nil", MetricPE, Inputs{}},
		{"pb nil", MetricPB, Inputs{}},
		{"pbfair without roe", MetricPBFair, Inputs{PB: f(1)}},
		{"dy nil", MetricDY, Inputs{}},
		{"earn two years", MetricEarnCons, Inputs{ProfitYears: []float64{1, 2}}},
		{"llr", MetricLLR, perfectBank("VCB")},
		{"margin lev", MetricMarginLev, perfectInputs("SSI")},
		{"combined", MetricCombined, perfectInputs("BVH")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if r := scoreMetric(tc.metric, tc.in, SectorBank, c); r.ok {
				t.Errorf("metric should be missing, got %+v", r)
			}
		})
	}
}

func TestProfileWeightsSumTo100(t *testing.T) {
	for sec, p := range profiles {
		var total float64
		for _, pl := range p {
			for _, m := range pl.metrics {
				total += m.weight
			}
		}
		if total != 100 {
			t.Errorf("%s weights sum to %v", sec, total)
		}
	}
}

func TestScoreStockPerfect(t *testing.T) {
	s := ScoreStock(perfectInputs("FPT"), testConfig())
	if !s.Valid || s.Value != 100 || s.Tier != TierStable || s.Coverage != 100 || s.Confidence != ConfidenceHigh {
		t.Fatalf("perfect stock: %+v", s)
	}
	var sum float64
	for _, m := range s.Metrics {
		sum += m.Weight
	}
	if !approx(sum, 100) {
		t.Errorf("effective weights sum to %v", sum)
	}
	if len(s.RedFlags) != 0 {
		t.Errorf("red flags: %v", s.RedFlags)
	}
}

func TestScoreStockBankWithoutLLR(t *testing.T) {
	s := ScoreStock(perfectBank("VCB"), testConfig())
	if !s.Valid || s.Value != 100 || s.Coverage != 95 {
		t.Fatalf("bank: value %d coverage %v valid %v notes %v", s.Value, s.Coverage, s.Valid, s.Notes)
	}
	llr := metric(t, s, MetricLLR)
	if !llr.Dropped || llr.Weight != 0 {
		t.Errorf("LLR not dropped: %+v", llr)
	}
	if !hasText(s.Notes, "LLR not available from any source") {
		t.Errorf("notes: %v", s.Notes)
	}
	// Fundamentals keep their 35 points: ROE gets 12/30 of them.
	if roe := metric(t, s, MetricROE); !approx(roe.Weight, 14) {
		t.Errorf("ROE effective weight = %v, want 14", roe.Weight)
	}
	if npl := metric(t, s, MetricNPL); !approx(npl.Weight, 35.0*10/30) {
		t.Errorf("NPL effective weight = %v", npl.Weight)
	}
}

func TestScoreStockRedistributionWithinPillar(t *testing.T) {
	in := perfectInputs("FPT")
	in.VOL = f(0.60) // scores 0, weight 15 of 40
	in.MDD = nil     // dropped, weight 10 of 40
	s := ScoreStock(in, testConfig())
	// Price pillar: (0×15 + 100×10 + 100×5) / 30 = 50.
	if got := s.Pillars[PillarPrice]; !approx(got, 50) {
		t.Errorf("price pillar = %v, want 50", got)
	}
	// Total = 50×0.40 + 100×0.35 + 100×0.25 = 80.
	if s.Value != 80 || s.Coverage != 90 {
		t.Errorf("value %d coverage %v", s.Value, s.Coverage)
	}
	if vol := metric(t, s, MetricVOL); !approx(vol.Weight, 20) {
		t.Errorf("VOL effective weight = %v, want 20", vol.Weight)
	}
	if !hasText(s.Notes, "MDD unavailable: weight redistributed within price") {
		t.Errorf("notes: %v", s.Notes)
	}
}

func TestScoreStockMissingPillar(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*Inputs)
		coverage float64
		valid    bool
		pillar   string // the missing pillar
	}{
		{"fundamentals absent", func(in *Inputs) { in.ROE, in.DE, in.ProfitYears = nil, nil, nil }, 65, true, PillarFundamentals},
		{"fundamentals under half", func(in *Inputs) { in.DE, in.ProfitYears = nil, nil }, 65, true, PillarFundamentals},
		{"fundamentals exactly half or more", func(in *Inputs) { in.ROE = nil }, 88, true, ""},
		{"coverage under 60", func(in *Inputs) { in.ROE, in.DE, in.ProfitYears, in.PB, in.DY = nil, nil, nil, nil, nil }, 40, false, PillarFundamentals},
		{"price pillar missing", func(in *Inputs) { in.VOL, in.Beta, in.MDD = nil, nil, nil }, 60, false, PillarPrice},
		{"coverage 55 without fundamentals or DY", func(in *Inputs) { in.ROE, in.DE, in.ProfitYears, in.DY = nil, nil, nil, nil }, 55, false, PillarFundamentals},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := perfectInputs("FPT")
			tc.mutate(&in)
			s := ScoreStock(in, testConfig())
			if !approx(s.Coverage, tc.coverage) || s.Valid != tc.valid {
				t.Fatalf("coverage %v valid %v, want %v %v; notes %v", s.Coverage, s.Valid, tc.coverage, tc.valid, s.Notes)
			}
			if tc.pillar != "" {
				if _, ok := s.Pillars[tc.pillar]; ok {
					t.Errorf("pillar %s should be missing: %v", tc.pillar, s.Pillars)
				}
				if !hasText(s.Notes, tc.pillar+" pillar missing") {
					t.Errorf("notes: %v", s.Notes)
				}
			}
			if !tc.valid && s.Tier != TierInsufficient {
				t.Errorf("tier = %q", s.Tier)
			}
			var sum float64
			for _, m := range s.Metrics {
				sum += m.Weight
			}
			if tc.coverage > 0 && !approx(sum, 100) {
				t.Errorf("effective weights sum to %v", sum)
			}
		})
	}
}

func TestScoreStockPillarUnderHalfDropsPresentMetrics(t *testing.T) {
	in := perfectInputs("FPT")
	in.DE, in.ProfitYears = nil, nil // ROE alone is 12 of 35
	s := ScoreStock(in, testConfig())
	roe := metric(t, s, MetricROE)
	if !roe.Dropped || roe.Weight != 0 {
		t.Errorf("ROE should be dropped with its pillar: %+v", roe)
	}
	if !hasText(s.Notes, "ROE dropped") {
		t.Errorf("notes: %v", s.Notes)
	}
}

func TestScoreStockCoverageBoundary(t *testing.T) {
	// Coverage exactly 60 with the price pillar present: price 40, PE 10 and
	// DY 10 (valuation 20 of 25), no fundamentals.
	in := perfectInputs("FPT")
	in.ROE, in.DE, in.ProfitYears, in.PB = nil, nil, nil, nil
	s := ScoreStock(in, testConfig())
	if s.Coverage != 60 || !s.Valid || s.Confidence != ConfidenceMedium {
		t.Fatalf("coverage %v valid %v confidence %s", s.Coverage, s.Valid, s.Confidence)
	}
	// One more point missing would go under 60: drop LIQ (5).
	in.ADTV = nil
	s = ScoreStock(in, testConfig())
	if s.Coverage != 55 || s.Valid || s.Tier != TierInsufficient {
		t.Fatalf("coverage %v valid %v tier %s", s.Coverage, s.Valid, s.Tier)
	}
}

func TestConfidence(t *testing.T) {
	c := testConfig()
	cases := []struct {
		cov  float64
		want string
	}{
		{100, ConfidenceHigh}, {85, ConfidenceHigh}, {84.9, ConfidenceMedium},
		{60, ConfidenceMedium}, {59.9, ConfidenceLow}, {0, ConfidenceLow},
	}
	for _, tc := range cases {
		if got := confidence(tc.cov, c); got != tc.want {
			t.Errorf("confidence(%v) = %s, want %s", tc.cov, got, tc.want)
		}
	}
}

func TestTier(t *testing.T) {
	c := testConfig()
	cases := []struct {
		value int
		valid bool
		want  string
	}{
		{100, true, TierStable}, {75, true, TierStable}, {74, true, TierModerate},
		{60, true, TierModerate}, {59, true, TierWeak}, {45, true, TierWeak},
		{44, true, TierSpeculative}, {0, true, TierSpeculative}, {90, false, TierInsufficient},
	}
	for _, tc := range cases {
		if got := Tier(Score{Value: tc.value, Valid: tc.valid}, c); got != tc.want {
			t.Errorf("Tier(%d, %v) = %s, want %s", tc.value, tc.valid, got, tc.want)
		}
	}
}

func TestSectorAdjustment(t *testing.T) {
	cases := []struct {
		name   string
		sector Sector
		mutate func(*Inputs)
		value  int
		adj    int
	}{
		{"real estate", SectorRealEstate, nil, 95, -5},
		{"securities", SectorSecurities, func(in *Inputs) { in.PB = f(1.0) }, 95, -5},
		{"utilities", SectorUtilities, nil, 100, 0},
		{"floored at zero", SectorRealEstate, func(in *Inputs) {
			in.VOL, in.Beta, in.MDD, in.ADTV = f(1), f(2), f(0.9), f(1e8)
			in.ROE, in.DE, in.PE, in.PB, in.DY = f(0), f(5), f(-1), f(10), f(0)
			in.DivPaidYears = 0
			in.ProfitYears = []float64{-1, -1, -1}
		}, 0, -5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := perfectInputs("X")
			in.Sector = tc.sector
			if tc.mutate != nil {
				tc.mutate(&in)
			}
			s := ScoreStock(in, testConfig())
			if s.Value != tc.value || s.SectorAdj != tc.adj {
				t.Errorf("value %d adj %d, want %d %d (metrics %+v)", s.Value, s.SectorAdj, tc.value, tc.adj, s.Metrics)
			}
		})
	}
}

func TestScoreStockNotScoredAndUnknownSector(t *testing.T) {
	c := testConfig()
	etf := perfectInputs("E1VFVN30")
	etf.Type = "ETF"
	s := ScoreStock(etf, c)
	if s.Valid || s.Tier != TierNotScored || !hasText(s.Notes, NoteNotScored) || len(s.Metrics) != 0 {
		t.Errorf("ETF: %+v", s)
	}

	in := perfectInputs("ABC")
	in.Sector = ""
	s = ScoreStock(in, c)
	if s.Sector != SectorGeneral || !hasText(s.Notes, NoteSectorUnknown) || s.Value != 100 {
		t.Errorf("unknown sector: %+v", s)
	}

	in.HistoryReturns = 80
	s = ScoreStock(in, c)
	if !hasText(s.Notes, "insufficient history") {
		t.Errorf("notes: %v", s.Notes)
	}
}

func TestRedFlags(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Inputs)
		flag   string // "" means no flag
	}{
		{"clean", func(*Inputs) {}, ""},
		{"status unknown is a note", func(in *Inputs) { in.Status = "" }, ""},
		{"status listed any case", func(in *Inputs) { in.Status = " Listed " }, ""},
		{"suspended", func(in *Inputs) { in.Status = "suspended" }, "listing status"},
		{"delisted", func(in *Inputs) { in.Status = "delisted" }, "listing status"},
		{"negative equity", func(in *Inputs) { in.NegEquity = true }, "negative equity"},
		{"ttm loss only", func(in *Inputs) { in.TTMLoss = true }, ""},
		{"previous loss only", func(in *Inputs) { in.PrevYearLoss = true }, ""},
		{"two losses", func(in *Inputs) { in.TTMLoss, in.PrevYearLoss = true, true }, "TTM loss"},
		{"adtv at 1bn", func(in *Inputs) { in.ADTV = f(1e9) }, ""},
		{"adtv below 1bn", func(in *Inputs) { in.ADTV = f(9.99e8) }, "ADTV"},
		{"10 gaps", func(in *Inputs) { in.GapSessions = 10 }, ""},
		{"11 gaps", func(in *Inputs) { in.GapSessions = 11 }, "gap sessions"},
		{"5 stale", func(in *Inputs) { in.StaleSessions = 5 }, ""},
		{"6 stale", func(in *Inputs) { in.StaleSessions = 6 }, "stale or suspended"},
		{"fundamentals 200 days", func(in *Inputs) { in.FundAge = 200 * 24 * time.Hour }, ""},
		{"fundamentals 201 days", func(in *Inputs) { in.FundAge = 201 * 24 * time.Hour }, "fundamentals 201 days old"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := perfectInputs("FPT")
			tc.mutate(&in)
			s := ScoreStock(in, testConfig())
			if tc.flag == "" {
				if len(s.RedFlags) != 0 {
					t.Errorf("unexpected flags %v", s.RedFlags)
				}
				return
			}
			if len(s.RedFlags) != 1 || !hasText(s.RedFlags, tc.flag) {
				t.Errorf("flags %v, want one containing %q", s.RedFlags, tc.flag)
			}
		})
	}
}

func TestStaleFlagAcrossHoliday(t *testing.T) {
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	// The index trades up to Fri 13 Feb, closes 9 days for Tet, and reopens
	// Mon 23 Feb for three sessions. The stock's last bar is 13 Feb.
	var index, stock research.Series
	for _, d := range []time.Time{day(2, 9), day(2, 10), day(2, 11), day(2, 12), day(2, 13)} {
		index.Bars = append(index.Bars, research.Bar{Date: d, Close: 1000})
		stock.Bars = append(stock.Bars, research.Bar{Date: d, Close: 10})
	}
	for _, d := range []time.Time{day(2, 23), day(2, 24), day(2, 25)} {
		index.Bars = append(index.Bars, research.Bar{Date: d, Close: 1000})
	}
	in := perfectInputs("FPT")
	in.StaleSessions = research.StaleSessions(stock, index)
	if in.StaleSessions != 3 {
		t.Fatalf("stale sessions = %d, want 3", in.StaleSessions)
	}
	if s := ScoreStock(in, testConfig()); len(s.RedFlags) != 0 {
		t.Errorf("holiday gap raised %v", s.RedFlags)
	}
}

func TestEarningsConsistency(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
		ok   bool
	}{
		{"none", nil, 0, false},
		{"two years", []float64{1, 2}, 0, false},
		{"flat", []float64{5, 5, 5}, 100, true},
		{"all losses", []float64{-5, -5, -5}, 30, true},
		{"zero mean", []float64{5, -5, 5, -5}, 35, true},
		{"four years", []float64{30, 20, 10, 20}, 70 + (1-math.Sqrt(50)/20)*30, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := EarningsConsistency(tc.in, 3)
			if ok != tc.ok || !approx(got, round1(tc.want)) {
				t.Errorf("got %v %v, want %v %v", got, ok, round1(tc.want), tc.ok)
			}
		})
	}
}

func TestMedianPE(t *testing.T) {
	c := testConfig()
	cases := []struct {
		name string
		in   []float64
		want *float64
	}{
		{"too few", []float64{10, 12, 14, 16}, nil},
		{"losses excluded", []float64{10, 12, 14, 16, -3, 0}, nil},
		{"odd", []float64{16, 10, 14, 12, 18}, f(14)},
		{"even", []float64{10, 12, 14, 16, 18, 20, -1}, f(15)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MedianPE(tc.in, c)
			switch {
			case tc.want == nil && got != nil:
				t.Errorf("got %v, want nil", *got)
			case tc.want != nil && (got == nil || !approx(*got, *tc.want)):
				t.Errorf("got %v, want %v", got, *tc.want)
			}
		})
	}
}

func TestPiecewise(t *testing.T) {
	pts := []Point{{1, 100}, {2, 50}, {4, 0}}
	cases := []struct{ x, want float64 }{
		{0, 100}, {1, 100}, {1.5, 75}, {2, 50}, {3, 25}, {4, 0}, {9, 0},
	}
	for _, tc := range cases {
		if got := piecewise(tc.x, pts); !approx(got, tc.want) {
			t.Errorf("piecewise(%v) = %v, want %v", tc.x, got, tc.want)
		}
	}
	if got := piecewise(1, nil); got != 0 {
		t.Errorf("empty curve = %v", got)
	}
}
