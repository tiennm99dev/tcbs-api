package advisor

import (
	"testing"
	"time"
)

func TestScreen(t *testing.T) {
	c := testConfig()
	good := func() (Inputs, Score) {
		in := perfectInputs("AAA")
		return in, ScoreStock(in, c)
	}
	cases := []struct {
		name    string
		mutate  func(*Inputs, *Score, map[Sector]float64)
		ok      bool
		note    string
		notNote string
	}{
		{"baseline", func(*Inputs, *Score, map[Sector]float64) {}, true, "", "unknown"},
		{"score 70", func(_ *Inputs, s *Score, _ map[Sector]float64) { s.Value = 70 }, true, "", ""},
		{"score 69", func(_ *Inputs, s *Score, _ map[Sector]float64) { s.Value = 69 }, false, "", ""},
		{"invalid", func(_ *Inputs, s *Score, _ map[Sector]float64) { s.Valid = false }, false, "", ""},
		{"coverage 80", func(_ *Inputs, s *Score, _ map[Sector]float64) { s.Coverage = 80 }, true, "", ""},
		{"coverage 79", func(_ *Inputs, s *Score, _ map[Sector]float64) { s.Coverage = 79 }, false, "", ""},
		{"vol 0.40", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.VOL = f(0.40) }, true, "", ""},
		{"vol 0.41", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.VOL = f(0.41) }, false, "", ""},
		{"vol missing", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.VOL = nil }, false, "", ""},
		{"red flag", func(_ *Inputs, s *Score, _ map[Sector]float64) { s.RedFlags = []string{"negative equity"} }, false, "", ""},
		{"listed 1 year", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.Listed = testNow.AddDate(-1, 0, 0) }, false, "", ""},
		{"listed 2 years", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.Listed = testNow.AddDate(-2, 0, -1) }, true, "", ""},
		{"listing date unknown", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.Listed = time.Time{} }, true, "listing date unknown", ""},
		{"recent loss", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.ProfitYears = []float64{10, -1, 10} }, false, "", ""},
		{"zero profit", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.ProfitYears = []float64{10, 10, 0} }, false, "", ""},
		{"old loss ignored", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.ProfitYears = []float64{10, 10, 10, -5} }, true, "", ""},
		{"profit unknown", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.ProfitYears = nil }, true, "profit history unknown", ""},
		{"short profit history", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.ProfitYears = []float64{10, 10} }, true, "only 2 years of profit history", ""},
		{"de 1.49", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.DE = f(1.49) }, true, "", ""},
		{"de 1.5", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.DE = f(1.5) }, false, "", ""},
		{"de missing", func(in *Inputs, _ *Score, _ map[Sector]float64) { in.DE = nil }, true, "", ""},
		{"real estate de 1.9", func(in *Inputs, s *Score, _ map[Sector]float64) { s.Sector = SectorRealEstate; in.DE = f(1.9) }, true, "", ""},
		{"real estate de 2.0", func(in *Inputs, s *Score, _ map[Sector]float64) { s.Sector = SectorRealEstate; in.DE = f(2.0) }, false, "", ""},
		{"utilities de 2.4", func(in *Inputs, s *Score, _ map[Sector]float64) { s.Sector = SectorUtilities; in.DE = f(2.4) }, true, "", ""},
		{"utilities de 2.5", func(in *Inputs, s *Score, _ map[Sector]float64) { s.Sector = SectorUtilities; in.DE = f(2.5) }, false, "", ""},
		{"bank ignores de", func(in *Inputs, s *Score, _ map[Sector]float64) {
			s.Sector = SectorBank
			in.DE = f(9)
			in.NPL = f(0.01)
		}, true, "", ""},
		{"bank npl 3%", func(in *Inputs, s *Score, _ map[Sector]float64) { s.Sector = SectorBank; in.NPL = f(0.03) }, true, "", ""},
		{"bank npl 3.1%", func(in *Inputs, s *Score, _ map[Sector]float64) { s.Sector = SectorBank; in.NPL = f(0.031) }, false, "", ""},
		{"bank npl missing", func(in *Inputs, s *Score, _ map[Sector]float64) { s.Sector = SectorBank; in.NPL = nil }, false, "", ""},
		{"sector weight 39%", func(_ *Inputs, _ *Score, w map[Sector]float64) { w[SectorGeneral] = 0.39 }, true, "", ""},
		{"sector weight 40%", func(_ *Inputs, _ *Score, w map[Sector]float64) { w[SectorGeneral] = 0.40 }, false, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in, s := good()
			w := map[Sector]float64{}
			tc.mutate(&in, &s, w)
			notes, ok := screen(in, s, w, c)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v (notes %v)", ok, tc.ok, notes)
			}
			if tc.note != "" && !hasText(notes, tc.note) {
				t.Errorf("notes %v, want %q", notes, tc.note)
			}
			if tc.notNote != "" && hasText(notes, tc.notNote) {
				t.Errorf("notes %v, do not want %q", notes, tc.notNote)
			}
		})
	}
}

func TestCandidates(t *testing.T) {
	c := testConfig()
	sum := summary(300_000_000, 0)

	t.Run("excludes held, funds and duplicates", func(t *testing.T) {
		etf := perfectInputs("ETF1")
		etf.Type = "ETF"
		universe := []Inputs{perfectInputs("FPT"), perfectInputs("HPG"), etf, perfectInputs("hpg")}
		got, note := Candidates(universe, map[string]bool{"FPT": true}, sum, nil, c)
		if len(got) != 1 || got[0].Score.Symbol != "HPG" || note != "" {
			t.Fatalf("got %+v note %q", got, note)
		}
		cand := got[0]
		if cand.LotCost != 5_005_000 || cand.MaxAlloc != 83_333_333 || len(cand.Reasons) != 3 || cand.RankScore != 105 {
			t.Errorf("candidate %+v", cand)
		}
	})

	t.Run("filters reject", func(t *testing.T) {
		volatile := perfectInputs("VOLA")
		volatile.VOL = f(0.45)
		flagged := perfectInputs("FLAG")
		flagged.NegEquity = true
		got, _ := Candidates([]Inputs{volatile, flagged}, nil, sum, nil, c)
		if len(got) != 0 {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("notes for unknown data", func(t *testing.T) {
		in := perfectInputs("NEW")
		in.ProfitYears = nil
		in.EarnCons = f(100)
		got, _ := Candidates([]Inputs{in}, nil, sum, nil, c)
		if len(got) != 1 || !hasText(got[0].Score.Notes, "profit history unknown") {
			t.Fatalf("got %+v", got)
		}
	})

	t.Run("per-sector limit and ranking", func(t *testing.T) {
		a, b, d := perfectInputs("AAA"), perfectInputs("BBB"), perfectInputs("CCC")
		b.DY = f(0.042) // 70 + 10 bonus = 80 → value 98
		bank := perfectBank("VCB")
		bank.DY = f(0.042)
		universe := []Inputs{d, bank, b, a}

		got, _ := Candidates(universe, nil, sum, map[Sector]float64{SectorGeneral: 0.20}, c)
		syms := symbols(got)
		// Bank 98 + 5 bonus outranks general 100 without one; one general dropped.
		if want := []string{"VCB", "AAA", "CCC"}; !equal(syms, want) {
			t.Errorf("order %v, want %v", syms, want)
		}

		got, _ = Candidates(universe, nil, sum, map[Sector]float64{SectorGeneral: 0.10}, c)
		if want := []string{"AAA", "CCC", "VCB"}; !equal(symbols(got), want) {
			t.Errorf("order %v, want %v", symbols(got), want)
		}

		c2 := c
		c2.CandidateCount = 2
		got, _ = Candidates(universe, nil, sum, nil, c2)
		if want := []string{"AAA", "CCC"}; !equal(symbols(got), want) {
			t.Errorf("top 2 %v, want %v", symbols(got), want)
		}
	})

	t.Run("sector at cap excluded", func(t *testing.T) {
		got, _ := Candidates([]Inputs{perfectInputs("AAA"), perfectBank("VCB")}, nil, sum, map[Sector]float64{SectorGeneral: 0.41}, c)
		if want := []string{"VCB"}; !equal(symbols(got), want) {
			t.Errorf("got %v", symbols(got))
		}
	})

	t.Run("watchlist only", func(t *testing.T) {
		low := summary(52_000_000, 0) // deployable 2m < one 5,005,000 lot
		got, note := Candidates([]Inputs{perfectInputs("AAA")}, nil, low, nil, c)
		if note != NoteWatchlistOnly || len(got) != 1 || got[0].MaxAlloc != 666_666 {
			t.Errorf("got %+v note %q", got, note)
		}
		got, note = Candidates([]Inputs{perfectInputs("AAA")}, nil, summary(10_000_000, 0), nil, c)
		if note != NoteWatchlistOnly || got[0].MaxAlloc != 0 {
			t.Errorf("negative deployable: %+v note %q", got, note)
		}
		if got, note := Candidates(nil, nil, low, nil, c); len(got) != 0 || note != "" {
			t.Errorf("empty universe: %+v %q", got, note)
		}
	})
}

func symbols(cs []Candidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Score.Symbol
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
