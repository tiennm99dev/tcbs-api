// Package research fetches public Vietnamese market data (daily bars,
// ratios, dividends, profiles and index constituents) from VNDirect, Vietcap
// and KBS, caches it on disk and computes price metrics from it.
//
// Nothing in this package talks to TCBS or handles credentials.
package research

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// UserAgent is sent on every request. Vietcap and KBS reject requests
	// without a browser-like User-Agent.
	UserAgent = "Mozilla/5.0 (X11; Linux aarch64) stock-advisor"

	// acceptHeader asks for JSON but also accepts text/plain: VNDirect dchart
	// serves its JSON as text/plain and answers 406 to a JSON-only Accept.
	acceptHeader = "application/json, text/plain, */*"

	// DefaultTimeout bounds one HTTP request.
	DefaultTimeout = 20 * time.Second
	// DefaultRatePerHost is the request budget per host and second.
	DefaultRatePerHost = 3

	maxBodyBytes = 32 << 20

	// hostDownAfter is how many requests in a row to one host may fail with
	// a transient error (timeout, connection error, 429 or 5xx, after the
	// retries) before the host is skipped for the rest of the run.
	hostDownAfter = 3
)

// DefaultBackoff is the wait before each retry.
var DefaultBackoff = []time.Duration{500 * time.Millisecond, 1500 * time.Millisecond}

// ErrNoData reports that a source answered but had nothing for the request.
var ErrNoData = errors.New("research: no data")

// ErrHostDown reports a request refused without contacting the host,
// because its recent requests kept failing. Callers fall through to the next
// provider at once instead of waiting out the timeouts again.
var ErrHostDown = errors.New("research: host unresponsive, skipped")

// StatusError is returned for a non-2xx response.
type StatusError struct {
	Code int
	URL  string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("research: %s: HTTP %d", e.URL, e.Code)
}

// HTTPClient is the shared HTTP client: it sets the common headers,
// throttles each host and retries transient failures.
type HTTPClient struct {
	client  *http.Client
	rate    int
	backoff []time.Duration

	mu       sync.Mutex
	limiters map[string]*hostLimiter
	failures map[string]int // consecutive transient failures per host
	closed   bool
}

// NewHTTPClient builds a client. A nil transport means DefaultTransport();
// timeout <= 0 means DefaultTimeout; rate <= 0 means DefaultRatePerHost; a
// nil backoff means DefaultBackoff (an empty, non-nil slice disables
// retries).
func NewHTTPClient(transport http.RoundTripper, timeout time.Duration, rate int, backoff []time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if rate <= 0 {
		rate = DefaultRatePerHost
	}
	if backoff == nil {
		backoff = DefaultBackoff
	}
	if transport == nil {
		transport = DefaultTransport()
	}
	return &HTTPClient{
		client:   &http.Client{Transport: transport, Timeout: timeout},
		rate:     rate,
		backoff:  backoff,
		limiters: make(map[string]*hostLimiter),
		failures: make(map[string]int),
	}
}

// tls12Hosts only speak TLS 1.2 and reset the connection when offered Go's
// TLS 1.3 ClientHello, so they get a transport capped at TLS 1.2.
var tls12Hosts = map[string]bool{"dchart-api.vndirect.com.vn": true}

// DefaultTransport returns the production transport: a clone of
// http.DefaultTransport, with TLS capped at 1.2 for the hosts that need it.
func DefaultTransport() http.RoundTripper {
	base := http.DefaultTransport.(*http.Transport).Clone()
	capped := base.Clone()
	capped.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12}
	return &hostRouter{base: base, tls12: capped}
}

type hostRouter struct {
	base, tls12 http.RoundTripper
}

func (h *hostRouter) RoundTrip(r *http.Request) (*http.Response, error) {
	if tls12Hosts[r.URL.Hostname()] {
		return h.tls12.RoundTrip(r)
	}
	return h.base.RoundTrip(r)
}

// Close stops the per-host rate limiters.
func (c *HTTPClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	for _, l := range c.limiters {
		l.stop()
	}
}

// GetJSON fetches rawURL and decodes the JSON body into v.
func (c *HTTPClient) GetJSON(ctx context.Context, rawURL string, header http.Header, v any) error {
	return c.do(ctx, http.MethodGet, rawURL, nil, header, v)
}

// PostJSON sends body as JSON to rawURL and decodes the JSON response into v.
func (c *HTTPClient) PostJSON(ctx context.Context, rawURL string, header http.Header, body, v any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("research: encode request: %w", err)
	}
	return c.do(ctx, http.MethodPost, rawURL, b, header, v)
}

func (c *HTTPClient) do(ctx context.Context, method, rawURL string, body []byte, header http.Header, v any) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("research: parse url: %w", err)
	}
	label := u.Host + u.Path

	for attempt := 0; ; attempt++ {
		if c.hostDown(u.Host) {
			return fmt.Errorf("research: %s: %w", u.Host, ErrHostDown)
		}
		if attempt > 0 {
			if err := sleepCtx(ctx, c.backoff[attempt-1]); err != nil {
				return err
			}
		}
		data, retry, err := c.once(ctx, method, u, body, header, label)
		if err == nil {
			c.recordHost(u.Host, false)
			if len(bytes.TrimSpace(data)) == 0 {
				return fmt.Errorf("%s: %w", label, ErrNoData)
			}
			if err := json.Unmarshal(data, v); err != nil {
				return fmt.Errorf("research: decode %s: %w", label, err)
			}
			return nil
		}
		if ctx.Err() != nil {
			return err
		}
		if !retry || attempt >= len(c.backoff) {
			// A non-retryable answer (4xx) still shows the host is alive.
			c.recordHost(u.Host, retry)
			return err
		}
	}
}

// hostDown reports whether host has failed hostDownAfter requests in a row.
func (c *HTTPClient) hostDown(host string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failures[host] >= hostDownAfter
}

// recordHost counts a transient failure of a whole request to host, or
// resets the count when the host answered.
func (c *HTTPClient) recordHost(host string, failed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if failed {
		c.failures[host]++
	} else if c.failures[host] < hostDownAfter {
		c.failures[host] = 0
	}
}

// once performs one attempt and reports whether a failure is retryable.
func (c *HTTPClient) once(ctx context.Context, method string, u *url.URL, body []byte, header http.Header, label string) ([]byte, bool, error) {
	if err := c.limiter(u.Host).wait(ctx); err != nil {
		return nil, false, err
	}

	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), rd)
	if err != nil {
		return nil, false, fmt.Errorf("research: build request: %w", err)
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", acceptHeader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		return nil, true, fmt.Errorf("research: %s: %w", label, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, true, fmt.Errorf("research: read %s: %w", label, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return nil, retry, &StatusError{Code: resp.StatusCode, URL: label}
	}
	return data, false, nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *HTTPClient) limiter(host string) *hostLimiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, ok := c.limiters[host]
	if !ok {
		l = newHostLimiter(c.rate)
		if c.closed {
			l.stop()
		}
		c.limiters[host] = l
	}
	return l
}

// hostLimiter is a token bucket refilled by a time.Ticker: it holds up to
// rate tokens and gains one every 1/rate seconds.
type hostLimiter struct {
	tokens chan struct{}
	ticker *time.Ticker
	done   chan struct{}
	once   sync.Once
}

func newHostLimiter(rate int) *hostLimiter {
	l := &hostLimiter{
		tokens: make(chan struct{}, rate),
		ticker: time.NewTicker(time.Second / time.Duration(rate)),
		done:   make(chan struct{}),
	}
	for i := 0; i < rate; i++ {
		l.tokens <- struct{}{}
	}
	go func() {
		for {
			select {
			case <-l.ticker.C:
				select {
				case l.tokens <- struct{}{}:
				default:
				}
			case <-l.done:
				return
			}
		}
	}()
	return l
}

func (l *hostLimiter) wait(ctx context.Context) error {
	select {
	case <-l.tokens:
		return nil
	case <-l.done:
		return errors.New("research: http client closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *hostLimiter) stop() {
	l.once.Do(func() {
		l.ticker.Stop()
		close(l.done)
	})
}

// joinURL appends path to a base URL that may or may not end in a slash.
func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + path
}
