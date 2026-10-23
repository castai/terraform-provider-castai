package client

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

func GetHttpClient(apiToken, userAgent string) (*http.Client, []func(ctx context.Context, req *http.Request) error) {
	apiToken = strings.TrimSpace(apiToken)

	// The API occasionally returns transient gateway errors from its
	// proxy, so requests are retried for up to ~15s instead of failing
	// the terraform operation outright.
	retry := retryablehttp.NewClient()
	retry.RetryWaitMin = 1 * time.Second
	retry.RetryWaitMax = 15 * time.Second
	retry.RetryMax = 4

	client := retry.StandardClient()
	client.Timeout = 1 * time.Minute
	client.Transport = logging.NewSubsystemLoggingHTTPTransport("CAST.AI", client.Transport)

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
