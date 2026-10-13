// Package portfolio pulls TCBS holdings, cash and margin debt into a
// Snapshot and aggregates it into per-symbol positions and portfolio totals.
// All money is in VND as int64; quantities are int64.
package portfolio

import (
	"strings"
	"time"
)

// Snapshot is the portfolio state across sub-accounts at one point in time.
type Snapshot struct {
	AsOf     time.Time `json:"asOf"`
	Custody  string    `json:"custody"` // masked on save: 105C***1234
	Accounts []Account `json:"accounts"`
}

// Account is one TCBS sub-account.
type Account struct {
	AccountNo string    `json:"accountNo"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Cash      Cash      `json:"cash"`
	Debt      Debt      `json:"debt"` // zero for non-margin accounts
	Holdings  []Holding `json:"holdings"`
}

// Holding is one stock line in a sub-account.
type Holding struct {
	Symbol          string `json:"symbol"`
	TotalQty        int64  `json:"totalQty"`
	SellableQty     int64  `json:"sellableQty"`
	ReceivingQty    int64  `json:"receivingQty"`
	RestrictedQty   int64  `json:"restrictedQty"`
	PendingStockDiv int64  `json:"pendingStockDiv"`
	PendingCashDiv  int64  `json:"pendingCashDiv"` // VND; informational, never added to NAV (sCashDividend already covers it)
	AvgCost         int64  `json:"avgCost"`        // VND per share; 0 means unknown (gifted or transferred shares)
	RefPrice        int64  `json:"refPrice"`       // VND, TCBS currentPrice (reference)
	SecType         string `json:"secType"`        // from secType; ETFs and covered warrants are valued but not scored
}

// Cash is the cash side of a sub-account.
type Cash struct {
	Balance           int64 `json:"balance"`
	Withdrawable      int64 `json:"withdrawable"`
	Receivable        int64 `json:"receivable"` // unsettled sale proceeds; outside NAV
	IncomingDividends int64 `json:"incomingDividends"`
	BankLinked        int64 `json:"bankLinked"` // IA linked-bank balance, held at the bank
	CustodyFeeDebt    int64 `json:"custodyFeeDebt"`
}

// Debt is the margin debt of a sub-account.
type Debt struct {
	Outstanding     int64   `json:"outstanding"`
	AccruedInterest int64   `json:"accruedInterest"`
	FeeDebt         int64   `json:"feeDebt"`
	RTT             float64 `json:"rtt"`
	RiskCode        string  `json:"riskCode"`
}

// Position is a holding aggregated by symbol across accounts.
type Position struct {
	Symbol                                    string
	Qty, Sellable, Receiving, PendingStockDiv int64
	AvgCost, Price                            int64  // AvgCost is the qty-weighted mean; 0 means unknown cost and unknown P&L
	PriceSource                               string // "tcbs-match", "ref", or the public provider name
	MarketValue, CostValue, UnrealizedPnL     int64
	PnLPct, Weight                            float64 // Weight = MV / NAV
	SecType                                   string  // first non-empty secType across accounts
	Scored                                    bool    // false for ETFs, covered warrants and other non-stock types
}

// Summary is the aggregated portfolio.
type Summary struct {
	Positions                                   []Position
	StockValue, Cash, ReceivableCash, Debt, NAV int64
	IncomingDividends                           int64 // in NAV
	BankCash                                    int64 // IA linked-bank balance; in Cash and NAV only when BankCashInNAV
	BankCashInNAV                               bool
	PendingStockDivValue                        int64 // pending stock-dividend shares × price; not in NAV
	CashRatio, MarginRatio, HHI, EffectiveN     float64
	Notes                                       []string
}

// Price sources set on Position.PriceSource by this package.
const (
	SourceTCBSMatch = "tcbs-match"
	SourceRef       = "ref"
)

// TCBS secType codes for common and preferred shares. Everything else (fund
// certificates 008, covered warrants 011, bonds, futures) is valued but not
// scored.
var scoredSecTypes = map[string]bool{"": true, "001": true, "002": true}

// IsScored reports whether a secType is a listed stock that the advisor scores.
func IsScored(secType string) bool {
	return scoredSecTypes[strings.TrimSpace(secType)]
}

// MaskCustody keeps the 105C prefix and the last 4 characters of a custody code.
func MaskCustody(code string) string {
	code = strings.TrimSpace(code)
	switch {
	case code == "":
		return ""
	case strings.Contains(code, "***"):
		return code
	case len(code) <= 8:
		return "***"
	}
	return code[:4] + "***" + code[len(code)-4:]
}
