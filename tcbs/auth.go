package tcbs

import "context"

// TokenRequest represents the request body for exchanging API Key + OTP for JWT Token.
type TokenRequest struct {
	OTP    string `json:"otp"`
	APIKey string `json:"apiKey"`
}

// TokenResponse represents a successful token exchange response.
type TokenResponse struct {
	Token string `json:"token"`
}

// TokenErrorResponse represents a failed token exchange response.
type TokenErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GetToken exchanges an API Key and OTP for a JWT token.
func (c *Client) GetToken(ctx context.Context, apiKey, otp string) (*TokenResponse, error) {
	var resp TokenResponse
	err := c.doRequest(ctx, "POST", "/gaia/v1/oauth2/openapi/token", nil, &TokenRequest{
		APIKey: apiKey,
		OTP:    otp,
	}, &resp)
	if err != nil {
		return nil, err
	}
	c.SetToken(resp.Token)
	return &resp, nil
}
