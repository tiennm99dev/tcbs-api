package research

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// KBSURL is the default KBS host and path prefix.
const KBSURL = "https://kbbuddywts.kbsec.com.vn/iis-server"

// KBS is the second fallback for daily bars and constituents, and the
// source of annual net profit.
type KBS struct {
	HTTP    *HTTPClient
	BaseURL string
}

// Name implements Provider.
func (k *KBS) Name() string { return "kbs" }

type kbsBar struct {
	T string  `json:"t"`
	O flexNum `json:"o"`
	H flexNum `json:"h"`
	L flexNum `json:"l"`
	C flexNum `json:"c"`
	V flexNum `json:"v"`
}

// Daily implements Provider. KBS prices are in VND and probably not
// adjusted for corporate actions, so the series is marked unadjusted.
func (k *KBS) Daily(ctx context.Context, symbol string, from, to time.Time) (Series, error) {
	symbol = strings.ToUpper(symbol)
	kind := "stocks"
	if IsIndex(symbol) {
		kind = "index"
	}
	q := url.Values{}
	q.Set("sdate", from.In(marketTZ).Format("02-01-2006"))
	q.Set("edate", to.In(marketTZ).Format("02-01-2006"))
	path := "/investment/" + kind + "/" + url.PathEscape(symbol) + "/data_day?"

	var r struct {
		DataDay []kbsBar `json:"data_day"`
	}
	if err := k.HTTP.GetJSON(ctx, joinURL(k.BaseURL, path)+q.Encode(), nil, &r); err != nil {
		return Series{}, err
	}
	bars := make([]Bar, 0, len(r.DataDay))
	for _, row := range r.DataDay {
		d, err := parseDay(time.DateOnly, row.T)
		if err != nil || !row.C.Valid {
			continue
		}
		bars = append(bars, Bar{Date: d, Open: row.O.V, High: row.H.V, Low: row.L.V, Close: row.C.V, Volume: row.V.V})
	}
	if len(bars) == 0 {
		return Series{}, fmt.Errorf("kbs: %s daily: %w", symbol, ErrNoData)
	}
	sortBars(bars) // rows come newest first
	return Series{Symbol: symbol, Source: k.Name(), Adjusted: false, Bars: filterBars(bars, from, to)}, nil
}

// IndexMembers lists the constituents of a KBS index group: "30" for VN30,
// "100" for VN100.
func (k *KBS) IndexMembers(ctx context.Context, group string) ([]string, error) {
	var r struct {
		Status int      `json:"status"`
		Data   []string `json:"data"`
	}
	path := "/investment/index/" + url.PathEscape(group) + "/stocks"
	if err := k.HTTP.GetJSON(ctx, joinURL(k.BaseURL, path), nil, &r); err != nil {
		return nil, err
	}
	out := normSymbols(r.Data)
	if len(out) == 0 {
		return nil, fmt.Errorf("kbs: index group %s: %w", group, ErrNoData)
	}
	sort.Strings(out)
	return out, nil
}

type kbsKQKD struct {
	Head []struct {
		YearPeriod int `json:"YearPeriod"`
	} `json:"Head"`
	Content map[string][]kbsNorm `json:"Content"`
}

type kbsNorm struct {
	NameEn string  `json:"NameEn"`
	Value1 flexNum `json:"Value1"`
	Value2 flexNum `json:"Value2"`
	Value3 flexNum `json:"Value3"`
	Value4 flexNum `json:"Value4"`
}

// minProfitYears is the fewest fiscal years worth reporting.
const minProfitYears = 3

// profitRowNames are matched against NameEn, in order of preference: the
// parent-company row (non-banks), then net profit after tax (banks).
var profitRowNames = []string{
	"profit after tax for shareholders of parent company",
	"net profit after tax",
}

// NetProfitYears returns up to 4 fiscal years of net profit, oldest first,
// in thousand VND. It returns an empty slice when fewer than 3 years are
// available.
func (k *KBS) NetProfitYears(ctx context.Context, symbol string) ([]float64, error) {
	symbol = strings.ToUpper(symbol)
	q := url.Values{}
	q.Set("page", "1")
	q.Set("pageSize", "4")
	q.Set("type", "KQKD")
	q.Set("unit", "1000")
	q.Set("termtype", "1")
	q.Set("languageid", "1")
	path := "/investment/stock/finance-info/" + url.PathEscape(symbol) + "?"

	var r kbsKQKD
	if err := k.HTTP.GetJSON(ctx, joinURL(k.BaseURL, path)+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	return parseNetProfit(r), nil
}

func parseNetProfit(r kbsKQKD) []float64 {
	row := findProfitRow(r.Content)
	if row == nil {
		return nil
	}
	values := []flexNum{row.Value1, row.Value2, row.Value3, row.Value4}
	type year struct {
		year  int
		value float64
	}
	var years []year
	for i, v := range values {
		if i >= len(r.Head) || !v.Valid || r.Head[i].YearPeriod == 0 {
			continue
		}
		years = append(years, year{r.Head[i].YearPeriod, v.V})
	}
	if len(years) < minProfitYears {
		return nil
	}
	sort.Slice(years, func(i, j int) bool { return years[i].year < years[j].year })
	out := make([]float64, len(years))
	for i, y := range years {
		out[i] = y.value
	}
	return out
}

func findProfitRow(content map[string][]kbsNorm) *kbsNorm {
	// Group keys are Vietnamese labels; iterate them in a fixed order.
	groups := make([]string, 0, len(content))
	for g := range content {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	for _, want := range profitRowNames {
		for _, g := range groups {
			rows := content[g]
			for i := range rows {
				if strings.Contains(strings.ToLower(rows[i].NameEn), want) {
					return &rows[i]
				}
			}
		}
	}
	return nil
}

// kbsGroup maps an index name to its KBS group code.
func kbsGroup(index string) (string, bool) {
	switch strings.ToUpper(index) {
	case "VN30":
		return "30", true
	case "VN100":
		return "100", true
	}
	return "", false
}
