package tcbs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer creates a test HTTP server and a Client pointing at it.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClient(WithBaseURL(srv.URL), WithToken("test-token"))
	return client, srv
}

// writeJSON is a test helper to write JSON responses.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode response: %v", err)
	}
}

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient()
	if c.baseURL != ProductionBaseURL {
		t.Errorf("expected base URL %s, got %s", ProductionBaseURL, c.baseURL)
	}
	if c.httpClient == nil {
		t.Error("expected non-nil http client")
	}
}

func TestNewClient_Options(t *testing.T) {
	c := NewClient(WithBaseURL(SITBaseURL), WithToken("tok"))
	if c.baseURL != SITBaseURL {
		t.Errorf("expected base URL %s, got %s", SITBaseURL, c.baseURL)
	}
	if c.currentToken() != "tok" {
		t.Errorf("expected token 'tok', got %q", c.currentToken())
	}
}

func TestSetToken(t *testing.T) {
	c := NewClient()
	c.SetToken("abc")
	if c.currentToken() != "abc" {
		t.Errorf("expected token 'abc', got %q", c.currentToken())
	}
}

func TestAPIError(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	})
	err := client.get(context.Background(), "/fail", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestAuthorizationHeader(t *testing.T) {
	var gotAuth string
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		writeJSON(t, w, MarketStockInfoResponse{})
	})
	_, _ = client.GetStockPrices(context.Background(), []string{"FPT"})
	if gotAuth != "Bearer test-token" {
		t.Errorf("expected 'Bearer test-token', got %q", gotAuth)
	}
}

func TestAPIError_CodeMessage(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"203074","message":"The API Key is invalid"}`))
	})
	err := client.get(context.Background(), "/fail", nil, nil)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != "203074" {
		t.Errorf("expected code 203074, got %q", apiErr.Code)
	}
	if apiErr.Message != "The API Key is invalid" {
		t.Errorf("unexpected message %q", apiErr.Message)
	}
	if got := apiErr.Error(); !strings.Contains(got, "203074") || !strings.Contains(got, "The API Key is invalid") {
		t.Errorf("unexpected Error(): %s", got)
	}
	if strings.Contains(apiErr.Error(), "test-token") {
		t.Error("Error() must not include the bearer token")
	}
}

func TestAPIError_TruncatesBody(t *testing.T) {
	e := &APIError{StatusCode: 500, Body: strings.Repeat("x", 2000)}
	if got := e.Error(); len(got) > 600 {
		t.Errorf("expected truncated error, got %d bytes", len(got))
	}
}
