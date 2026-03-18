package tcbs

import "context"

// TokenRequest represents the request body for exchanging API Key + OTP for JWT Token.
type TokenRequest struct {
	APIKey string `json:"apiKey"`
	OTP    string `json:"otp"`
}

// TokenResponse represents a successful token exchange response.
type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int64  `json:"expiresIn"`
}

// TokenErrorResponse represents a failed token exchange response.
type TokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// GetToken exchanges an API Key and OTP for a JWT token.
// The returned token is valid for up to 8 hours.
func (c *Client) GetToken(ctx context.Context, apiKey, otp string) (*TokenResponse, error) {
	var resp TokenResponse
	err := c.doRequest(ctx, "POST", "/gaia/v1/oauth2/openapi/token", nil, &TokenRequest{
		APIKey: apiKey,
		OTP:    otp,
	}, &resp)
	if err != nil {
		return nil, err
	}
	c.token = resp.AccessToken
	return &resp, nil
}
