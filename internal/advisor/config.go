// Package advisor turns research metrics and a portfolio summary into
// deterministic, rule-based stability scores and per-holding actions. Every
// function is pure: no I/O apart from LoadOverrides reading the rules file.
// Ratios are fractions (0.25 = 25%); money is VND.
package advisor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Band maps a metric linearly onto 0-100: Best scores 100, Worst scores 0,
// and values past either end are clamped. Best may be above or below Worst.
type Band struct {
	Best, Worst float64
}

// Point is one knot of a piecewise-linear score curve.
type Point struct {
	X, Score float64
}

// Range is a winsorisation range for a raw metric.
type Range struct {
	Min, Max float64
}

// Scoring holds the metric bands and data-quality thresholds.
type Scoring struct {
	VOL, Beta, MDD, LIQ   Band
	ROE, ROEBank          Band
	DE, DERealEstate      Band
	DEUtilities           Band
	NPL, CASA, DY         Band
	PERelative, PEAbs     []Point // PE/median and absolute PE curves above the low-PE step
	PEBank                []Point
	PBGeneral, PBFair     []Point // PB and PB/fairPB curves
	PELowScore            float64 // score for 0 < PE < first PEAbs knot (possible one-off profit)
	WinsorVOL, WinsorBeta Range
	WinsorPE, WinsorPB    Range
	WinsorDE              Range

	MinPeers            int     // peers needed for a sector-relative PE
	LowBeta             float64 // beta below this on an illiquid stock is capped
	LowBetaCap          float64
	IlliquidADTV        float64 // VND; below this, prefer the weekly beta
	WeeklyBetaR2        float64 // daily R² below this prefers the weekly beta
	ValueTrapPB         float64 // PB below this with ROE below ValueTrapROE is capped
	ValueTrapROE        float64
	ValueTrapCap        float64
	DYSpecial           float64 // trailing yield above this is likely a special dividend
	DYSpecialCap        float64 // raw yield used instead
	DYConsistentBonus   float64 // added when cash was paid in all of the last 3 years
	DYConsistentYears   int
	MinEarnYears        int // fiscal years EARN_CONS needs
	MinHistoryReturns   int // daily returns below this get the short-history note
	PillarMinPresent    float64
	MinCoverage         float64 // below this the score is invalid
	HighConfidence      float64
	MediumConfidence    float64
	StableScore         float64
	ModerateScore       float64
	WeakTierScore       float64
	SectorAdjRealEst    int
	SectorAdjSecurities int

	RedFlagADTV    float64       // VND
	RedFlagGaps    int           // gap sessions in the last 60
	RedFlagStale   int           // VN-Index sessions behind the index's last bar
	RedFlagFundAge time.Duration // fundamentals older than this
	ListedStatus   string
	StockType      string
}

// Config holds every threshold the advisor uses. Default returns the
// documented defaults; LoadOverrides changes only the fields a rules file
// names.
type Config struct {
	SingleSoftCap  float64
	SingleHardCap  float64
	SectorCap      float64
	CashMin        float64
	CashHigh       float64
	MarginMax      float64
	AddMinScore    float64
	AddStrongScore float64
	WeakScore      float64
	ReviewScore    float64
	ReviewLossPct  float64
	WeakLossPct    float64
	ReviewWeakLoss float64 // score below this with pnl ≤ WeakLossPct forces REVIEW
	TakeProfitPct  float64
	TakeProfitPart float64 // share of the position sold on a take-profit trim
	WeakMaxWeight  float64 // weak-score holdings are trimmed to this weight
	AddRoomBuffer  float64
	AddMinValue    float64 // valuation pillar an ADD needs
	ConcentratedN  float64 // effectiveN below this is "concentrated"
	MinHoldings    int     // fewer holdings than this is "under-diversified"
	CoE            float64
	FeeRate        float64
	SellTax        float64
	DividendTax    float64

	CandidateMinScore        float64
	CandidateMaxVol          float64
	CandidateMinCoverage     float64
	CandidateMinListedYears  float64
	CandidatePerSector       int
	CandidateLowSectorBonus  float64
	CandidateLowSectorWeight float64
	CandidateMaxNPL          float64
	CandidateProfitYears     int
	CandidateCount           int // top N shown
	CandidateAllocSplit      float64

	Scoring Scoring

	// Now is the clock for listing age and the market phase. Zero means
	// time.Now. It is never read from a rules file.
	Now time.Time `json:"-"`
}

// Default returns the documented defaults.
func Default() Config {
	return Config{
		SingleSoftCap:  0.20,
		SingleHardCap:  0.25,
		SectorCap:      0.40,
		CashMin:        0.05,
		CashHigh:       0.40,
		MarginMax:      0.20,
		AddMinScore:    70,
		AddStrongScore: 75,
		WeakScore:      60,
		ReviewScore:    45,
		ReviewLossPct:  -0.20,
		WeakLossPct:    -0.15,
		ReviewWeakLoss: 50,
		TakeProfitPct:  0.30,
		TakeProfitPart: 1.0 / 3,
		WeakMaxWeight:  0.10,
		AddRoomBuffer:  0.05,
		AddMinValue:    50,
		ConcentratedN:  4,
		MinHoldings:    5,
		CoE:            0.13,
		FeeRate:        0.001,
		SellTax:        0.001,
		DividendTax:    0.05,

		CandidateMinScore:        70,
		CandidateMaxVol:          0.40,
		CandidateMinCoverage:     80,
		CandidateMinListedYears:  2,
		CandidatePerSector:       2,
		CandidateLowSectorBonus:  5,
		CandidateLowSectorWeight: 0.15,
		CandidateMaxNPL:          0.03,
		CandidateProfitYears:     3,
		CandidateCount:           10,
		CandidateAllocSplit:      3,

		Scoring: Scoring{
			VOL:          Band{Best: 0.20, Worst: 0.60},
			Beta:         Band{Best: 0.9, Worst: 1.5},
			MDD:          Band{Best: 0.15, Worst: 0.50},
			LIQ:          Band{Best: 2e10, Worst: 1e9},
			ROE:          Band{Best: 0.20, Worst: 0.05},
			ROEBank:      Band{Best: 0.20, Worst: 0.08},
			DE:           Band{Best: 0.3, Worst: 1.5},
			DERealEstate: Band{Best: 0.5, Worst: 2.0},
			DEUtilities:  Band{Best: 0.8, Worst: 2.5},
			NPL:          Band{Best: 0.010, Worst: 0.035},
			CASA:         Band{Best: 0.35, Worst: 0.10},
			DY:           Band{Best: 0.06, Worst: 0},
			PERelative:   []Point{{0.8, 100}, {1.0, 70}, {1.3, 40}, {1.6, 0}},
			PEAbs:        []Point{{4, 100}, {12, 100}, {18, 60}, {30, 0}},
			PEBank:       []Point{{4, 100}, {10, 100}, {15, 50}, {20, 0}},
			PBGeneral:    []Point{{1.5, 100}, {3, 60}, {6, 0}},
			PBFair:       []Point{{0.8, 100}, {1.0, 75}, {1.25, 40}, {1.5, 0}},
			PELowScore:   50,
			WinsorVOL:    Range{0, 2.0},
			WinsorBeta:   Range{-1, 3},
			WinsorPE:     Range{-100, 200},
			WinsorPB:     Range{0, 20},
			WinsorDE:     Range{0, 10},

			MinPeers:            5,
			LowBeta:             0.3,
			LowBetaCap:          70,
			IlliquidADTV:        5e9,
			WeeklyBetaR2:        0.1,
			ValueTrapPB:         0.7,
			ValueTrapROE:        0.08,
			ValueTrapCap:        50,
			DYSpecial:           0.10,
			DYSpecialCap:        0.08,
			DYConsistentBonus:   10,
			DYConsistentYears:   3,
			MinEarnYears:        3,
			MinHistoryReturns:   120,
			PillarMinPresent:    0.5,
			MinCoverage:         60,
			HighConfidence:      85,
			MediumConfidence:    60,
			StableScore:         75,
			ModerateScore:       60,
			WeakTierScore:       45,
			SectorAdjRealEst:    -5,
			SectorAdjSecurities: -5,

			RedFlagADTV:    1e9,
			RedFlagGaps:    10,
			RedFlagStale:   5,
			RedFlagFundAge: 200 * 24 * time.Hour,
			ListedStatus:   "listed",
			StockType:      "STOCK",
		},
	}
}

// now returns c.Now, or the wall clock when it is unset.
func (c Config) now() time.Time {
	if c.Now.IsZero() {
		return time.Now()
	}
	return c.Now
}

// LoadOverrides reads a JSON rules file and overrides only the fields it
// names, nested ones included. Unknown field names are an error so a typo
// does not silently keep a default.
func LoadOverrides(path string, c *Config) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("advisor: read rules: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	next := *c
	if err := dec.Decode(&next); err != nil {
		return fmt.Errorf("advisor: parse rules %s: %w", path, err)
	}
	if next.SingleSoftCap > next.SingleHardCap {
		return fmt.Errorf("advisor: rules: SingleSoftCap %g is above SingleHardCap %g", next.SingleSoftCap, next.SingleHardCap)
	}
	next.Now = c.Now
	*c = next
	return nil
}
