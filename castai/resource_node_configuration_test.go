package castai

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func Test_resourceNodeConfigurationRead(t *testing.T) {
	t.Run("flattern to node configs", func(t *testing.T) {
		result := flattenEKSConfig(&sdk.NodeconfigV1EKSConfig{
			TargetGroup: &sdk.NodeconfigV1TargetGroup{
				Arn:  toPtr("arn:aws:iam::aws:policy/AdministratorAccess"),
				Port: toPtr(int32(80)),
			},
		})

		got := toEKSConfig(result[0])
		require.Equal(t, got, &sdk.NodeconfigV1EKSConfig{
			TargetGroup: &sdk.NodeconfigV1TargetGroup{
				Arn:  toPtr("arn:aws:iam::aws:policy/AdministratorAccess"),
				Port: toPtr(int32(80)),
			},
		})
	})
}

func Test_resourceNodeConfigurationCreate(t *testing.T) {
	type args struct {
		tuneMock func(m *mock_sdk.MockClientInterface)
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "success",
			args: args{
				tuneMock: func(m *mock_sdk.MockClientInterface) {
					m.EXPECT().NodeConfigurationAPICreateConfiguration(gomock.Any(), gomock.Any(), gomock.Any()).
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
    "subnet-0ede99883f8d65813",
    "subnet-0f7cb7d2702533af0",
    "subnet-0beea7ca69ceb165b"
  ],
  "tags": {
    "k8s.io/cluster/valentyna-0326-1": "owned"
  },
  "eks": {
    "securityGroups": [
      "sg-04dffb0bec1821a92",
      "sg-008614a8aad956a53",
      "sg-031d074b817c04773",
      "sg-053e9a1980987985a",
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
			_ = resourceNodeConfigurationCreate(context.Background(), data, provider)
		})
	}
}
