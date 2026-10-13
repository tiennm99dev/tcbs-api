package tcbs

import (
	"context"
	"net/http"
	"testing"
)

func TestGetToken(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/gaia/v1/oauth2/openapi/token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, TokenResponse{Token: "jwt-123"})
	})

	resp, err := client.GetToken(context.Background(), "key", "otp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "jwt-123" {
		t.Errorf("expected token 'jwt-123', got %q", resp.Token)
	}
	if client.currentToken() != "jwt-123" {
		t.Errorf("expected client token updated to 'jwt-123', got %q", client.currentToken())
	}
}
