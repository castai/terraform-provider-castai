package client

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"
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
		req.Body.Close()
		getBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	bo := t.newBackoff()

	var (
		resp    *http.Response
		attempt int
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

		var err error
		resp, err = t.wrapped.RoundTrip(cloned)
		if err != nil {
			return nil, err
		}

		attempt++

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

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	secs, err := strconv.Atoi(header)
	if err != nil {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// drainBody reads any remaining data from the response body and closes it.
// This is necessary to allow the underlying TCP connection to be reused.
// See: https://golang.org/pkg/net/http/#Response
func drainBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
