package portfolio

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Quote is a current price in VND and its source.
type Quote struct {
	Price    int64
	Source   string
	Intraday bool
}

// These switch receivable sale proceeds and pending stock-dividend shares
// into NAV once a live dump shows TCBS does not already count them in
// cashBalance and totalQtty. Until then they are reported on their own lines.
const (
	receivableInNAV      = false
	pendingStockDivInNAV = false
)

// Aggregate folds a snapshot into per-symbol positions and portfolio totals.
// prices maps a symbol to its current price in VND and the source of that
// price; a symbol without a quote falls back to the TCBS reference price.
// Ratios (Weight, PnLPct, CashRatio, MarginRatio, HHI) are fractions, not
// percentages.
func Aggregate(s Snapshot, prices map[string]Quote, includeBankCash bool) Summary {
	type acc struct {
		pos         Position
		costValue   int64
		unknownCost bool
		refPrice    int64
	}
	bySym := map[string]*acc{}
	var sum Summary
	sum.BankCashInNAV = includeBankCash

	for _, a := range s.Accounts {
		sum.Cash += a.Cash.Balance
		sum.ReceivableCash += a.Cash.Receivable
		sum.IncomingDividends += a.Cash.IncomingDividends
		sum.BankCash += a.Cash.BankLinked
		sum.Debt += a.Debt.Outstanding + a.Debt.AccruedInterest + a.Debt.FeeDebt + a.Cash.CustodyFeeDebt

		for _, h := range a.Holdings {
			sym := strings.ToUpper(strings.TrimSpace(h.Symbol))
			if sym == "" {
				continue
			}
			p := bySym[sym]
			if p == nil {
				p = &acc{pos: Position{Symbol: sym}}
				bySym[sym] = p
			}
			p.pos.Qty += h.TotalQty
			p.pos.Sellable += h.SellableQty
			p.pos.Receiving += h.ReceivingQty
			p.pos.PendingStockDiv += h.PendingStockDiv
			if h.TotalQty > 0 {
				if h.AvgCost > 0 {
					p.costValue += h.TotalQty * h.AvgCost
				} else {
					p.unknownCost = true
				}
			}
			if p.refPrice == 0 && h.RefPrice > 0 {
				p.refPrice = h.RefPrice
			}
			if p.pos.SecType == "" {
				p.pos.SecType = strings.TrimSpace(h.SecType)
			}
		}
	}
	if includeBankCash {
		sum.Cash += sum.BankCash
	}

	syms := make([]string, 0, len(bySym))
	for sym := range bySym {
		syms = append(syms, sym)
	}
	sort.Strings(syms)

	for _, sym := range syms {
		p := bySym[sym]
		pos := p.pos
		pos.Scored = IsScored(pos.SecType)

		if q, ok := prices[sym]; ok && q.Price > 0 {
			pos.Price, pos.PriceSource = q.Price, q.Source
		} else {
			pos.Price, pos.PriceSource = p.refPrice, SourceRef
			if p.refPrice > 0 {
				sum.Notes = append(sum.Notes, fmt.Sprintf("%s: no current quote; valued at the TCBS reference price", sym))
			} else {
				sum.Notes = append(sum.Notes, fmt.Sprintf("%s: no price available; valued at 0", sym))
			}
		}

		pos.MarketValue = pos.Qty * pos.Price
		if p.unknownCost {
			sum.Notes = append(sum.Notes, fmt.Sprintf("%s: average cost unknown (gifted or transferred shares); P&L not computed", sym))
		} else if pos.Qty > 0 && p.costValue > 0 {
			pos.CostValue = p.costValue
			pos.AvgCost = int64(math.Round(float64(p.costValue) / float64(pos.Qty)))
			pos.UnrealizedPnL = pos.MarketValue - pos.CostValue
			pos.PnLPct = float64(pos.UnrealizedPnL) / float64(pos.CostValue)
		}

		sum.PendingStockDivValue += pos.PendingStockDiv * pos.Price
		sum.StockValue += pos.MarketValue

		if !pos.Scored {
			sum.Notes = append(sum.Notes, fmt.Sprintf("%s: fund/warrant (secType %s), valued but not scored", sym, pos.SecType))
		}
		sum.Positions = append(sum.Positions, pos)
	}

	sum.NAV = sum.StockValue + sum.Cash + sum.IncomingDividends - sum.Debt
	if receivableInNAV {
		sum.NAV += sum.ReceivableCash
	}
	if pendingStockDivInNAV {
		sum.NAV += sum.PendingStockDivValue
	}

	if sum.NAV > 0 {
		nav := float64(sum.NAV)
		for i := range sum.Positions {
			sum.Positions[i].Weight = float64(sum.Positions[i].MarketValue) / nav
		}
		sum.CashRatio = float64(sum.Cash) / nav
		sum.MarginRatio = float64(sum.Debt) / nav
	} else if len(s.Accounts) > 0 {
		sum.Notes = append(sum.Notes, "NAV is not positive; weights and ratios are not computed")
	}

	if sum.StockValue > 0 {
		sv := float64(sum.StockValue)
		for _, pos := range sum.Positions {
			w := float64(pos.MarketValue) / sv
			sum.HHI += w * w
		}
		sum.EffectiveN = 1 / sum.HHI
	}

	sort.SliceStable(sum.Positions, func(i, j int) bool {
		a, b := sum.Positions[i], sum.Positions[j]
		if a.MarketValue != b.MarketValue {
			return a.MarketValue > b.MarketValue
		}
		return a.Symbol < b.Symbol
	})
	return sum
}
