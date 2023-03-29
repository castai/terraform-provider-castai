package castai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestAutoscalerResource_PoliciesUpdateAction(t *testing.T) {
	armTenantId := os.Getenv("ARM_TENANT_ID")
	fmt.Println("plain", armTenantId)
	fmt.Println("base64", base64.StdEncoding.EncodeToString([]byte(armTenantId)))
	fmt.Println("hex", hex.EncodeToString([]byte(armTenantId)))

	currentPolicies := `
		{
		    "enabled": true,
		    "isScopedMode": false,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 32,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": false
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "spotInstances": {
		        "enabled": true,
		        "clouds": [
		            "azure"
		        ],
		        "maxReclaimRate": 0,
		        "spotBackups": {
		            "enabled": false,
		            "spotBackupRestoreRateSeconds": 1800
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`

	// 1. enable scope mode
	// 2. enable node constraints and change max CPU
	// 3. enable spot backups
	// 4. change spot cloud to aws - just to test if we can do change on arrays
	policyChanges := `{
		"isScopedMode":true,
		"unschedulablePods": {
			"nodeConstraints": {
				"enabled": true,
				"maxCpuCores": 96
			}
		},
		"spotInstances": {
			"clouds": ["aws"],
			"spotBackups": {
				"enabled": true
			}
		}
	}`

	updatedPolicies := `
		{
		    "enabled": true,
		    "isScopedMode": true,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 96,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": true
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "spotInstances": {
		        "enabled": true,
		        "clouds": [
		            "aws"
		        ],
		        "maxReclaimRate": 0,
		        "spotBackups": {
		            "enabled": true,
		            "spotBackupRestoreRateSeconds": 1800
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`

	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceAutoscaler()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldAutoscalerPoliciesJSON: cty.StringVal(policyChanges),
		FieldClusterId:              cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	body := io.NopCloser(bytes.NewReader([]byte(currentPolicies)))
	response := &http.Response{StatusCode: 200, Body: body}

	policiesUpdated := false

	mockClient.EXPECT().PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().PoliciesAPIUpsertClusterPoliciesWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {
			got, _ := io.ReadAll(body)
			expected := []byte(updatedPolicies)

			eq, err := JSONBytesEqual(got, expected)
			r.NoError(err)
			r.True(eq, fmt.Sprintf("got:      %v\n"+
				"expected: %v\n", string(got), string(expected)))

			policiesUpdated = true

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			}, nil
		}).Times(1)

	result := resource.UpdateContext(ctx, data, provider)
	r.Nil(result)
	r.True(policiesUpdated)
}

func TestAutoscalerResource_PoliciesUpdateAction_Fail(t *testing.T) {
	currentPolicies := `
		{
		    "enabled": true,
		    "isScopedMode": false,
		    "unschedulablePods": {
		        "enabled": true,
		        "headroom": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "headroomSpot": {
		            "cpuPercentage": 10,
		            "memoryPercentage": 10,
		            "enabled": true
		        },
		        "nodeConstraints": {
		            "minCpuCores": 2,
		            "maxCpuCores": 32,
		            "minRamMib": 4096,
		            "maxRamMib": 262144,
		            "enabled": false
		        },
		        "diskGibToCpuRatio": 25
		    },
		    "clusterLimits": {
		        "enabled": false,
		        "cpu": {
		            "minCores": 1,
		            "maxCores": 20
		        }
		    },
		    "spotInstances": {
		        "enabled": true,
		        "clouds": [
		            "azure"
		        ],
		        "maxReclaimRate": 0,
		        "spotBackups": {
		            "enabled": false,
		            "spotBackupRestoreRateSeconds": 1800
		        }
		    },
		    "nodeDownscaler": {
		        "emptyNodes": {
		            "enabled": false,
		            "delaySeconds": 0
		        }
		    }
		}`

	policyChanges := `{
		"isScopedMode":true,
		"unschedulablePods": {
			"nodeConstraints": {
				"enabled": true,
				"maxCpuCores": 96
			}
		},
		"spotInstances": {
			"clouds": ["aws"],
			"spotBackups": {
				"enabled": true
			}
		}
	}`

	r := require.New(t)
	mockctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(mockctrl)

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	resource := resourceAutoscaler()

	clusterId := "cluster_id"
	val := cty.ObjectVal(map[string]cty.Value{
		FieldAutoscalerPoliciesJSON: cty.StringVal(policyChanges),
		FieldClusterId:              cty.StringVal(clusterId),
	})
	state := terraform.NewInstanceStateShimmedFromValue(val, 0)
	data := resource.Data(state)

	body := io.NopCloser(bytes.NewReader([]byte(currentPolicies)))
	response := &http.Response{StatusCode: 200, Body: body}

	mockClient.EXPECT().PoliciesAPIGetClusterPolicies(gomock.Any(), clusterId, gomock.Any()).Return(response, nil).Times(1)
	mockClient.EXPECT().PoliciesAPIUpsertClusterPoliciesWithBody(gomock.Any(), clusterId, "application/json", gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterId string, contentType string, body io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"message":"policies config: Evictor policy management is not allowed: Evictor installed externally. Uninstall Evictor first and try again.","fieldViolations":[]`))),
			}, nil
		}).Times(1)

	result := resource.UpdateContext(ctx, data, provider)
	r.NotNil(result)
	r.True(result.HasError())
	r.Equal(`expected status code 200, received: status=400 body={"message":"policies config: Evictor policy management is not allowed: Evictor installed externally. Uninstall Evictor first and try again.","fieldViolations":[]`, result[0].Summary)
}

func JSONBytesEqual(a, b []byte) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(a, &j); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}
