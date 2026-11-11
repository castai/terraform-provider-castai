package policies

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPolicies(t *testing.T) {
	t.Run("IAM policy", func(t *testing.T) {
		iamPolicy, err := GetIAMPolicy("testaccount", "testpartition", nil)
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

	t.Run("IAM policy without pass role ARNs is unchanged", func(t *testing.T) {
		want, err := os.ReadFile("testdata/iam-policy-default.golden.json")
		if err != nil {
			t.Fatalf("reading golden file: %v", err)
		}

		for _, roleARNs := range [][]string{nil, {}} {
			got, err := GetIAMPolicy("123456789012", "aws", roleARNs)
			if err != nil {
				t.Fatalf("couldn't generate IAM policy: %v", err)
			}
			if got != string(want) {
				t.Fatalf("IAM policy differs from golden file:\n%s", got)
			}
		}
	})

	t.Run("IAM policy with pass role ARNs", func(t *testing.T) {
		const (
			roleA = "arn:aws:iam::123456789012:role/node-role"
			roleB = "arn:aws:iam::123456789012:role/path/to/other-role"
		)

		tests := []struct {
			name string
			arns []string
			want any
		}{
			{name: "one ARN", arns: []string{roleA}, want: roleA},
			{name: "multiple ARNs", arns: []string{roleA, roleB}, want: []any{roleA, roleB}},
		}

		var defaultPolicy map[string]any
		def, _ := os.ReadFile("testdata/iam-policy-default.golden.json")
		if err := json.Unmarshal(def, &defaultPolicy); err != nil {
			t.Fatalf("parsing golden file: %v", err)
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := GetIAMPolicy("123456789012", "aws", tt.arns)
				if err != nil {
					t.Fatalf("couldn't generate IAM policy: %v", err)
				}

				var policy map[string]any
				if err := json.Unmarshal([]byte(got), &policy); err != nil {
					t.Fatalf("generated IAM policy is not valid JSON: %v", err)
				}

				statements := policy["Statement"].([]any)
				defaultStatements := defaultPolicy["Statement"].([]any)
				if len(statements) != len(defaultStatements) {
					t.Fatalf("expected %d statements, got %d", len(defaultStatements), len(statements))
				}

				for i, raw := range statements {
					statement := raw.(map[string]any)
					expected := defaultStatements[i].(map[string]any)

					if sid := statement["Sid"]; sid == "PassRoleEC2" || sid == "PassRoleEKS" {
						if !reflect.DeepEqual(statement["Resource"], tt.want) {
							t.Fatalf("%v: expected Resource %v, got %v", sid, tt.want, statement["Resource"])
						}
						// Everything but the resource must stay the same.
						delete(statement, "Resource")
						expected = copyWithout(expected, "Resource")
					}

					if !reflect.DeepEqual(statement, expected) {
						t.Fatalf("statement %v changed: %v", statement["Sid"], statement)
					}
				}
			})
		}
	})

	t.Run("IAM policy with invalid pass role ARNs", func(t *testing.T) {
		tests := map[string]string{
			"not an ARN":            "node-role",
			"wrong service":         "arn:aws:s3:::bucket",
			"not a role":            "arn:aws:iam::123456789012:user/bob",
			"bad account":           "arn:aws:iam::1234:role/node-role",
			"empty name":            "arn:aws:iam::123456789012:role/",
			"trailing slash":        "arn:aws:iam::123456789012:role/path/",
			"wildcard":              "arn:aws:iam::123456789012:role/*",
			"quote injection":       `arn:aws:iam::123456789012:role/a"b`,
			"unsupported partition": "arn:aws-cn:iam::123456789012:role/node-role",
			"unknown partition":     "arn:foo:iam::123456789012:role/node-role",
			"other account":         "arn:aws:iam::210987654321:role/node-role",
		}

		for name, arn := range tests {
			t.Run(name, func(t *testing.T) {
				if _, err := GetIAMPolicy("123456789012", "aws", []string{"arn:aws:iam::123456789012:role/ok", arn}); err == nil {
					t.Fatalf("expected error for %q", arn)
				}
			})
		}
	})

	t.Run("IAM policy pass role ARNs must match partition", func(t *testing.T) {
		if _, err := GetIAMPolicy("123456789012", "aws-us-gov", []string{"arn:aws-us-gov:iam::123456789012:role/node-role"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := GetIAMPolicy("123456789012", "aws-us-gov", []string{"arn:aws:iam::123456789012:role/node-role"}); err == nil {
			t.Fatalf("expected partition mismatch error")
		}
	})
}

func copyWithout(m map[string]any, key string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if k != key {
			out[k] = v
		}
	}
	return out
}
