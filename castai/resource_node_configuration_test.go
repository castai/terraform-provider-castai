package castai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func Test_resourceNodeConfigurationCreate(t *testing.T) {
	type args struct {
		tuneMock func(m *mock_sdk.MockClientInterface)
		dataSet  map[string]interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "success",
			args: args{
				dataSet: map[string]interface{}{
					"eks": []map[string]interface{}{
						{
							FieldNodeConfigurationEKSTargetGroup: []interface{}{
								map[string]interface{}{
									"arn": "test",
								},
								map[string]interface{}{
									"arn":  "test2",
									"port": 80,
								},
							},
						},
					},
				},
				tuneMock: func(m *mock_sdk.MockClientInterface) {
					m.EXPECT().NodeConfigurationAPICreateConfiguration(gomock.Any(), gomock.Any(), sdk.NodeconfigV1NewNodeConfiguration{
						Eks: &sdk.NodeconfigV1EKSConfig{
							TargetGroups: &[]sdk.NodeconfigV1TargetGroup{
								{
									Arn:  toPtr("test"),
									Port: nil,
								},
								{
									Arn:  toPtr("test2"),
									Port: toPtr(int32(80)),
								},
							},
							ImdsHopLimit: toPtr(int32(0)),
							ImdsV1:       toPtr(false),
						},
						DiskCpuRatio:    toPtr(int32(0)),
						DrainTimeoutSec: toPtr(int32(0)),
						MinDiskSize:     toPtr(int32(0)),
					}).
						Return(
							&http.Response{
								StatusCode: 200,
								Header:     map[string][]string{"Content-Type": {"json"}},
								Body:       io.NopCloser(bytes.NewReader([]byte(`{"id": "id-1"}`))),
							}, nil)
					m.EXPECT().NodeConfigurationAPIGetConfiguration(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(
							&http.Response{
								StatusCode: 200,
								Header:     map[string][]string{"Content-Type": {"json"}},
								Body: io.NopCloser(bytes.NewReader([]byte(`{
  "id": "765fdc7b-2577-4ae8-a6b8-e3b60afbc33a",
  "name": "test4",
  "version": 1,
  "createdAt": "2024-03-26T12:23:57.501529Z",
  "updatedAt": "2024-03-26T12:23:57.503428Z",
  "default": false,
  "diskCpuRatio": 0,
  "subnets": [
    "subnet-0beea7ca69ceb165b"
  ],
  "tags": {
    "k8s.io/cluster/valentyna-0326-1": "owned"
  },
  "eks": {
    "securityGroups": [
      "sg-04dffb0bec1821a92",
      "sg-084e6537aff751bd6"
    ],
    "instanceProfileArn": "arn:aws:iam::028075177508:instance-profile/cast-valentyna-0326-1-eks-0626f5c2",
    "imdsV1": true,
    "imdsHopLimit": 2,
    "targetGroup": {
      "arn": "test"
    }
  },
  "subnetDetails": [
    {
      "id": "subnet-0ede99883f8d65813",
      "cidr": "10.0.1.0/24",
      "zone": {
        "id": "euc1-az2",
        "name": "eu-central-1a"
      },
      "tags": {}
    },
    {
      "id": "subnet-0f7cb7d2702533af0",
      "cidr": "10.0.2.0/24",
      "zone": {
        "id": "euc1-az3",
        "name": "eu-central-1b"
      },
      "tags": {}
    },
    {
      "id": "subnet-0beea7ca69ceb165b",
      "cidr": "10.0.3.0/24",
      "zone": {
        "id": "euc1-az1",
        "name": "eu-central-1c"
      },
      "tags": {}
    }
  ],
  "minDiskSize": 100
}
`))),
							}, nil)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
			if tt.args.tuneMock != nil {
				tt.args.tuneMock(mockClient)
			}
			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			data := resourceNodeConfiguration().Data(terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0))
			for k, v := range tt.args.dataSet {
				require.NoError(t, data.Set(k, v))
			}

			_ = resourceNodeConfigurationCreate(context.Background(), data, provider)
		})
	}
}

func Test_NodeConfiguration_UpdateContext(t *testing.T) {
	type args struct {
		tuneMock func(m *mock_sdk.MockClientInterface)
		updated  *sdk.NodeconfigV1EKSConfig
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "success",
			args: args{
				updated: &sdk.NodeconfigV1EKSConfig{
					TargetGroups: &[]sdk.NodeconfigV1TargetGroup{
						{
							Arn:  toPtr("test2"),
							Port: toPtr(int32(80)),
						},
						{
							Arn: toPtr("test"),
						},
					},
				},
				tuneMock: func(m *mock_sdk.MockClientInterface) {
					m.EXPECT().NodeConfigurationAPIUpdateConfiguration(gomock.Any(), "",
						"765fdc7b-2577-4ae8-a6b8-e3b60afbc33a",
						sdk.NodeconfigV1NodeConfigurationUpdate{
							Eks: &sdk.NodeconfigV1EKSConfig{
								TargetGroups: &[]sdk.NodeconfigV1TargetGroup{
									{
										Arn:  toPtr("test2"),
										Port: toPtr(int32(80)),
									},
									{
										Arn: toPtr("test"),
									},
								},
								ImdsHopLimit: toPtr(int32(0)),
								ImdsV1:       toPtr(false),
							},
							DiskCpuRatio:    toPtr(int32(0)),
							DrainTimeoutSec: toPtr(int32(0)),
							MinDiskSize:     toPtr(int32(100)),
						}).
						Return(
							&http.Response{
								StatusCode: 200,
								Header:     map[string][]string{"Content-Type": {"json"}},
								Body: io.NopCloser(bytes.NewReader([]byte(`{
						  "id": "765fdc7b-2577-4ae8-a6b8-e3b60afbc33a",
						  "name": "test4"
						}
						`))),
							}, nil)

					m.EXPECT().NodeConfigurationAPIGetConfiguration(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(
							&http.Response{
								StatusCode: 200,
								Header:     map[string][]string{"Content-Type": {"json"}},
								Body: io.NopCloser(bytes.NewReader([]byte(`{
						  "id": "765fdc7b-2577-4ae8-a6b8-e3b60afbc33a",
						  "name": "test4",
						  "tags": {},
						  "eks": {
							"targetGroups": [
							{
							  "arn": "test2",
							  "port": 80
							}
							]
						  }
						}
						`))),
							}, nil)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))
			if tt.args.tuneMock != nil {
				tt.args.tuneMock(mockClient)
			}
			provider := &ProviderConfig{
				api: &sdk.ClientWithResponses{
					ClientInterface: mockClient,
				},
			}

			resource := resourceNodeConfiguration()
			raw := make(map[string]interface{})
			data := schema.TestResourceDataRaw(t, resource.Schema, raw)
			require.NoError(t, data.Set("eks", flattenEKSConfig(tt.args.updated)))
			data.SetId("765fdc7b-2577-4ae8-a6b8-e3b60afbc33a")
			updateResult := resource.UpdateContext(context.Background(), data, provider)
			require.Nil(t, updateResult)
			require.False(t, updateResult.HasError())
		})
	}
}
