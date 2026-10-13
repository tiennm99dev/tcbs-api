package advisor

import "strings"

// Sector is a scoring profile.
type Sector string

// Sectors. SectorFund is not a scoring profile: it groups ETFs and covered
// warrants in the sector weights so they never count toward a stock
// sector's cap.
const (
	SectorBank       Sector = "bank"
	SectorSecurities Sector = "securities"
	SectorInsurance  Sector = "insurance"
	SectorRealEstate Sector = "real_estate"
	SectorUtilities  Sector = "utilities_infra"
	SectorGeneral    Sector = "general"
	SectorFund       Sector = "fund"
)

// NoteSectorUnknown is added to a score whose sector could not be classified.
const NoteSectorUnknown = "sector unknown"

// bankTickers is the fallback when no ICB data is available.
var bankTickers = map[string]bool{}

func init() {
	for _, s := range strings.Fields("VCB BID CTG TCB MBB ACB VPB HDB STB VIB TPB SHB MSB LPB OCB EIB SSB NAB ABB BAB BVB KLB NVB PGB SGB VAB VBB") {
		bankTickers[s] = true
	}
}

// ClassifySector maps ICB level 1, 2 and 4 names (case-insensitive,
// trimmed) to a sector. Without any ICB data it falls back to a fixed list
// of bank tickers. ok is false when neither source classifies the symbol;
// the sector is then general, and the caller should leave Inputs.Sector
// empty or add NoteSectorUnknown.
func ClassifySector(icb1, icb2, icb4, symbol string) (Sector, bool) {
	n1 := strings.ToLower(strings.TrimSpace(icb1))
	n2 := strings.ToLower(strings.TrimSpace(icb2))
	n4 := strings.ToLower(strings.TrimSpace(icb4))
	if n1 == "" && n2 == "" && n4 == "" {
		if bankTickers[strings.ToUpper(strings.TrimSpace(symbol))] {
			return SectorBank, true
		}
		return SectorGeneral, false
	}
	switch {
	case n2 == "banks":
		return SectorBank, true
	case n2 == "insurance":
		return SectorInsurance, true
	case n2 == "financial services" && (strings.Contains(n4, "brokerage") || strings.Contains(n4, "investment services")):
		return SectorSecurities, true
	case n2 == "real estate":
		return SectorRealEstate, true
	case n1 == "utilities":
		return SectorUtilities, true
	case strings.Contains(n4, "transportation services") || strings.Contains(n4, "marine"):
		return SectorUtilities, true
	}
	return SectorGeneral, true
}
