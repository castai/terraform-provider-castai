package castai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestDataSourceWorkloadScalingPoliciesRead_APIError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	}

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"

	mockClient.EXPECT().
		WorkloadOptimizationAPIListWorkloadScalingPolicies(ctx, clusterID).
		Return(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"internal error"}`))),
		}, nil)

	ds := dataSourceWorkloadScalingPolicies()
	state := sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
	}), 0)
	data := ds.Data(state)

	diags := dataSourceWorkloadScalingPoliciesRead(ctx, data, provider)
	r.True(diags.HasError())
}

func TestDataSourceWorkloadScalingPoliciesRead(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	}

	clusterID := "b6bfc074-a267-400f-b8f1-db0850c36gf4"

	listResponse := sdk.WorkloadoptimizationV1ListWorkloadScalingPoliciesResponse{
		Items: []sdk.WorkloadoptimizationV1WorkloadScalingPolicy{
			{Id: "policy-1-id", Name: "readonly", IsReadonly: true, IsCastware: true, IsDefault: false},
			{Id: "policy-2-id", Name: "balanced", IsReadonly: false, IsCastware: false, IsDefault: true},
			{Id: "policy-3-id", Name: "custom", IsReadonly: false, IsCastware: false, IsDefault: false},
		},
	}
	body, _ := json.Marshal(listResponse)
	mockClient.EXPECT().
		WorkloadOptimizationAPIListWorkloadScalingPolicies(ctx, clusterID).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil)

	ds := dataSourceWorkloadScalingPolicies()
	state := sdkterraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"cluster_id": cty.StringVal(clusterID),
	}), 0)
	data := ds.Data(state)

	diags := dataSourceWorkloadScalingPoliciesRead(ctx, data, provider)
	r.Empty(diags)
	r.Equal(clusterID, data.Id())

	r.Equal("policy-1-id", data.Get("policies.0.id"))
	r.Equal("readonly", data.Get("policies.0.name"))
	r.Equal(true, data.Get("policies.0.is_readonly"))
	r.Equal(true, data.Get("policies.0.is_castware"))
	r.Equal(false, data.Get("policies.0.is_default"))

	r.Equal("policy-2-id", data.Get("policies.1.id"))
	r.Equal("balanced", data.Get("policies.1.name"))
	r.Equal(false, data.Get("policies.1.is_readonly"))
	r.Equal(false, data.Get("policies.1.is_castware"))
	r.Equal(true, data.Get("policies.1.is_default"))

	r.Equal("policy-3-id", data.Get("policies.2.id"))
	r.Equal("custom", data.Get("policies.2.name"))
}
