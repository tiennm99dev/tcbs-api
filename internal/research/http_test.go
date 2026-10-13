package research

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPRetriesTransientErrors(t *testing.T) {
	for _, code := range []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway} {
		var calls atomic.Int32
		srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) < 3 {
				http.Error(w, "busy", code)
				return
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		})
		var v struct{ OK bool }
		if err := newTestHTTP(t).GetJSON(context.Background(), srv.URL, nil, &v); err != nil || !v.OK {
			t.Errorf("HTTP %d: err = %v, v = %+v", code, err, v)
		}
		if calls.Load() != 3 {
			t.Errorf("HTTP %d: calls = %d, want 3 (two retries)", code, calls.Load())
		}
	}
}

func TestHTTPGivesUpAfterTwoRetries(t *testing.T) {
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "down", http.StatusServiceUnavailable)
	})
	var v any
	err := newTestHTTP(t).GetJSON(context.Background(), srv.URL+"/x", nil, &v)
	var se *StatusError
	if !errors.As(err, &se) || se.Code != http.StatusServiceUnavailable {
		t.Errorf("err = %v, want StatusError 503", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestHTTPNoRetryOnClientError(t *testing.T) {
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "bad", http.StatusBadRequest)
	})
	var v any
	if err := newTestHTTP(t).GetJSON(context.Background(), srv.URL, nil, &v); err == nil {
		t.Error("want an error")
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestHTTPRetriesNetworkErrors(t *testing.T) {
	var calls atomic.Int32
	failing := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("connection reset")
	})
	c := NewHTTPClient(failing, time.Second, 1000, []time.Duration{time.Millisecond, time.Millisecond})
	defer c.Close()
	var v any
	if err := c.GetJSON(context.Background(), "http://example.invalid/x", nil, &v); err == nil {
		t.Error("want an error")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHTTPHeadersAndPost(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != UserAgent || r.Header.Get("Accept") != acceptHeader {
			t.Errorf("headers = %v", r.Header)
		}
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Extra") != "1" {
			t.Errorf("post headers = %v", r.Header)
		}
		_, _ = w.Write([]byte(`[1,2]`))
	})
	h := http.Header{}
	h.Set("X-Extra", "1")
	var v []int
	if err := newTestHTTP(t).PostJSON(context.Background(), srv.URL, h, map[string]int{"a": 1}, &v); err != nil || len(v) != 2 {
		t.Errorf("err = %v, v = %v", err, v)
	}
}

func TestHTTPThrottlesPerHost(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })
	c := NewHTTPClient(nil, time.Second, 10, []time.Duration{})
	defer c.Close()

	start := time.Now()
	for i := 0; i < 13; i++ {
		var v any
		if err := c.GetJSON(context.Background(), srv.URL, nil, &v); err != nil {
			t.Fatal(err)
		}
	}
	// 10 tokens up front, then one per 100ms: the last 3 wait about 300ms.
	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Errorf("13 requests at 10/s took %v, want at least ~300ms", elapsed)
	}
}

func TestHTTPHonoursContext(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	})
	c := NewHTTPClient(nil, time.Second, 1000, []time.Duration{time.Hour})
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	var v any
	if err := c.GetJSON(ctx, srv.URL, nil, &v); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want deadline exceeded during backoff", err)
	}
}

func TestDefaultTransportCapsTLSForDchart(t *testing.T) {
	rt, ok := DefaultTransport().(*hostRouter)
	if !ok {
		t.Fatalf("DefaultTransport = %T", DefaultTransport())
	}
	capped := rt.tls12.(*http.Transport)
	if capped.TLSClientConfig == nil || capped.TLSClientConfig.MaxVersion != tls.VersionTLS12 {
		t.Errorf("dchart transport TLS config = %+v", capped.TLSClientConfig)
	}
	if base := rt.base.(*http.Transport); base.TLSClientConfig != nil && base.TLSClientConfig.MaxVersion != 0 {
		t.Errorf("other hosts must keep the default TLS versions")
	}
	if !tls12Hosts["dchart-api.vndirect.com.vn"] || tls12Hosts["api-finfo.vndirect.com.vn"] {
		t.Errorf("tls12Hosts = %v", tls12Hosts)
	}
}

func TestHTTPSkipsUnresponsiveHost(t *testing.T) {
	// A host that accepts the request but never answers in time.
	var calls atomic.Int32
	hung := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.URL.Host == "ok.example" {
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: http.Header{}}, nil
		}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	c := NewHTTPClient(hung, 20*time.Millisecond, 1000, []time.Duration{time.Millisecond, time.Millisecond})
	defer c.Close()
	var v any
	for i := 0; i < hostDownAfter; i++ {
		if err := c.GetJSON(context.Background(), "http://hung.example/x", nil, &v); err == nil || errors.Is(err, ErrHostDown) {
			t.Fatalf("request %d: err = %v, want a timeout", i, err)
		}
	}
	if got := calls.Load(); got != 3*hostDownAfter {
		t.Fatalf("calls = %d, want %d", got, 3*hostDownAfter)
	}

	start := time.Now()
	err := c.GetJSON(context.Background(), "http://hung.example/y", nil, &v)
	if !errors.Is(err, ErrHostDown) {
		t.Errorf("err = %v, want ErrHostDown", err)
	}
	if calls.Load() != 3*hostDownAfter || time.Since(start) > 10*time.Millisecond {
		t.Errorf("skipped host was contacted (calls %d, %v)", calls.Load(), time.Since(start))
	}
	// Other hosts are unaffected.
	if err := c.GetJSON(context.Background(), "http://ok.example/x", nil, &v); errors.Is(err, ErrHostDown) {
		t.Errorf("healthy host skipped: %v", err)
	}
}

func TestHTTPHostFailureCountResets(t *testing.T) {
	var calls atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		// Every third request answers; the others fail with 503.
		if calls.Add(1)%3 == 0 {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		http.Error(w, "busy", http.StatusServiceUnavailable)
	})
	c := NewHTTPClient(nil, time.Second, 1000, []time.Duration{})
	defer c.Close()
	var v any
	for i := 0; i < 3*hostDownAfter; i++ {
		if err := c.GetJSON(context.Background(), srv.URL, nil, &v); errors.Is(err, ErrHostDown) {
			t.Fatalf("request %d: host skipped though it keeps answering", i)
		}
	}
	// A 4xx answer also proves the host is up.
	bad := newServer(t, func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "bad", http.StatusNotFound) })
	for i := 0; i < 2*hostDownAfter; i++ {
		if err := c.GetJSON(context.Background(), bad.URL, nil, &v); errors.Is(err, ErrHostDown) {
			t.Fatalf("request %d: host skipped after 4xx answers", i)
		}
	}
}
