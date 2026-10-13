package research

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultWorkers bounds the per-symbol fetches running at once.
	DefaultWorkers = 4
	// batchSize is the most symbols sent in one VNDirect list query.
	batchSize = 40
)

// ServiceOptions configures NewService. Zero values select the production
// defaults.
type ServiceOptions struct {
	Transport   http.RoundTripper // nil = http.DefaultTransport
	Timeout     time.Duration     // per request; 0 = DefaultTimeout
	RatePerHost int               // requests per second per host; 0 = DefaultRatePerHost
	Backoff     []time.Duration   // nil = DefaultBackoff
	Workers     int               // 0 = DefaultWorkers

	Cache Cache
	Now   func() time.Time

	VNDirectChartURL string
	VNDirectFinfoURL string
	VietcapChartURL  string
	VietcapIQURL     string
	KBSURL           string
}

// Service combines the providers: it falls back from one source to the
// next, batches list queries, caches results and records coverage warnings.
type Service struct {
	http      *HTTPClient
	vnd       *VNDirect
	vci       *Vietcap
	kbs       *KBS
	providers []Provider
	cache     Cache
	now       func() time.Time
	workers   int

	mu       sync.Mutex
	warnings []string
}

// NewService builds a Service. Call Close when done.
func NewService(opts ServiceOptions) *Service {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	if opts.Cache.Now == nil {
		opts.Cache.Now = now
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = DefaultWorkers
	}
	h := NewHTTPClient(opts.Transport, opts.Timeout, opts.RatePerHost, opts.Backoff)
	s := &Service{
		http:    h,
		vnd:     &VNDirect{HTTP: h, ChartURL: or(opts.VNDirectChartURL, VNDirectChartURL), FinfoURL: or(opts.VNDirectFinfoURL, VNDirectFinfoURL), Now: now},
		vci:     &Vietcap{HTTP: h, ChartURL: or(opts.VietcapChartURL, VietcapChartURL), IQURL: or(opts.VietcapIQURL, VietcapIQURL)},
		kbs:     &KBS{HTTP: h, BaseURL: or(opts.KBSURL, KBSURL)},
		cache:   opts.Cache,
		now:     now,
		workers: workers,
	}
	s.providers = []Provider{s.vnd, s.vci, s.kbs}
	return s
}

func or(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// Close releases the HTTP rate limiters.
func (s *Service) Close() { s.http.Close() }

// Warnings returns the coverage notes recorded so far, in order.
func (s *Service) Warnings() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.warnings...)
}

func (s *Service) warnf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warnings = append(s.warnings, fmt.Sprintf(format, args...))
}

func (s *Service) put(key string, v any) {
	if err := s.cache.Put(key, v); err != nil {
		s.warnf("cache: %v", err)
	}
}

// ForEach runs fn for every symbol with at most Workers calls in flight. It
// stops starting new calls once ctx is done.
func (s *Service) ForEach(ctx context.Context, symbols []string, fn func(ctx context.Context, symbol string)) {
	sem := make(chan struct{}, s.workers)
	var wg sync.WaitGroup
	for _, sym := range symbols {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return
		}
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(ctx, sym)
		}(sym)
	}
	wg.Wait()
}

// Daily returns symbol's daily bars in [from, to], trying each provider in
// order (VNDirect, Vietcap, KBS) until one returns bars.
func (s *Service) Daily(ctx context.Context, symbol string, from, to time.Time) (Series, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	key := cacheKey("bars", "daily", symbol+"-"+dayUTC(from).Format("20060102")+"-"+dayUTC(to).Format("20060102"))
	var cached Series
	if s.cache.GetFresh(key, DailyFresh, &cached) {
		return cached, nil
	}

	var errs []error
	for _, p := range s.providers {
		series, err := p.Daily(ctx, symbol, from, to)
		if err == nil && len(series.Bars) == 0 {
			err = fmt.Errorf("%s: %s daily: %w", p.Name(), symbol, ErrNoData)
		}
		if err != nil {
			if ctx.Err() != nil {
				return Series{}, ctx.Err()
			}
			errs = append(errs, err)
			continue
		}
		if len(errs) > 0 {
			s.warnf("%s: daily bars from %s after %v", symbol, p.Name(), errors.Join(errs...))
		}
		s.put(key, series)
		return series, nil
	}
	err := fmt.Errorf("research: %s daily bars: %w", symbol, errors.Join(errs...))
	s.warnf("%s: no daily bars from any source", symbol)
	return Series{}, err
}

// DailyBatch fetches daily bars for many symbols in parallel. Symbols with
// no data are left out and noted in Warnings.
func (s *Service) DailyBatch(ctx context.Context, symbols []string, from, to time.Time) map[string]Series {
	out := make(map[string]Series)
	var mu sync.Mutex
	s.ForEach(ctx, normSymbols(symbols), func(ctx context.Context, sym string) {
		series, err := s.Daily(ctx, sym, from, to)
		if err != nil {
			return
		}
		mu.Lock()
		out[sym] = series
		mu.Unlock()
	})
	return out
}

// Fundamentals returns the VNDirect ratios of every symbol. NPL, CASA and
// NetProfitYears are left for BankExtras and NetProfitYears, which only
// holdings and universe members need. Symbols are
// queried in batches of 40; a failed batch is retried symbol by symbol, and
// a symbol that still fails gets an entry with every field nil.
func (s *Service) Fundamentals(ctx context.Context, symbols []string) (map[string]Fundamentals, error) {
	symbols = normSymbols(symbols)
	out := make(map[string]Fundamentals, len(symbols))
	var missing []string
	for _, sym := range symbols {
		var f Fundamentals
		if s.cache.Get(cacheKey("vndirect", "ratios", sym), TTLRatios, &f) {
			out[sym] = f
			continue
		}
		missing = append(missing, sym)
	}

	store := func(sym string, got map[string]Fundamentals) {
		f, ok := got[sym]
		if !ok {
			f = Fundamentals{Symbol: sym}
		}
		out[sym] = f
		s.put(cacheKey("vndirect", "ratios", sym), f)
	}

	var retry []string
	for _, chunk := range chunks(missing, batchSize) {
		got, err := s.vnd.Ratios(ctx, chunk)
		if err != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			retry = append(retry, chunk...)
			continue
		}
		for _, sym := range chunk {
			store(sym, got)
		}
	}

	var mu sync.Mutex
	s.ForEach(ctx, retry, func(ctx context.Context, sym string) {
		got, err := s.vnd.Ratios(ctx, []string{sym})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			out[sym] = Fundamentals{Symbol: sym}
			s.warnf("%s: ratios unavailable: %v", sym, err)
			return
		}
		store(sym, got)
	})
	return out, ctx.Err()
}

// BankExtras adds Vietcap's NPL and CASA to f and fills its PE, PB, ROE and
// DE when VNDirect had none. Call it only for banks.
func (s *Service) BankExtras(ctx context.Context, f Fundamentals) (Fundamentals, error) {
	sym := strings.ToUpper(f.Symbol)
	key := cacheKey("vietcap", "bank", sym)
	var b BankRatios
	if !s.cache.Get(key, TTLRatios, &b) {
		var err error
		b, err = s.vci.BankRatios(ctx, sym)
		if err != nil {
			s.warnf("%s: bank ratios (NPL, CASA) unavailable: %v", sym, err)
			return f, err
		}
		s.put(key, b)
	}
	f.NPL, f.CASA = b.NPL, b.CASA
	fill := func(dst **float64, src *float64) {
		if *dst == nil && src != nil {
			*dst = src
		}
	}
	fill(&f.PE, b.PE)
	fill(&f.PB, b.PB)
	fill(&f.ROE, b.ROE)
	fill(&f.DE, b.DE)
	if f.AsOf.IsZero() || (!b.AsOf.IsZero() && b.AsOf.Before(f.AsOf)) {
		f.AsOf = b.AsOf
	}
	return f, nil
}

// NetProfitYears returns symbol's annual net profit from KBS, oldest year
// first. It is empty when fewer than 3 years are available.
func (s *Service) NetProfitYears(ctx context.Context, symbol string) ([]float64, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	key := cacheKey("kbs", "profit", sym)
	var years []float64
	if s.cache.Get(key, TTLRatios, &years) {
		return years, nil
	}
	years, err := s.kbs.NetProfitYears(ctx, sym)
	if err != nil {
		s.warnf("%s: annual profit unavailable: %v", sym, err)
		return nil, err
	}
	s.put(key, years)
	return years, nil
}

// Dividends returns symbol's cash and stock dividends with ex-dates on or
// after since, newest first, from VNDirect or else Vietcap.
func (s *Service) Dividends(ctx context.Context, symbol string, since time.Time) ([]Dividend, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	key := cacheKey("events", "dividends", sym+"-"+dayUTC(since).Format("20060102"))
	var divs []Dividend
	if s.cache.Get(key, TTLEvents, &divs) {
		return divs, nil
	}

	all, err := s.vnd.Dividends(ctx, sym)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var err2 error
		all, err2 = s.vci.Dividends(ctx, sym, since, s.now().AddDate(1, 0, 0))
		if err2 != nil {
			s.warnf("%s: dividends unavailable", sym)
			return nil, fmt.Errorf("research: %s dividends: %w", sym, errors.Join(err, err2))
		}
		s.warnf("%s: dividends from vietcap after %v", sym, err)
	}
	lo := dayUTC(since)
	divs = []Dividend{}
	for _, d := range all {
		if !d.ExDate.Before(lo) {
			divs = append(divs, d)
		}
	}
	s.put(key, divs)
	return divs, nil
}

// Profiles returns the listing and industry data of every symbol VNDirect
// knows. Unknown symbols are left out and noted in Warnings.
func (s *Service) Profiles(ctx context.Context, symbols []string) (map[string]Profile, error) {
	symbols = normSymbols(symbols)
	out := make(map[string]Profile, len(symbols))
	var missing []string
	for _, sym := range symbols {
		var p Profile
		if s.cache.Get(cacheKey("vndirect", "profile", sym), TTLProfile, &p) {
			out[sym] = p
			continue
		}
		missing = append(missing, sym)
	}

	listed := make(map[string]Profile)
	for _, chunk := range chunks(missing, batchSize) {
		got, err := s.vnd.Listings(ctx, chunk)
		if err != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			s.warnf("profiles unavailable for %s: %v", strings.Join(chunk, ","), err)
			continue
		}
		for sym, p := range got {
			listed[sym] = p
		}
	}

	var found []string
	for _, sym := range missing {
		if _, ok := listed[sym]; ok {
			found = append(found, sym)
		} else {
			s.warnf("%s: no listing data", sym)
		}
	}

	var mu sync.Mutex
	s.ForEach(ctx, found, func(ctx context.Context, sym string) {
		p := listed[sym]
		icb1, icb2, icb4, peers, err := s.vnd.Industry(ctx, sym)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			// Keep the listing; leave it uncached so the next run retries.
			s.warnf("%s: industry classification unavailable: %v", sym, err)
			out[sym] = p
			return
		}
		p.ICB1, p.ICB2, p.ICB4, p.Peers = icb1, icb2, icb4, peers
		out[sym] = p
		s.put(cacheKey("vndirect", "profile", sym), p)
	})
	return out, ctx.Err()
}

// Universe returns the constituents of group, "VN30" or "VN100", sorted.
// VN30 comes from VNDirect, else KBS. VN100 comes from KBS only; when that
// fails it falls back to VN30 and records a warning.
func (s *Service) Universe(ctx context.Context, group string) ([]string, error) {
	group = strings.ToUpper(strings.TrimSpace(group))
	switch group {
	case "VN30", "VN100":
	default:
		return nil, fmt.Errorf("research: unknown universe %q (want VN30 or VN100)", group)
	}
	key := cacheKey("universe", "constituents", group)
	var members []string
	if s.cache.Get(key, TTLConstituents, &members) {
		return members, nil
	}

	var err error
	if group == "VN100" {
		members, err = s.kbsMembers(ctx, group)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			s.warnf("VN100 list unavailable (%v); using VN30 instead", err)
			return s.Universe(ctx, "VN30")
		}
	} else {
		members, err = s.vnd.IndexMembers(ctx, group)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var err2 error
			members, err2 = s.kbsMembers(ctx, group)
			if err2 != nil {
				return nil, fmt.Errorf("research: %s constituents: %w", group, errors.Join(err, err2))
			}
		}
	}
	sort.Strings(members)
	s.put(key, members)
	return members, nil
}

func (s *Service) kbsMembers(ctx context.Context, index string) ([]string, error) {
	g, ok := kbsGroup(index)
	if !ok {
		return nil, fmt.Errorf("research: no KBS group for %s", index)
	}
	return s.kbs.IndexMembers(ctx, g)
}
