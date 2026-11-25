package castai

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

func TestToEdgeClusterSpecForMode(t *testing.T) {
	t.Parallel()
	r := &edgeLocationResource{}
	controlPlane := &controlPlaneModel{}

	tests := []struct {
		name     string
		mode     omni.EdgeLocationControlPlaneMode
		wantSpec bool
	}{
		{name: "shared keeps edge cluster spec", mode: omni.EdgeLocationControlPlaneModeSHARED, wantSpec: true},
		{name: "dedicated keeps edge cluster spec", mode: omni.EdgeLocationControlPlaneModeDEDICATED, wantSpec: true},
		{name: "native skips edge cluster spec", mode: omni.EdgeLocationControlPlaneModeNATIVE, wantSpec: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, diags := r.toEdgeClusterSpecForMode(context.Background(), string(tt.mode), controlPlane, nil, nil, nil)
			require.Empty(t, diags.Errors())
			require.Equal(t, tt.wantSpec, spec != nil)
		})
	}
}

func TestNativeEdgeClusterWarning(t *testing.T) {
	t.Parallel()
	r := &edgeLocationResource{}

	diags := r.nativeEdgeClusterWarning(edgeLocationModel{
		ControlPlaneMode: types.StringValue(string(omni.EdgeLocationControlPlaneModeNATIVE)),
		ControlPlane:     &controlPlaneModel{},
	})
	require.Empty(t, diags.Errors())
	require.Len(t, diags.Warnings(), 1)

	diags = r.nativeEdgeClusterWarning(edgeLocationModel{
		ControlPlaneMode: types.StringValue(string(omni.EdgeLocationControlPlaneModeSHARED)),
		ControlPlane:     &controlPlaneModel{},
	})
	require.Empty(t, diags.Errors())
	require.Empty(t, diags.Warnings())
}
