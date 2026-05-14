package client

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"
)

type mockRoundTripper struct {
	responses []*http.Response
	index     int
	calls     int
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	m.calls++
	if m.index >= len(m.responses) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

type mockErrRoundTripper struct {
	errors []error
	index  int
	calls  int
}

func (m *mockErrRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	m.calls++
	if m.index >= len(m.errors) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	err := m.errors[m.index]
	m.index++
	return nil, err
}

type timeoutError struct {
	msg string
}

func (e *timeoutError) Error() string   { return e.msg }
func (e *timeoutError) Temporary() bool { return false }
func (e *timeoutError) Timeout() bool   { return true }

func makeResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func fastConfig() RetryTransportConfig {
	return RetryTransportConfig{
		MaxRetries:      3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
	}
}

func TestRetryTransport_NoRetryOnSuccess(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{makeResp(200, "ok")},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls)
	}
}

func TestRetryTransport_NoRetryOnClientError(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"400", 400},
		{"401", 401},
		{"403", 403},
		{"404", 404},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockRoundTripper{
				responses: []*http.Response{makeResp(tc.status, "")},
			}
			rt := NewRetryTransport(mock, fastConfig())

			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
			resp, err := rt.RoundTrip(req)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, resp.StatusCode)
			}
			if mock.calls != 1 {
				t.Fatalf("expected 1 call, got %d", mock.calls)
			}
		})
	}
}

func TestRetryTransport_RetriesOn500(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{
			makeResp(500, "err"),
			makeResp(500, "err"),
			makeResp(200, "ok"),
		},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Fatalf("expected 3 calls, got %d", mock.calls)
	}
}

func TestRetryTransport_ExhaustsRetriesOn500(t *testing.T) {
	cfg := fastConfig()
	// maxRetries=3 means 3 retries after the first attempt = 4 total
	responses := make([]*http.Response, int(cfg.MaxRetries)+2)
	for i := range responses {
		responses[i] = makeResp(503, "err")
	}
	mock := &mockRoundTripper{responses: responses}
	rt := NewRetryTransport(mock, cfg)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 503 {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
	expectedCalls := int(cfg.MaxRetries) + 1
	if mock.calls != expectedCalls {
		t.Fatalf("expected %d calls, got %d", expectedCalls, mock.calls)
	}
}

func TestRetryTransport_RetriesOn429(t *testing.T) {
	mock := &mockRoundTripper{
		responses: []*http.Response{
			makeResp(429, "rate limited"),
			makeResp(200, "ok"),
		},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls)
	}
}

func TestRetryTransport_RespectsRetryAfterHeader(t *testing.T) {
	r429 := makeResp(429, "")
	r429.Header.Set("Retry-After", "0")
	mock := &mockRoundTripper{
		responses: []*http.Response{
			r429,
			makeResp(200, "ok"),
		},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRetryTransport_ResendsBodyOnRetry(t *testing.T) {
	var bodies []string
	mock := &mockRoundTripper{}
	mock.responses = []*http.Response{
		makeResp(500, ""),
		makeResp(200, "ok"),
	}

	captureTransport := &capturingTransport{
		wrapped: mock,
		capture: &bodies,
	}
	rt := NewRetryTransport(captureTransport, fastConfig())

	// Use http.NewRequest which sets GetBody for retryable requests
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("hello"))
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected 2 captured bodies, got %d", len(bodies))
	}
	for i, b := range bodies {
		if b != "hello" {
			t.Fatalf("attempt %d: expected body 'hello', got %q", i, b)
		}
	}
}

func TestRetryTransport_ResendsBodyOnRetryWithoutGetBody(t *testing.T) {
	var bodies []string
	mock := &mockRoundTripper{}
	mock.responses = []*http.Response{
		makeResp(500, ""),
		makeResp(200, "ok"),
	}

	captureTransport := &capturingTransport{
		wrapped: mock,
		capture: &bodies,
	}
	rt := NewRetryTransport(captureTransport, fastConfig())

	// Manually create request without GetBody set (simulates some custom request builders)
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", nil)
	req.Body = io.NopCloser(strings.NewReader("hello"))
	req.GetBody = nil // Ensure GetBody is not set

	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected 2 captured bodies, got %d", len(bodies))
	}
	for i, b := range bodies {
		if b != "hello" {
			t.Fatalf("attempt %d: expected body 'hello', got %q", i, b)
		}
	}
}

func TestRetryTransport_RetriesOnNetworkError(t *testing.T) {
	mock := &mockErrRoundTripper{
		errors: []error{
			&timeoutError{msg: "connection refused"},
			&timeoutError{msg: "connection refused"},
		},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Fatalf("expected 3 calls (2 errors + 1 success), got %d", mock.calls)
	}
}

func TestRetryTransport_ExhaustsRetriesOnNetworkError(t *testing.T) {
	cfg := fastConfig()
	// maxRetries=3 means 3 retries after the first attempt = 4 total
	errs := make([]error, int(cfg.MaxRetries)+2)
	for i := range errs {
		errs[i] = &timeoutError{msg: "connection refused"}
	}
	mock := &mockErrRoundTripper{errors: errs}
	rt := NewRetryTransport(mock, cfg)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)

	if err == nil {
		t.Fatal("expected error got nil")
	}
	expectedCalls := int(cfg.MaxRetries) + 1
	if mock.calls != expectedCalls {
		t.Fatalf("expected %d calls, got %d", expectedCalls, mock.calls)
	}
}

func TestRetryTransport_NoRetryOnNonTransientError(t *testing.T) {
	mock := &mockErrRoundTripper{
		errors: []error{errors.New("permanent failure")},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)

	if err == nil {
		t.Fatal("expected error got nil")
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call (no retries for non-transient error), got %d", mock.calls)
	}
}

func TestRetryTransport_ContextCancellationNotRetried(t *testing.T) {
	mock := &mockErrRoundTripper{
		errors: []error{context.Canceled},
	}
	rt := NewRetryTransport(mock, fastConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled got %v", err)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls)
	}
}

func TestRetryTransport_NoRetryOnTLSError(t *testing.T) {
	// TLS errors are not retryable (not temporary)
	mock := &mockErrRoundTripper{
		errors: []error{&net.OpError{Op: "remote error", Err: errors.New("tls: bad certificate")}},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_, err := rt.RoundTrip(req)

	if err == nil {
		t.Fatal("expected error got nil")
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call (no retries for TLS error), got %d", mock.calls)
	}
}

func TestRetryTransport_SyscallErrorRetry(t *testing.T) {
	mock := &mockErrRoundTripper{
		errors: []error{
			syscall.Errno(syscall.ECONNREFUSED),
			syscall.Errno(syscall.ECONNRESET),
		},
	}
	rt := NewRetryTransport(mock, fastConfig())

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Fatalf("expected 3 calls, got %d", mock.calls)
	}
}

func TestRetryTransport_CloseIdleConnectionsDelegates(t *testing.T) {
	mock := &mockClosableTransport{}
	rt := NewRetryTransport(mock, fastConfig())

	type closeIdler interface{ CloseIdleConnections() }
	if tr, ok := rt.(closeIdler); ok {
		tr.CloseIdleConnections()
	} else {
		t.Fatal("retryTransport does not implement CloseIdleConnections")
	}

	if !mock.closeCalled {
		t.Fatal("expected CloseIdleConnections to be called on wrapped transport")
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"", 0},
		{"0", 0},
		{"5", 5 * time.Second},
		{"120", 120 * time.Second},
		{"invalid", 0},
	}
	for _, tc := range tests {
		got := parseRetryAfter(tc.input)
		if got != tc.expected {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	// Use a date in the future
	future := time.Now().UTC().Add(5 * time.Second)
	got := parseRetryAfter(future.Format(http.TimeFormat))
	// Allow 1 second tolerance due to parsing overhead
	if got < 4*time.Second || got > 6*time.Second {
		t.Errorf("parseRetryAfter(date) = %v, want ~5s", got)
	}

	// Past date should return 0
	past := time.Now().UTC().Add(-5 * time.Second)
	got = parseRetryAfter(past.Format(http.TimeFormat))
	if got != 0 {
		t.Errorf("parseRetryAfter(past date) = %v, want 0", got)
	}
}

type capturingTransport struct {
	wrapped http.RoundTripper
	capture *[]string
}

func (c *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		*c.capture = append(*c.capture, string(b))
		req.Body = io.NopCloser(strings.NewReader(string(b)))
	}
	return c.wrapped.RoundTrip(req)
}

type mockClosableTransport struct {
	mockRoundTripper
	closeCalled bool
}

func (m *mockClosableTransport) CloseIdleConnections() {
	m.closeCalled = true
}
