package research

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Default VNDirect hosts.
const (
	VNDirectChartURL = "https://dchart-api.vndirect.com.vn"
	VNDirectFinfoURL = "https://api-finfo.vndirect.com.vn"
)

// ratioCodes maps VNDirect ratio codes to Fundamentals fields.
var ratioCodes = map[string]func(f *Fundamentals) **float64{
	"PRICE_TO_EARNINGS": func(f *Fundamentals) **float64 { return &f.PE },
	"PRICE_TO_BOOK":     func(f *Fundamentals) **float64 { return &f.PB },
	"ROAE_TR_AVG5Q":     func(f *Fundamentals) **float64 { return &f.ROE },
	"ROAA_TR_AVG5Q":     func(f *Fundamentals) **float64 { return &f.ROA },
	"DIVIDEND_YIELD":    func(f *Fundamentals) **float64 { return &f.DivYield },
	"BETA":              func(f *Fundamentals) **float64 { return &f.Beta },
	"DEBT_TO_EQUITY_AQ": func(f *Fundamentals) **float64 { return &f.DE },
	"EPS_TR":            func(f *Fundamentals) **float64 { return &f.EPS },
	"BVPS_CR":           func(f *Fundamentals) **float64 { return &f.BVPS },
	"MARKETCAP":         func(f *Fundamentals) **float64 { return &f.MarketCap },
}

// ratioCodeList is ratioCodes' keys in a fixed order for the request.
var ratioCodeList = func() []string {
	out := make([]string, 0, len(ratioCodes))
	for k := range ratioCodes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}()

// VNDirect is the primary source: dchart for daily bars, finfo for ratios,
// events, listings and industry classification.
type VNDirect struct {
	HTTP     *HTTPClient
	ChartURL string
	FinfoURL string
	Now      func() time.Time
}

// Name implements Provider.
func (v *VNDirect) Name() string { return "vndirect" }

func (v *VNDirect) now() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now()
}

type dchartResponse struct {
	S string    `json:"s"`
	T []int64   `json:"t"`
	O []float64 `json:"o"`
	H []float64 `json:"h"`
	L []float64 `json:"l"`
	C []float64 `json:"c"`
	V []float64 `json:"v"`
}

// Daily implements Provider. dchart prices are adjusted and quoted in
// thousand VND; index series are in points.
func (v *VNDirect) Daily(ctx context.Context, symbol string, from, to time.Time) (Series, error) {
	symbol = strings.ToUpper(symbol)
	q := url.Values{}
	q.Set("resolution", "D")
	q.Set("symbol", symbol)
	q.Set("from", strconv.FormatInt(from.Unix(), 10))
	q.Set("to", strconv.FormatInt(to.Unix(), 10))

	var r dchartResponse
	if err := v.HTTP.GetJSON(ctx, joinURL(v.ChartURL, "/dchart/history?")+q.Encode(), nil, &r); err != nil {
		return Series{}, err
	}
	if r.S != "ok" {
		return Series{}, fmt.Errorf("vndirect: %s daily: status %q: %w", symbol, r.S, ErrNoData)
	}
	n := len(r.T)
	if len(r.O) != n || len(r.H) != n || len(r.L) != n || len(r.C) != n || len(r.V) != n {
		return Series{}, fmt.Errorf("vndirect: %s daily: column lengths differ", symbol)
	}

	scale := func(x float64) float64 { return math.Round(x * 1000) }
	if IsIndex(symbol) {
		scale = func(x float64) float64 { return x }
	}
	bars := make([]Bar, 0, n)
	for i := 0; i < n; i++ {
		bars = append(bars, Bar{
			Date:   dayUTC(time.Unix(r.T[i], 0)),
			Open:   scale(r.O[i]),
			High:   scale(r.H[i]),
			Low:    scale(r.L[i]),
			Close:  scale(r.C[i]),
			Volume: r.V[i],
		})
	}
	sortBars(bars)
	return Series{Symbol: symbol, Source: v.Name(), Adjusted: true, Bars: filterBars(bars, from, to)}, nil
}

type finfoRatio struct {
	Code       string  `json:"code"`
	RatioCode  string  `json:"ratioCode"`
	Value      flexNum `json:"value"`
	ReportDate string  `json:"reportDate"`
}

// Ratios fetches the latest ratios for up to a few dozen symbols in one
// call. Symbols the source does not know are absent from the result.
func (v *VNDirect) Ratios(ctx context.Context, symbols []string) (map[string]Fundamentals, error) {
	symbols = normSymbols(symbols)
	if len(symbols) == 0 {
		return map[string]Fundamentals{}, nil
	}
	q := url.Values{}
	q.Set("filter", "ratioCode:"+strings.Join(ratioCodeList, ","))
	q.Set("where", "code:"+strings.Join(symbols, ","))
	q.Set("order", "reportDate")
	q.Set("fields", "code,ratioCode,value,reportDate")

	var r struct {
		Data []finfoRatio `json:"data"`
	}
	if err := v.HTTP.GetJSON(ctx, joinURL(v.FinfoURL, "/v4/ratios/latest?")+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	return parseRatios(r.Data), nil
}

func parseRatios(rows []finfoRatio) map[string]Fundamentals {
	out := make(map[string]Fundamentals)
	for _, row := range rows {
		field, ok := ratioCodes[row.RatioCode]
		if !ok || !row.Value.Valid {
			continue
		}
		sym := strings.ToUpper(row.Code)
		f := out[sym]
		f.Symbol, f.Source = sym, "vndirect"
		*field(&f) = row.Value.ptr()
		if d, err := parseDay(time.DateOnly, row.ReportDate); err == nil && (f.AsOf.IsZero() || d.Before(f.AsOf)) {
			f.AsOf = d
		}
		out[sym] = f
	}
	return out
}

type finfoEvent struct {
	Code          string  `json:"code"`
	Type          string  `json:"type"`
	Dividend      flexNum `json:"dividend"`
	Ratio         flexNum `json:"ratio"`
	EffectiveDate string  `json:"effectiveDate"`
}

// Dividends returns the cash and stock dividends of symbol, newest first.
// Stock payouts come as STOCKDIV or, for bonus-share issues, KINDDIV.
// Events dated more than a year ahead are placeholders and are dropped.
func (v *VNDirect) Dividends(ctx context.Context, symbol string) ([]Dividend, error) {
	symbol = strings.ToUpper(symbol)
	q := url.Values{}
	q.Set("q", "code:"+symbol+"~type:DIVIDEND,STOCKDIV,KINDDIV~locale:EN_GB")
	q.Set("sort", "effectiveDate:desc")
	q.Set("size", "50")

	var r struct {
		Data []finfoEvent `json:"data"`
	}
	if err := v.HTTP.GetJSON(ctx, joinURL(v.FinfoURL, "/v4/events?")+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	return parseFinfoEvents(r.Data, v.now()), nil
}

func parseFinfoEvents(rows []finfoEvent, now time.Time) []Dividend {
	limit := dayUTC(now).AddDate(1, 0, 0)
	var out []Dividend
	for _, e := range rows {
		ex, err := parseDay(time.DateOnly, e.EffectiveDate)
		if err != nil || ex.After(limit) {
			continue
		}
		switch e.Type {
		case "DIVIDEND":
			cash := e.Dividend.V
			if cash == 0 {
				cash = e.Ratio.V * 100 // percent of the 10,000 VND par value
			}
			if cash > 0 {
				out = append(out, Dividend{ExDate: ex, CashPerShare: cash})
			}
		case "STOCKDIV", "KINDDIV":
			if e.Ratio.V > 0 {
				out = append(out, Dividend{ExDate: ex, StockRatio: e.Ratio.V / 100})
			}
		}
	}
	sortDividends(out)
	return out
}

type finfoStock struct {
	Code           string `json:"code"`
	Type           string `json:"type"`
	Floor          string `json:"floor"`
	Status         string `json:"status"`
	CompanyName    string `json:"companyName"`
	CompanyNameEng string `json:"companyNameEng"`
	ShortNameEng   string `json:"shortNameEng"`
	ListedDate     string `json:"listedDate"`
}

// Listings returns the exchange, type, status, listing date and name of
// each symbol, without industry data.
func (v *VNDirect) Listings(ctx context.Context, symbols []string) (map[string]Profile, error) {
	symbols = normSymbols(symbols)
	out := make(map[string]Profile, len(symbols))
	if len(symbols) == 0 {
		return out, nil
	}
	rows, err := v.stocks(ctx, "code:"+strings.Join(symbols, ","))
	if err != nil {
		return nil, err
	}
	for _, s := range rows {
		p := Profile{
			Symbol:   strings.ToUpper(s.Code),
			Name:     firstNonEmpty(s.CompanyNameEng, s.ShortNameEng, s.CompanyName),
			Exchange: strings.TrimSpace(s.Floor),
			Type:     strings.TrimSpace(s.Type),
			Status:   strings.TrimSpace(s.Status),
		}
		if d, err := parseDay(time.DateOnly, s.ListedDate); err == nil {
			p.ListedDate = d
		}
		out[p.Symbol] = p
	}
	return out, nil
}

// IndexMembers lists the stocks whose VNDirect indexCode is index (VN30
// works; each stock carries a single index code, so VN100 does not).
func (v *VNDirect) IndexMembers(ctx context.Context, index string) ([]string, error) {
	rows, err := v.stocks(ctx, "indexCode:"+strings.ToUpper(index))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, s := range rows {
		out = append(out, s.Code)
	}
	out = normSymbols(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("vndirect: index %s: %w", index, ErrNoData)
	}
	sort.Strings(out)
	return out, nil
}

func (v *VNDirect) stocks(ctx context.Context, query string) ([]finfoStock, error) {
	q := url.Values{}
	q.Set("q", query)
	q.Set("size", "100")
	var r struct {
		Data []finfoStock `json:"data"`
	}
	if err := v.HTTP.GetJSON(ctx, joinURL(v.FinfoURL, "/v4/stocks?")+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	return r.Data, nil
}

type finfoIndustry struct {
	IndustryLevel string `json:"industryLevel"`
	EnglishName   string `json:"englishName"`
	CodeList      string `json:"codeList"`
}

// Industry returns the ICB level 1, 2 and 4 names of symbol and its level-4
// peers (symbol itself excluded).
func (v *VNDirect) Industry(ctx context.Context, symbol string) (icb1, icb2, icb4 string, peers []string, err error) {
	symbol = strings.ToUpper(symbol)
	q := url.Values{}
	q.Set("q", "codeList:"+symbol)
	var r struct {
		Data []finfoIndustry `json:"data"`
	}
	if err := v.HTTP.GetJSON(ctx, joinURL(v.FinfoURL, "/v4/industry_classification?")+q.Encode(), nil, &r); err != nil {
		return "", "", "", nil, err
	}
	if len(r.Data) == 0 {
		return "", "", "", nil, fmt.Errorf("vndirect: %s industry: %w", symbol, ErrNoData)
	}
	for _, row := range r.Data {
		name := strings.TrimSpace(row.EnglishName)
		switch strings.TrimSpace(row.IndustryLevel) {
		case "1":
			icb1 = name
		case "2":
			icb2 = name
		case "4":
			icb4 = name
			for _, p := range normSymbols(strings.Split(row.CodeList, ",")) {
				if p != symbol {
					peers = append(peers, p)
				}
			}
		}
	}
	return icb1, icb2, icb4, peers, nil
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	return ""
}

func sortBars(bars []Bar) {
	sort.SliceStable(bars, func(i, j int) bool { return bars[i].Date.Before(bars[j].Date) })
}

// sortDividends orders newest first.
func sortDividends(divs []Dividend) {
	sort.SliceStable(divs, func(i, j int) bool { return divs[i].ExDate.After(divs[j].ExDate) })
}
