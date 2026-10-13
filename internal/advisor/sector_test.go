package advisor

import "testing"

func TestClassifySector(t *testing.T) {
	cases := []struct {
		name                     string
		icb1, icb2, icb4, symbol string
		want                     Sector
		ok                       bool
	}{
		{"bank", "FINANCIALS ", "Banks", "Banks", "VCB", SectorBank, true},
		{"bank case and space", "financials", "  BANKS ", "banks", "XYZ", SectorBank, true},
		{"insurance", "Financials", "Insurance", "Full Line Insurance", "BVH", SectorInsurance, true},
		{"brokerage", "Financials", "Financial Services", "Investment Services", "SSI", SectorSecurities, true},
		{"brokerage word", "Financials", "Financial Services", "Securities Brokerage", "VND", SectorSecurities, true},
		{"other financial services", "Financials", "Financial Services", "Consumer Finance", "EVF", SectorGeneral, true},
		{"real estate", "Financials", "Real Estate", "Real Estate Holding & Development", "VHM", SectorRealEstate, true},
		{"utilities", "Utilities", "Utilities", "Conventional Electricity", "POW", SectorUtilities, true},
		{"transport services", "Industrials", "Industrial Goods & Services", "Transportation Services", "GMD", SectorUtilities, true},
		{"marine", "Industrials", "Industrial Goods & Services", "Marine Transportation", "HAH", SectorUtilities, true},
		{"technology", "Technology", "Technology", "Computer Services", "FPT", SectorGeneral, true},
		{"no icb bank ticker", "", "", "", "tcb", SectorBank, true},
		{"no icb unknown", " ", "", "", "FPT", SectorGeneral, false},
		{"icb overrides ticker list", "Technology", "Technology", "Software", "VCB", SectorGeneral, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ClassifySector(tc.icb1, tc.icb2, tc.icb4, tc.symbol)
			if got != tc.want || ok != tc.ok {
				t.Errorf("got %s %v, want %s %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}
