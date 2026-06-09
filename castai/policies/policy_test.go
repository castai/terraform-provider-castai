package policies

import (
	"strings"
	"testing"
)

func TestPolicies(t *testing.T) {
	t.Run("IAM policy", func(t *testing.T) {
		iamPolicy, err := GetIAMPolicy("testaccount", "testpartition")
		if err != nil {
			t.Fatalf("couldn't generate IAM policy")
		}

		resource := "arn:testpartition:ec2:*:testaccount:security-group/*"

		if !strings.Contains(iamPolicy, resource) {
			t.Fatalf("generated IAM policy does not contain required resource")
		}

		if strings.Contains(iamPolicy, ".AccountNumber") || strings.Contains(iamPolicy, ".Partition") {
			t.Fatalf("Incorrectly formatted template")
		}
	})

	t.Run("User policy", func(t *testing.T) {
		userpolicy, err := GetUserInlinePolicy("clustername", "testarn", "testvpc", "testpartition", "")
		if err != nil || userpolicy == "" {
			t.Fatalf("couldn't generate user policy")
		}

		vpcResource := "arn:testpartition:ec2:testarn:vpc/testvpc"

		if !strings.Contains(userpolicy, vpcResource) {
			t.Fatalf("generated User policy does not contain required resource")
		}

		if strings.Contains(userpolicy, ".ARN") || strings.Contains(userpolicy, ".Partition") {
			t.Fatalf("Incorrectly formatted template")
		}
	})

	t.Run("User policy with shared VPC account", func(t *testing.T) {
		userpolicy, err := GetUserInlinePolicy("clustername", "testarn", "testvpc", "testpartition", "sharedvpcarn")
		if err != nil || userpolicy == "" {
			t.Fatalf("couldn't generate user policy")
		}

		vpcResource := "arn:testpartition:ec2:sharedvpcarn:vpc/testvpc"
		subnetResource := "arn:testpartition:ec2:sharedvpcarn:subnet/*"
		instanceResource := "arn:testpartition:ec2:testarn:instance/*"

		if !strings.Contains(userpolicy, vpcResource) {
			t.Fatalf("generated User policy does not contain shared VPC resource")
		}

		if !strings.Contains(userpolicy, subnetResource) {
			t.Fatalf("generated User policy does not contain shared subnet resource")
		}

		if !strings.Contains(userpolicy, instanceResource) {
			t.Fatalf("generated User policy should still use main ARN for instance resources")
		}
	})

	t.Run("Managed policies", func(t *testing.T) {
		managedPolicies := GetManagedPolicies("testpartition")

		resource := "arn:testpartition:iam::aws:policy/"

		for _, policy := range managedPolicies {
			if !strings.HasPrefix(policy, resource) {
				t.Fatalf("Generated managed policies do not contain required resource")
			}
		}
	})
}
