# Portfolio stability report

Generated 2026-10-13 15:00 +07 (Asia/Ho_Chi_Minh). Prices: last close.

> Informational, rule-based output; not financial advice. Verify prices and rules with your broker before trading.

## Portfolio summary

Sub-accounts: ****001N (normal), ****001G (margin).

| Item | VND |
|---|--:|
| NAV | 354,651,100 |
| Stock value | 264,651,100 |
| Cash | 90,000,000 |
| Incoming cash dividends (in NAV) | 450,000 |
| Debt | 0 |
| IA bank cash (not in NAV) | 10,000,000 |
| Receivable sale proceeds (not in NAV) | 12,000,000 |
| Pending stock-dividend value (not in NAV) | 27,067,500 |

- Cash ratio 25.4%, margin ratio 0.0%.
- HHI 0.30, effective N 3.33.
- Pending stock-dividend shares (not in NAV): FPT 225 shares.
- Receivable sale proceeds (not in NAV): 12,000,000 VND.
- Market: VN-Index 1719.42 on 2026-10-13; 1-year return 0.2%, 1-year volatility 20.5%.

Portfolio messages:

- Portfolio concentrated: effectiveN 3.3 < 4.
- HPG: average cost unknown (gifted or transferred shares)

## Sector weights

| Sector | Weight | Status |
|---|--:|---|
| general | 59.6% | above the 40.0% cap |
| fund | 14.8% | funds and warrants, no cap |

## Holdings

| Symbol | Qty (sellable / receiving) | Avg cost | Price (source) | Market value | Weight | P&L | Breakeven | Score | Tier | Action | Rule |
|---|--:|--:|--:|--:|--:|--:|--:|--:|---|---|---|
| FPT | 1,537 (1,437 / 100) | 100,000 | 120,300 (vndirect) | 184,901,100 | 52.0% | 20.3% | 100,301 | 81 | Stable | TRIM | TRIM: weight 52.0% > hard cap 25%; trim to 20% |
| HPG | 1,000 (1,000 / 0) | unknown | 26,950 (ref) | 26,950,000 | 7.6% | unknown | unknown | n/a | Insufficient data | REVIEW | REVIEW: red flag \| ADTV 0.5bn VND < 1bn VND (hard to exit) |
| E1VFVN30 | 2,000 (2,000 / 0) | 24,100 | 26,400 (tcbs-match) | 52,800,000 | 14.8% | 9.5% | 24,173 | n/a | Not scored | HOLD | HOLD: not scored (fund/warrant) |

## Holding detail

### FPT - FPT Corporation

- Exchange HOSE, sector general (ICB4 Computer Services).
- 52-week range: 95,000 - 130,500 VND.
- Dividends, last 3 years: 2026-05-28 stock 15.0%; 2025-06-12 cash 1,000 VND/share.
- Action: **TRIM**. TRIM: weight 52.0% > hard cap 25%; trim to 20%.
- Reason: sell 9 lots (900 shares).
- Also triggered: ADD blocked by the trim.
- Suggested TRIM: 900 shares in board lots of 100, estimated cash impact +108,053,820 VND (fee and tax included).
- Odd lot: 37 shares trade on the odd-lot board.

Score 81, tier Stable, coverage 90, confidence High.

| Metric | Pillar | Raw | Score | Weight | Contribution |
|---|---|--:|--:|--:|--:|
| VOL | price | 31.9% | 70.3 | 15.0 | 10.5 |
| LIQ | price | 380bn VND | 100.0 | 5.0 | 5.0 |
| DY | valuation | 3.6% | 69.2 | 10.0 | 6.9 |

- Pillars: price 70.4, fundamentals 93.6, valuation 81.3.
- Sector adjustment: 0.
- DY: +10: cash paid in each of the last 3 years.
- Dropped metrics: EARN_CONS.
- Red flags: none.
- Note: EARN_CONS unavailable: weight redistributed within fundamentals.

### HPG - Hoa Phat Group

- Exchange HOSE, sector n/a.
- 52-week range: n/a.
- Dividends, last 3 years: unavailable.
- Action: **REVIEW**. REVIEW: red flag | ADTV 0.5bn VND < 1bn VND (hard to exit).

Score n/a, tier Insufficient data, coverage 40, confidence Low.

- Pillars: price missing, fundamentals missing, valuation missing.
- Sector adjustment: 0.
- Red flags: ADTV 0.5bn VND < 1bn VND (hard to exit).

### E1VFVN30

- No research data.
- Action: **HOLD**. HOLD: not scored (fund/warrant).

Score: not scored (fund/warrant).

## Stable candidates

Universe: VN30 (30 stocks), minus the symbols already held.

For watchlist only.

| # | Symbol | Sector | Score | Tier | VOL | BETA | DY | ROE | PE | PB | Price | Lot cost | Max alloc | Reasons |
|--:|---|---|--:|---|--:|--:|--:|--:|--:|--:|--:|--:|--:|---|
| 1 | VNM | general | 91 | Stable | 29.1% | 0.49 | 8.1% | 30.4% | 11.13 | 3.83 | 58,000 | 5,805,800 | 13,738,003 | DE 0.24 (score 100); ROE 30.4% (score 100) |

## Data sources and timestamps

| Source | Data | As of |
|---|---|---|
| offline holdings file | holdings, cash, margin | 2026-10-13 15:00 +07 |
| vndirect | daily bars (latest bar) | 2026-10-13 |

The public sources (VNDirect, Vietcap, KBS) are undocumented and can change without notice.

Not covered: the exchange warning, control and restricted lists. The public sources do not expose them, so only a listing status other than "listed" and price-history gaps are flagged.

## Config in use

| Setting | Value |
|---|---|
| Single-stock caps (soft / hard) | 20.0% / 25.0% |
| Sector cap | 40.0% |
| Cash band (min / high) | 5.0% / 40.0% |
| Margin max | 20.0% |
| ADD score (min / strong) | 70 / 75 |
| Weak / review score | 60 / 45 |
| Review loss / weak loss | -20.0% / -15.0% |
| Take-profit gain | 30.0% |
| Fee per side / sales tax | 0.1% / 0.1% |
| Cost of equity (fair PB) | 13.0% |
| Candidates (count / min score / max VOL) | 10 / 70 / 40.0% |

## Warnings

- ABC: no daily bars from any source
- ZZZ: no listing data

---

Informational, rule-based output; not financial advice. Verify prices and rules with your broker before trading.
