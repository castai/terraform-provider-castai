package castai

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestEKSSettingsInstanceProfileRoleArns(t *testing.T) {
	read := func(region string, roleARNs []interface{}) (*schema.ResourceData, error) {
		raw := map[string]interface{}{
			EKSSettingsFieldAccountId: "123456789012",
			EKSSettingsFieldRegion:    region,
			EKSSettingsFieldVpc:       "vpc-1",
			EKSSettingsFieldCluster:   "cluster",
		}
		if roleARNs != nil {
			raw[EKSSettingsFieldInstanceProfileRoleArns] = roleARNs
		}
		data := schema.TestResourceDataRaw(t, dataSourceEKSSettings().Schema, raw)
		diags := dataSourceCastaiEKSSettingsRead(context.Background(), data, nil)
		if diags.HasError() {
			return nil, &diagError{diags[0].Summary}
		}
		return data, nil
	}

	t.Run("defaults to wildcard", func(t *testing.T) {
		data, err := read("eu-west-1", nil)
		require.NoError(t, err)
		require.Contains(t, data.Get(EKSSettingsFieldIamPolicyJson), `"Resource": "arn:aws:iam::*:role/*"`)
	})

	t.Run("limits pass role resources", func(t *testing.T) {
		data, err := read("eu-west-1", []interface{}{"arn:aws:iam::123456789012:role/node-role"})
		require.NoError(t, err)
		policy := data.Get(EKSSettingsFieldIamPolicyJson).(string)
		require.Equal(t, 2, strings.Count(policy, `"Resource": "arn:aws:iam::123456789012:role/node-role"`))
		require.NotContains(t, policy, "role/*")
	})

	t.Run("ignores duplicate ARNs", func(t *testing.T) {
		data, err := read("eu-west-1", []interface{}{
			"arn:aws:iam::123456789012:role/node-role",
			"arn:aws:iam::123456789012:role/node-role",
		})
		require.NoError(t, err)
		policy := data.Get(EKSSettingsFieldIamPolicyJson).(string)
		require.Equal(t, 2, strings.Count(policy, `"Resource": "arn:aws:iam::123456789012:role/node-role"`))
	})

	t.Run("renders the same policy regardless of ARN order", func(t *testing.T) {
		roleA := "arn:aws:iam::123456789012:role/a-role"
		roleB := "arn:aws:iam::123456789012:role/b-role"

		first, err := read("eu-west-1", []interface{}{roleA, roleB})
		require.NoError(t, err)
		second, err := read("eu-west-1", []interface{}{roleB, roleA})
		require.NoError(t, err)

		require.Equal(t, first.Get(EKSSettingsFieldIamPolicyJson), second.Get(EKSSettingsFieldIamPolicyJson))
	})

	t.Run("rejects invalid ARN", func(t *testing.T) {
		_, err := read("eu-west-1", []interface{}{"not-an-arn"})
		require.ErrorContains(t, err, "not a valid IAM role ARN")
	})

	t.Run("rejects ARN from another account", func(t *testing.T) {
		_, err := read("eu-west-1", []interface{}{"arn:aws:iam::210987654321:role/node-role"})
		require.ErrorContains(t, err, "does not belong to account")
	})

	t.Run("rejects ARN from another partition", func(t *testing.T) {
		_, err := read("us-gov-west-1", []interface{}{"arn:aws:iam::123456789012:role/node-role"})
		require.ErrorContains(t, err, "does not belong to partition")
	})
}

type diagError struct{ msg string }

func (e *diagError) Error() string { return e.msg }
