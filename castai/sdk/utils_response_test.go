package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_IsCredentialsError(t *testing.T) {
	t.Run("credentials error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		body := `{
	"message": "Forbidden",
	"fieldViolations":
	[{"field": "credentials", "description": ""}]
}`
		resp := ExternalClusterAPIReconcileClusterResponse{
			Body: []byte(body),
		}
		result := IsCredentialsError(resp)
		r.True(result)
	})
	t.Run("not credentials error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		body := `{
	"message": "something",
	"fieldViolations":
	[{"field": "credentials", "description": ""}]
}`
		resp := ExternalClusterAPIReconcileClusterResponse{
			Body: []byte(body),
		}
		result := IsCredentialsError(resp)
		r.False(result)
	})
	t.Run("no fields in message", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)

		body := `{
	"message": "something",
	"fieldViolations":[]
}`
		resp := ExternalClusterAPIReconcileClusterResponse{
			Body: []byte(body),
		}
		result := IsCredentialsError(resp)
		r.False(result)
	})
}
