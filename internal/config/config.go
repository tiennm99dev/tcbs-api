// Package config resolves the stock-advisor CLI settings from flags, the
// process environment and an optional .env file.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

// Environment keys read from the process env or the .env file.
const (
	EnvAPIKey      = "TCBS_API_KEY"
	EnvCustodyCode = "TCBS_CUSTODY_CODE"
)

// Universe group names accepted by --universe.
const (
	UniverseVN30  = "VN30"
	UniverseVN100 = "VN100"
)

const (
	defaultEnvFile    = ".env"
	defaultCandidates = 10
	defaultOut        = "-"
	defaultCacheDir   = ".cache/stock-advisor"
	defaultFeeRate    = 0.001
	defaultTimeout    = 20 * time.Second
)

// Config holds the resolved CLI settings.
type Config struct {
	APIKey, CustodyCode                                              string // APIKey is never printed
	Accounts                                                         []string
	Candidates                                                       int
	Universe                                                         []string // a single group name (VN30, VN100) or a ticker list
	Out, OfflineHoldings, SaveHoldings, DumpRaw, CacheDir, RulesFile string
	Refresh, IncludeBankCash, Verbose                                bool
	FeeRate                                                          float64
	Timeout                                                          time.Duration
}

// UniverseGroup returns the group name when Universe names a group, or "".
func (c Config) UniverseGroup() string {
	if len(c.Universe) == 1 && (c.Universe[0] == UniverseVN30 || c.Universe[0] == UniverseVN100) {
		return c.Universe[0]
	}
	return ""
}

// flagValues holds raw flag input before precedence is applied.
type flagValues struct {
	custody, envFile, accounts, universe string
	cfg                                  Config
}

func newFlagSet(v *flagValues) *flag.FlagSet {
	fs := flag.NewFlagSet("stock-advisor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&v.custody, "custody", "", "custody code (105C...); overrides "+EnvCustodyCode)
	fs.StringVar(&v.envFile, "env-file", defaultEnvFile, "path of the .env file (it is fine if the default is missing)")
	fs.StringVar(&v.accounts, "accounts", "", "comma list of sub-account numbers to include (default: all active)")
	fs.IntVar(&v.cfg.Candidates, "candidates", defaultCandidates, "number of candidates to show")
	fs.StringVar(&v.universe, "universe", UniverseVN100, "VN30, VN100, or a comma list of tickers")
	fs.StringVar(&v.cfg.Out, "out", defaultOut, "report path; - means stdout")
	fs.StringVar(&v.cfg.OfflineHoldings, "offline-holdings", "", "read a portfolio snapshot JSON instead of TCBS")
	fs.StringVar(&v.cfg.SaveHoldings, "save-holdings", "", "write the pulled snapshot as JSON (mode 0600)")
	fs.StringVar(&v.cfg.DumpRaw, "dump-raw", "", "dir for the raw TCBS JSON responses (debug; delete after inspection)")
	fs.StringVar(&v.cfg.CacheDir, "cache-dir", defaultCacheDir, "public-data cache dir")
	fs.BoolVar(&v.cfg.Refresh, "refresh", false, "ignore cache freshness and refetch")
	fs.Float64Var(&v.cfg.FeeRate, "fee-rate", defaultFeeRate, "broker fee per side")
	fs.BoolVar(&v.cfg.IncludeBankCash, "include-bank-cash", false, "count the IA linked-bank balance in NAV")
	fs.StringVar(&v.cfg.RulesFile, "rules", "", "JSON file overriding the advisor thresholds")
	fs.DurationVar(&v.cfg.Timeout, "timeout", defaultTimeout, "per-request HTTP timeout")
	fs.BoolVar(&v.cfg.Verbose, "v", false, "progress on stderr")
	return fs
}

// PrintUsage writes the flag help to w.
func PrintUsage(w io.Writer) {
	fs := newFlagSet(&flagValues{})
	fmt.Fprintln(w, "Usage: stock-advisor [flags]")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s and %s are read from the process env or the .env file.\n\n", EnvAPIKey, EnvCustodyCode)
	fs.SetOutput(w)
	fs.PrintDefaults()
}

// Parse resolves the config. Precedence: flag, then process env, then the
// .env file, then the default. It returns flag.ErrHelp for -h/--help.
func Parse(args []string, getenv func(string) string) (Config, error) {
	var v flagValues
	fs := newFlagSet(&v)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Config{}, err
		}
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("config: unexpected argument %q", fs.Arg(0))
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	dotenv, err := LoadDotEnv(v.envFile)
	if err != nil {
		if !isNotExist(err) || set["env-file"] {
			return Config{}, fmt.Errorf("config: %w", err)
		}
		dotenv = nil
	}
	lookup := func(key string) string {
		if getenv != nil {
			if val := strings.TrimSpace(getenv(key)); val != "" {
				return val
			}
		}
		return strings.TrimSpace(dotenv[key])
	}

	c := v.cfg
	c.APIKey = lookup(EnvAPIKey)
	c.CustodyCode = strings.TrimSpace(v.custody)
	if !set["custody"] {
		c.CustodyCode = lookup(EnvCustodyCode)
	}
	c.CustodyCode = strings.ToUpper(c.CustodyCode)
	c.Accounts = splitList(v.accounts, false)
	c.Universe = splitList(v.universe, true)

	if err := c.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return c, nil
}

func (c Config) validate() error {
	if c.OfflineHoldings == "" {
		if c.APIKey == "" {
			return fmt.Errorf("%s is required (process env or .env) unless --offline-holdings is set", EnvAPIKey)
		}
		if c.CustodyCode == "" {
			return fmt.Errorf("--custody or %s is required unless --offline-holdings is set", EnvCustodyCode)
		}
	}
	if c.Candidates < 0 {
		return errors.New("--candidates must be 0 or more")
	}
	if len(c.Universe) == 0 {
		return errors.New("--universe must name VN30, VN100, or at least one ticker")
	}
	if c.FeeRate < 0 || c.FeeRate >= 0.1 {
		return errors.New("--fee-rate must be in [0, 0.1)")
	}
	if c.Timeout <= 0 {
		return errors.New("--timeout must be positive")
	}
	if c.Out == "" {
		return errors.New("--out must not be empty; use - for stdout")
	}
	return nil
}

// splitList splits a comma list, trimming blanks and dropping duplicates.
func splitList(s string, upper bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if upper {
			part = strings.ToUpper(part)
		}
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

// String renders the config for logs, with the API key redacted as "***".
func (c Config) String() string {
	key := ""
	if c.APIKey != "" {
		key = "***"
	}
	return fmt.Sprintf("Config{APIKey:%s CustodyCode:%s Accounts:%v Candidates:%d Universe:%v Out:%s OfflineHoldings:%s SaveHoldings:%s DumpRaw:%s CacheDir:%s RulesFile:%s Refresh:%t IncludeBankCash:%t Verbose:%t FeeRate:%g Timeout:%s}",
		key, c.CustodyCode, c.Accounts, c.Candidates, c.Universe, c.Out, c.OfflineHoldings,
		c.SaveHoldings, c.DumpRaw, c.CacheDir, c.RulesFile, c.Refresh, c.IncludeBankCash, c.Verbose,
		c.FeeRate, c.Timeout)
}

// GoString keeps %#v from printing the raw API key.
func (c Config) GoString() string { return c.String() }
