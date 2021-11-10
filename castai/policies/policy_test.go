package policies

import (
"testing"
)

func TestPolicies(t *testing.T) {
	t.Run("IAM policy", func(t *testing.T) {
		_, err := GetIAMPolicy("testaccount")
		if err != nil {
			t.Fatalf("couldn't generate IAM policy")
		}
	})

	t.Run("User policy", func(t *testing.T) {
		userpolicy, err := GetUserInlinePolicy("clustername", "arn", "vpc")
		if err != nil || userpolicy == "" {
			t.Fatalf("couldn't generate user policy")
		}
	})
}
