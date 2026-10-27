package castai

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestRoleDataSource_Found(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		Return(rolesTestResponse(t, rolesTestPage(
			rolesTestRole("owner-id", "Owner", false)+","+rolesTestRole("viewer-id", "Viewer", false),
			"",
		)), nil)

	data, _ := rolesTestRead(t, dataSourceRole(), rolesTestState(t, "Viewer", "ALL"), mockClient)

	r.Equal("viewer-id", data.Id())
	r.Equal("description of Viewer", data.Get("description"))
	r.Equal(true, data.Get("default"))
	r.Equal(false, data.Get("deprecated"))
	r.Equal(10, data.Get("level"))

	permissions, ok := data.Get("permissions").([]interface{})
	r.True(ok, "permissions should be a list")
	r.Len(permissions, 1)
	permission, ok := permissions[0].(map[string]any)
	r.True(ok, "permission should be a map")
	r.Equal("EFFECT_ALLOW", permission["effect"])
	r.Equal([]interface{}{"/v1/organizations"}, permission["endpoints"])
	r.Equal([]interface{}{"rbac"}, permission["groups"])
}

func TestRoleDataSource_NotFoundErrors(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		Return(rolesTestResponse(t, rolesTestPage(rolesTestRole("owner-id", "Owner", false), "")), nil)

	resource := dataSourceRole()
	data := resource.Data(rolesTestState(t, "Missing", "ALL"))
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	r.True(result.HasError())
	r.Contains(result[0].Summary, `no ALL role with name "Missing" found`)
}

func TestRoleDataSource_DuplicateNameErrors(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		Return(rolesTestResponse(t, rolesTestPage(
			rolesTestRole("default-operator-id", "Operator", false)+","+rolesTestRole("custom-operator-id", "Operator", false),
			"",
		)), nil)

	resource := dataSourceRole()
	data := resource.Data(rolesTestState(t, "Operator", "ALL"))
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	r.True(result.HasError())
	r.Contains(result[0].Summary, `expected a single role with name "Operator"`)
}
