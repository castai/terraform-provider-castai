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

func TestOrganizationGroupReadContext(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	organizationID := "b6bfc024-a267-400f-b8f1-db0850c369b2"
	groupID := "e9a3f787-15d4-4850-ae7c-b4864809aa54"

	body := io.NopCloser(bytes.NewReader([]byte(`{	
		"id": "e9a3f787-15d4-4850-ae7c-b4864809aa54",
		"organizationId": "b6bfc024-a267-400f-b8f1-db0850c369b2",
		"name": "test group",
		"description": "test group description",
		"definition": {
			"members": [
				{
					"id": "5d832285-c263-4d27-9ba5-7d8cf5759782",
					"email": "test-user-1@test.com"
				},
				{
					"id": "5d832285-c263-4d27-9ba5-7d8cf5759783",
					"email": "test-user-2@test.com"
				}
			]
		}
	}`)))

	mockClient.EXPECT().
		RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	stateValue := cty.ObjectVal(map[string]cty.Value{
		"organization_id": cty.StringVal(organizationID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	state.ID = groupID

	resource := resourceOrganizationGroup()
	data := resource.Data(state)

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = e9a3f787-15d4-4850-ae7c-b4864809aa54
description = test group description
members.# = 1
members.0.member.# = 2
members.0.member.0.email = test-user-1@test.com
members.0.member.0.id = 5d832285-c263-4d27-9ba5-7d8cf5759782
members.0.member.0.kind = 
members.0.member.1.email = test-user-2@test.com
members.0.member.1.id = 5d832285-c263-4d27-9ba5-7d8cf5759783
members.0.member.1.kind = 
name = test group
organization_id = b6bfc024-a267-400f-b8f1-db0850c369b2
Tainted = false
`, data.State().String())
}

func TestOrganizationGroupReadContext_WithoutOrganizationInDefinition(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	organizationID := "b6bfc024-a267-400f-b8f1-db0850c369b4"
	groupID := "e9a3f787-15d4-4850-ae7c-b4864809aa55"

	organizationsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "organizations": [
    {
      "id": "b6bfc024-a267-400f-b8f1-db0850c369b4",
      "name": "Test 1",
      "createdAt": "2023-04-18T16:03:18.800099Z",
      "role": "owner"
    }
  ]
}`)))

	mockClient.EXPECT().
		UsersAPIListOrganizations(gomock.Any(), gomock.Any()).
		Return(&http.Response{StatusCode: 200, Body: organizationsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	body := io.NopCloser(bytes.NewReader([]byte(`{
		"id": "e9a3f787-15d4-4850-ae7c-b4864809aa55",
		"organizationId": "b6bfc024-a267-400f-b8f1-db0850c369b4",
		"name": "test group",
		"description": "test group description",
		"definition": {
			"members": [
				{
					"id": "5d832285-c263-4d27-9ba5-7d8cf5759782",
					"email": "test-user-1@test.com"
				},
				{
					"id": "5d832285-c263-4d27-9ba5-7d8cf5759783",
					"email": "test-user-2@test.com"
				}
			]
		}
	}`)))

	mockClient.EXPECT().
		RbacServiceAPIGetGroup(gomock.Any(), organizationID, groupID).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	stateValue := cty.ObjectVal(map[string]cty.Value{})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	state.ID = groupID

	resource := resourceOrganizationGroup()
	data := resource.Data(state)

	result := resource.ReadContext(ctx, data, provider)
	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = e9a3f787-15d4-4850-ae7c-b4864809aa55
description = test group description
members.# = 1
members.0.member.# = 2
members.0.member.0.email = test-user-1@test.com
members.0.member.0.id = 5d832285-c263-4d27-9ba5-7d8cf5759782
members.0.member.0.kind = 
members.0.member.1.email = test-user-2@test.com
members.0.member.1.id = 5d832285-c263-4d27-9ba5-7d8cf5759783
members.0.member.1.kind = 
name = test group
organization_id = b6bfc024-a267-400f-b8f1-db0850c369b4
Tainted = false
`, data.State().String())
}
