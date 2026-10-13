package tcbs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	ProductionBaseURL = "https://openapi.tcbs.com.vn"
	SITBaseURL        = "https://openapisit.tcbs.com.vn"

	defaultTimeout = 30 * time.Second
)

// Client is the TCBS API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex
	token      string
}

// Option configures the Client.
type Option func(*Client)

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithToken sets the Bearer token directly.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// NewClient creates a new TCBS API client.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL: ProductionBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SetToken sets the Bearer token for authentication.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// currentToken returns the current Bearer token.
func (c *Client) currentToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// maxErrorBody caps how much of a response body APIError.Error prints.
const maxErrorBody = 512

// APIError represents an error response from the TCBS API.
// Code and Message are filled when the body is a {code,message} JSON object.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Code != "" || e.Message != "" {
		return fmt.Sprintf("tcbs: API error status=%d code=%s message=%s", e.StatusCode, e.Code, e.Message)
	}
	body := e.Body
	if len(body) > maxErrorBody {
		body = body[:maxErrorBody] + "...(truncated)"
	}
	return fmt.Sprintf("tcbs: API error status=%d body=%s", e.StatusCode, body)
}

// newAPIError builds an APIError, decoding {code,message} from the body when present.
func newAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode, Body: string(body)}
	var payload struct {
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		apiErr.Code = rawCode(payload.Code)
		apiErr.Message = payload.Message
	}
	return apiErr
}

// rawCode renders a code that may arrive as a JSON string or number.
func rawCode(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values, body any, result any) error {
	u := c.baseURL + path
	if query != nil {
		u += "?" + query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("tcbs: marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return fmt.Errorf("tcbs: create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if token := c.currentToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tcbs: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("tcbs: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return newAPIError(resp.StatusCode, respBody)
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("tcbs: unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values, result any) error {
	return c.doRequest(ctx, http.MethodGet, path, query, nil, result)
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	return c.doRequest(ctx, http.MethodPost, path, nil, body, result)
}

func (c *Client) put(ctx context.Context, path string, body any, result any) error {
	return c.doRequest(ctx, http.MethodPut, path, nil, body, result)
}
