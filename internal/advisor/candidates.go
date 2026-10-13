package advisor

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/miti99dev/stock-advisor/internal/portfolio"
)

// NoteWatchlistOnly is the candidate note when deployable cash cannot buy
// one lot of the cheapest candidate.
const NoteWatchlistOnly = "For watchlist only."

// Candidate is a stable stock the portfolio does not hold. RankScore orders
// candidates and never changes the displayed score. LotCost is one board lot
// with the buy fee; MaxAlloc is the indicative maximum allocation in VND.
type Candidate struct {
	Score     Score
	Inputs    Inputs
	RankScore float64
	Reasons   []string
	LotCost   int64
	MaxAlloc  int64
}

// Candidates screens and ranks the universe minus held symbols. It returns
// at most c.CandidateCount candidates, at most c.CandidatePerSector per
// sector, and a note for the whole list ("" or NoteWatchlistOnly).
func Candidates(universe []Inputs, held map[string]bool, sum portfolio.Summary, sectorW map[Sector]float64, c Config) ([]Candidate, string) {
	nav := float64(sum.NAV)
	deployable := float64(sum.Cash) - c.CashMin*nav
	maxAlloc := int64(math.Max(0, math.Floor(math.Min(c.SingleSoftCap*nav, deployable/math.Max(1, c.CandidateAllocSplit)))))

	var pool []Candidate
	seen := map[string]bool{}
	for _, in := range universe {
		sym := strings.ToUpper(strings.TrimSpace(in.Symbol))
		if sym == "" || seen[sym] || held[sym] || held[in.Symbol] || !IsStockType(in.Type, c) {
			continue
		}
		seen[sym] = true
		s := ScoreStock(in, c)
		notes, ok := screen(in, s, sectorW, c)
		if !ok {
			continue
		}
		s.Notes = append(s.Notes, notes...)
		rank := float64(s.Value)
		if below(sectorW[s.Sector], c.CandidateLowSectorWeight) {
			rank += c.CandidateLowSectorBonus
		}
		pool = append(pool, Candidate{
			Score:     s,
			Inputs:    in,
			RankScore: rank,
			Reasons:   candidateReasons(s),
			LotCost:   BuyCost(BoardLot, int64(math.Round(in.Price)), c.FeeRate),
			MaxAlloc:  maxAlloc,
		})
	}
	sort.SliceStable(pool, func(i, j int) bool {
		a, b := pool[i], pool[j]
		if a.RankScore != b.RankScore {
			return a.RankScore > b.RankScore
		}
		if a.Score.Value != b.Score.Value {
			return a.Score.Value > b.Score.Value
		}
		if a.Score.Coverage != b.Score.Coverage {
			return a.Score.Coverage > b.Score.Coverage
		}
		return a.Score.Symbol < b.Score.Symbol
	})

	var out []Candidate
	perSector := map[Sector]int{}
	for _, cand := range pool {
		if len(out) >= c.CandidateCount {
			break
		}
		if perSector[cand.Score.Sector] >= c.CandidatePerSector {
			continue
		}
		perSector[cand.Score.Sector]++
		out = append(out, cand)
	}

	note := ""
	if len(out) > 0 {
		cheapest := out[0].LotCost
		for _, cand := range out[1:] {
			cheapest = min(cheapest, cand.LotCost)
		}
		if deployable < float64(cheapest) {
			note = NoteWatchlistOnly
		}
	}
	return out, note
}

// screen applies the candidate filters. It returns notes for data it could
// not check and false when the candidate fails a filter.
func screen(in Inputs, s Score, sectorW map[Sector]float64, c Config) ([]string, bool) {
	var notes []string
	if !s.Valid || float64(s.Value) < c.CandidateMinScore || s.Coverage < c.CandidateMinCoverage {
		return nil, false
	}
	if in.VOL == nil || above(*in.VOL, c.CandidateMaxVol) {
		return nil, false
	}
	if len(s.RedFlags) > 0 {
		return nil, false
	}
	if in.Listed.IsZero() {
		notes = append(notes, "listing date unknown")
	} else if years := c.now().Sub(in.Listed).Hours() / 24 / 365.25; years < c.CandidateMinListedYears {
		return nil, false
	}
	switch n := min(len(in.ProfitYears), c.CandidateProfitYears); {
	case n == 0:
		notes = append(notes, "profit history unknown")
	default:
		for _, p := range in.ProfitYears[:n] {
			if p <= 0 {
				return nil, false
			}
		}
		if n < c.CandidateProfitYears {
			notes = append(notes, fmt.Sprintf("only %d years of profit history", n))
		}
	}
	if worst, ok := deWorst(s.Sector, c); ok && in.DE != nil && *in.DE >= worst {
		return nil, false
	}
	if s.Sector == SectorBank && (in.NPL == nil || above(*in.NPL, c.CandidateMaxNPL)) {
		return nil, false
	}
	if !below(sectorW[s.Sector], c.SectorCap) {
		return nil, false
	}
	return notes, true
}

// deWorst is the DE threshold that scores 0 for a sector, when the sector
// scores DE at all.
func deWorst(sec Sector, c Config) (float64, bool) {
	switch sec {
	case SectorGeneral:
		return c.Scoring.DE.Worst, true
	case SectorRealEstate:
		return c.Scoring.DERealEstate.Worst, true
	case SectorUtilities:
		return c.Scoring.DEUtilities.Worst, true
	}
	return 0, false
}

// candidateReasons are the three largest metric contributions.
func candidateReasons(s Score) []string {
	var used []MetricScore
	for _, m := range s.Metrics {
		if !m.Dropped {
			used = append(used, m)
		}
	}
	top := topContributions(used, 3)
	out := make([]string, len(top))
	for i, m := range top {
		out[i] = fmt.Sprintf("%s %s (score %.0f)", m.Name, FormatRaw(m.Name, m.Raw), m.Score)
	}
	return out
}
