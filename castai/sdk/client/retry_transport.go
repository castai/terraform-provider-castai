package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	defaultMaxRetries      = 3
	defaultInitialInterval = 100 * time.Millisecond
	defaultMaxInterval     = 30 * time.Second
)

type RetryTransportConfig struct {
	MaxRetries      uint64
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

func DefaultRetryTransportConfig() RetryTransportConfig {
	return RetryTransportConfig{
		MaxRetries:      defaultMaxRetries,
		InitialInterval: defaultInitialInterval,
		MaxInterval:     defaultMaxInterval,
	}
}

type retryTransport struct {
	wrapped http.RoundTripper
	cfg     RetryTransportConfig
}

func NewRetryTransport(wrapped http.RoundTripper, cfg RetryTransportConfig) http.RoundTripper {
	return &retryTransport{
		wrapped: wrapped,
		cfg:     cfg,
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// GetBody is set by the standard library for retryable requests (e.g., from NewRequest).
	// If not available, we need to buffer the body ourselves to allow retries.
	var getBody func() (io.ReadCloser, error)
	if req.GetBody != nil {
		getBody = req.GetBody
	} else if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
		getBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	bo := t.newBackoff()

	var (
		resp    *http.Response
		attempt int
		lastErr error
	)

	for {
		cloned := req.Clone(req.Context())
		if getBody != nil {
			body, err := getBody()
			if err != nil {
				return nil, err
			}
			cloned.Body = body
		}

		resp, lastErr = t.wrapped.RoundTrip(cloned)
		attempt++

		if lastErr != nil {
			if !isRetryableError(lastErr) {
				return nil, lastErr
			}
			wait := bo.NextBackOff()
			if wait == backoff.Stop {
				log.Printf("[WARN] Exhausted retries for error %v after %d attempts", lastErr, attempt)
				return nil, lastErr
			}
			log.Printf("[WARN] Received transient error %v, retrying in %s (attempt %d)", lastErr, wait, attempt)
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(wait):
			}
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			wait := bo.NextBackOff()
			if wait == backoff.Stop {
				log.Printf("[WARN] Exhausted retries for status %d after %d attempts", resp.StatusCode, attempt)
				return resp, nil
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After")); retryAfter > 0 {
					wait = retryAfter
				}
			}
			log.Printf("[WARN] Received status %d, retrying in %s (attempt %d)", resp.StatusCode, wait, attempt)
			drainBody(resp)
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(wait):
			}
			continue
		}

		return resp, nil
	}
}

func (t *retryTransport) newBackoff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = t.cfg.InitialInterval
	b.MaxInterval = t.cfg.MaxInterval
	b.Reset()
	return backoff.WithMaxRetries(b, t.cfg.MaxRetries)
}

func (t *retryTransport) CloseIdleConnections() {
	if tr, ok := t.wrapped.(interface{ CloseIdleConnections() }); ok {
		tr.CloseIdleConnections()
	}
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	secs, err := strconv.Atoi(header)
	if err == nil {
		return time.Duration(secs) * time.Second
	}
	t, err := http.ParseTime(header)
	if err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) {
		return syscallErr == syscall.ECONNREFUSED || syscallErr == syscall.ETIMEDOUT || syscallErr == syscall.ECONNRESET
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		return isRetryableError(urlErr.Err)
	}

	return false
}

// drainBody reads any remaining data from the response body and closes it.
// This is necessary to allow the underlying TCP connection to be reused.
// See: https://golang.org/pkg/net/http/#Response
func drainBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}
