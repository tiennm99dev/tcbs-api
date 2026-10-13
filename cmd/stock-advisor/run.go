package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/miti99dev/stock-advisor/internal/advisor"
	"github.com/miti99dev/stock-advisor/internal/config"
	"github.com/miti99dev/stock-advisor/internal/portfolio"
	"github.com/miti99dev/stock-advisor/internal/report"
	"github.com/miti99dev/stock-advisor/internal/research"
	"github.com/miti99dev/stock-advisor/tcbs"
)

// Exit codes.
const (
	exitOK    = 0
	exitFatal = 1
	exitUsage = 2
)

// Env looks up a process environment variable; os.Getenv in production.
type Env func(key string) string

// deps are the endpoints and clock behind run. main uses productionDeps;
// tests point every URL at local fixture servers.
type deps struct {
	tcbsURL string // "" means the SDK's production URL

	vndChartURL, vndFinfoURL string // "" means the research defaults
	vciChartURL, vciIQURL    string
	kbsURL                   string

	// transport carries every HTTP request, TCBS and public research alike.
	transport   http.RoundTripper
	ratePerHost int             // 0 means research.DefaultRatePerHost
	backoff     []time.Duration // nil means research.DefaultBackoff
	now         func() time.Time
}

func productionDeps() deps {
	return deps{transport: research.DefaultTransport(), now: time.Now}
}

// run is the whole CLI behind main, with its inputs and outputs injected.
func run(ctx context.Context, d deps, args []string, env Env, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := config.Parse(args, env)
	if errors.Is(err, flag.ErrHelp) {
		config.PrintUsage(stdout)
		return exitOK
	}
	if err != nil {
		fmt.Fprintf(stderr, "stock-advisor: %v\n", err)
		fmt.Fprintln(stderr, "Run stock-advisor -h for usage.")
		return exitUsage
	}
	if d.now == nil {
		d.now = time.Now
	}

	r := &runner{cfg: cfg, deps: d, stdin: stdin, stderr: stderr}
	r.secrets = append(r.secrets, cfg.APIKey)
	r.logf("config: %s", cfg)

	acfg := advisor.Default()
	acfg.FeeRate = cfg.FeeRate
	acfg.CandidateCount = cfg.Candidates
	if cfg.RulesFile != "" {
		if err := advisor.LoadOverrides(cfg.RulesFile, &acfg); err != nil {
			r.fail(err)
			return exitUsage
		}
	}
	acfg.Now = d.now()

	if cfg.CacheDir != "" {
		if err := os.MkdirAll(cfg.CacheDir, 0o700); err != nil {
			r.fail(fmt.Errorf("create cache dir: %w", err))
			return exitFatal
		}
	}

	snap, client, err := r.snapshot(ctx)
	if err != nil {
		r.fail(err)
		return exitFatal
	}
	if cfg.SaveHoldings != "" {
		if err := portfolio.Save(cfg.SaveHoldings, snap); err != nil {
			r.fail(err)
			return exitFatal
		}
		r.logf("saved the snapshot to %s", cfg.SaveHoldings)
	}

	data, err := r.analyse(ctx, snap, client, acfg)
	if err != nil {
		r.fail(err)
		return exitFatal
	}
	if err := r.write(stdout, func(w io.Writer) error { return report.Render(w, data) }); err != nil {
		r.fail(err)
		return exitFatal
	}
	return exitOK
}

type runner struct {
	cfg     config.Config
	deps    deps
	stdin   io.Reader
	stderr  io.Writer
	secrets []string // redacted from everything written to stderr or the report
}

// snapshot reads the portfolio from the offline file or from TCBS. The
// client is nil in offline mode.
func (r *runner) snapshot(ctx context.Context) (portfolio.Snapshot, *tcbs.Client, error) {
	if r.cfg.OfflineHoldings != "" {
		r.logf("reading holdings from %s", r.cfg.OfflineHoldings)
		snap, err := portfolio.FileSource{Path: r.cfg.OfflineHoldings}.Snapshot(ctx)
		return snap, nil, err
	}

	client := portfolio.NewTCBSClient(r.deps.tcbsURL, r.deps.transport, r.cfg.Timeout, r.cfg.DumpRaw)
	otp, err := config.PromptOTP(r.stdin, stdinFD(r.stdin), r.stderr)
	if err != nil {
		return portfolio.Snapshot{}, nil, err
	}
	r.secrets = append(r.secrets, otp)
	r.logf("signing in to TCBS")
	tok, err := client.GetToken(ctx, r.cfg.APIKey, otp)
	if err != nil {
		return portfolio.Snapshot{}, nil, fmt.Errorf("sign in: %w", err)
	}
	r.secrets = append(r.secrets, tok.Token)

	r.logf("reading sub-accounts, holdings, cash and margin")
	snap, err := portfolio.TCBSSource{
		Client:   client,
		Custody:  r.cfg.CustodyCode,
		Accounts: r.cfg.Accounts,
		RawDir:   r.cfg.DumpRaw,
	}.Snapshot(ctx)
	if err != nil {
		return portfolio.Snapshot{}, nil, err
	}
	if r.cfg.DumpRaw != "" {
		fmt.Fprintf(r.stderr, "Raw TCBS responses are in %s. They hold your name and custody code; delete them after inspection.\n", r.cfg.DumpRaw)
	}
	return snap, client, nil
}

// analyse fetches the public data, scores every symbol and assembles the
// report. Only a cancelled context is fatal: missing data becomes a warning.
func (r *runner) analyse(ctx context.Context, snap portfolio.Snapshot, client *tcbs.Client, acfg advisor.Config) (report.Data, error) {
	now := r.deps.now()
	svc := research.NewService(research.ServiceOptions{
		Transport:        r.deps.transport,
		Timeout:          r.cfg.Timeout,
		RatePerHost:      r.deps.ratePerHost,
		Backoff:          r.deps.backoff,
		Cache:            research.Cache{Dir: r.cfg.CacheDir, Refresh: r.cfg.Refresh},
		Now:              r.deps.now,
		VNDirectChartURL: r.deps.vndChartURL,
		VNDirectFinfoURL: r.deps.vndFinfoURL,
		VietcapChartURL:  r.deps.vciChartURL,
		VietcapIQURL:     r.deps.vciIQURL,
		KBSURL:           r.deps.kbsURL,
	})
	defer svc.Close()

	var warnings []string
	held, heldTypes := heldSymbols(snap)

	quotes := map[string]portfolio.Quote{}
	if client != nil && len(held) > 0 {
		r.logf("reading TCBS quotes for %d symbols", len(held))
		rows, err := client.GetStockPrices(ctx, held)
		if err != nil {
			warnings = append(warnings, r.redact(fmt.Sprintf("TCBS quotes unavailable, using public last close: %v", err)))
		} else {
			quotes = portfolio.TCBSQuotes(rows)
		}
	}

	universe, universeLabel, err := r.universe(ctx, svc)
	if err != nil {
		return report.Data{}, err
	}

	g := &gatherer{svc: svc, cfg: acfg, now: now, logf: r.logf}
	md, err := g.gather(ctx, held, heldTypes, universe)
	if err != nil {
		return report.Data{}, err
	}
	warnings = append(warnings, md.warnings...)

	for _, sym := range held {
		if _, ok := quotes[sym]; ok {
			continue
		}
		if s, ok := md.bars[sym]; ok && len(s.Bars) > 0 {
			quotes[sym] = portfolio.Quote{Price: int64(math.Round(s.Bars[len(s.Bars)-1].Close)), Source: s.Source}
		}
	}
	sum := portfolio.Aggregate(snap, quotes, r.cfg.IncludeBankCash)

	scores := map[string]advisor.Score{}
	sectors := map[string]advisor.Sector{}
	heldSet := map[string]bool{}
	resBlocks := map[string]report.HoldingResearch{}
	for _, p := range sum.Positions {
		heldSet[p.Symbol] = true
		in := md.inputs(p.Symbol, float64(p.Price), acfg)
		if p.Scored {
			s := advisor.ScoreStock(in, acfg)
			scores[p.Symbol] = s
			if in.Sector != "" {
				sectors[p.Symbol] = in.Sector
			}
		}
		sector := in.Sector
		if !p.Scored {
			sector = advisor.SectorFund
		}
		resBlocks[p.Symbol] = md.holdingResearch(p.Symbol, sector)
	}
	advice := advisor.Recommend(sum, scores, sectors, acfg)

	var pool []advisor.Inputs
	for _, sym := range universe {
		if heldSet[sym] {
			continue
		}
		pool = append(pool, md.inputs(sym, md.lastClose(sym), acfg))
	}
	cands, note := advisor.Candidates(pool, heldSet, sum, advice.SectorWeights, acfg)

	accounts := make([]report.Account, len(snap.Accounts))
	for i, a := range snap.Accounts {
		accounts[i] = report.Account{No: a.AccountNo, Type: a.Type}
	}
	warnings = append(warnings, svc.Warnings()...)
	for i, w := range warnings {
		warnings[i] = r.redact(w)
	}

	return report.Data{
		GeneratedAt:   now,
		Phase:         advisor.MarketPhase(now),
		Accounts:      accounts,
		Summary:       sum,
		Advice:        advice,
		Scores:        scores,
		Candidates:    cands,
		CandidateNote: note,
		Universe:      universeLabel,
		Sources:       r.stamps(snap, client != nil && len(quotes) > 0, md, now),
		Config:        acfg,
		Warnings:      warnings,
		Market:        md.market(),
		Research:      resBlocks,
	}, nil
}

// universe resolves --universe to a sorted ticker list and a label for the
// report. A failed constituent fetch leaves the universe empty with a
// warning in the service.
func (r *runner) universe(ctx context.Context, svc *research.Service) ([]string, string, error) {
	group := r.cfg.UniverseGroup()
	if group == "" {
		list := append([]string(nil), r.cfg.Universe...)
		sort.Strings(list)
		return list, "the --universe list (" + strings.Join(list, ", ") + ")", nil
	}
	r.logf("reading the %s constituents", group)
	members, err := svc.Universe(ctx, group)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		return nil, group + " (constituents unavailable)", nil
	}
	return members, fmt.Sprintf("%s (%d stocks)", group, len(members)), nil
}

// stamps lists the data sources used and how fresh each one is.
func (r *runner) stamps(snap portfolio.Snapshot, tcbsQuotes bool, md *marketData, now time.Time) []report.SourceStamp {
	var out []report.SourceStamp
	if r.cfg.OfflineHoldings != "" {
		out = append(out, report.SourceStamp{Name: "offline holdings file", Kind: "holdings, cash, margin", AsOf: snap.AsOf})
	} else {
		out = append(out, report.SourceStamp{Name: "TCBS OpenAPI", Kind: "holdings, cash, margin", AsOf: snap.AsOf})
	}
	if tcbsQuotes {
		out = append(out, report.SourceStamp{Name: "TCBS tickerCommons", Kind: "quotes for held symbols", AsOf: now})
	}
	return append(out, md.stamps(now)...)
}

// heldSymbols returns the held symbols, sorted, and the TCBS secType of each.
func heldSymbols(snap portfolio.Snapshot) ([]string, map[string]string) {
	types := map[string]string{}
	for _, a := range snap.Accounts {
		for _, h := range a.Holdings {
			sym := strings.ToUpper(strings.TrimSpace(h.Symbol))
			if sym == "" {
				continue
			}
			if types[sym] == "" {
				types[sym] = h.SecType // first non-empty secType across accounts
			}
		}
	}
	out := make([]string, 0, len(types))
	for sym := range types {
		out = append(out, sym)
	}
	sort.Strings(out)
	return out, types
}

// stdinFD returns the file descriptor behind in, or -1 when in is not a file.
func stdinFD(in io.Reader) int {
	if f, ok := in.(*os.File); ok {
		return int(f.Fd())
	}
	return -1
}

// write renders to --out (mode 0600) or to stdout for "-".
func (r *runner) write(stdout io.Writer, render func(io.Writer) error) error {
	if r.cfg.Out == "-" {
		return render(stdout)
	}
	if dir := filepath.Dir(r.cfg.Out); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}
	f, err := os.OpenFile(r.cfg.Out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open output: %w", err)
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return fmt.Errorf("open output: %w", err)
	}
	if err := render(f); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	r.logf("wrote %s", r.cfg.Out)
	return nil
}

func (r *runner) logf(format string, a ...any) {
	if r.cfg.Verbose {
		fmt.Fprintln(r.stderr, r.redact(fmt.Sprintf(format, a...)))
	}
}

func (r *runner) fail(err error) {
	fmt.Fprintf(r.stderr, "stock-advisor: %s\n", r.redact(err.Error()))
}

// redact masks every known secret in s.
func (r *runner) redact(s string) string {
	for _, secret := range r.secrets {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "***")
		}
	}
	return s
}
