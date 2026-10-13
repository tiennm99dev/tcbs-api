// Package report renders the advisor's results as a Markdown report. Render
// is deterministic: maps are walked in sorted order and every number uses a
// fixed format, so the same Data always gives the same bytes.
package report

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/miti99dev/stock-advisor/internal/advisor"
	"github.com/miti99dev/stock-advisor/internal/portfolio"
	"github.com/miti99dev/stock-advisor/internal/research"
)

// Disclaimer opens and closes every report.
const Disclaimer = "Informational, rule-based output; not financial advice. Verify prices and rules with your broker before trading."

// notCovered names the risk checks no public source supports.
const notCovered = "Not covered: the exchange warning, control and restricted lists. The public sources do not expose them, so only a listing status other than \"listed\" and price-history gaps are flagged."

// Data is everything the report shows.
type Data struct {
	GeneratedAt time.Time
	Phase       string    // intraday / last close
	Accounts    []Account // sub-accounts; numbers are masked when rendered

	Summary portfolio.Summary
	Advice  advisor.PortfolioAdvice

	Scores        map[string]advisor.Score
	Candidates    []advisor.Candidate
	CandidateNote string
	Universe      string // how the candidate universe was chosen, e.g. "VN100 (kbs)"

	Sources  []SourceStamp
	Config   advisor.Config
	Warnings []string

	Market   MarketContext
	Research map[string]HoldingResearch // keyed by held symbol
}

// Account is one sub-account line. No is masked to its last 4 characters.
type Account struct {
	No, Type string
}

// MarketContext is the VN-Index line, computed from the fetched bars.
type MarketContext struct {
	IndexClose, Return1Y, Vol1Y float64
	AsOf                        time.Time // zero when the index bars are missing
	HasReturn, HasVol           bool
}

// HoldingResearch is the research block of one held symbol.
type HoldingResearch struct {
	Name, Exchange   string
	Sector           advisor.Sector
	ICB4             string
	Low52W, High52W  float64
	Dividends        []research.Dividend // last 3 years, cash and stock
	DividendsMissing bool                // the dividend sources failed
}

// SourceStamp names one data source, what it supplied and how fresh it is.
type SourceStamp struct {
	Name, Kind string
	AsOf       time.Time
}

// zone is Asia/Ho_Chi_Minh. Vietnam has no daylight saving, so the fixed
// fallback is exact when the tz database is unavailable.
var zone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Ho_Chi_Minh"); err == nil {
		return loc
	}
	return time.FixedZone("+07", 7*3600)
}()

// Render writes the Markdown report to w.
func Render(w io.Writer, d Data) error {
	r := &renderer{w: w, d: d}
	r.header()
	r.summary()
	r.sectors()
	r.holdings()
	r.details()
	r.candidates()
	r.sources()
	r.config()
	r.warnings()
	r.line("---")
	r.line("")
	r.line(Disclaimer)
	return r.err
}

type renderer struct {
	w   io.Writer
	d   Data
	err error
}

func (r *renderer) printf(format string, a ...any) {
	if r.err != nil {
		return
	}
	_, r.err = fmt.Fprintf(r.w, format, a...)
}

func (r *renderer) line(s string) { r.printf("%s\n", s) }

// row writes one Markdown table row.
func (r *renderer) row(cells ...string) {
	for i, c := range cells {
		cells[i] = cell(c)
	}
	r.line("| " + strings.Join(cells, " | ") + " |")
}

func (r *renderer) header() {
	r.line("# Portfolio stability report")
	r.line("")
	r.printf("Generated %s (Asia/Ho_Chi_Minh). Prices: %s.\n\n", stamp(r.d.GeneratedAt), r.d.Phase)
	r.printf("> %s\n\n", Disclaimer)
}

func (r *renderer) summary() {
	s := r.d.Summary
	r.line("## Portfolio summary")
	r.line("")
	if len(r.d.Accounts) > 0 {
		parts := make([]string, len(r.d.Accounts))
		for i, a := range r.d.Accounts {
			parts[i] = MaskAccount(a.No)
			if a.Type != "" {
				parts[i] += " (" + a.Type + ")"
			}
		}
		r.printf("Sub-accounts: %s.\n\n", strings.Join(parts, ", "))
	}

	r.row("Item", "VND")
	r.line("|---|--:|")
	r.row("NAV", VND(s.NAV))
	r.row("Stock value", VND(s.StockValue))
	r.row("Cash", VND(s.Cash))
	r.row("Incoming cash dividends (in NAV)", VND(s.IncomingDividends))
	r.row("Debt", VND(s.Debt))
	if s.BankCashInNAV {
		r.row("IA bank cash (in cash and NAV)", VND(s.BankCash))
	} else {
		r.row("IA bank cash (not in NAV)", VND(s.BankCash))
	}
	r.row("Receivable sale proceeds (not in NAV)", VND(s.ReceivableCash))
	r.row("Pending stock-dividend value (not in NAV)", VND(s.PendingStockDivValue))
	r.line("")

	r.printf("- Cash ratio %s, margin ratio %s.\n", Pct(s.CashRatio), Pct(s.MarginRatio))
	r.printf("- HHI %s, effective N %s.\n", Ratio(s.HHI), Ratio(s.EffectiveN))
	var pending []string
	for _, p := range s.Positions {
		if p.PendingStockDiv > 0 {
			pending = append(pending, fmt.Sprintf("%s %s shares", p.Symbol, VND(p.PendingStockDiv)))
		}
	}
	if len(pending) > 0 {
		r.printf("- Pending stock-dividend shares (not in NAV): %s.\n", strings.Join(pending, ", "))
	}
	if s.ReceivableCash > 0 {
		r.printf("- Receivable sale proceeds (not in NAV): %s VND.\n", VND(s.ReceivableCash))
	}
	r.printf("- %s\n", r.marketLine())
	r.line("")

	msgs := append(append([]string(nil), r.d.Advice.Messages...), s.Notes...)
	if len(msgs) > 0 {
		r.line("Portfolio messages:")
		r.line("")
		for _, m := range msgs {
			r.printf("- %s\n", m)
		}
		r.line("")
	}
}

func (r *renderer) marketLine() string {
	m := r.d.Market
	if m.AsOf.IsZero() {
		return "Market: VN-Index bars unavailable."
	}
	ret, vol := "n/a", "n/a"
	if m.HasReturn {
		ret = Pct(m.Return1Y)
	}
	if m.HasVol {
		vol = Pct(m.Vol1Y)
	}
	return fmt.Sprintf("Market: VN-Index %s on %s; 1-year return %s, 1-year volatility %s.",
		Ratio(m.IndexClose), day(m.AsOf), ret, vol)
}

func (r *renderer) sectors() {
	sw := r.d.Advice.SectorWeights
	if len(sw) == 0 {
		return
	}
	secs := make([]advisor.Sector, 0, len(sw))
	for s := range sw {
		secs = append(secs, s)
	}
	sort.Slice(secs, func(i, j int) bool {
		if sw[secs[i]] != sw[secs[j]] {
			return sw[secs[i]] > sw[secs[j]]
		}
		return secs[i] < secs[j]
	})
	r.line("## Sector weights")
	r.line("")
	r.row("Sector", "Weight", "Status")
	r.line("|---|--:|---|")
	capPct := Pct(r.d.Config.SectorCap)
	for _, s := range secs {
		status := "within the " + capPct + " cap"
		switch {
		case s == advisor.SectorFund:
			status = "funds and warrants, no cap"
		case sw[s] > r.d.Config.SectorCap+1e-9:
			status = "above the " + capPct + " cap"
		}
		r.row(string(s), Pct(sw[s]), status)
	}
	r.line("")
}

// advice returns the advice for symbol.
func (r *renderer) advice(symbol string) (advisor.Advice, bool) {
	for _, a := range r.d.Advice.Holdings {
		if a.Symbol == symbol {
			return a, true
		}
	}
	return advisor.Advice{}, false
}

func (r *renderer) holdings() {
	r.line("## Holdings")
	r.line("")
	if len(r.d.Summary.Positions) == 0 {
		r.line("No holdings.")
		r.line("")
		return
	}
	r.row("Symbol", "Qty (sellable / receiving)", "Avg cost", "Price (source)", "Market value", "Weight", "P&L", "Breakeven", "Score", "Tier", "Action", "Rule")
	r.line("|---|--:|--:|--:|--:|--:|--:|--:|--:|---|---|---|")
	for _, p := range r.d.Summary.Positions {
		a, _ := r.advice(p.Symbol)
		score, tier := r.scoreCells(p.Symbol)
		cost, pnl, be := "unknown", "unknown", "unknown"
		if p.AvgCost > 0 {
			cost, pnl = VND(p.AvgCost), Pct(p.PnLPct)
		}
		if a.Breakeven > 0 {
			be = VND(a.Breakeven)
		}
		r.row(p.Symbol,
			fmt.Sprintf("%s (%s / %s)", VND(p.Qty), VND(p.Sellable), VND(p.Receiving)),
			cost,
			fmt.Sprintf("%s (%s)", VND(p.Price), p.PriceSource),
			VND(p.MarketValue), Pct(p.Weight), pnl, be, score, tier,
			string(a.Action), a.Rule)
	}
	r.line("")
}

// scoreCells returns the score and tier columns of a held symbol.
func (r *renderer) scoreCells(symbol string) (score, tier string) {
	s, ok := r.d.Scores[symbol]
	switch {
	case !ok:
		return "n/a", advisor.TierNotScored
	case !s.Valid:
		return "n/a", s.Tier
	}
	return strconv.Itoa(s.Value), s.Tier
}

func (r *renderer) details() {
	if len(r.d.Summary.Positions) == 0 {
		return
	}
	r.line("## Holding detail")
	r.line("")
	for _, p := range r.d.Summary.Positions {
		r.detail(p)
	}
}

func (r *renderer) detail(p portfolio.Position) {
	res, hasRes := r.d.Research[p.Symbol]
	title := p.Symbol
	if hasRes && res.Name != "" {
		title += " - " + res.Name
	}
	r.printf("### %s\n\n", title)

	if hasRes {
		exch := orNA(res.Exchange)
		sector := orNA(string(res.Sector))
		if res.ICB4 != "" {
			sector += " (ICB4 " + res.ICB4 + ")"
		}
		r.printf("- Exchange %s, sector %s.\n", exch, sector)
		if res.High52W > 0 {
			r.printf("- 52-week range: %s - %s VND.\n", VNDf(res.Low52W), VNDf(res.High52W))
		} else {
			r.line("- 52-week range: n/a.")
		}
		r.printf("- Dividends, last 3 years: %s.\n", dividends(res))
	} else {
		r.line("- No research data.")
	}

	a, ok := r.advice(p.Symbol)
	if ok {
		r.printf("- Action: **%s**. %s\n", a.Action, sentence(a.Rule))
		if a.Reason != "" {
			r.printf("- Reason: %s\n", sentence(a.Reason))
		}
		for _, s := range a.Secondary {
			r.printf("- Also triggered: %s\n", sentence(s))
		}
		if a.Qty > 0 {
			sign := ""
			if a.EstCash > 0 {
				sign = "+"
			}
			r.printf("- Suggested %s: %s shares in board lots of %d, estimated cash impact %s%s VND (fee and tax included).\n",
				a.Action, VND(a.Qty), advisor.BoardLot, sign, VND(a.EstCash))
		}
		if a.OddLot > 0 {
			r.printf("- Odd lot: %s shares trade on the odd-lot board.\n", VND(a.OddLot))
		}
		for _, n := range a.Notes {
			r.printf("- Note: %s\n", sentence(n))
		}
	}
	r.line("")

	s, ok := r.d.Scores[p.Symbol]
	if !ok || s.Tier == advisor.TierNotScored {
		r.printf("Score: %s.\n\n", advisor.NoteNotScored)
		return
	}
	r.scoreBlock(s)
}

func (r *renderer) scoreBlock(s advisor.Score) {
	value := "n/a"
	if s.Valid {
		value = strconv.Itoa(s.Value)
	}
	r.printf("Score %s, tier %s, coverage %.0f, confidence %s.\n\n", value, s.Tier, s.Coverage, s.Confidence)

	var used, dropped []advisor.MetricScore
	for _, m := range s.Metrics {
		if m.Dropped {
			dropped = append(dropped, m)
		} else {
			used = append(used, m)
		}
	}
	if len(used) > 0 {
		r.row("Metric", "Pillar", "Raw", "Score", "Weight", "Contribution")
		r.line("|---|---|--:|--:|--:|--:|")
		for _, m := range used {
			r.row(m.Name, m.Pillar, advisor.FormatRaw(m.Name, m.Raw), num1(m.Score), num1(m.Weight), num1(m.Contribution))
		}
		r.line("")
	}

	pillars := []string{advisor.PillarPrice, advisor.PillarFundamentals, advisor.PillarValuation}
	parts := make([]string, 0, len(pillars))
	for _, name := range pillars {
		if v, ok := s.Pillars[name]; ok {
			parts = append(parts, name+" "+num1(v))
		} else {
			parts = append(parts, name+" missing")
		}
	}
	r.printf("- Pillars: %s.\n", strings.Join(parts, ", "))
	r.printf("- Sector adjustment: %d.\n", s.SectorAdj)
	for _, m := range used {
		if m.Note != "" {
			r.printf("- %s: %s.\n", m.Name, strings.TrimSuffix(m.Note, "."))
		}
	}
	if len(dropped) > 0 {
		names := make([]string, len(dropped))
		for i, m := range dropped {
			names[i] = m.Name
		}
		r.printf("- Dropped metrics: %s.\n", strings.Join(names, ", "))
	}
	if len(s.RedFlags) > 0 {
		r.printf("- Red flags: %s.\n", strings.Join(s.RedFlags, "; "))
	} else {
		r.line("- Red flags: none.")
	}
	for _, n := range s.Notes {
		r.printf("- Note: %s\n", sentence(n))
	}
	r.line("")
}

func dividends(res HoldingResearch) string {
	if res.DividendsMissing {
		return "unavailable"
	}
	if len(res.Dividends) == 0 {
		return "none recorded"
	}
	divs := append([]research.Dividend(nil), res.Dividends...)
	sort.SliceStable(divs, func(i, j int) bool { return divs[i].ExDate.After(divs[j].ExDate) })
	parts := make([]string, 0, len(divs))
	for _, d := range divs {
		switch {
		case d.CashPerShare > 0:
			parts = append(parts, fmt.Sprintf("%s cash %s VND/share", day(d.ExDate), VNDf(d.CashPerShare)))
		case d.StockRatio > 0:
			parts = append(parts, fmt.Sprintf("%s stock %s", day(d.ExDate), Pct(d.StockRatio)))
		}
	}
	return strings.Join(parts, "; ")
}

func (r *renderer) candidates() {
	r.line("## Stable candidates")
	r.line("")
	if r.d.Universe != "" {
		r.printf("Universe: %s, minus the symbols already held.\n\n", r.d.Universe)
	}
	if len(r.d.Candidates) == 0 {
		r.line("No stock in the universe passed the candidate screen.")
		r.line("")
		return
	}
	if r.d.CandidateNote != "" {
		r.printf("%s\n\n", r.d.CandidateNote)
	}
	r.row("#", "Symbol", "Sector", "Score", "Tier", "VOL", "BETA", "DY", "ROE", "PE", "PB", "Price", "Lot cost", "Max alloc", "Reasons")
	r.line("|--:|---|---|--:|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|---|")
	for i, c := range r.d.Candidates {
		in := c.Inputs
		beta := in.Beta
		if raw, ok := metricRaw(c.Score, advisor.MetricBeta); ok {
			beta = &raw
		}
		r.row(strconv.Itoa(i+1), c.Score.Symbol, string(c.Score.Sector), strconv.Itoa(c.Score.Value), c.Score.Tier,
			optPct(in.VOL), optRatio(beta), optPct(in.DY), optPct(in.ROE), optRatio(in.PE), optRatio(in.PB),
			VNDf(in.Price), VND(c.LotCost), VND(c.MaxAlloc), strings.Join(c.Reasons, "; "))
	}
	r.line("")
}

// metricRaw returns the raw value a score used for a metric.
func metricRaw(s advisor.Score, name string) (float64, bool) {
	for _, m := range s.Metrics {
		if m.Name == name && !m.Dropped {
			return m.Raw, true
		}
	}
	return 0, false
}

func (r *renderer) sources() {
	r.line("## Data sources and timestamps")
	r.line("")
	if len(r.d.Sources) > 0 {
		r.row("Source", "Data", "As of")
		r.line("|---|---|---|")
		for _, s := range r.d.Sources {
			at := "n/a"
			switch u := s.AsOf.UTC(); {
			case s.AsOf.IsZero():
			case u.Hour() == 0 && u.Minute() == 0 && u.Second() == 0 && u.Nanosecond() == 0:
				at = day(s.AsOf) // a trading or report date, not an instant
			default:
				at = stamp(s.AsOf)
			}
			r.row(s.Name, s.Kind, at)
		}
		r.line("")
	}
	r.line("The public sources (VNDirect, Vietcap, KBS) are undocumented and can change without notice.")
	r.line("")
	r.line(notCovered)
	r.line("")
}

func (r *renderer) config() {
	c := r.d.Config
	r.line("## Config in use")
	r.line("")
	r.row("Setting", "Value")
	r.line("|---|---|")
	r.row("Single-stock caps (soft / hard)", Pct(c.SingleSoftCap)+" / "+Pct(c.SingleHardCap))
	r.row("Sector cap", Pct(c.SectorCap))
	r.row("Cash band (min / high)", Pct(c.CashMin)+" / "+Pct(c.CashHigh))
	r.row("Margin max", Pct(c.MarginMax))
	r.row("ADD score (min / strong)", num0(c.AddMinScore)+" / "+num0(c.AddStrongScore))
	r.row("Weak / review score", num0(c.WeakScore)+" / "+num0(c.ReviewScore))
	r.row("Review loss / weak loss", Pct(c.ReviewLossPct)+" / "+Pct(c.WeakLossPct))
	r.row("Take-profit gain", Pct(c.TakeProfitPct))
	r.row("Fee per side / sales tax", Pct(c.FeeRate)+" / "+Pct(c.SellTax))
	r.row("Cost of equity (fair PB)", Pct(c.CoE))
	r.row("Candidates (count / min score / max VOL)", fmt.Sprintf("%d / %s / %s", c.CandidateCount, num0(c.CandidateMinScore), Pct(c.CandidateMaxVol)))
	r.line("")
}

func (r *renderer) warnings() {
	ws := sortedUnique(r.d.Warnings)
	r.line("## Warnings")
	r.line("")
	if len(ws) == 0 {
		r.line("None.")
		r.line("")
		return
	}
	for _, w := range ws {
		r.printf("- %s\n", strings.Join(strings.Fields(strings.ReplaceAll(w, "\n", "; ")), " "))
	}
	r.line("")
}

func sortedUnique(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	n := 0
	for i, s := range out {
		if i > 0 && s == out[n-1] {
			continue
		}
		out[n] = s
		n++
	}
	return out[:n]
}

// MaskAccount keeps the last 4 characters of an account number.
func MaskAccount(no string) string {
	no = strings.TrimSpace(no)
	if len(no) <= 4 {
		return "****"
	}
	return "****" + no[len(no)-4:]
}

// VND formats an integer with thousands separators: 1,234,500.
func VND(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// VNDf rounds f to the dong and formats it like VND.
func VNDf(f float64) string { return VND(int64(math.Round(f))) }

// Pct formats a fraction as a percentage with 1 decimal place.
func Pct(f float64) string { return strconv.FormatFloat(f*100, 'f', 1, 64) + "%" }

// Ratio formats a ratio with 2 decimal places.
func Ratio(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }

func num0(f float64) string { return strconv.FormatFloat(f, 'f', 0, 64) }
func num1(f float64) string { return strconv.FormatFloat(f, 'f', 1, 64) }

func optPct(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return Pct(*p)
}

func optRatio(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return Ratio(*p)
}

func orNA(s string) string {
	if strings.TrimSpace(s) == "" {
		return "n/a"
	}
	return s
}

// stamp formats a time in Asia/Ho_Chi_Minh.
func stamp(t time.Time) string { return t.In(zone).Format("2006-01-02 15:04 -07") }

// day formats a trading-day date (stored at 00:00 UTC).
func day(t time.Time) string { return t.UTC().Format(time.DateOnly) }

// sentence ends s with a full stop.
func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

// cell escapes a value for a Markdown table cell.
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.Join(strings.Fields(s), " ")
}
