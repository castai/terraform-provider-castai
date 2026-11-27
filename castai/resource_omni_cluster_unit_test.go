package castai

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

func TestToRegisteredClusterStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		status        *omniClusterStatusModel
		expectedMode  *omni.RegisteredClusterStatusProvisioningMode
		expectedAgent *string
	}{
		{
			name: "legacy agent",
			status: &omniClusterStatusModel{
				OmniAgentVersion: types.StringValue("1.19.16"),
				PodCIDR:          types.StringValue("10.244.0.0/16"),
			},
			expectedAgent: stringPointer("1.19.16"),
		},
		{
			name: "explicit agent",
			status: &omniClusterStatusModel{
				ProvisioningMode: types.StringValue("VIRTUALNODE"),
				OmniAgentVersion: types.StringValue("1.19.16"),
				PodCIDR:          types.StringValue("10.244.0.0/16"),
			},
			expectedMode:  provisioningModePointer(omni.RegisteredClusterStatusProvisioningMode("VIRTUALNODE")),
			expectedAgent: stringPointer("1.19.16"),
		},
		{
			name: "omni connect",
			status: &omniClusterStatusModel{
				ProvisioningMode: types.StringValue("NATIVE"),
				OmniAgentVersion: types.StringNull(),
				PodCIDR:          types.StringValue("10.244.0.0/16"),
			},
			expectedMode: provisioningModePointer(omni.RegisteredClusterStatusProvisioningMode("NATIVE")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toRegisteredClusterStatus(tt.status)
			require.Equal(t, tt.expectedMode, got.ProvisioningMode)
			require.Equal(t, tt.expectedAgent, got.OmniAgentVersion)
			require.Equal(t, "10.244.0.0/16", got.PodCidr)
		})
	}
}

func TestToRegisteredClusterStatusWithoutStatus(t *testing.T) {
	t.Parallel()
	require.Nil(t, toRegisteredClusterStatus(nil))
}

func TestValidateOmniClusterStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         *omniClusterStatusModel
		expectedErrors int
	}{
		{name: "no status", expectedErrors: 0},
		{name: "legacy agent", status: &omniClusterStatusModel{OmniAgentVersion: types.StringValue("1.19.16")}, expectedErrors: 0},
		{name: "native without agent version", status: &omniClusterStatusModel{
			ProvisioningMode: types.StringValue("NATIVE"),
			OmniAgentVersion: types.StringNull(),
		}, expectedErrors: 0},
		{name: "virtual node without version", status: &omniClusterStatusModel{
			ProvisioningMode: types.StringValue("VIRTUALNODE"),
			OmniAgentVersion: types.StringNull(),
		}, expectedErrors: 1},
		{name: "legacy empty version", status: &omniClusterStatusModel{
			OmniAgentVersion: types.StringValue(""),
		}, expectedErrors: 1},
		{name: "native with agent version", status: &omniClusterStatusModel{
			ProvisioningMode: types.StringValue("NATIVE"),
			OmniAgentVersion: types.StringValue("1.19.16"),
		}, expectedErrors: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateOmniClusterStatus(tt.status)
			require.Len(t, diags.Errors(), tt.expectedErrors)
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func provisioningModePointer(value omni.RegisteredClusterStatusProvisioningMode) *omni.RegisteredClusterStatusProvisioningMode {
	return &value
}
