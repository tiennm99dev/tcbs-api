package advisor

import (
	"math"
	"time"
)

// VN market rules the report uses. Tick sizes and price bands are not needed
// because the CLI shows no limit prices.
const (
	// BoardLot is the HOSE/HNX/UPCoM board lot. Odd lots (1-99 shares)
	// trade on a separate board.
	BoardLot = 100
	// SettlementDays is the T+N settlement cycle, used only in notes.
	SettlementDays = 2
)

// Market phase labels returned by MarketPhase.
const (
	PhaseIntraday  = "intraday"
	PhaseLastClose = "last close"
)

// vnTZ is Asia/Ho_Chi_Minh. Vietnam has no daylight saving, so a fixed zone
// avoids depending on the system tz database.
var vnTZ = time.FixedZone("ICT", 7*3600)

// MarketPhase returns "intraday" on weekdays from 09:00 to 15:00
// Asia/Ho_Chi_Minh and "last close" otherwise. There is no holiday calendar.
func MarketPhase(now time.Time) string {
	t := now.In(vnTZ)
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return PhaseLastClose
	}
	mins := t.Hour()*60 + t.Minute()
	if mins >= 9*60 && mins < 15*60 {
		return PhaseIntraday
	}
	return PhaseLastClose
}

// FloorLot rounds qty down to whole board lots.
func FloorLot(qty int64) int64 {
	if qty <= 0 {
		return 0
	}
	return qty / BoardLot * BoardLot
}

// OddLot is the part of qty below one board lot.
func OddLot(qty int64) int64 {
	if qty <= 0 {
		return 0
	}
	return qty % BoardLot
}

// BuyCost is the cash a buy of qty shares at price costs, fee included.
func BuyCost(qty, price int64, fee float64) int64 {
	return int64(math.Round(float64(qty) * float64(price) * (1 + fee)))
}

// SellProceeds is the cash a sale of qty shares at price returns, after the
// fee and the sales tax.
func SellProceeds(qty, price int64, fee, sellTax float64) int64 {
	return int64(math.Round(float64(qty) * float64(price) * (1 - fee - sellTax)))
}

// Breakeven is the sale price that recovers avgCost after the buy fee, the
// sell fee and the sales tax: avgCost × (1+fee)/(1−fee−sellTax), rounded up
// to the dong. It is 0 when the cost is unknown.
func Breakeven(avgCost int64, fee, sellTax float64) int64 {
	if avgCost <= 0 || fee+sellTax >= 1 {
		return 0
	}
	return int64(math.Ceil(float64(avgCost)*(1+fee)/(1-fee-sellTax) - 1e-9))
}
