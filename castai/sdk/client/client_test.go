package client

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestGetHttpClientTrimsAPITokenWhitespace(t *testing.T) {
	tests := []struct {
		give string
		want string
	}{
		{give: "token", want: "token"},
		{give: "token\n", want: "token"},
		{give: "token\r\n", want: "token"},
		{give: " token ", want: "token"},
		{give: "\t\ntoken\n\t", want: "token"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("trim_%q", tt.give), func(t *testing.T) {
			_, editors := GetHttpClient(tt.give, "test-agent")

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.cast.ai", nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, editor := range editors {
				if err := editor(context.Background(), req); err != nil {
					t.Fatal(err)
				}
			}

			if got := req.Header.Get("X-Api-Key"); got != tt.want {
				t.Fatalf("X-Api-Key header = %q, want %q", got, tt.want)
			}
		})
	}
}
