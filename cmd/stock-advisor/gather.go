package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/miti99dev/stock-advisor/internal/advisor"
	"github.com/miti99dev/stock-advisor/internal/portfolio"
	"github.com/miti99dev/stock-advisor/internal/report"
	"github.com/miti99dev/stock-advisor/internal/research"
)

const (
	// maxPeers caps the extra symbols fetched only for the sector median P/E.
	maxPeers = 40
	// barYears of daily bars feed the 1-year metrics and the 2-year weekly beta.
	barYears = 2
	// dividendYears of events feed the trailing yield and the payout streak.
	dividendYears = 3
	// yearSessions is one year of trading sessions.
	yearSessions = 250
	// liquiditySessions is the ADTV window.
	liquiditySessions = 60
	// gapWindow is the window, in index sessions, checked for missing bars.
	gapWindow = 60
	// weeklyBetaWeeks is the weekly beta window (2 years).
	weeklyBetaWeeks = 104
	// indexSymbol is the market index used for beta and staleness.
	indexSymbol = "VNINDEX"
)

// gatherer fetches the public data for the held symbols and the universe.
type gatherer struct {
	svc  *research.Service
	cfg  advisor.Config
	now  time.Time
	logf func(format string, a ...any)
}

// marketData is the public data of one run, keyed by symbol.
type marketData struct {
	now        time.Time
	profiles   map[string]research.Profile
	funds      map[string]research.Fundamentals
	bars       map[string]research.Series
	divs       map[string][]research.Dividend
	divsFailed map[string]bool
	profits    map[string][]float64 // oldest year first, as the service returns them
	banks      int                  // banks whose Vietcap extras were requested
	index      research.Series
	core       []string // held symbols, then universe members
	warnings   []string
}

// gather fetches profiles and ratios for the held symbols, the universe and
// up to maxPeers industry peers; daily bars, dividends, annual profit and
// bank ratios for the held symbols and the universe only; and the VN-Index
// bars once.
func (g *gatherer) gather(ctx context.Context, held []string, heldTypes map[string]string, universe []string) (*marketData, error) {
	md := &marketData{
		now:        g.now,
		divs:       map[string][]research.Dividend{},
		divsFailed: map[string]bool{},
		profits:    map[string][]float64{},
	}
	md.core = union(held, universe)

	g.logf("reading profiles for %d symbols", len(md.core))
	profiles, err := g.svc.Profiles(ctx, md.core)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		md.warnings = append(md.warnings, fmt.Sprintf("profiles: %v", err))
	}
	md.profiles = profiles

	peers, capped := pickPeers(md.core, profiles)
	if capped {
		md.warnings = append(md.warnings, fmt.Sprintf("industry peers capped at %d symbols; some sector median P/Es use fewer peers", maxPeers))
	}
	g.logf("reading ratios for %d symbols and %d peers", len(md.core), len(peers))
	funds, err := g.svc.Fundamentals(ctx, append(append([]string(nil), md.core...), peers...))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		md.warnings = append(md.warnings, fmt.Sprintf("ratios: %v", err))
	}
	md.funds = funds

	from := g.now.AddDate(-barYears, 0, 0)
	g.logf("reading daily bars for %d symbols", len(md.core))
	md.bars = g.svc.DailyBatch(ctx, md.core, from, g.now)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if idx, err := g.svc.Daily(ctx, indexSymbol, from, g.now); err == nil {
		md.index = idx
	} else if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	for _, sym := range sortedKeys(md.bars) {
		s := md.bars[sym]
		if s.Adjusted {
			continue
		}
		if adj, ok := research.AdjustJumps(s, research.PriceBand(profiles[sym].Exchange)); ok {
			md.bars[sym] = adj
			md.warnings = append(md.warnings, fmt.Sprintf("%s: %s bars are not adjusted for corporate actions; moves beyond the daily price band were back-adjusted", sym, s.Source))
		}
	}

	var stocks []string
	for _, sym := range md.core {
		if t, ok := heldTypes[sym]; ok && !portfolio.IsScored(t) {
			continue
		}
		if p, ok := profiles[sym]; ok && !advisor.IsStockType(p.Type, g.cfg) {
			continue
		}
		stocks = append(stocks, sym)
	}
	g.logf("reading dividends, annual profit and bank ratios for %d stocks", len(stocks))
	since := g.now.AddDate(-dividendYears, 0, 0)
	var mu sync.Mutex
	g.svc.ForEach(ctx, stocks, func(ctx context.Context, sym string) {
		divs, derr := g.svc.Dividends(ctx, sym, since)
		profits, _ := g.svc.NetProfitYears(ctx, sym)
		var bank research.Fundamentals
		isBank := md.sector(sym) == advisor.SectorBank
		if isBank {
			mu.Lock()
			f := md.funds[sym]
			mu.Unlock()
			if f.Symbol == "" {
				f.Symbol = sym
			}
			bank, _ = g.svc.BankExtras(ctx, f) // a failure keeps f and is warned by the service
		}
		mu.Lock()
		defer mu.Unlock()
		if derr != nil {
			md.divsFailed[sym] = true
		} else {
			md.divs[sym] = divs
		}
		if len(profits) > 0 {
			md.profits[sym] = profits
		}
		if isBank {
			md.funds[sym] = bank
			md.banks++
		}
	})
	return md, ctx.Err()
}

// sector classifies sym from its profile, or returns "" when neither the ICB
// data nor the bank ticker list knows it.
func (md *marketData) sector(sym string) advisor.Sector {
	p := md.profiles[sym]
	if sec, ok := advisor.ClassifySector(p.ICB1, p.ICB2, p.ICB4, sym); ok {
		return sec
	}
	return ""
}

// lastClose is the latest close of sym's daily bars, or 0.
func (md *marketData) lastClose(sym string) float64 {
	s := md.bars[sym]
	if len(s.Bars) == 0 {
		return 0
	}
	return s.Bars[len(s.Bars)-1].Close
}

// inputs builds the advisor inputs of sym at price (VND).
func (md *marketData) inputs(sym string, price float64, c advisor.Config) advisor.Inputs {
	p := md.profiles[sym]
	in := advisor.Inputs{
		Symbol:   sym,
		Sector:   md.sector(sym),
		Exchange: p.Exchange,
		Price:    price,
		Status:   p.Status,
		Type:     p.Type,
		Listed:   p.ListedDate,
	}

	if s, ok := md.bars[sym]; ok && len(s.Bars) > 0 {
		if v, ok := research.Volatility(s, yearSessions); ok {
			in.VOL = &v
		}
		if v, ok := research.MaxDrawdown(s, yearSessions); ok {
			in.MDD = &v
		}
		adtv := research.ADTV(s, liquiditySessions)
		in.ADTV = &adtv
		in.HistoryReturns = len(research.LogReturns(s))
		if len(md.index.Bars) > 0 {
			if b, r2, ok := research.Beta(s, md.index, yearSessions); ok {
				in.Beta, in.BetaR2 = &b, &r2
			}
			if b, ok := research.WeeklyBeta(s, md.index, weeklyBetaWeeks); ok {
				in.BetaWeekly = &b
			}
			in.GapSessions = research.GapSessions(s, md.index, gapWindow)
			in.StaleSessions = research.StaleSessions(s, md.index)
		}
	}

	f := md.funds[sym]
	in.ROE, in.DE, in.PE, in.PB = f.ROE, f.DE, f.PE, f.PB
	in.NPL, in.CASA = f.NPL, f.CASA
	if !f.AsOf.IsZero() {
		in.FundAge = md.now.Sub(f.AsOf)
	}
	in.NegEquity = f.BVPS != nil && *f.BVPS < 0
	in.TTMLoss = f.EPS != nil && *f.EPS < 0

	if years := md.profits[sym]; len(years) > 0 {
		in.ProfitYears = make([]float64, len(years))
		for i, v := range years {
			in.ProfitYears[len(years)-1-i] = v // newest first
		}
		in.PrevYearLoss = in.ProfitYears[0] < 0
	}

	if divs, ok := md.divs[sym]; ok && price > 0 {
		y, paid := research.TrailingCashYield(divs, price, md.now)
		in.DY, in.DivPaidYears = &y, paid
	}

	if in.Sector != advisor.SectorBank {
		var pes []float64
		for _, peer := range p.Peers {
			if pf, ok := md.funds[peer]; ok && pf.PE != nil {
				pes = append(pes, *pf.PE)
			}
		}
		in.SectorMedianPE = advisor.MedianPE(pes, c)
	}
	return in
}

// holdingResearch is the research block of a held symbol.
func (md *marketData) holdingResearch(sym string, sector advisor.Sector) report.HoldingResearch {
	p := md.profiles[sym]
	hr := report.HoldingResearch{
		Name:             p.Name,
		Exchange:         p.Exchange,
		Sector:           sector,
		ICB4:             p.ICB4,
		Dividends:        md.divs[sym],
		DividendsMissing: md.divsFailed[sym],
	}
	if s, ok := md.bars[sym]; ok {
		hr.Low52W, hr.High52W = research.Range(s, yearSessions)
	}
	return hr
}

// market is the VN-Index context line.
func (md *marketData) market() report.MarketContext {
	bars := md.index.Bars
	if len(bars) == 0 {
		return report.MarketContext{}
	}
	m := report.MarketContext{IndexClose: bars[len(bars)-1].Close, AsOf: bars[len(bars)-1].Date}
	m.Return1Y, m.HasReturn = research.Return(md.index, yearSessions)
	m.Vol1Y, m.HasVol = research.Volatility(md.index, yearSessions)
	return m
}

// stamps lists the public sources and the freshness of their data. Daily
// bars carry their latest bar date and ratios their oldest report date;
// the other rows show when the data was read (cached entries can be older,
// up to their cache lifetime).
func (md *marketData) stamps(now time.Time) []report.SourceStamp {
	var out []report.SourceStamp
	latest := map[string]time.Time{}
	note := func(s research.Series) {
		if len(s.Bars) == 0 {
			return
		}
		if d := s.Bars[len(s.Bars)-1].Date; d.After(latest[s.Source]) {
			latest[s.Source] = d
		}
	}
	for _, s := range md.bars {
		note(s)
	}
	note(md.index)
	for _, src := range sortedKeys(latest) {
		out = append(out, report.SourceStamp{Name: src, Kind: "daily bars (latest bar)", AsOf: latest[src]})
	}

	var oldest time.Time
	for _, sym := range md.core {
		if a := md.funds[sym].AsOf; !a.IsZero() && (oldest.IsZero() || a.Before(oldest)) {
			oldest = a
		}
	}
	if !oldest.IsZero() {
		out = append(out, report.SourceStamp{Name: "vndirect", Kind: "ratios (oldest report date)", AsOf: oldest})
	}
	if len(md.profiles) > 0 {
		out = append(out, report.SourceStamp{Name: "vndirect", Kind: "listing and ICB industry (read)", AsOf: now})
	}
	if len(md.divs) > 0 {
		out = append(out, report.SourceStamp{Name: "vndirect, vietcap fallback", Kind: "dividend events (read)", AsOf: now})
	}
	if len(md.profits) > 0 {
		out = append(out, report.SourceStamp{Name: "kbs", Kind: "annual net profit (read)", AsOf: now})
	}
	if md.banks > 0 {
		out = append(out, report.SourceStamp{Name: "vietcap", Kind: "bank NPL and CASA (read)", AsOf: now})
	}
	return out
}

// pickPeers returns the industry peers of the core symbols that are not core
// symbols themselves, in a fixed order, capped at maxPeers. capped reports
// whether some peers were left out.
func pickPeers(core []string, profiles map[string]research.Profile) (peers []string, capped bool) {
	seen := map[string]bool{}
	for _, s := range core {
		seen[s] = true
	}
	for _, sym := range core {
		ps := append([]string(nil), profiles[sym].Peers...)
		sort.Strings(ps)
		for _, p := range ps {
			p = strings.ToUpper(strings.TrimSpace(p))
			if p == "" || seen[p] {
				continue
			}
			seen[p] = true
			if len(peers) == maxPeers {
				capped = true
				continue
			}
			peers = append(peers, p)
		}
	}
	return peers, capped
}

// union returns a then b, upper-cased, without duplicates.
func union(a, b []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, list := range [][]string{a, b} {
		for _, s := range list {
			s = strings.ToUpper(strings.TrimSpace(s))
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
