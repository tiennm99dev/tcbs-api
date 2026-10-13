# Phase 4: stability scoring and recommendation rules

All code in this phase is pure, with no I/O. Inputs are `research` metrics plus
a `portfolio.Summary`. Outputs are typed results that the report renders.

## Files

- `internal/advisor/config.go`: `Config` with the defaults below, plus
  `LoadOverrides(path string, c *Config) error` (a JSON file that overrides
  only the fields it names)
- `internal/advisor/market.go`: VN market rules (lot, fees and tax, breakeven, session phase)
- `internal/advisor/sector.go`: sector classification
- `internal/advisor/score.go`: metric normalisation, weights, redistribution,
  tiers, red flags
- `internal/advisor/recommend.go`: per-holding actions, portfolio messages
- `internal/advisor/candidates.go`: screening and ranking of candidates
- table tests for each file

## API

```go
type Sector string // bank, securities, insurance, real_estate, utilities_infra, general
type Inputs struct {
    Symbol string; Sector Sector; Exchange string
    Price float64 // VND
    VOL, Beta, BetaR2, MDD, ADTV, ROE, DE, PE, PB, DY, NPL, CASA, EarnCons *float64
    DivPaidYears int; SectorMedianPE *float64; Status, Type string
    HistoryReturns, GapSessions, StaleSessions int; FundAge time.Duration
    ProfitYears []float64 // newest first; empty = unknown
    TTMLoss, PrevYearLoss, NegEquity bool; Listed time.Time
}
type MetricScore struct { Name string; Raw float64; Score float64; Weight, Contribution float64; Dropped bool; Note string }
type Score struct {
    Symbol string; Sector Sector; Value int; Valid bool; Tier string; Confidence string; Coverage float64
    Pillars map[string]float64; Metrics []MetricScore; SectorAdj int; RedFlags []string; Notes []string
}
func ScoreStock(in Inputs, c Config) Score
func ClassifySector(icb1, icb2, icb4, symbol string) (Sector, bool)

type Action string // REVIEW, TRIM, ADD, HOLD
type Advice struct {
    Symbol string; Action Action; Rule string; Reason string; Secondary []string
    Lots, Qty int64; EstCash int64 // + proceeds / − cost, after fee and tax
    Breakeven int64; OddLot int64
}
type PortfolioAdvice struct { Messages []string; Holdings []Advice; SectorWeights map[Sector]float64 }
func Recommend(sum portfolio.Summary, scores map[string]Score, sectors map[string]Sector, c Config) PortfolioAdvice
type Candidate struct { Score Score; Inputs Inputs; RankScore float64; Reasons []string; LotCost int64; MaxAlloc int64 }
func Candidates(universe []Inputs, held map[string]bool, sum portfolio.Summary, sectorW map[Sector]float64, c Config) ([]Candidate, string /* note e.g. "For watchlist only." */)
```

## Scoring spec

The defaults below live in `Config`.

**Normalisation.** `s = 100 × clamp((x − worst)/(best − worst), 0, 1)`. Metric
scores keep 1 decimal place; the final score is rounded to an integer.

**Winsorisation** before scoring:

| Metric | Range |
|---|---|
| VOL | 0 to 2.0 |
| BETA | −1 to 3 |
| PE | −100 to 200 |
| PB | 0 to 20 |
| DE | 0 to 10 |

### Metric bands

| Metric | best → 100 | worst → 0 | notes |
|---|---|---|---|
| VOL (1y) | ≤ 0.20 | ≥ 0.60 | |
| BETA | ≤ 0.9 | ≥ 1.5 | Cap the score at 70 if β < 0.3 and ADTV < 5bn. Use weekly 2y beta when R² < 0.1 or ADTV < 5bn. |
| MDD (1y) | ≤ 0.15 | ≥ 0.50 | |
| LIQ (ADTV, 60 sessions) | ≥ 2e10 | ≤ 1e9 | log10 scale |
| ROE general | ≥ 0.20 | ≤ 0.05 | |
| ROE bank | ≥ 0.20 | ≤ 0.08 | |
| EARN_CONS | - | - | Already 0-100: `pos/n×70 + (1−min(CV,1))×30`, where n is the number of fiscal years available (KBS returns at most 4); needs n ≥ 3, otherwise dropped |
| DE general | ≤ 0.3 | ≥ 1.5 | |
| DE real_estate | ≤ 0.5 | ≥ 2.0 | |
| DE utilities_infra | ≤ 0.8 | ≥ 2.5 | |
| NPL (bank) | ≤ 0.010 | ≥ 0.035 | |
| CASA (bank) | ≥ 0.35 | ≤ 0.10 | |
| DY | ≥ 0.06 | 0 | +10 (cap 100) if paid cash in all of the last 3 years. If DY > 0.10, cap the raw value at 0.08 and add the note "high trailing yield, likely special dividend". |

**PE for non-banks.** When the sector median comes from at least 5 peers, score
the ratio PE/median at these points, linear in between:

| PE/median | Score |
|---|---|
| ≤ 0.8 | 100 |
| 1.0 | 70 |
| 1.3 | 40 |
| ≥ 1.6 | 0 |

Otherwise use absolute bands:

| PE | Score |
|---|---|
| ≤ 0 | 0 |
| 0 to 4 | 50 |
| 4 to 12 | 100 |
| 12 to 18 | 100 → 60 |
| 18 to 30 | 60 → 0 |
| ≥ 30 | 0 |

**PE for banks.**

| PE | Score |
|---|---|
| 4 to 10 | 100 |
| 10 to 15 | 100 → 50 |
| 15 to 20 | 50 → 0 |
| ≥ 20 | 0 |

**PB general.**

| PB | Score |
|---|---|
| ≤ 1.5 | 100 |
| 1.5 to 3 | 100 → 60 |
| 3 to 6 | 60 → 0 |
| ≥ 6 | 0 |

If PB < 0.7 and ROE < 0.08, cap the score at 50.

**PB for banks and securities.** `fairPB = ROE / CoE`, with CoE = 0.13.
Score the ratio PB/fairPB:

| PB/fairPB | Score |
|---|---|
| ≤ 0.8 | 100 |
| 1.0 | 75 |
| 1.25 | 40 |
| ≥ 1.5 | 0 |

LLR, MARGIN_LEV and COMBINED come from no available source. They are defined
but always missing, so they are dropped and listed in the notes.

### Weights (each profile sums to 100)

| Profile | Price 40 | Fundamentals 35 | Valuation 25 | Adj |
|---|---|---|---|---|
| general | VOL15 BETA10 MDD10 LIQ5 | ROE12 EARN_CONS10 DE13 | PE10 PB5 DY10 | 0 |
| real_estate | same | same (RE DE bands) | same | −5 |
| utilities_infra | same | same (util DE bands) | same | 0 |
| bank | same | ROE12 NPL10 LLR5 CASA8 | PBfair12 PE5 DY8 | 0 |
| securities | same | ROE12 EARN_CONS13 MARGIN_LEV10 | PBfair12 PE5 DY8 | −5 |
| insurance | same | ROE12 EARN_CONS13 COMBINED10 | PB8 PE9 DY8 | 0 |

### Missing data

Never impute zero for a missing value.

1. A missing metric's weight is redistributed across the remaining metrics in
   its pillar, in proportion to their weights.
2. If less than 50% of a pillar's weight is present, the pillar is marked
   missing and its weight is redistributed across the remaining pillars.
3. Coverage is the sum of the original weights that were used. Confidence is
   High at 85 or more, Medium at 60 to 84, and Low below 60.
4. If coverage is below 60 or the price pillar is missing, the score is
   invalid and the tier is "Insufficient data".
5. Every dropped metric produces a note.

The sector adjustment is applied after the weighted sum and floored at 0.

### Tiers

| Score | Tier |
|---|---|
| ≥ 75 | Stable |
| 60 to 74 | Moderate |
| 45 to 59 | Weak/Volatile |
| < 45 | Speculative |

### Red flags

Any one of these forces REVIEW:

- the listing status is anything other than `listed` (suspended,
  delisted). The public sources do not expose the warning/control/restricted
  lists, so the report footer says that check is not covered.
- negative equity
- a TTM loss together with a loss in the previous year
- ADTV below 1e9
- more than 10 gap sessions in the last 60
- the stock's last bar is more than 5 VN-Index sessions behind the index's
  last bar (stale or suspended), so long holidays do not trigger it
- fundamentals older than 2 quarters (more than 200 days)

### Sector classification

`ClassifySector` maps the ICB names, case-insensitive and trimmed:

| ICB match | Sector |
|---|---|
| ICB2 "Banks" | bank |
| ICB2 "Insurance" | insurance |
| ICB2 "Financial Services" with ICB4 containing "Brokerage" or "Investment Services" | securities |
| ICB2 "Real Estate" | real_estate |
| ICB1 "Utilities" | utilities_infra |
| ICB4 containing "Transportation Services" or "Marine" | utilities_infra |
| anything else | general |

If the ICB data is absent, fall back to a fixed list of bank tickers:

```
VCB BID CTG TCB MBB ACB VPB HDB STB VIB TPB SHB MSB LPB OCB EIB SSB NAB ABB BAB BVB KLB NVB PGB SGB VAB VBB
```

If there is still no match, the sector is `general` with the note "sector unknown".

## Portfolio rules (`Config` defaults)

| Field | Default |
|---|---|
| SingleSoftCap | 0.20 |
| SingleHardCap | 0.25 |
| SectorCap | 0.40 |
| CashMin | 0.05 |
| CashHigh | 0.40 |
| MarginMax | 0.20 |
| AddMinScore | 70 |
| AddStrongScore | 75 |
| WeakScore | 60 |
| ReviewScore | 45 |
| ReviewLossPct | −0.20 |
| WeakLossPct | −0.15 |
| TakeProfitPct | 0.30 |
| AddRoomBuffer | 0.05 |
| CoE | 0.13 |
| FeeRate | 0.001 (from `--fee-rate`) |
| SellTax | 0.001 |
| DividendTax | 0.05 |
| CandidateMinScore | 70 |
| CandidateMaxVol | 0.40 |
| CandidateMinCoverage | 80 |
| CandidateMinListedYears | 2 |
| CandidatePerSector | 2 |
| CandidateLowSectorBonus | 5 |
| CandidateLowSectorWeight | 0.15 |

## Actions per holding

Rules run in order; the first match is the primary action. Every other
triggered rule goes into `Secondary`. The `Rule` string cites the numbers, for
example "TRIM: weight 27.4% > hard cap 25%".

**1. REVIEW** if any of these hold:

- there is a red flag
- the score is invalid
- S < 45
- pnl ≤ −20% and S < 60
- pnl ≤ −15% and S < 50

**2. TRIM** if any of these hold, checked in this order:

1. w > 25%: trim to 20%.
2. The sector weight is above 40%: trim the lowest-score holdings in that
   sector until it is back to 40%.
3. S < 60 and pnl ≥ 30%: sell 1/3.
4. 45 ≤ S < 60 and w > 10%: trim to 10%.

For every trim:

- `qty = floor(excessValue / price / 100) × 100`
- cap the qty at the sellable quantity, rounded down to a whole lot
- if receiving shares cut the trim short, add the note "N shares settle T+2"
- `EstCash = qty × price × (1 − fee − sellTax)`

**3. ADD** only if all of these hold:

- S ≥ 70
- w < 15% (soft cap minus buffer)
- the sector weight after the add stays at or below 40%
- the cash ratio is above 5%
- the margin ratio is at or below 20%
- the valuation pillar is at least 50
- pnl > −15%, or S ≥ 75

The amount is `min((softCap − w) × NAV, deployable / max(1, number of ADD candidates))`,
where `deployable = cash − cashMin × NAV`. Round it down to whole lots at
`price × (1 + fee)`. If that is less than 1 lot, the action becomes HOLD with
the note "insufficient cash for one lot".

ETFs and covered warrants (`Type` not `STOCK`, or `/se` `secType` marks them)
skip scoring and rules 1 to 3: they get HOLD with the note "not scored
(fund/warrant)" and still count in NAV and weights.

**4. HOLD** otherwise. The reason names the top 2 metric contributions, plus
the weakest metric when its score is below 40.

## Portfolio messages

- The cash ratio is below 5%: "Cash buffer below 5%: ..."
- The cash ratio is above 40%: suggest deploying in 2-3 tranches.
- There is margin debt: a warning. If margin is above 20% of NAV, suppress
  all ADDs and say "reduce margin first".
- The weight of every sector, with any above the cap flagged.
- "concentrated" when effectiveN is below 4.
- "under-diversified" when there are fewer than 5 holdings.
- Odd lots.
- Prices labelled "intraday, not closing" during trading hours.

## Candidates

The universe is the `--universe` list minus held tickers. A candidate must meet
all of these:

- a valid score of at least 70
- coverage of at least 80
- VOL at or below 0.40
- no red flags
- listed for at least 2 years
- positive profit in each of the last 3 years (`ProfitYears`), if known; when unknown, add a note and still allow it
- DE better than the sector's worst threshold
- for banks, NPL at or below 3%
- its sector weight in the portfolio below the 40% cap

Ranking:

- `RankScore` is S, plus 5 if the sector's portfolio weight is below 15%.
- At most 2 candidates per sector.
- Take the top N.

`LotCost` is `100 × price × (1 + fee)`. `MaxAlloc` is
`min(softCap × NAV, deployable/3)`. If deployable is less than the cheapest
lot, the note is "For watchlist only."

## Market rules (`market.go`)

- The lot is 100 shares. Odd lots (1-99) are reported separately.
- Only what the report uses lives here. The CLI shows no limit prices, so tick
  sizes and price bands are not implemented; `AdjustJumps` in research takes
  its band (7% HOSE, 10% HNX, 15% UPCoM) from a small table there.
- `func MarketPhase(now time.Time) string` returns "intraday" on weekdays from
  09:00 to 15:00 Asia/Ho_Chi_Minh, otherwise "last close". There is no holiday
  calendar.
- `BuyCost`, `SellProceeds` and `Breakeven` use `avgCost × (1+fee)/(1−fee−sellTax)`.
- Settlement is T+2. This is only used for notes.

## Tests

- Table tests for every band edge, including best, worst, the midpoint and
  past both ends.
- Redistribution: a bank without LLR, and a stock that loses a whole pillar.
- Coverage below 60 gives an invalid score.
- Each tier boundary.
- Each red flag.
- Each recommendation rule in isolation, plus the priority order: a stock
  that matches both REVIEW and TRIM gets REVIEW, with TRIM as secondary.
- Lot rounding and the sellable cap.
- Margin suppression.
- Candidate filters and the per-sector limit.
- ETFs skip scoring and get HOLD with the "not scored" note.
- The staleness red flag across a 9-day holiday gap in the index does not
  fire.
