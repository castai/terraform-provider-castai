package client

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

func GetHttpClient(apiToken, userAgent string) (*http.Client, []func(ctx context.Context, req *http.Request) error) {
	transport := NewRetryTransport(
		logging.NewSubsystemLoggingHTTPTransport("CAST.AI", http.DefaultTransport),
		DefaultRetryTransportConfig(),
	)
	client := &http.Client{
		Transport: transport,
		Timeout:   1 * time.Minute,
	}
	requestEditors := []func(ctx context.Context, req *http.Request) error{
		func(_ context.Context, req *http.Request) error {
			req.Header.Set("user-agent", userAgent)
			return nil
		},
		func(ctx context.Context, req *http.Request) error {
			req.Header.Set("X-API-Key", apiToken)
			return nil
		},
	}

	return client, requestEditors
}
