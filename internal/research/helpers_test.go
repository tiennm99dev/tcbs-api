package research

import (
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testNow is a fixed clock inside the fixtures' capture day, 15:00 in
// Vietnam.
var testNow = time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

// newTestHTTP returns a client with no real throttling or backoff.
func newTestHTTP(t *testing.T) *HTTPClient {
	t.Helper()
	c := NewHTTPClient(nil, 5*time.Second, 1000, []time.Duration{time.Millisecond, time.Millisecond})
	t.Cleanup(c.Close)
	return c
}

func newServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func serveFixture(t *testing.T, name string) http.HandlerFunc {
	b := fixture(t, name)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func near(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

func wantFloat(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s = nil, want %v", name, want)
		return
	}
	if !near(*got, want) {
		t.Errorf("%s = %v, want %v", name, *got, want)
	}
}
