package advisor

import (
	"testing"
	"time"

	"github.com/miti99dev/stock-advisor/internal/portfolio"
)

const testNAV = 1_000_000_000

// pos builds a fully sellable position; weight follows from qty × price / NAV.
func pos(sym string, qty, price, avg int64) portfolio.Position {
	p := portfolio.Position{
		Symbol: sym, Qty: qty, Sellable: qty, Price: price, AvgCost: avg,
		MarketValue: qty * price, Scored: true,
	}
	p.Weight = float64(p.MarketValue) / testNAV
	if avg > 0 {
		p.CostValue = qty * avg
		p.UnrealizedPnL = p.MarketValue - p.CostValue
		p.PnLPct = float64(price-avg) / float64(avg)
	}
	return p
}

// summary builds a portfolio with NAV fixed at testNAV.
func summary(cash, debt int64, ps ...portfolio.Position) portfolio.Summary {
	s := portfolio.Summary{Positions: ps, Cash: cash, Debt: debt, NAV: testNAV, EffectiveN: 10}
	s.CashRatio = float64(cash) / testNAV
	s.MarginRatio = float64(debt) / testNAV
	return s
}

// score builds a valid score with a valuation pillar and a few metrics.
func score(sym string, v int, valuation float64) Score {
	return Score{
		Symbol: sym, Sector: SectorGeneral, Value: v, Valid: true, Coverage: 100, Tier: TierStable,
		Pillars: map[string]float64{PillarPrice: 80, PillarValuation: valuation},
		Metrics: []MetricScore{
			{Name: MetricVOL, Raw: 0.18, Score: 100, Weight: 15, Contribution: 15},
			{Name: MetricROE, Raw: 0.22, Score: 100, Weight: 12, Contribution: 12},
			{Name: MetricPE, Raw: 25, Score: 20, Weight: 10, Contribution: 2},
			{Name: MetricLLR, Dropped: true},
		},
	}
}

func adviceFor(t *testing.T, pa PortfolioAdvice, sym string) Advice {
	t.Helper()
	for _, a := range pa.Holdings {
		if a.Symbol == sym {
			return a
		}
	}
	t.Fatalf("no advice for %s in %+v", sym, pa.Holdings)
	return Advice{}
}

func TestRecommendSingleHolding(t *testing.T) {
	const cash = 300_000_000
	cases := []struct {
		name      string
		pos       portfolio.Position
		score     *Score // nil means no score at all
		cash      int64
		debt      int64
		action    Action
		rule      string
		secondary string
		note      string
		qty       int64
		estCash   int64
	}{
		{name: "red flag", pos: pos("A", 5000, 10000, 10000), score: func() *Score { s := score("A", 90, 80); s.RedFlags = []string{"negative equity"}; return &s }(), action: ActionReview, rule: "red flag: negative equity"},
		{name: "invalid score", pos: pos("A", 5000, 10000, 10000), score: func() *Score { s := score("A", 90, 80); s.Valid = false; s.Coverage = 55; return &s }(), action: ActionReview, rule: "score invalid (coverage 55"},
		{name: "no score", pos: pos("A", 5000, 10000, 10000), action: ActionReview, rule: "no score data"},
		{name: "score 44", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 44, 80)), action: ActionReview, rule: "score 44 < 45"},
		{name: "score 45", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 45, 80)), action: ActionHold},
		{name: "loss 20 weak", pos: pos("A", 5000, 8000, 10000), score: ptrScore(score("A", 59, 80)), action: ActionReview, rule: "P&L -20.0% ≤ -20% and score 59 < 60"},
		{name: "loss 20 score 60", pos: pos("A", 5000, 8000, 10000), score: ptrScore(score("A", 60, 80)), action: ActionHold},
		{name: "loss 19.9 weak", pos: pos("A", 5000, 8010, 10000), score: ptrScore(score("A", 55, 80)), action: ActionHold},
		{name: "loss 15 score 49", pos: pos("A", 5000, 8500, 10000), score: ptrScore(score("A", 49, 80)), action: ActionReview, rule: "P&L -15.0% ≤ -15% and score 49 < 50"},
		{name: "loss 15 score 50", pos: pos("A", 5000, 8500, 10000), score: ptrScore(score("A", 50, 80)), action: ActionHold},
		{name: "unknown cost skips loss rules", pos: pos("A", 5000, 8000, 0), score: ptrScore(score("A", 55, 80)), action: ActionHold},

		{name: "hard cap", pos: pos("A", 27400, 10000, 10000), score: ptrScore(score("A", 80, 80)), action: ActionTrim, rule: "TRIM: weight 27.4% > hard cap 25%; trim to 20%", qty: 7400, estCash: 73_852_000},
		{name: "at hard cap", pos: pos("A", 25000, 10000, 10000), score: ptrScore(score("A", 80, 80)), action: ActionHold},
		{name: "take profit", pos: pos("A", 3800, 13000, 10000), score: ptrScore(score("A", 55, 80)), action: ActionTrim, rule: "P&L 30.0% ≥ 30%; sell 1/3", qty: 1200, estCash: 15_568_800},
		{name: "profit 29", pos: pos("A", 3800, 12900, 10000), score: ptrScore(score("A", 55, 80)), action: ActionHold},
		{name: "take profit needs weak score", pos: pos("A", 3800, 13000, 10000), score: ptrScore(score("A", 60, 80)), action: ActionHold},
		{name: "weak overweight", pos: pos("A", 12000, 10000, 10000), score: ptrScore(score("A", 50, 80)), action: ActionTrim, rule: "score 50 in [45, 60) and weight 12.0% > 10%; trim to 10%", qty: 2000},
		{name: "weak at 10%", pos: pos("A", 10000, 10000, 10000), score: ptrScore(score("A", 50, 80)), action: ActionHold},
		{name: "score 60 overweight", pos: pos("A", 12000, 10000, 10000), score: ptrScore(score("A", 60, 80)), action: ActionHold},
		{name: "review beats trim", pos: pos("A", 30000, 10000, 10000), score: ptrScore(score("A", 40, 80)), action: ActionReview, rule: "score 40 < 45", secondary: "TRIM: weight 30.0% > hard cap 25%"},
		{name: "trim rules secondary", pos: pos("A", 30000, 13000, 10000), score: ptrScore(score("A", 55, 80)), action: ActionTrim, rule: "hard cap", secondary: "sell 1/3"},

		{name: "add", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 75, 60)), action: ActionAdd, rule: "ADD: score 75 ≥ 70, weight 5.0% < 15%", qty: 14900, estCash: -149_149_000},
		{name: "add score 69", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 69, 60)), action: ActionHold},
		{name: "add score 70", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 70, 60)), action: ActionAdd, qty: 14900},
		{name: "add weight 15", pos: pos("A", 15000, 10000, 10000), score: ptrScore(score("A", 80, 60)), action: ActionHold},
		{name: "add weight 14.9", pos: pos("A", 14900, 10000, 10000), score: ptrScore(score("A", 80, 60)), action: ActionAdd, qty: 5000},
		{name: "add cash at 5%", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 80, 60)), cash: 50_000_000, action: ActionHold},
		{name: "add valuation 49", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 80, 49)), action: ActionHold},
		{name: "add valuation 50", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 80, 50)), action: ActionAdd},
		{name: "add no valuation pillar", pos: pos("A", 5000, 10000, 10000), score: func() *Score { s := score("A", 80, 60); delete(s.Pillars, PillarValuation); return &s }(), action: ActionHold},
		{name: "average down needs stable", pos: pos("A", 5000, 8500, 10000), score: ptrScore(score("A", 74, 60)), action: ActionHold},
		{name: "average down stable", pos: pos("A", 5000, 8500, 10000), score: ptrScore(score("A", 75, 60)), action: ActionAdd},
		{name: "margin above max", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 80, 60)), debt: 210_000_000, action: ActionHold, note: "ADD suppressed: margin 21.0% of NAV > 20%; reduce margin first"},
		{name: "margin at max", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 80, 60)), debt: 200_000_000, action: ActionAdd},
		{name: "insufficient cash", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 80, 60)), cash: 50_500_000, action: ActionHold, note: "insufficient cash for one lot"},
		{name: "cash-limited add", pos: pos("A", 5000, 10000, 10000), score: ptrScore(score("A", 80, 60)), cash: 60_000_000, action: ActionAdd, qty: 900, estCash: -9_009_000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scores := map[string]Score{}
			if tc.score != nil {
				scores["A"] = *tc.score
			}
			c := tc.cash
			if c == 0 {
				c = cash
			}
			pa := Recommend(summary(c, tc.debt, tc.pos), scores, nil, testConfig())
			a := adviceFor(t, pa, "A")
			if a.Action != tc.action {
				t.Fatalf("action %s, want %s (%+v)", a.Action, tc.action, a)
			}
			if tc.rule != "" && !hasText([]string{a.Rule}, tc.rule) {
				t.Errorf("rule %q, want %q", a.Rule, tc.rule)
			}
			if tc.secondary != "" && !hasText(a.Secondary, tc.secondary) {
				t.Errorf("secondary %v, want %q", a.Secondary, tc.secondary)
			}
			if tc.note != "" && !hasText(a.Notes, tc.note) {
				t.Errorf("notes %v, want %q", a.Notes, tc.note)
			}
			if tc.qty != 0 && (a.Qty != tc.qty || a.Lots != tc.qty/BoardLot) {
				t.Errorf("qty %d lots %d, want %d", a.Qty, a.Lots, tc.qty)
			}
			if tc.estCash != 0 && a.EstCash != tc.estCash {
				t.Errorf("est cash %d, want %d", a.EstCash, tc.estCash)
			}
			if a.Action == ActionReview && (a.Qty != 0 || a.EstCash != 0) {
				t.Errorf("REVIEW carries a trade: %+v", a)
			}
		})
	}
}

func ptrScore(s Score) *Score { return &s }

func TestRecommendTrimSellableCap(t *testing.T) {
	cases := []struct {
		name      string
		sellable  int64
		receiving int64
		qty       int64
		note      string
	}{
		{"receiving shares settle later", 5050, 3000, 5000, "2,400 shares settle T+2"},
		{"few receiving", 5050, 1000, 5000, "1,000 shares settle T+2"},
		{"no receiving", 5050, 0, 5000, "capped at the sellable 5,050 shares"},
		{"enough sellable", 27400, 0, 7400, ""},
		{"nothing sellable", 50, 27350, 0, "7,400 shares settle T+2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := pos("A", 27400, 10000, 10000)
			p.Sellable, p.Receiving = tc.sellable, tc.receiving
			pa := Recommend(summary(300_000_000, 0, p), map[string]Score{"A": score("A", 80, 60)}, nil, testConfig())
			a := adviceFor(t, pa, "A")
			if a.Action != ActionTrim || a.Qty != tc.qty || a.Lots != tc.qty/BoardLot {
				t.Fatalf("got %+v, want TRIM %d", a, tc.qty)
			}
			if a.EstCash != SellProceeds(tc.qty, 10000, 0.001, 0.001) {
				t.Errorf("est cash %d", a.EstCash)
			}
			if tc.note != "" && !hasText(a.Notes, tc.note) {
				t.Errorf("notes %v, want %q", a.Notes, tc.note)
			}
			if tc.note == "" && len(a.Notes) != 0 {
				t.Errorf("unexpected notes %v", a.Notes)
			}
		})
	}
}

func TestRecommendSectorTrim(t *testing.T) {
	a := pos("ACB", 22000, 10000, 10000) // 22%
	b := pos("VCB", 23000, 10000, 10000) // 23%
	scores := map[string]Score{"ACB": score("ACB", 70, 60), "VCB": score("VCB", 80, 60)}
	sectors := map[string]Sector{"ACB": SectorBank, "VCB": SectorBank}
	pa := Recommend(summary(300_000_000, 0, a, b), scores, sectors, testConfig())
	if w := pa.SectorWeights[SectorBank]; !approx(w, 0.45) {
		t.Fatalf("bank weight %v", w)
	}
	low := adviceFor(t, pa, "ACB")
	if low.Action != ActionTrim || !hasText([]string{low.Rule}, "sector bank above cap 40%") || low.Qty != 5000 {
		t.Errorf("lowest score: %+v", low)
	}
	high := adviceFor(t, pa, "VCB")
	if high.Action != ActionHold {
		t.Errorf("higher score: %+v", high)
	}
	if !hasText(pa.Messages, "bank 45.0% (above the 40% cap)") {
		t.Errorf("messages %v", pa.Messages)
	}

	// The excess spills to the next-lowest score when the lowest is too small.
	small := pos("ACB", 3000, 10000, 10000) // 3%, score 70
	mid := pos("CTG", 18000, 10000, 10000)  // 18%, score 75
	big := pos("VCB", 24000, 10000, 10000)  // 24%, score 80
	scores["CTG"] = score("CTG", 75, 60)
	sectors["CTG"] = SectorBank
	pa = Recommend(summary(300_000_000, 0, small, mid, big), scores, sectors, testConfig())
	// 45% − 40% = 50m: ACB sells its whole 30m, CTG the other 20m.
	if a := adviceFor(t, pa, "ACB"); a.Action != ActionTrim || a.Qty != 3000 {
		t.Errorf("ACB: %+v", a)
	}
	if a := adviceFor(t, pa, "CTG"); a.Action != ActionTrim || a.Qty != 2000 {
		t.Errorf("CTG: %+v", a)
	}
	if a := adviceFor(t, pa, "VCB"); a.Action != ActionHold {
		t.Errorf("VCB: %+v", a)
	}
}

func TestRecommendAddSizing(t *testing.T) {
	c := testConfig()
	t.Run("split across candidates", func(t *testing.T) {
		a := pos("A", 5000, 10000, 10000)
		b := pos("B", 5000, 10000, 10000)
		scores := map[string]Score{"A": score("A", 80, 60), "B": score("B", 80, 60)}
		pa := Recommend(summary(300_000_000, 0, a, b), scores, nil, c)
		// deployable 250m / 2 = 125m → 124 lots at 1,001,000.
		for _, sym := range []string{"A", "B"} {
			if got := adviceFor(t, pa, sym); got.Action != ActionAdd || got.Lots != 124 {
				t.Errorf("%s: %+v", sym, got)
			}
		}
	})
	t.Run("capped by sector room", func(t *testing.T) {
		a := pos("A", 5000, 10000, 10000)
		o1 := pos("O1", 17000, 10000, 10000)
		o2 := pos("O2", 17000, 10000, 10000)
		scores := map[string]Score{"A": score("A", 80, 60), "O1": score("O1", 65, 60), "O2": score("O2", 65, 60)}
		pa := Recommend(summary(300_000_000, 0, a, o1, o2), scores, nil, c)
		got := adviceFor(t, pa, "A")
		// sector 39% → room 10m → 9 lots.
		if got.Action != ActionAdd || got.Lots != 9 {
			t.Fatalf("A: %+v", got)
		}
		after := pa.SectorWeights[SectorGeneral] + float64(got.Qty*10000)/testNAV
		if after > c.SectorCap {
			t.Errorf("sector after add %v", after)
		}
	})
	t.Run("sector room under one lot", func(t *testing.T) {
		a := pos("A", 5000, 10000, 10000)
		o1 := pos("O1", 17490, 10000, 10000)
		o2 := pos("O2", 17490, 10000, 10000)
		scores := map[string]Score{"A": score("A", 80, 60), "O1": score("O1", 65, 60), "O2": score("O2", 65, 60)}
		pa := Recommend(summary(300_000_000, 0, a, o1, o2), scores, nil, c)
		got := adviceFor(t, pa, "A")
		if got.Action != ActionHold || !hasText(got.Notes, "under one lot of room") {
			t.Fatalf("A: %+v", got)
		}
	})
	t.Run("sector at cap", func(t *testing.T) {
		a := pos("A", 5000, 10000, 10000)
		o1 := pos("O1", 18000, 10000, 10000)
		o2 := pos("O2", 17000, 10000, 10000)
		scores := map[string]Score{"A": score("A", 80, 60), "O1": score("O1", 65, 60), "O2": score("O2", 65, 60)}
		pa := Recommend(summary(300_000_000, 0, a, o1, o2), scores, nil, c)
		if got := adviceFor(t, pa, "A"); got.Action != ActionHold {
			t.Fatalf("A: %+v", got)
		}
	})
}

func TestRecommendMarginSuppressesAllAdds(t *testing.T) {
	a := pos("A", 5000, 10000, 10000)
	b := pos("B", 5000, 10000, 10000)
	scores := map[string]Score{"A": score("A", 80, 60), "B": score("B", 90, 60)}
	pa := Recommend(summary(300_000_000, 250_000_000, a, b), scores, nil, testConfig())
	for _, adv := range pa.Holdings {
		if adv.Action == ActionAdd {
			t.Errorf("%s: ADD under high margin", adv.Symbol)
		}
		if !hasText(adv.Notes, "reduce margin first") {
			t.Errorf("%s notes %v", adv.Symbol, adv.Notes)
		}
	}
	if !hasText(pa.Messages, "all ADDs suppressed; reduce margin first") {
		t.Errorf("messages %v", pa.Messages)
	}
}

func TestRecommendNotScoredAndHold(t *testing.T) {
	etf := pos("E1VFVN30", 30000, 10000, 10000) // 30%: over the hard cap but never trimmed
	etf.Scored = false
	etf.SecType = "008"
	stock := pos("FPT", 5050, 10000, 10000)
	scores := map[string]Score{"FPT": score("FPT", 72, 40)}
	pa := Recommend(summary(300_000_000, 0, etf, stock), scores, nil, testConfig())

	e := adviceFor(t, pa, "E1VFVN30")
	if e.Action != ActionHold || !hasText(e.Notes, NoteNotScored) || e.Reason != NoteNotScored {
		t.Errorf("ETF: %+v", e)
	}
	if !approx(pa.SectorWeights[SectorFund], 0.30) {
		t.Errorf("fund weight %v", pa.SectorWeights)
	}
	if hasText(pa.Messages, "fund 30.0% (above") {
		t.Errorf("fund flagged against the sector cap: %v", pa.Messages)
	}

	h := adviceFor(t, pa, "FPT")
	if h.Action != ActionHold || h.Rule != "HOLD: no action rule triggered" {
		t.Fatalf("FPT: %+v", h)
	}
	want := "strongest: VOL 18.0% (score 100), ROE 22.0% (score 100); weakest: PE 25.00 (score 20)"
	if h.Reason != want {
		t.Errorf("reason %q, want %q", h.Reason, want)
	}
	if h.OddLot != 50 || h.Breakeven != 10031 {
		t.Errorf("odd lot %d breakeven %d", h.OddLot, h.Breakeven)
	}

	// An ETF typed by research but not by secType is also skipped.
	notScored := Score{Symbol: "FUEVFVND", Tier: TierNotScored, Notes: []string{NoteNotScored}}
	fund := pos("FUEVFVND", 1000, 10000, 10000)
	pa = Recommend(summary(300_000_000, 0, fund), map[string]Score{"FUEVFVND": notScored}, nil, testConfig())
	if a := adviceFor(t, pa, "FUEVFVND"); a.Action != ActionHold || !hasText(a.Notes, NoteNotScored) {
		t.Errorf("research-typed fund: %+v", a)
	}
}

func TestPortfolioMessages(t *testing.T) {
	five := []portfolio.Position{
		pos("A", 1000, 10000, 10000), pos("B", 1000, 10000, 10000), pos("C", 1000, 10000, 10000),
		pos("D", 1000, 10000, 10000), pos("E", 1000, 10000, 10000),
	}
	intraday := time.Date(2026, 10, 13, 3, 0, 0, 0, time.UTC) // 10:00 in Vietnam, Tuesday
	cases := []struct {
		name    string
		sum     portfolio.Summary
		now     time.Time
		want    string
		notWant string
	}{
		{"cash low", summary(40_000_000, 0, five...), testNow, "Cash buffer below 5%", ""},
		{"cash at min", summary(50_000_000, 0, five...), testNow, "", "Cash buffer"},
		{"cash high", summary(410_000_000, 0, five...), testNow, "consider deploying into Stable candidates in 2-3 tranches", ""},
		{"cash at high", summary(400_000_000, 0, five...), testNow, "", "tranches"},
		{"margin", summary(300_000_000, 100_000_000, five...), testNow, "Margin debt 100,000,000 VND (10.0% of NAV)", "suppressed"},
		{"margin high", summary(300_000_000, 210_000_000, five...), testNow, "reduce margin first", ""},
		{"no margin", summary(300_000_000, 0, five...), testNow, "", "Margin debt"},
		{"sector weights", summary(300_000_000, 0, five...), testNow, "Sector weights: general 5.0%.", ""},
		{"concentrated", func() portfolio.Summary { s := summary(300_000_000, 0, five...); s.EffectiveN = 3.9; return s }(), testNow, "concentrated: effectiveN 3.9 < 4", ""},
		{"not concentrated", func() portfolio.Summary { s := summary(300_000_000, 0, five...); s.EffectiveN = 4; return s }(), testNow, "", "concentrated"},
		{"under-diversified", summary(300_000_000, 0, five[:4]...), testNow, "under-diversified: 4 holdings < 5", ""},
		{"diversified", summary(300_000_000, 0, five...), testNow, "", "under-diversified"},
		{"odd lots", summary(300_000_000, 0, pos("A", 1050, 10000, 10000), pos("B", 7, 10000, 10000)), testNow, "Odd lots (sell on the odd-lot board): A 50, B 7.", ""},
		{"intraday", summary(300_000_000, 0, five...), intraday, "Prices are intraday, not closing.", ""},
		{"after close", summary(300_000_000, 0, five...), testNow, "", "intraday"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := testConfig()
			c.Now = tc.now
			scores := map[string]Score{}
			for _, p := range tc.sum.Positions {
				scores[p.Symbol] = score(p.Symbol, 65, 60)
			}
			pa := Recommend(tc.sum, scores, nil, c)
			if tc.want != "" && !hasText(pa.Messages, tc.want) {
				t.Errorf("messages %v, want %q", pa.Messages, tc.want)
			}
			if tc.notWant != "" && hasText(pa.Messages, tc.notWant) {
				t.Errorf("messages %v, do not want %q", pa.Messages, tc.notWant)
			}
		})
	}
}

func TestGroupInt(t *testing.T) {
	cases := map[int64]string{0: "0", 999: "999", 1000: "1,000", 1234500: "1,234,500", -73852000: "-73,852,000"}
	for in, want := range cases {
		if got := groupInt(in); got != want {
			t.Errorf("groupInt(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRecommendSectorCapAcrossAdds(t *testing.T) {
	// Three banks at 10% each leave 10% of room below the 40% cap; the adds
	// together must stay inside it.
	c := testConfig()
	var ps []portfolio.Position
	scores := map[string]Score{}
	sectors := map[string]Sector{}
	for _, sym := range []string{"AAA", "BBB", "CCC"} {
		ps = append(ps, pos(sym, 1000, 100000, 100000))
		scores[sym] = score(sym, 80, 80)
		sectors[sym] = SectorBank
	}
	pa := Recommend(summary(700_000_000, 0, ps...), scores, sectors, c)
	added := 0.0
	nAdd := 0
	for _, a := range pa.Holdings {
		if a.Action == ActionAdd {
			nAdd++
			added += float64(a.Qty*100000) / testNAV
		}
	}
	if nAdd == 0 {
		t.Fatalf("no ADD suggested: %+v", pa.Holdings)
	}
	if after := pa.SectorWeights[SectorBank] + added; above(after, c.SectorCap) {
		t.Errorf("bank sector after adds %.3f > cap %.2f (%+v)", after, c.SectorCap, pa.Holdings)
	}
	// 100m of room buys 9 lots with fee; the 10m left is under one lot.
	if nAdd != 1 || !approx(added, 0.09) {
		t.Errorf("%d adds use %.3f of NAV, want one add of 9%%", nAdd, added)
	}
	for _, sym := range []string{"BBB", "CCC"} {
		if a := adviceFor(t, pa, sym); a.Action != ActionHold || !hasText(a.Notes, "under one lot of room") {
			t.Errorf("%s after the room is used: %+v", sym, a)
		}
	}
}

func TestRecommendTrimCoversLargestExcess(t *testing.T) {
	// AAA is over the hard cap (excess 100m to 20%) and also carries the
	// whole 140m sector excess; one sale must bring the sector back to 40%.
	a := pos("AAA", 3000, 100000, 100000) // 30%
	b := pos("BBB", 2400, 100000, 100000) // 24%
	scores := map[string]Score{"AAA": score("AAA", 70, 60), "BBB": score("BBB", 80, 60)}
	sectors := map[string]Sector{"AAA": SectorBank, "BBB": SectorBank}
	pa := Recommend(summary(300_000_000, 0, a, b), scores, sectors, testConfig())
	got := adviceFor(t, pa, "AAA")
	if got.Action != ActionTrim || got.Qty != 1400 {
		t.Fatalf("AAA: %+v, want TRIM 1400", got)
	}
	if !hasText([]string{got.Rule}, "hard cap 25%") || !hasText(got.Secondary, "sector bank above cap 40%") {
		t.Errorf("rules %q / %v", got.Rule, got.Secondary)
	}
}

func TestRecommendSectorTrimSkipsReview(t *testing.T) {
	// AAA has an invalid score and becomes REVIEW, so the 18% sector excess
	// must land on the valid members, lowest score first.
	a := pos("AAA", 2000, 100000, 100000) // 20%
	b := pos("BBB", 1900, 100000, 100000) // 19%
	cc := pos("CCC", 1900, 100000, 100000)
	invalid := score("AAA", 90, 60)
	invalid.Valid = false
	scores := map[string]Score{"AAA": invalid, "BBB": score("BBB", 80, 60), "CCC": score("CCC", 82, 60)}
	sectors := map[string]Sector{"AAA": SectorBank, "BBB": SectorBank, "CCC": SectorBank}
	pa := Recommend(summary(300_000_000, 0, a, b, cc), scores, sectors, testConfig())
	if got := adviceFor(t, pa, "AAA"); got.Action != ActionReview || got.Qty != 0 {
		t.Errorf("AAA: %+v", got)
	}
	bb, ccc := adviceFor(t, pa, "BBB"), adviceFor(t, pa, "CCC")
	if bb.Action != ActionTrim || bb.Qty != 1800 {
		t.Errorf("BBB: %+v, want TRIM 1800", bb)
	}
	if ccc.Action != ActionHold {
		t.Errorf("CCC: %+v", ccc)
	}
}
