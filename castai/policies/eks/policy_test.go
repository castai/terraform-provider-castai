package eks

import (
	"strings"
	"testing"
)

func TestPolicies(t *testing.T) {
	t.Run("IAM policy", func(t *testing.T) {
		iamPolicy, err := GetIAMPolicy("testaccount")
		if err != nil {
			t.Fatalf("couldn't generate IAM policy")
		}

		resource := "arn:aws:ec2:*:testaccount:security-group/*"

		if !strings.Contains(iamPolicy, resource) {
			t.Fatalf("generated IAM policy does not contain required resource")
		}

		if strings.Contains(iamPolicy, ".AccountNumber") {
			t.Fatalf("Incorrectly formatted template")
		}
	})

	t.Run("User policy", func(t *testing.T) {
		userpolicy, err := GetUserInlinePolicy("clustername", "testarn", "testvpc")
		if err != nil || userpolicy == "" {
			t.Fatalf("couldn't generate user policy")
		}

		vpcResource := "arn:aws:ec2:testarn:vpc/testvpc"

		if !strings.Contains(userpolicy, vpcResource) {
			t.Fatalf("generated User policy does not contain required resource")
		}

		if strings.Contains(userpolicy, ".ARN") {
			t.Fatalf("Incorrectly formatted template")
		}
	})
}
