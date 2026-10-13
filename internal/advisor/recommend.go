package advisor

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/miti99dev/stock-advisor/internal/portfolio"
)

// Action is the suggested action for one holding.
type Action string

// Actions, in rule priority order.
const (
	ActionReview Action = "REVIEW"
	ActionTrim   Action = "TRIM"
	ActionAdd    Action = "ADD"
	ActionHold   Action = "HOLD"
)

// Advice is the action for one holding. Rule cites the primary rule with its
// numbers; Secondary lists every other rule that also triggered. Qty and
// Lots are the suggested trade in whole board lots; EstCash is the cash
// impact after fee and tax (+ proceeds, − cost). Breakeven is 0 when the
// cost is unknown.
type Advice struct {
	Symbol    string
	Action    Action
	Rule      string
	Reason    string
	Secondary []string
	Notes     []string
	Lots, Qty int64
	EstCash   int64
	Breakeven int64
	OddLot    int64
}

// PortfolioAdvice is the advice for a whole portfolio.
type PortfolioAdvice struct {
	Messages      []string
	Holdings      []Advice
	SectorWeights map[Sector]float64
}

// floorEps keeps a float result such as 1999.9999999999998 from losing a
// whole lot when it is floored.
const floorEps = 1e-6

// ratioEps absorbs float error in weights and ratios, so 0.20 − 0.05 still
// equals a 15% weight.
const ratioEps = 1e-9

// above reports x > limit beyond float error.
func above(x, limit float64) bool { return x > limit+ratioEps }

// below reports x < limit beyond float error.
func below(x, limit float64) bool { return x < limit-ratioEps }

// trimRule is one triggered TRIM rule and the value it would sell.
type trimRule struct {
	rule   string
	excess float64 // VND
}

// holdingState is what the rules need about one position.
type holdingState struct {
	pos      portfolio.Position
	sector   Sector
	score    Score
	hasScore bool
	scored   bool
	review   []string
	trims    []trimRule
	addOK    bool
	addWhy   string // why an otherwise eligible ADD was suppressed
}

// Recommend applies the portfolio rules to every position of sum. scores and
// sectors are keyed by symbol; a symbol missing from sectors takes its
// score's sector, then general.
func Recommend(sum portfolio.Summary, scores map[string]Score, sectors map[string]Sector, c Config) PortfolioAdvice {
	nav := float64(sum.NAV)
	out := PortfolioAdvice{SectorWeights: map[Sector]float64{}}

	states := make([]*holdingState, 0, len(sum.Positions))
	for _, p := range sum.Positions {
		st := &holdingState{pos: p}
		st.score, st.hasScore = scores[p.Symbol]
		st.scored = p.Scored && !(st.hasScore && st.score.Tier == TierNotScored)
		switch {
		case !st.scored:
			st.sector = SectorFund
		case sectors[p.Symbol] != "":
			st.sector = sectors[p.Symbol]
		case st.hasScore && st.score.Sector != "":
			st.sector = st.score.Sector
		default:
			st.sector = SectorGeneral
		}
		out.SectorWeights[st.sector] += p.Weight
		if st.scored {
			st.review = reviewRules(st, c)
		}
		states = append(states, st)
	}

	sectorTrim := sectorTrims(states, out.SectorWeights, nav, c)
	marginBlocked := above(sum.MarginRatio, c.MarginMax)

	nAdd := 0
	for _, st := range states {
		if !st.scored {
			continue
		}
		st.trims = trimRules(st, sectorTrim[st.pos.Symbol], nav, c)
		if len(st.review) == 0 && len(st.trims) == 0 {
			st.addOK = addEligible(st, out.SectorWeights[st.sector], sum, c)
			if st.addOK && marginBlocked {
				st.addOK = false
				st.addWhy = fmt.Sprintf("ADD suppressed: margin %s of NAV > %s; reduce margin first", pct(sum.MarginRatio), pct0(c.MarginMax))
			}
			if st.addOK {
				nAdd++
			}
		}
	}

	deployable := float64(sum.Cash) - c.CashMin*nav
	// sectorRoom is each sector's VND room below the cap, shrunk by every
	// ADD sized so far so that several adds in one sector share it.
	sectorRoom := map[Sector]float64{}
	for sec, w := range out.SectorWeights {
		sectorRoom[sec] = (c.SectorCap - w) * nav
	}
	for _, st := range states {
		out.Holdings = append(out.Holdings, adviseHolding(st, sectorRoom, nav, deployable, nAdd, c))
	}
	out.Messages = portfolioMessages(sum, states, out.SectorWeights, c)
	return out
}

// sectorTrims spreads each over-cap sector's excess over its lowest-score
// holdings first, so the sector falls back to the cap. Holdings headed for
// REVIEW never get a trim quantity, so they take the excess only after every
// other member is used up.
func sectorTrims(states []*holdingState, sw map[Sector]float64, nav float64, c Config) map[string]float64 {
	out := map[string]float64{}
	for sec, w := range sw {
		if sec == SectorFund || !above(w, c.SectorCap) {
			continue
		}
		var members []*holdingState
		for _, st := range states {
			if st.sector == sec && st.pos.MarketValue > 0 {
				members = append(members, st)
			}
		}
		sort.SliceStable(members, func(i, j int) bool {
			a, b := members[i], members[j]
			if ar, br := len(a.review) > 0, len(b.review) > 0; ar != br {
				return !ar
			}
			av, bv := a.hasScore && a.score.Valid, b.hasScore && b.score.Valid
			if av != bv {
				return !av // invalid scores rank lowest
			}
			if a.score.Value != b.score.Value {
				return a.score.Value < b.score.Value
			}
			return a.pos.Symbol < b.pos.Symbol
		})
		remaining := (w - c.SectorCap) * nav
		for _, st := range members {
			if remaining <= 0 {
				break
			}
			take := math.Min(remaining, float64(st.pos.MarketValue))
			out[st.pos.Symbol] = take
			remaining -= take
		}
	}
	return out
}

func reviewRules(st *holdingState, c Config) []string {
	var r []string
	if !st.hasScore {
		return []string{"REVIEW: no score data"}
	}
	s := st.score
	for _, f := range s.RedFlags {
		r = append(r, "REVIEW: red flag: "+f)
	}
	if !s.Valid {
		r = append(r, fmt.Sprintf("REVIEW: score invalid (coverage %.0f, needs %.0f and price data)", s.Coverage, c.Scoring.MinCoverage))
		return r
	}
	v := float64(s.Value)
	pnl, known := pnlOf(st.pos)
	if v < c.ReviewScore {
		r = append(r, fmt.Sprintf("REVIEW: score %d < %.0f", s.Value, c.ReviewScore))
	}
	if known && !above(pnl, c.ReviewLossPct) && v < c.WeakScore {
		r = append(r, fmt.Sprintf("REVIEW: P&L %s ≤ %s and score %d < %.0f; re-check the thesis, consider a stop-loss rule", pct(pnl), pct0(c.ReviewLossPct), s.Value, c.WeakScore))
	}
	if known && !above(pnl, c.WeakLossPct) && v < c.ReviewWeakLoss {
		r = append(r, fmt.Sprintf("REVIEW: P&L %s ≤ %s and score %d < %.0f", pct(pnl), pct0(c.WeakLossPct), s.Value, c.ReviewWeakLoss))
	}
	return r
}

func trimRules(st *holdingState, sectorExcess, nav float64, c Config) []trimRule {
	var r []trimRule
	p := st.pos
	w := p.Weight
	if above(w, c.SingleHardCap) {
		r = append(r, trimRule{fmt.Sprintf("TRIM: weight %s > hard cap %s; trim to %s", pct(w), pct0(c.SingleHardCap), pct0(c.SingleSoftCap)), (w - c.SingleSoftCap) * nav})
	}
	if sectorExcess > 0 {
		r = append(r, trimRule{fmt.Sprintf("TRIM: sector %s above cap %s; trim the lowest-score holdings back to the cap", st.sector, pct0(c.SectorCap)), sectorExcess})
	}
	if !st.hasScore || !st.score.Valid {
		return r
	}
	v := float64(st.score.Value)
	if pnl, known := pnlOf(p); known && v < c.WeakScore && !below(pnl, c.TakeProfitPct) {
		r = append(r, trimRule{fmt.Sprintf("TRIM: score %d < %.0f and P&L %s ≥ %s; sell %s", st.score.Value, c.WeakScore, pct(pnl), pct0(c.TakeProfitPct), fraction(c.TakeProfitPart)), float64(p.MarketValue) * c.TakeProfitPart})
	}
	if v >= c.ReviewScore && v < c.WeakScore && above(w, c.WeakMaxWeight) {
		r = append(r, trimRule{fmt.Sprintf("TRIM: score %d in [%.0f, %.0f) and weight %s > %s; trim to %s", st.score.Value, c.ReviewScore, c.WeakScore, pct(w), pct0(c.WeakMaxWeight), pct0(c.WeakMaxWeight)), (w - c.WeakMaxWeight) * nav})
	}
	return r
}

// addEligible checks every ADD condition except the amount. A sector
// already at the cap fails here; the add amount is later capped to the
// sector's remaining room.
func addEligible(st *holdingState, sectorW float64, sum portfolio.Summary, c Config) bool {
	s := st.score
	v := float64(s.Value)
	if !s.Valid || v < c.AddMinScore {
		return false
	}
	if !below(st.pos.Weight, c.SingleSoftCap-c.AddRoomBuffer) {
		return false
	}
	if !below(sectorW, c.SectorCap) {
		return false
	}
	if !above(sum.CashRatio, c.CashMin) {
		return false
	}
	if val, ok := s.Pillars[PillarValuation]; !ok || val < c.AddMinValue {
		return false
	}
	if pnl, known := pnlOf(st.pos); known && !above(pnl, c.WeakLossPct) && v < c.AddStrongScore {
		return false
	}
	if st.pos.Price <= 0 {
		return false
	}
	return true
}

func adviseHolding(st *holdingState, sectorRoom map[Sector]float64, nav, deployable float64, nAdd int, c Config) Advice {
	p := st.pos
	a := Advice{
		Symbol:    p.Symbol,
		OddLot:    OddLot(p.Qty),
		Breakeven: Breakeven(p.AvgCost, c.FeeRate, c.SellTax),
	}
	if !st.scored {
		a.Action = ActionHold
		a.Rule = "HOLD: " + NoteNotScored
		a.Reason = NoteNotScored
		a.Notes = []string{NoteNotScored}
		return a
	}
	trimTexts := make([]string, len(st.trims))
	for i, t := range st.trims {
		trimTexts[i] = t.rule
	}
	switch {
	case len(st.review) > 0:
		a.Action = ActionReview
		a.Rule = st.review[0]
		a.Reason = "re-check the holding before acting"
		a.Secondary = append(append([]string{}, st.review[1:]...), trimTexts...)
		return a
	case len(st.trims) > 0:
		a.Action = ActionTrim
		a.Rule = st.trims[0].rule
		a.Secondary = trimTexts[1:]
		trimQty(&a, p, largestExcess(st.trims), c)
		a.Reason = fmt.Sprintf("sell %d lots (%s shares), est. +%s VND after fee and tax", a.Lots, groupInt(a.Qty), groupInt(a.EstCash))
		return a
	case st.addOK:
		if addQty(&a, st, sectorRoom, nav, deployable, nAdd, c) {
			return a
		}
	}
	a.Action = ActionHold
	a.Rule = "HOLD: no action rule triggered"
	a.Reason = holdReason(st.score)
	if st.addWhy != "" {
		a.Notes = append(a.Notes, st.addWhy)
	}
	return a
}

// largestExcess is the biggest excess among the triggered TRIM rules, so one
// sale satisfies all of them.
func largestExcess(trims []trimRule) float64 {
	m := 0.0
	for _, t := range trims {
		m = math.Max(m, t.excess)
	}
	return m
}

// trimQty fills the lots to sell for an excess value: whole lots, capped at
// the sellable quantity.
func trimQty(a *Advice, p portfolio.Position, excess float64, c Config) {
	if p.Price <= 0 {
		a.Notes = append(a.Notes, "no price: trim quantity not computed")
		return
	}
	want := FloorLot(int64(math.Floor(excess/float64(p.Price) + floorEps)))
	if want == 0 {
		a.Notes = append(a.Notes, "the excess is under one board lot")
	}
	qty := want
	if sellable := FloorLot(p.Sellable); qty > sellable {
		qty = sellable
		if p.Receiving > 0 {
			n := min(p.Receiving, want-sellable)
			a.Notes = append(a.Notes, fmt.Sprintf("%s shares settle T+%d; trim the remainder after settlement", groupInt(n), SettlementDays))
		} else {
			a.Notes = append(a.Notes, fmt.Sprintf("capped at the sellable %s shares", groupInt(p.Sellable)))
		}
	}
	a.Qty = qty
	a.Lots = qty / BoardLot
	a.EstCash = SellProceeds(qty, p.Price, c.FeeRate, c.SellTax)
}

// addQty sizes an ADD. It returns false, with a note when cash is short,
// when the result is under one lot. An ADD uses up its value of the sector's
// room in sectorRoom.
func addQty(a *Advice, st *holdingState, sectorRoom map[Sector]float64, nav, deployable float64, nAdd int, c Config) bool {
	p := st.pos
	amount := math.Min((c.SingleSoftCap-p.Weight)*nav, deployable/float64(max(1, nAdd)))
	room := sectorRoom[st.sector]
	unit := float64(p.Price) * (1 + c.FeeRate) * BoardLot
	if room < amount {
		amount = room
		if amount < unit {
			st.addWhy = fmt.Sprintf("sector %s has under one lot of room below the %s cap", st.sector, pct0(c.SectorCap))
			return false
		}
	}
	lots := int64(math.Floor(amount/unit + floorEps))
	if lots < 1 {
		st.addWhy = "insufficient cash for one lot"
		return false
	}
	a.Action = ActionAdd
	a.Rule = fmt.Sprintf("ADD: score %d ≥ %.0f, weight %s < %s, valuation %.0f ≥ %.0f", st.score.Value, c.AddMinScore, pct(p.Weight), pct0(c.SingleSoftCap-c.AddRoomBuffer), st.score.Pillars[PillarValuation], c.AddMinValue)
	a.Lots = lots
	a.Qty = lots * BoardLot
	a.EstCash = -BuyCost(a.Qty, p.Price, c.FeeRate)
	sectorRoom[st.sector] = room - float64(a.Qty*p.Price)
	a.Reason = fmt.Sprintf("buy %d lots (%s shares), est. %s VND with fee", a.Lots, groupInt(a.Qty), groupInt(a.EstCash))
	return true
}

// holdReason names the two largest contributions and the weakest metric
// when it scores below 40.
func holdReason(s Score) string {
	var used []MetricScore
	for _, m := range s.Metrics {
		if !m.Dropped {
			used = append(used, m)
		}
	}
	if len(used) == 0 {
		return "no metric data"
	}
	top := topContributions(used, 2)
	parts := make([]string, len(top))
	for i, m := range top {
		parts[i] = fmt.Sprintf("%s %s (score %.0f)", m.Name, FormatRaw(m.Name, m.Raw), m.Score)
	}
	reason := "strongest: " + strings.Join(parts, ", ")
	weak := used[0]
	for _, m := range used[1:] {
		if m.Score < weak.Score {
			weak = m
		}
	}
	if weak.Score < 40 {
		reason += fmt.Sprintf("; weakest: %s %s (score %.0f)", weak.Name, FormatRaw(weak.Name, weak.Raw), weak.Score)
	}
	return reason
}

// topContributions returns up to n metrics by contribution, largest first.
func topContributions(ms []MetricScore, n int) []MetricScore {
	s := append([]MetricScore(nil), ms...)
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Contribution != s[j].Contribution {
			return s[i].Contribution > s[j].Contribution
		}
		return s[i].Name < s[j].Name
	})
	return s[:min(n, len(s))]
}

func portfolioMessages(sum portfolio.Summary, states []*holdingState, sw map[Sector]float64, c Config) []string {
	var m []string
	if sum.NAV <= 0 {
		m = append(m, "NAV is not positive: weight and cash rules are not reliable")
	}
	if below(sum.CashRatio, c.CashMin) {
		m = append(m, fmt.Sprintf("Cash buffer below %s: cash is %s of NAV; no new adds suggested, trims fund the buffer first.", pct0(c.CashMin), pct(sum.CashRatio)))
	}
	if above(sum.CashRatio, c.CashHigh) {
		m = append(m, fmt.Sprintf("Cash %s of NAV is above %s: consider deploying into Stable candidates in 2-3 tranches.", pct(sum.CashRatio), pct0(c.CashHigh)))
	}
	if sum.Debt > 0 {
		m = append(m, fmt.Sprintf("Margin debt %s VND (%s of NAV): interest cost and forced-sell risk.", groupInt(sum.Debt), pct(sum.MarginRatio)))
		if above(sum.MarginRatio, c.MarginMax) {
			m = append(m, fmt.Sprintf("Margin %s of NAV is above %s: all ADDs suppressed; reduce margin first.", pct(sum.MarginRatio), pct0(c.MarginMax)))
		}
	}
	if len(sw) > 0 {
		secs := make([]Sector, 0, len(sw))
		for s := range sw {
			secs = append(secs, s)
		}
		sort.Slice(secs, func(i, j int) bool {
			if sw[secs[i]] != sw[secs[j]] {
				return sw[secs[i]] > sw[secs[j]]
			}
			return secs[i] < secs[j]
		})
		parts := make([]string, len(secs))
		for i, s := range secs {
			parts[i] = fmt.Sprintf("%s %s", s, pct(sw[s]))
			if s != SectorFund && above(sw[s], c.SectorCap) {
				parts[i] += fmt.Sprintf(" (above the %s cap)", pct0(c.SectorCap))
			}
		}
		m = append(m, "Sector weights: "+strings.Join(parts, ", ")+".")
	}
	if sum.EffectiveN > 0 && sum.EffectiveN < c.ConcentratedN {
		m = append(m, fmt.Sprintf("Portfolio concentrated: effectiveN %.1f < %.0f.", sum.EffectiveN, c.ConcentratedN))
	}
	held := 0
	var odd []string
	for _, st := range states {
		if st.pos.Qty > 0 {
			held++
		}
		if n := OddLot(st.pos.Qty); n > 0 {
			odd = append(odd, fmt.Sprintf("%s %d", st.pos.Symbol, n))
		}
	}
	if held > 0 && held < c.MinHoldings {
		m = append(m, fmt.Sprintf("Portfolio under-diversified: %d holdings < %d.", held, c.MinHoldings))
	}
	if len(odd) > 0 {
		m = append(m, "Odd lots (sell on the odd-lot board): "+strings.Join(odd, ", ")+".")
	}
	if MarketPhase(c.now()) == PhaseIntraday {
		m = append(m, "Prices are intraday, not closing.")
	}
	return m
}

// pnlOf returns the position's P&L fraction, and false when the cost is
// unknown.
func pnlOf(p portfolio.Position) (float64, bool) {
	return p.PnLPct, p.AvgCost > 0
}

// FormatRaw formats a metric's raw value for messages: fractions as
// percentages, ADTV in VND, everything else with 2 decimals.
func FormatRaw(name string, raw float64) string {
	switch name {
	case MetricVOL, MetricMDD, MetricROE, MetricDY, MetricNPL, MetricCASA:
		return pct(raw)
	case MetricLIQ:
		return compactVND(raw) + " VND"
	case MetricEarnCons:
		return fmt.Sprintf("%.0f", raw)
	}
	return fmt.Sprintf("%.2f", raw)
}

// pct formats a fraction as a percentage with 1 decimal.
func pct(f float64) string {
	return strconv.FormatFloat(f*100, 'f', 1, 64) + "%"
}

// pct0 formats a threshold fraction as a percentage without trailing zeros.
func pct0(f float64) string {
	return strconv.FormatFloat(math.Round(f*1000)/10, 'f', -1, 64) + "%"
}

func fraction(f float64) string {
	if math.Abs(f-1.0/3) < 1e-9 {
		return "1/3"
	}
	return pct0(f)
}

// groupInt formats n with thousands separators: 1,234,500.
func groupInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
