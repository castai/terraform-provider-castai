package castai

import (
	"testing"
)

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("internal consistency validation failed: %v", err)
	}
}
