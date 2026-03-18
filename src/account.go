package tcbs

import (
	"context"
	"fmt"
	"net/url"
)

// GetSubAccountInfo retrieves sub-account information for the given custody code.
// The fields parameter specifies which info to retrieve, e.g. "basicInfo,personalInfo,bankSubAccounts,bankAccounts".
func (c *Client) GetSubAccountInfo(ctx context.Context, custodyCode string, fields string) (*AccountInformationResponse, error) {
	query := url.Values{}
	query.Set("fields", fields)

	var resp AccountInformationResponse
	err := c.get(ctx, fmt.Sprintf("/eros/v2/get-profile/by-username/%s", custodyCode), query, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
