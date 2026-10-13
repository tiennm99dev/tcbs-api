package research

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Units: prices are VND, money is VND, ratios are fractions. Index series are
// in index points. Every date (Bar.Date, Dividend.ExDate, ...) is the trading
// day at 00:00 UTC, whatever the source's own time stamp, so series from
// different providers line up on the same keys.

// Bar is one daily OHLCV bar.
type Bar struct {
	Date                   time.Time
	Open, High, Low, Close float64
	Volume                 float64
}

// Series is a daily price history, oldest bar first.
type Series struct {
	Symbol, Source string
	Adjusted       bool
	Bars           []Bar
}

// Fundamentals holds the latest ratios for one symbol. A nil pointer means
// the value is missing.
type Fundamentals struct {
	Symbol string
	AsOf   time.Time // oldest report date among the ratios used
	Source string

	PE, PB, ROE, ROA, DivYield, Beta, DE, EPS, BVPS, MarketCap, NPL, CASA *float64

	NetProfitYears []float64 // most recent last; empty = missing
}

// Dividend is one corporate action that pays shareholders.
type Dividend struct {
	ExDate       time.Time
	CashPerShare float64 // VND per share; 0 for stock dividends
	StockRatio   float64 // new shares per held share (0.15 for 100:15); 0 for cash
}

// Profile describes a listed security.
type Profile struct {
	Symbol, Name, Exchange string
	Type                   string // STOCK, ETF, ...
	Status                 string // "listed" for normal stocks
	ICB1, ICB2, ICB4       string
	Peers                  []string // other symbols in the same ICB level-4 industry
	ListedDate             time.Time
}

// Provider is a source of daily bars.
type Provider interface {
	Name() string
	Daily(ctx context.Context, symbol string, from, to time.Time) (Series, error)
}

// indexSymbols are quoted in points, not prices.
var indexSymbols = map[string]bool{
	"VNINDEX": true, "VN30": true, "VN100": true,
	"HNXINDEX": true, "HNX30": true, "UPCOMINDEX": true,
}

// IsIndex reports whether symbol is a market index rather than a security.
func IsIndex(symbol string) bool {
	return indexSymbols[strings.ToUpper(symbol)]
}

// dayUTC returns the calendar day of t, as seen in exchange time, at
// 00:00 UTC.
func dayUTC(t time.Time) time.Time {
	y, m, d := t.In(marketTZ).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// parseDay parses the leading date of s in the given layout into a day at
// 00:00 UTC.
func parseDay(layout, s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) > len(layout) {
		s = s[:len(layout)]
	}
	t, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
}

// flexNum decodes a JSON number, a numeric string or null. Valid is false
// for null, "" and missing values.
type flexNum struct {
	V     float64
	Valid bool
}

func (n *flexNum) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	*n = flexNum{}
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return nil
	}
	s := string(b)
	if b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return fmt.Errorf("research: decode number: %w", err)
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("research: decode number %q: %w", s, err)
	}
	*n = flexNum{V: f, Valid: true}
	return nil
}

func (n flexNum) ptr() *float64 {
	if !n.Valid {
		return nil
	}
	v := n.V
	return &v
}

func ptr(v float64) *float64 { return &v }

func normSymbols(symbols []string) []string {
	seen := make(map[string]bool, len(symbols))
	out := make([]string, 0, len(symbols))
	for _, s := range symbols {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func chunks(items []string, size int) [][]string {
	var out [][]string
	for len(items) > 0 {
		n := min(size, len(items))
		out = append(out, items[:n])
		items = items[n:]
	}
	return out
}

// filterBars keeps the bars whose day lies within [from, to].
func filterBars(bars []Bar, from, to time.Time) []Bar {
	lo, hi := dayUTC(from), dayUTC(to)
	out := bars[:0:0]
	for _, b := range bars {
		if b.Date.Before(lo) || b.Date.After(hi) {
			continue
		}
		out = append(out, b)
	}
	return out
}
