package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGetHttpClientRetriesGatewayErrors(t *testing.T) {
	tests := []struct {
		name         string
		giveStatuses []int
		giveMethod   string
		wantStatus   int
		wantCalls    int
	}{
		{
			name:         "success on first attempt",
			giveStatuses: []int{http.StatusOK},
			wantStatus:   http.StatusOK,
			wantCalls:    1,
		},
		{
			name:         "retries 502 then succeeds",
			giveStatuses: []int{http.StatusBadGateway, http.StatusOK},
			wantStatus:   http.StatusOK,
			wantCalls:    2,
		},
		{
			name:         "replays request body on retry",
			giveStatuses: []int{http.StatusBadGateway, http.StatusOK},
			giveMethod:   http.MethodPost,
			wantStatus:   http.StatusOK,
			wantCalls:    2,
		},
		{
			name:         "does not retry 404",
			giveStatuses: []int{http.StatusNotFound},
			wantStatus:   http.StatusNotFound,
			wantCalls:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := int(calls.Add(1))
				status := tt.giveStatuses[len(tt.giveStatuses)-1]
				if n <= len(tt.giveStatuses) {
					status = tt.giveStatuses[n-1]
				}
				w.WriteHeader(status)
			}))
			defer server.Close()

			client, editors := GetHttpClient("test-token", "test-agent/1.0")

			method := tt.giveMethod
			if method == "" {
				method = http.MethodGet
			}
			var body io.Reader
			if method == http.MethodPost {
				body = strings.NewReader(`{"test":true}`)
			}
			req, err := http.NewRequestWithContext(context.Background(), method, server.URL, body)
			if err != nil {
				t.Fatal(err)
			}
			for _, editor := range editors {
				if err := editor(context.Background(), req); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if got := int(calls.Load()); got != tt.wantCalls {
				t.Errorf("server calls = %d, want %d", got, tt.wantCalls)
			}
		})
	}
}
