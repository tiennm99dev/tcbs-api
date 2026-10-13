package tcbs

import (
	"context"
	"strings"
)

// accountStatusClosed marks a closed sub-account in MarginQuotaResponse.AccountStatus.
const accountStatusClosed = "C"

// internalAccountTypes are TCBS-internal sub-account types (covered warrants
// and structured products) that hold no regular stock positions.
var internalAccountTypes = map[string]bool{
	"CW":  true,
	"SPW": true,
	"SPT": true,
}

// ListAccounts returns the sub-accounts under a custody code (105C...).
func (c *Client) ListAccounts(ctx context.Context, custodyCode string) ([]MarginQuotaResponse, error) {
	return c.GetMarginQuota(ctx, custodyCode)
}

// ActiveStockAccounts drops closed accounts (accountStatus "C") and the
// internal CW/SPW/SPT types.
func ActiveStockAccounts(in []MarginQuotaResponse) []MarginQuotaResponse {
	var out []MarginQuotaResponse
	for _, a := range in {
		if strings.EqualFold(strings.TrimSpace(a.AccountStatus), accountStatusClosed) {
			continue
		}
		if internalAccountTypes[strings.ToUpper(strings.TrimSpace(a.AccountType))] {
			continue
		}
		out = append(out, a)
	}
	return out
}
