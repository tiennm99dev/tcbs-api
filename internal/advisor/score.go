package advisor

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Inputs are the raw metrics of one symbol. A nil pointer means the value is
// missing; a missing metric is dropped, never scored as zero.
type Inputs struct {
	Symbol   string
	Sector   Sector // empty means unknown: scored as general with a note
	Exchange string
	Price    float64 // VND

	VOL, Beta, BetaR2, MDD, ADTV *float64
	BetaWeekly                   *float64 // 2y weekly beta, preferred when the daily fit is poor or the stock is illiquid
	ROE, DE, PE, PB, DY          *float64
	NPL, CASA                    *float64
	EarnCons                     *float64 // already 0-100; nil means compute it from ProfitYears

	DivPaidYears   int      // of the last 3 years, how many paid cash
	SectorMedianPE *float64 // from MedianPE; nil when fewer than MinPeers peers
	Status, Type   string   // listing status ("listed") and security type ("STOCK")

	HistoryReturns int           // daily returns available for the price metrics
	GapSessions    int           // index sessions without a bar, of the last 60
	StaleSessions  int           // index sessions after the stock's last bar
	FundAge        time.Duration // age of the fundamentals; 0 means unknown

	ProfitYears  []float64 // annual net profit, newest first; empty = unknown
	TTMLoss      bool
	PrevYearLoss bool
	NegEquity    bool
	Listed       time.Time // zero means unknown
}

// MetricScore is one metric's contribution to a score. Weight is the
// effective weight after redistribution, in points out of 100.
type MetricScore struct {
	Name         string
	Pillar       string
	Raw          float64
	Score        float64
	Weight       float64
	Contribution float64
	Dropped      bool
	Note         string
}

// Score is the stability score of one symbol. Value is meaningful only when
// Valid. Pillars holds the 0-100 score of every pillar with enough data.
type Score struct {
	Symbol     string
	Sector     Sector
	Value      int
	Valid      bool
	Tier       string
	Confidence string
	Coverage   float64
	Pillars    map[string]float64
	Metrics    []MetricScore
	SectorAdj  int
	RedFlags   []string
	Notes      []string
}

// Pillars.
const (
	PillarPrice        = "price"
	PillarFundamentals = "fundamentals"
	PillarValuation    = "valuation"
)

// Metric names.
const (
	MetricVOL       = "VOL"
	MetricBeta      = "BETA"
	MetricMDD       = "MDD"
	MetricLIQ       = "LIQ"
	MetricROE       = "ROE"
	MetricEarnCons  = "EARN_CONS"
	MetricDE        = "DE"
	MetricNPL       = "NPL"
	MetricLLR       = "LLR"
	MetricCASA      = "CASA"
	MetricMarginLev = "MARGIN_LEV"
	MetricCombined  = "COMBINED"
	MetricPE        = "PE"
	MetricPB        = "PB"
	MetricPBFair    = "PB_FAIR"
	MetricDY        = "DY"
)

// Tiers and confidence levels.
const (
	TierStable       = "Stable"
	TierModerate     = "Moderate"
	TierWeak         = "Weak/Volatile"
	TierSpeculative  = "Speculative"
	TierInsufficient = "Insufficient data"
	TierNotScored    = "Not scored"

	ConfidenceHigh   = "High"
	ConfidenceMedium = "Medium"
	ConfidenceLow    = "Low"

	// NoteNotScored marks ETFs and covered warrants.
	NoteNotScored = "not scored (fund/warrant)"
)

type metricDef struct {
	name   string
	weight float64
}

type pillarDef struct {
	name    string
	metrics []metricDef
}

var pricePillar = pillarDef{PillarPrice, []metricDef{{MetricVOL, 15}, {MetricBeta, 10}, {MetricMDD, 10}, {MetricLIQ, 5}}}

var generalProfile = []pillarDef{
	pricePillar,
	{PillarFundamentals, []metricDef{{MetricROE, 12}, {MetricEarnCons, 10}, {MetricDE, 13}}},
	{PillarValuation, []metricDef{{MetricPE, 10}, {MetricPB, 5}, {MetricDY, 10}}},
}

var profiles = map[Sector][]pillarDef{
	SectorGeneral:    generalProfile,
	SectorRealEstate: generalProfile,
	SectorUtilities:  generalProfile,
	SectorBank: {
		pricePillar,
		{PillarFundamentals, []metricDef{{MetricROE, 12}, {MetricNPL, 10}, {MetricLLR, 5}, {MetricCASA, 8}}},
		{PillarValuation, []metricDef{{MetricPBFair, 12}, {MetricPE, 5}, {MetricDY, 8}}},
	},
	SectorSecurities: {
		pricePillar,
		{PillarFundamentals, []metricDef{{MetricROE, 12}, {MetricEarnCons, 13}, {MetricMarginLev, 10}}},
		{PillarValuation, []metricDef{{MetricPBFair, 12}, {MetricPE, 5}, {MetricDY, 8}}},
	},
	SectorInsurance: {
		pricePillar,
		{PillarFundamentals, []metricDef{{MetricROE, 12}, {MetricEarnCons, 13}, {MetricCombined, 10}}},
		{PillarValuation, []metricDef{{MetricPB, 8}, {MetricPE, 9}, {MetricDY, 8}}},
	},
}

// noSource lists metrics no available data source provides.
var noSource = map[string]bool{MetricLLR: true, MetricMarginLev: true, MetricCombined: true}

// metricResult is one scored metric before weighting.
type metricResult struct {
	raw, score float64
	ok         bool
	note       string
}

// IsStockType reports whether a security type is scored. An empty type is
// treated as a stock.
func IsStockType(typ string, c Config) bool {
	t := strings.TrimSpace(typ)
	return t == "" || strings.EqualFold(t, c.Scoring.StockType)
}

// ScoreStock scores one symbol. ETFs and covered warrants are not scored.
func ScoreStock(in Inputs, c Config) Score {
	sc := c.Scoring
	sec := in.Sector
	s := Score{Symbol: in.Symbol, Sector: sec, Pillars: map[string]float64{}}
	if !IsStockType(in.Type, c) {
		s.Tier = TierNotScored
		s.Notes = []string{NoteNotScored}
		return s
	}
	if _, ok := profiles[sec]; !ok {
		if sec != "" {
			s.Notes = append(s.Notes, fmt.Sprintf("sector %q has no profile; scored as general", sec))
		} else {
			s.Notes = append(s.Notes, NoteSectorUnknown)
		}
		sec = SectorGeneral
		s.Sector = sec
	}
	switch sec {
	case SectorRealEstate:
		s.SectorAdj = sc.SectorAdjRealEst
		s.Notes = append(s.Notes, fmt.Sprintf("sector adjustment %d: real estate, high leverage and lumpy earnings", s.SectorAdj))
	case SectorSecurities:
		s.SectorAdj = sc.SectorAdjSecurities
		s.Notes = append(s.Notes, fmt.Sprintf("sector adjustment %d: securities, earnings track market turnover", s.SectorAdj))
	}
	if in.HistoryReturns < sc.MinHistoryReturns {
		s.Notes = append(s.Notes, fmt.Sprintf("recent listing / insufficient history (%d daily returns < %d)", in.HistoryReturns, sc.MinHistoryReturns))
	}

	var weighted, presentPillarW float64
	type pillarOut struct {
		def      pillarDef
		results  []metricResult
		present  float64
		score    float64
		included bool
	}
	var pillars []pillarOut
	for _, p := range profiles[sec] {
		po := pillarOut{def: p}
		var total, sum float64
		for _, m := range p.metrics {
			r := scoreMetric(m.name, in, sec, c)
			po.results = append(po.results, r)
			total += m.weight
			if r.ok {
				po.present += m.weight
				sum += r.score * m.weight
			}
		}
		if po.present > 0 && po.present >= sc.PillarMinPresent*total {
			po.included = true
			po.score = sum / po.present
			presentPillarW += total
			s.Pillars[p.name] = round1(po.score)
		}
		pillars = append(pillars, po)
	}

	for _, po := range pillars {
		var total float64
		for _, m := range po.def.metrics {
			total += m.weight
		}
		for i, m := range po.def.metrics {
			r := po.results[i]
			ms := MetricScore{Name: m.name, Pillar: po.def.name, Raw: r.raw, Score: r.score, Note: r.note}
			switch {
			case !r.ok:
				ms.Dropped = true
				if noSource[m.name] {
					ms.Note = fmt.Sprintf("%s not available from any source: weight redistributed within %s", m.name, po.def.name)
				} else {
					ms.Note = joinNote(fmt.Sprintf("%s unavailable: weight redistributed within %s", m.name, po.def.name), r.note)
				}
				s.Notes = append(s.Notes, ms.Note)
			case !po.included:
				ms.Dropped = true
				ms.Note = joinNote(fmt.Sprintf("%s pillar under %.0f%% data: weight redistributed to the other pillars", po.def.name, sc.PillarMinPresent*100), r.note)
				s.Notes = append(s.Notes, fmt.Sprintf("%s dropped: %s", m.name, ms.Note))
			default:
				ms.Weight = m.weight / po.present * total / presentPillarW * 100
				ms.Contribution = ms.Score * ms.Weight / 100
				weighted += ms.Contribution
				s.Coverage += m.weight
			}
			s.Metrics = append(s.Metrics, ms)
		}
		if !po.included {
			s.Notes = append(s.Notes, fmt.Sprintf("%s pillar missing (%.0f of %.0f weight present)", po.def.name, po.present, total))
		}
	}

	if presentPillarW > 0 {
		s.Value = max(0, int(math.Round(weighted+float64(s.SectorAdj))))
	}
	_, pricePresent := s.Pillars[PillarPrice]
	s.Valid = s.Coverage >= sc.MinCoverage && pricePresent
	s.Confidence = confidence(s.Coverage, c)
	s.Tier = Tier(s, c)
	s.RedFlags, s.Notes = redFlags(in, c, s.Notes)
	return s
}

// confidence grades a coverage (0-100).
func confidence(coverage float64, c Config) string {
	switch {
	case coverage >= c.Scoring.HighConfidence:
		return ConfidenceHigh
	case coverage >= c.Scoring.MediumConfidence:
		return ConfidenceMedium
	}
	return ConfidenceLow
}

// Tier names the band of a score.
func Tier(s Score, c Config) string {
	sc := c.Scoring
	v := float64(s.Value)
	switch {
	case !s.Valid:
		return TierInsufficient
	case v >= sc.StableScore:
		return TierStable
	case v >= sc.ModerateScore:
		return TierModerate
	case v >= sc.WeakTierScore:
		return TierWeak
	}
	return TierSpeculative
}

func redFlags(in Inputs, c Config, notes []string) (flags, outNotes []string) {
	sc := c.Scoring
	switch st := strings.ToLower(strings.TrimSpace(in.Status)); {
	case st == "":
		notes = append(notes, "listing status unknown")
	case st != strings.ToLower(sc.ListedStatus):
		flags = append(flags, fmt.Sprintf("listing status %q is not %q", in.Status, sc.ListedStatus))
	}
	if in.NegEquity {
		flags = append(flags, "negative equity")
	}
	if in.TTMLoss && in.PrevYearLoss {
		flags = append(flags, "TTM loss after a loss in the previous year")
	}
	if in.ADTV != nil && *in.ADTV < sc.RedFlagADTV {
		flags = append(flags, fmt.Sprintf("ADTV %s VND < %s VND (hard to exit)", compactVND(*in.ADTV), compactVND(sc.RedFlagADTV)))
	}
	if in.GapSessions > sc.RedFlagGaps {
		flags = append(flags, fmt.Sprintf("%d gap sessions in the last 60 > %d (possible suspension)", in.GapSessions, sc.RedFlagGaps))
	}
	if in.StaleSessions > sc.RedFlagStale {
		flags = append(flags, fmt.Sprintf("last bar %d VN-Index sessions behind > %d (stale or suspended)", in.StaleSessions, sc.RedFlagStale))
	}
	if in.FundAge > sc.RedFlagFundAge {
		flags = append(flags, fmt.Sprintf("fundamentals %d days old > %d (stale)", int(in.FundAge.Hours()/24), int(sc.RedFlagFundAge.Hours()/24)))
	}
	return flags, notes
}

// scoreMetric scores one metric for a sector profile.
func scoreMetric(name string, in Inputs, sec Sector, c Config) metricResult {
	sc := c.Scoring
	switch name {
	case MetricVOL:
		return banded(in.VOL, &sc.WinsorVOL, sc.VOL)
	case MetricBeta:
		return scoreBeta(in, sc)
	case MetricMDD:
		return banded(in.MDD, nil, sc.MDD)
	case MetricLIQ:
		if in.ADTV == nil {
			return metricResult{}
		}
		return metricResult{raw: *in.ADTV, score: round1(logBandScore(*in.ADTV, sc.LIQ)), ok: true}
	case MetricROE:
		if sec == SectorBank {
			return banded(in.ROE, nil, sc.ROEBank)
		}
		return banded(in.ROE, nil, sc.ROE)
	case MetricEarnCons:
		if in.EarnCons != nil {
			v := clamp(*in.EarnCons, 0, 100)
			return metricResult{raw: v, score: round1(v), ok: true}
		}
		v, ok := EarningsConsistency(in.ProfitYears, sc.MinEarnYears)
		if !ok {
			return metricResult{note: fmt.Sprintf("needs %d fiscal years, have %d", sc.MinEarnYears, len(in.ProfitYears))}
		}
		return metricResult{raw: v, score: v, ok: true}
	case MetricDE:
		switch sec {
		case SectorRealEstate:
			return banded(in.DE, &sc.WinsorDE, sc.DERealEstate)
		case SectorUtilities:
			return banded(in.DE, &sc.WinsorDE, sc.DEUtilities)
		}
		return banded(in.DE, &sc.WinsorDE, sc.DE)
	case MetricNPL:
		return banded(in.NPL, nil, sc.NPL)
	case MetricCASA:
		return banded(in.CASA, nil, sc.CASA)
	case MetricPE:
		return scorePE(in, sec, sc)
	case MetricPB:
		return scorePB(in, sc)
	case MetricPBFair:
		return scorePBFair(in, c)
	case MetricDY:
		return scoreDY(in, sc)
	}
	return metricResult{}
}

func banded(v *float64, w *Range, b Band) metricResult {
	if v == nil || math.IsNaN(*v) {
		return metricResult{}
	}
	x := *v
	if w != nil {
		x = clamp(x, w.Min, w.Max)
	}
	return metricResult{raw: x, score: round1(bandScore(x, b)), ok: true}
}

func scoreBeta(in Inputs, sc Scoring) metricResult {
	b, note := in.Beta, ""
	poorFit := in.BetaR2 != nil && *in.BetaR2 < sc.WeeklyBetaR2
	illiquid := in.ADTV != nil && *in.ADTV < sc.IlliquidADTV
	if (poorFit || illiquid || b == nil) && in.BetaWeekly != nil {
		b = in.BetaWeekly
		switch {
		case poorFit:
			note = fmt.Sprintf("weekly 2y beta used (daily R² %.2f < %.2g)", *in.BetaR2, sc.WeeklyBetaR2)
		case illiquid:
			note = fmt.Sprintf("weekly 2y beta used (ADTV below %s VND)", compactVND(sc.IlliquidADTV))
		default:
			note = "weekly 2y beta used (daily beta unavailable)"
		}
	} else if poorFit || illiquid {
		note = "daily beta kept; weekly beta unavailable"
	}
	r := banded(b, &sc.WinsorBeta, sc.Beta)
	r.note = note
	if r.ok && r.raw < sc.LowBeta && illiquid && r.score > sc.LowBetaCap {
		r.score = sc.LowBetaCap
		r.note = joinNote(r.note, fmt.Sprintf("capped at %.0f: low beta on an illiquid stock", sc.LowBetaCap))
	}
	return r
}

func scorePE(in Inputs, sec Sector, sc Scoring) metricResult {
	if in.PE == nil || math.IsNaN(*in.PE) {
		return metricResult{}
	}
	pe := clamp(*in.PE, sc.WinsorPE.Min, sc.WinsorPE.Max)
	r := metricResult{raw: pe, ok: true}
	switch {
	case pe <= 0:
		r.note = "loss-making (PE ≤ 0)"
	case sec == SectorBank:
		r.score = absPEScore(pe, sc.PEBank, sc.PELowScore)
	case in.SectorMedianPE != nil && *in.SectorMedianPE > 0:
		ratio := pe / *in.SectorMedianPE
		r.score = piecewise(ratio, sc.PERelative)
		r.note = fmt.Sprintf("PE/sector median %.2f (median %.1f)", ratio, *in.SectorMedianPE)
	default:
		r.score = absPEScore(pe, sc.PEAbs, sc.PELowScore)
	}
	r.score = round1(r.score)
	return r
}

func absPEScore(pe float64, pts []Point, low float64) float64 {
	if len(pts) > 0 && pe < pts[0].X {
		return low
	}
	return piecewise(pe, pts)
}

func scorePB(in Inputs, sc Scoring) metricResult {
	if in.PB == nil || math.IsNaN(*in.PB) {
		return metricResult{}
	}
	pb := clamp(*in.PB, sc.WinsorPB.Min, sc.WinsorPB.Max)
	r := metricResult{raw: pb, ok: true}
	if pb <= 0 {
		r.note = "book value not positive"
		return r
	}
	r.score = round1(piecewise(pb, sc.PBGeneral))
	if pb < sc.ValueTrapPB && in.ROE != nil && *in.ROE < sc.ValueTrapROE && r.score > sc.ValueTrapCap {
		r.score = sc.ValueTrapCap
		r.note = fmt.Sprintf("capped at %.0f: PB < %.1f with ROE < %.0f%% (value trap)", sc.ValueTrapCap, sc.ValueTrapPB, sc.ValueTrapROE*100)
	}
	return r
}

func scorePBFair(in Inputs, c Config) metricResult {
	sc := c.Scoring
	if in.PB == nil || math.IsNaN(*in.PB) {
		return metricResult{}
	}
	if in.ROE == nil || math.IsNaN(*in.ROE) {
		return metricResult{note: "fair PB needs ROE"}
	}
	pb := clamp(*in.PB, sc.WinsorPB.Min, sc.WinsorPB.Max)
	r := metricResult{raw: pb, ok: true}
	if pb <= 0 {
		r.note = "book value not positive"
		return r
	}
	if *in.ROE <= 0 || c.CoE <= 0 {
		r.note = "ROE not positive: no fair PB"
		return r
	}
	fair := *in.ROE / c.CoE
	ratio := pb / fair
	r.score = round1(piecewise(ratio, sc.PBFair))
	r.note = fmt.Sprintf("PB/fairPB %.2f (fairPB %.2f = ROE / CoE %.0f%%)", ratio, fair, c.CoE*100)
	return r
}

func scoreDY(in Inputs, sc Scoring) metricResult {
	if in.DY == nil || math.IsNaN(*in.DY) {
		return metricResult{}
	}
	dy := *in.DY
	r := metricResult{ok: true}
	if dy > sc.DYSpecial {
		dy = sc.DYSpecialCap
		r.note = "high trailing yield, likely special dividend"
	}
	r.raw = dy
	s := bandScore(dy, sc.DY)
	if in.DivPaidYears >= sc.DYConsistentYears {
		s = math.Min(100, s+sc.DYConsistentBonus)
		r.note = joinNote(r.note, fmt.Sprintf("+%.0f: cash paid in each of the last %d years", sc.DYConsistentBonus, sc.DYConsistentYears))
	}
	r.score = round1(s)
	return r
}

// EarningsConsistency scores annual net profits (any order) on 0-100:
// pos/n × 70 + (1 − min(CV, 1)) × 30, where CV is the population standard
// deviation over the absolute mean. It needs at least minYears values.
func EarningsConsistency(profits []float64, minYears int) (float64, bool) {
	n := len(profits)
	if n == 0 || n < minYears {
		return 0, false
	}
	var pos int
	var sum float64
	for _, p := range profits {
		if p > 0 {
			pos++
		}
		sum += p
	}
	m := sum / float64(n)
	cv := 1.0
	if m != 0 {
		var ss float64
		for _, p := range profits {
			ss += (p - m) * (p - m)
		}
		cv = math.Sqrt(ss/float64(n)) / math.Abs(m)
	}
	return round1(float64(pos)/float64(n)*70 + (1-math.Min(cv, 1))*30), true
}

// MedianPE is the median of the positive peer PEs, or nil when fewer than
// c.Scoring.MinPeers peers have one. Loss-making peers are left out.
func MedianPE(pes []float64, c Config) *float64 {
	var pos []float64
	for _, p := range pes {
		if p > 0 && !math.IsNaN(p) && !math.IsInf(p, 0) {
			pos = append(pos, p)
		}
	}
	if len(pos) == 0 || len(pos) < c.Scoring.MinPeers {
		return nil
	}
	sort.Float64s(pos)
	n := len(pos)
	m := pos[n/2]
	if n%2 == 0 {
		m = (pos[n/2-1] + pos[n/2]) / 2
	}
	return &m
}

// bandScore is 100 × clamp((x − worst)/(best − worst), 0, 1).
func bandScore(x float64, b Band) float64 {
	if b.Best == b.Worst {
		if x == b.Best {
			return 100
		}
		return 0
	}
	return 100 * clamp((x-b.Worst)/(b.Best-b.Worst), 0, 1)
}

// logBandScore is bandScore on log10 values. Non-positive x scores 0.
func logBandScore(x float64, b Band) float64 {
	if x <= 0 || b.Best <= 0 || b.Worst <= 0 {
		return 0
	}
	return bandScore(math.Log10(x), Band{Best: math.Log10(b.Best), Worst: math.Log10(b.Worst)})
}

// piecewise interpolates linearly between knots sorted by X; values outside
// the knots take the nearest end's score.
func piecewise(x float64, pts []Point) float64 {
	if len(pts) == 0 {
		return 0
	}
	if x <= pts[0].X {
		return pts[0].Score
	}
	for i := 1; i < len(pts); i++ {
		if x <= pts[i].X {
			a, b := pts[i-1], pts[i]
			if b.X == a.X {
				return b.Score
			}
			return a.Score + (b.Score-a.Score)*(x-a.X)/(b.X-a.X)
		}
	}
	return pts[len(pts)-1].Score
}

func clamp(x, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, x))
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}

func joinNote(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + "; " + b
}

// compactVND formats an amount as 1.5bn or 250m.
func compactVND(v float64) string {
	switch {
	case math.Abs(v) >= 1e9:
		return trimZero(fmt.Sprintf("%.1f", v/1e9)) + "bn"
	case math.Abs(v) >= 1e6:
		return trimZero(fmt.Sprintf("%.1f", v/1e6)) + "m"
	}
	return fmt.Sprintf("%.0f", v)
}

func trimZero(s string) string {
	return strings.TrimSuffix(s, ".0")
}
