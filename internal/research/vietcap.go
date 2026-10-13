package research

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Default Vietcap hosts.
const (
	VietcapChartURL = "https://trading.vietcap.com.vn"
	VietcapIQURL    = "https://iq.vietcap.com.vn"

	vietcapOrigin    = "https://trading.vietcap.com.vn/"
	vietcapCountBack = 520 // about two years of sessions
)

// Vietcap is the first fallback for daily bars and the source of bank
// ratios (NPL, CASA) and a fallback for dividends.
type Vietcap struct {
	HTTP     *HTTPClient
	ChartURL string
	IQURL    string
}

// Name implements Provider.
func (v *Vietcap) Name() string { return "vietcap" }

type gapChartRow struct {
	Symbol string    `json:"symbol"`
	T      []flexNum `json:"t"`
	O      []float64 `json:"o"`
	H      []float64 `json:"h"`
	L      []float64 `json:"l"`
	C      []float64 `json:"c"`
	V      []float64 `json:"v"`
}

// Daily implements Provider. Prices are adjusted and already in VND; index
// series are in points.
func (v *Vietcap) Daily(ctx context.Context, symbol string, from, to time.Time) (Series, error) {
	symbol = strings.ToUpper(symbol)
	body := map[string]any{
		"timeFrame": "ONE_DAY",
		"symbols":   []string{symbol},
		"to":        to.Unix(),
		"countBack": vietcapCountBack,
	}
	h := http.Header{}
	h.Set("Origin", vietcapOrigin)
	h.Set("Referer", vietcapOrigin)

	var rows []gapChartRow
	if err := v.HTTP.PostJSON(ctx, joinURL(v.ChartURL, "/api/chart/OHLCChart/gap-chart"), h, body, &rows); err != nil {
		return Series{}, err
	}
	var r *gapChartRow
	for i := range rows {
		if strings.EqualFold(rows[i].Symbol, symbol) {
			r = &rows[i]
			break
		}
	}
	if r == nil {
		return Series{}, fmt.Errorf("vietcap: %s daily: %w", symbol, ErrNoData)
	}
	n := len(r.T)
	if len(r.O) != n || len(r.H) != n || len(r.L) != n || len(r.C) != n || len(r.V) != n {
		return Series{}, fmt.Errorf("vietcap: %s daily: column lengths differ", symbol)
	}
	bars := make([]Bar, 0, n)
	for i := 0; i < n; i++ {
		if !r.T[i].Valid {
			continue
		}
		bars = append(bars, Bar{
			Date:   dayUTC(time.Unix(int64(r.T[i].V), 0)),
			Open:   r.O[i],
			High:   r.H[i],
			Low:    r.L[i],
			Close:  r.C[i],
			Volume: r.V[i],
		})
	}
	sortBars(bars)
	return Series{Symbol: symbol, Source: v.Name(), Adjusted: true, Bars: filterBars(bars, from, to)}, nil
}

// BankRatios holds the Vietcap ratios used for banks.
type BankRatios struct {
	AsOf                       time.Time // quarter end of the row used
	NPL, CASA, PE, PB, ROE, DE *float64
}

type statFinRow struct {
	RatioType    string  `json:"ratioType"`
	YearReport   int     `json:"yearReport"`
	Quarter      int     `json:"quarter"`
	NPL          flexNum `json:"npl"`
	CASARatio    flexNum `json:"casaRatio"`
	PE           flexNum `json:"pe"`
	PB           flexNum `json:"pb"`
	ROE          flexNum `json:"roe"`
	DebtToEquity flexNum `json:"debtToEquity"`
}

// BankRatios reads the latest trailing-twelve-month row of symbol's
// statistics. Its dividendYield field reads 0 even for payers and is ignored.
func (v *Vietcap) BankRatios(ctx context.Context, symbol string) (BankRatios, error) {
	symbol = strings.ToUpper(symbol)
	var r struct {
		Data []statFinRow `json:"data"`
	}
	path := "/api/iq-insight-service/v1/company/" + url.PathEscape(symbol) + "/statistics-financial"
	if err := v.HTTP.GetJSON(ctx, joinURL(v.IQURL, path), nil, &r); err != nil {
		return BankRatios{}, err
	}
	var best *statFinRow
	for i := range r.Data {
		row := &r.Data[i]
		if row.RatioType != "RATIO_TTM" {
			continue
		}
		if best == nil || row.YearReport > best.YearReport ||
			(row.YearReport == best.YearReport && row.Quarter > best.Quarter) {
			best = row
		}
	}
	if best == nil {
		return BankRatios{}, fmt.Errorf("vietcap: %s statistics: %w", symbol, ErrNoData)
	}
	return BankRatios{
		AsOf: quarterEnd(best.YearReport, best.Quarter),
		NPL:  best.NPL.ptr(),
		CASA: best.CASARatio.ptr(),
		PE:   best.PE.ptr(),
		PB:   best.PB.ptr(),
		ROE:  best.ROE.ptr(),
		DE:   best.DebtToEquity.ptr(),
	}, nil
}

func quarterEnd(year, quarter int) time.Time {
	if year <= 0 || quarter < 1 || quarter > 4 {
		return time.Time{}
	}
	return time.Date(year, time.Month(quarter*3)+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
}

type vciEvent struct {
	EventCode     string  `json:"eventCode"`
	ExrightDate   string  `json:"exrightDate"`
	ValuePerShare flexNum `json:"valuePerShare"`
	ExerciseRatio flexNum `json:"exerciseRatio"`
}

// Dividends returns symbol's cash dividends (DIV) and share issues (ISS,
// as a stock ratio) with ex-dates in [from, to], newest first.
func (v *Vietcap) Dividends(ctx context.Context, symbol string, from, to time.Time) ([]Dividend, error) {
	symbol = strings.ToUpper(symbol)
	q := url.Values{}
	q.Set("ticker", symbol)
	q.Set("fromDate", from.In(marketTZ).Format("20060102"))
	q.Set("toDate", to.In(marketTZ).Format("20060102"))
	q.Set("eventCode", "DIV,ISS")
	q.Set("page", "0")
	q.Set("size", "50")

	var r struct {
		Data struct {
			Content []vciEvent `json:"content"`
		} `json:"data"`
	}
	if err := v.HTTP.GetJSON(ctx, joinURL(v.IQURL, "/api/iq-insight-service/v1/events?")+q.Encode(), nil, &r); err != nil {
		return nil, err
	}
	var out []Dividend
	for _, e := range r.Data.Content {
		ex, err := parseDay(time.DateOnly, e.ExrightDate)
		if err != nil {
			continue
		}
		switch e.EventCode {
		case "DIV":
			if e.ValuePerShare.V > 0 {
				out = append(out, Dividend{ExDate: ex, CashPerShare: e.ValuePerShare.V})
			}
		case "ISS":
			if e.ExerciseRatio.V > 0 {
				out = append(out, Dividend{ExDate: ex, StockRatio: e.ExerciseRatio.V})
			}
		}
	}
	sortDividends(out)
	return out, nil
}
