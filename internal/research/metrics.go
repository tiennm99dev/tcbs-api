package research

import (
	"math"
	"strings"
	"time"
)

const (
	// TradingDaysPerYear annualises daily volatility.
	TradingDaysPerYear = 250
	// minReturns is the fewest daily returns a volatility or beta needs.
	minReturns = 120
	// minWeeklyReturns is the fewest weekly returns a weekly beta needs.
	minWeeklyReturns = 52
	// jumpMargin is added to the price band before a daily move counts as a
	// corporate-action jump.
	jumpMargin = 0.01
)

// PriceBand returns the daily price limit of an exchange: 7% on HOSE, 10%
// on HNX, 15% on UPCoM. Unknown exchanges get the widest band, so the jump
// heuristic flags the fewest moves.
func PriceBand(exchange string) float64 {
	switch strings.ToUpper(strings.TrimSpace(exchange)) {
	case "HOSE", "HSX":
		return 0.07
	case "HNX":
		return 0.10
	default:
		return 0.15
	}
}

// LogReturns maps each bar's date to ln(close / previous close). Bars with
// a non-positive close break the chain.
func LogReturns(s Series) map[time.Time]float64 {
	out := make(map[time.Time]float64, len(s.Bars))
	for i := 1; i < len(s.Bars); i++ {
		prev, cur := s.Bars[i-1].Close, s.Bars[i].Close
		if prev > 0 && cur > 0 {
			out[s.Bars[i].Date] = math.Log(cur / prev)
		}
	}
	return out
}

// lastBars returns the bars needed for the last n returns.
func lastBars(bars []Bar, n int) []Bar {
	if n+1 < len(bars) {
		return bars[len(bars)-n-1:]
	}
	return bars
}

func returnsOf(closes []float64) []float64 {
	out := make([]float64, 0, len(closes))
	for i := 1; i < len(closes); i++ {
		if closes[i-1] > 0 && closes[i] > 0 {
			out = append(out, math.Log(closes[i]/closes[i-1]))
		}
	}
	return out
}

// Volatility is the annualised standard deviation of the last sessions
// daily log returns (stdev × sqrt(250)). It needs at least 120 returns.
func Volatility(s Series, sessions int) (float64, bool) {
	bars := lastBars(s.Bars, sessions)
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	r := returnsOf(closes)
	if len(r) < minReturns {
		return 0, false
	}
	return stdev(r) * math.Sqrt(TradingDaysPerYear), true
}

// Beta regresses the stock's daily log returns on the index's over the last
// sessions common trading days. Returns are taken between consecutive days
// both series traded, so a missing bar does not misalign them. It needs at
// least 120 returns.
func Beta(stock, index Series, sessions int) (beta, r2 float64, ok bool) {
	sc, ic := alignCloses(dailyCloses(stock), dailyCloses(index))
	if sessions+1 < len(sc) {
		sc, ic = sc[len(sc)-sessions-1:], ic[len(ic)-sessions-1:]
	}
	return regress(sc, ic, minReturns)
}

// WeeklyBeta is Beta on the last close of each week (normally Friday) over
// the last weeks weeks. It needs at least 52 weekly returns.
func WeeklyBeta(stock, index Series, weeks int) (float64, bool) {
	sc, ic := alignCloses(weeklyCloses(stock), weeklyCloses(index))
	if weeks+1 < len(sc) {
		sc, ic = sc[len(sc)-weeks-1:], ic[len(ic)-weeks-1:]
	}
	b, _, ok := regress(sc, ic, minWeeklyReturns)
	return b, ok
}

type datedClose struct {
	key   time.Time
	close float64
}

func dailyCloses(s Series) []datedClose {
	out := make([]datedClose, 0, len(s.Bars))
	for _, b := range s.Bars {
		if b.Close > 0 {
			out = append(out, datedClose{b.Date, b.Close})
		}
	}
	return out
}

// weeklyCloses keeps the last close of each ISO week, keyed by that week's
// Monday.
func weeklyCloses(s Series) []datedClose {
	var out []datedClose
	for _, b := range s.Bars {
		if b.Close <= 0 {
			continue
		}
		wd := (int(b.Date.Weekday()) + 6) % 7 // Monday = 0
		monday := b.Date.AddDate(0, 0, -wd)
		if n := len(out); n > 0 && out[n-1].key.Equal(monday) {
			out[n-1].close = b.Close
			continue
		}
		out = append(out, datedClose{monday, b.Close})
	}
	return out
}

// alignCloses keeps the keys present in both lists, in order.
func alignCloses(a, b []datedClose) (ac, bc []float64) {
	idx := make(map[time.Time]float64, len(b))
	for _, x := range b {
		idx[x.key] = x.close
	}
	for _, x := range a {
		if v, ok := idx[x.key]; ok {
			ac = append(ac, x.close)
			bc = append(bc, v)
		}
	}
	return ac, bc
}

func regress(stockCloses, indexCloses []float64, minN int) (beta, r2 float64, ok bool) {
	y, x := returnsOf(stockCloses), returnsOf(indexCloses)
	if len(x) != len(y) || len(x) < minN {
		return 0, 0, false
	}
	mx, my := mean(x), mean(y)
	var sxy, sxx, syy float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 {
		return 0, 0, false
	}
	beta = sxy / sxx
	if syy > 0 {
		r2 = sxy * sxy / (sxx * syy)
	}
	return beta, r2, true
}

// MaxDrawdown is the largest peak-to-trough fall of the close over the last
// sessions bars, as a positive fraction. It needs at least
// min(sessions, 120) bars.
func MaxDrawdown(s Series, sessions int) (float64, bool) {
	bars := s.Bars
	if sessions < len(bars) {
		bars = bars[len(bars)-sessions:]
	}
	if len(bars) < 2 || len(bars) < min(sessions, minReturns) {
		return 0, false
	}
	var peak, mdd float64
	for _, b := range bars {
		if b.Close <= 0 {
			continue
		}
		peak = math.Max(peak, b.Close)
		mdd = math.Max(mdd, 1-b.Close/peak)
	}
	return mdd, true
}

// ADTV is the average daily traded value (close × volume, VND) over the last
// sessions bars, normally 60.
func ADTV(s Series, sessions int) float64 {
	bars := s.Bars
	if sessions < len(bars) {
		bars = bars[len(bars)-sessions:]
	}
	if len(bars) == 0 {
		return 0
	}
	var sum float64
	for _, b := range bars {
		sum += b.Close * b.Volume
	}
	return sum / float64(len(bars))
}

// TrailingCashYield is the cash paid per share with ex-dates in the year up
// to now, divided by price. paidYears counts how many of the last three
// one-year windows had a cash payment. Stock dividends never count.
func TrailingCashYield(divs []Dividend, price float64, now time.Time) (yield float64, paidYears int) {
	today := dayUTC(now)
	var paid [3]bool
	var lastYear float64
	for _, d := range divs {
		if d.CashPerShare <= 0 || d.ExDate.After(today) {
			continue
		}
		for w := 0; w < 3; w++ {
			hi := today.AddDate(-w, 0, 0)
			lo := today.AddDate(-w-1, 0, 0)
			if d.ExDate.After(lo) && !d.ExDate.After(hi) {
				paid[w] = true
				if w == 0 {
					lastYear += d.CashPerShare
				}
			}
		}
	}
	for _, p := range paid {
		if p {
			paidYears++
		}
	}
	if price > 0 {
		yield = lastYear / price
	}
	return yield, paidYears
}

// AdjustJumps back-adjusts an unadjusted series for corporate actions. A
// daily close-to-close move larger than band + 1pp cannot be a trading move,
// so it is treated as a split or dividend: every earlier bar's prices are
// scaled by the jump ratio (volumes by its inverse). It reports whether any
// jump was adjusted. Adjusted series are returned unchanged.
func AdjustJumps(s Series, band float64) (Series, bool) {
	if s.Adjusted || len(s.Bars) < 2 {
		return s, false
	}
	bars := append([]Bar(nil), s.Bars...)
	adjusted := false
	for i := len(bars) - 1; i > 0; i-- {
		prev, cur := bars[i-1].Close, bars[i].Close
		if prev <= 0 || cur <= 0 {
			continue
		}
		if math.Abs(cur/prev-1) <= band+jumpMargin {
			continue
		}
		f := cur / prev
		for j := 0; j < i; j++ {
			bars[j].Open *= f
			bars[j].High *= f
			bars[j].Low *= f
			bars[j].Close *= f
			bars[j].Volume /= f
		}
		adjusted = true
	}
	out := s
	out.Bars = bars
	return out, adjusted
}

// GapSessions counts the index sessions among the last `last` on which the
// stock, already listed, has no bar. Sessions after the stock's last bar are
// StaleSessions, not gaps.
func GapSessions(s Series, index Series, last int) int {
	if len(s.Bars) == 0 {
		return 0
	}
	first, lastBar := s.Bars[0].Date, s.Bars[len(s.Bars)-1].Date
	have := make(map[time.Time]bool, len(s.Bars))
	for _, b := range s.Bars {
		have[b.Date] = true
	}
	ib := index.Bars
	if last < len(ib) {
		ib = ib[len(ib)-last:]
	}
	gaps := 0
	for _, b := range ib {
		if b.Date.Before(first) || b.Date.After(lastBar) {
			continue
		}
		if !have[b.Date] {
			gaps++
		}
	}
	return gaps
}

// StaleSessions counts the index sessions after the stock's last bar. It
// measures staleness in trading sessions, so holidays do not count.
func StaleSessions(s Series, index Series) int {
	if len(s.Bars) == 0 {
		return len(index.Bars)
	}
	lastBar := s.Bars[len(s.Bars)-1].Date
	n := 0
	for _, b := range index.Bars {
		if b.Date.After(lastBar) {
			n++
		}
	}
	return n
}

// Range is the lowest low and highest high over the last sessions bars
// (250 for the 52-week range). It is 0, 0 for an empty series.
func Range(s Series, sessions int) (low, high float64) {
	bars := s.Bars
	if sessions < len(bars) {
		bars = bars[len(bars)-sessions:]
	}
	for _, b := range bars {
		lo, hi := b.Low, b.High
		if lo <= 0 {
			lo = b.Close
		}
		if hi <= 0 {
			hi = b.Close
		}
		if lo > 0 && (low == 0 || lo < low) {
			low = lo
		}
		if hi > high {
			high = hi
		}
	}
	return low, high
}

// Return is the simple close-to-close return over the last sessions bars.
func Return(s Series, sessions int) (float64, bool) {
	n := len(s.Bars)
	if sessions <= 0 || n <= sessions {
		return 0, false
	}
	start, end := s.Bars[n-1-sessions].Close, s.Bars[n-1].Close
	if start <= 0 {
		return 0, false
	}
	return end/start - 1, true
}

func mean(x []float64) float64 {
	var s float64
	for _, v := range x {
		s += v
	}
	return s / float64(len(x))
}

// stdev is the sample standard deviation.
func stdev(x []float64) float64 {
	if len(x) < 2 {
		return 0
	}
	m := mean(x)
	var ss float64
	for _, v := range x {
		ss += (v - m) * (v - m)
	}
	return math.Sqrt(ss / float64(len(x)-1))
}
