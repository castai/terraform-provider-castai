package castai

import (
	"bytes"
	"context"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"io"
	"net/http"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNodeTemplatesResource_NodeTemplatesUpdateAction(t *testing.T) {
	currentNodeTemplates := `
		{
		  "items": [
			{
			  "template": {
				"configurationId": "7dc4f922-29c9-4377-889c-0c8c5fb8d497",
				"configurationName": "default",
				"name": "gpu",
				"constraints": {
				  "spot": false,
				  "useSpotFallbacks": false,
				  "fallbackRestoreRateSeconds": 0,
				  "storageOptimized": false,
				  "computeOptimized": false,
				  "instanceFamilies": {
					"include": [],
					"exclude": [
					  "p4d",
					  "p3dn",
					  "p2",
					  "g3s",
					  "g5g",
					  "g5",
					  "g3"
					]
				  },
				  "gpu": {
					"manufacturers": [
					  "NVIDIA"
					],
					"includeNames": [],
					"excludeNames": []
				  }
				},
				"version": "3",
				"shouldTaint": false,
				"rebalancingConfig": {
				  "minNodes": 0
				}
			  },
			  "stats": {
				"countOnDemand": 0,
				"countSpot": 0,
				"countFallback": 0
			  }
			}
		  ]
		}
	`

	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceNodeTemplate()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldClusterId:                   cty.StringVal(clusterId),
		FieldNodeTemplateName:            cty.StringVal("gpu"),
		FieldNodeTemplateConfigurationId: cty.StringVal("7dc4f922-29c9-4377-889c-0c8c5fb8d497"),
		FieldNodeTemplateShouldTaint:     cty.BoolVal(true),
	})

	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)
	body := io.NopCloser(bytes.NewReader([]byte(currentNodeTemplates)))
	response := &http.Response{StatusCode: 200, Body: body}
	nodeTemplatesUpdated := false

	mockClient.EXPECT().NodeTemplatesAPIListNodeTemplates(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().NodeTemplatesAPIUpdateNodeTemplateWithBody(gomock.Any(), clusterId, "gpu", "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {
			got, _ := io.ReadAll(body)
			expected := []byte(currentNodeTemplates)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			nodeTemplatesUpdated = true

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			}, nil
		}).Times(1)

	result := resource.UpdateContext(ctx, data, provider)
	r.Nil(result)
	r.True(nodeTemplatesUpdated)
}
