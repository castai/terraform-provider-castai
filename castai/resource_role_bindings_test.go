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

func TestRoleBindingsReadContext(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

	ctx := context.Background()
	provider := &ProviderConfig{
		api: &sdk.ClientWithResponses{
			ClientInterface: mockClient,
		},
	}

	organizationID := "4e4cd9eb-82eb-407e-a926-e5fef81cab50"
	roleBindingID := "a83b7bf2-5a99-45d9-bcac-b969386e751f"
	roleID := "4df39779-dfb2-48d3-91d8-7ee5bd2bca4b"
	userID := "671b2ebb-f361-42f0-aa2f-3049de93f8c1"
	serviceAccountID := "b11f5945-22ca-4101-a86e-d6e37f44a415"
	groupID := "844d2bf2-870d-42da-a81c-4e19befc78fc"

	body := io.NopCloser(bytes.NewReader([]byte(`{
  "id": "` + roleBindingID + `",
  "organizationId": "` + organizationID + `",
  "name": "role-binding-name",
  "description": "role-binding-description",
  "definition": {
    "roleId": "` + roleID + `",
    "scope": {
      "organization": {
        "id": "` + organizationID + `"
      } 
    },
    "subjects": [
      {
        "user": {
          "id": "` + userID + `"
        }
      },
      {
        "serviceAccount": {
          "id": "` + serviceAccountID + `"
		}
	  },
	  {
        "group": {
          "id": "` + groupID + `"
        }
      }
    ]
  }
}`)))

	mockClient.EXPECT().
		RbacServiceAPIGetRoleBinding(gomock.Any(), organizationID, roleBindingID).
		Return(&http.Response{StatusCode: 200, Body: body, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

	stateValue := cty.ObjectVal(map[string]cty.Value{
		"organization_id": cty.StringVal(organizationID),
	})
	state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
	state.ID = roleBindingID

	resource := resourceRoleBindings()
	data := resource.Data(state)

	result := resource.ReadContext(ctx, data, provider)

	r.Nil(result)
	r.False(result.HasError())
	r.Equal(`ID = `+roleBindingID+`
description = role-binding-description
name = role-binding-name
organization_id = `+organizationID+`
role_id = `+roleID+`
scope.# = 1
scope.0.kind = organization
scope.0.resource_id = `+organizationID+`
subjects.# = 1
subjects.0.subject.# = 3
subjects.0.subject.0.group_id = 
subjects.0.subject.0.kind = user
subjects.0.subject.0.service_account_id = 
subjects.0.subject.0.user_id = `+userID+`
subjects.0.subject.1.group_id = 
subjects.0.subject.1.kind = service_account
subjects.0.subject.1.service_account_id = `+serviceAccountID+`
subjects.0.subject.1.user_id = 
subjects.0.subject.2.group_id = `+groupID+`
subjects.0.subject.2.kind = group
subjects.0.subject.2.service_account_id = 
subjects.0.subject.2.user_id = 
Tainted = false
`, data.State().String())
}
