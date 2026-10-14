package castai

import (
	"bytes"
	"context"
	"fmt"
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

const rolesTestOrganizationID = "d09389f9-6a95-46a1-9ded-5aa76e7cbfa5"

func rolesTestState(t *testing.T, name, roleType string) *terraform.InstanceState {
	t.Helper()

	return terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		FieldRolesOrganizationID: cty.StringVal(rolesTestOrganizationID),
		FieldRolesName:           cty.StringVal(name),
		FieldRolesType:           cty.StringVal(roleType),
	}), 0)
}

func rolesTestResponse(t *testing.T, body string) *http.Response {
	t.Helper()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     map[string][]string{"Content-Type": {"json"}},
	}
}

func rolesTestPage(roles string, nextPageCursor string) string {
	nextPage := ""
	if nextPageCursor != "" {
		nextPage = fmt.Sprintf(`,"nextPage":{"cursor":%q,"limit":"100"}`, nextPageCursor)
	}
	return fmt.Sprintf(`{"roles":[%s]%s}`, roles, nextPage)
}

func rolesTestRole(id, name string, deprecated bool) string {
	return fmt.Sprintf(`{
		"id": %q,
		"name": %q,
		"description": "description of %s",
		"default": true,
		"deprecated": %t,
		"level": 10,
		"definition": {
			"permissions": [
				{"effect": "EFFECT_ALLOW", "endpoints": ["/v1/organizations"], "groups": ["rbac"]}
			]
		}
	}`, id, name, name, deprecated)
}

func rolesTestRead(t *testing.T, state *terraform.InstanceState, mockClient *mock_sdk.MockClientInterface) *schema.ResourceData {
	t.Helper()

	resource := dataSourceRoles()
	data := resource.Data(state)
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	require.False(t, result.HasError(), "unexpected diagnostics: %v", result)

	return data
}

func rolesFromState(t *testing.T, data *schema.ResourceData) []map[string]any {
	t.Helper()

	raw, ok := data.Get(FieldRolesRoles).([]interface{})
	require.True(t, ok, "roles should be a list")

	roles := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		role, ok := item.(map[string]any)
		require.True(t, ok, "role should be a map")
		roles = append(roles, role)
	}
	return roles
}

func TestRolesDataSource_NameMatchReturnsRole(t *testing.T) {
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

	data := rolesTestRead(t, rolesTestState(t, "Viewer", "ALL"), mockClient)
	roles := rolesFromState(t, data)

	r.Len(roles, 1)
	r.Equal("viewer-id", roles[0]["id"])
	r.Equal("Viewer", roles[0]["name"])
	r.Equal("description of Viewer", roles[0]["description"])
	r.Equal(true, roles[0]["default"])
	r.Equal(false, roles[0]["deprecated"])
	r.Equal(10, roles[0]["level"])

	permissions, ok := roles[0]["permissions"].([]interface{})
	r.True(ok, "permissions should be a list")
	r.Len(permissions, 1)
	permission, ok := permissions[0].(map[string]any)
	r.True(ok, "permission should be a map")
	r.Equal("EFFECT_ALLOW", permission["effect"])
	r.Equal([]interface{}{"/v1/organizations"}, permission["endpoints"])
	r.Equal([]interface{}{"rbac"}, permission["groups"])

	r.Equal(rolesTestOrganizationID, data.Id())
}

func TestRolesDataSource_NameCaseMismatchReturnsEmpty(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		Return(rolesTestResponse(t, rolesTestPage(rolesTestRole("viewer-id", "Viewer", false), "")), nil)

	data := rolesTestRead(t, rolesTestState(t, "viewer", "ALL"), mockClient)

	r.Empty(rolesFromState(t, data))
}

func TestRolesDataSource_TwoRolesWithSameNameBothReturned(t *testing.T) {
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

	data := rolesTestRead(t, rolesTestState(t, "Operator", "ALL"), mockClient)
	roles := rolesFromState(t, data)

	r.Len(roles, 2)
	ids := []string{roles[0]["id"].(string), roles[1]["id"].(string)}
	r.ElementsMatch([]string{"default-operator-id", "custom-operator-id"}, ids)
}

func TestRolesDataSource_TypeReachesParams(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedParams *sdk.RbacServiceAPIListRolesParams
	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListRolesParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedParams = params
			return rolesTestResponse(t, rolesTestPage(rolesTestRole("viewer-id", "Viewer", false), "")), nil
		})

	rolesTestRead(t, rolesTestState(t, "Viewer", "DEFAULT"), mockClient)

	r.NotNil(capturedParams)
	r.NotNil(capturedParams.Type)
	r.Equal(sdk.RbacServiceAPIListRolesParamsTypeDEFAULT, *capturedParams.Type)
}

func TestRolesDataSource_CursorLoop(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedCursors []*string
	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListRolesParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedCursors = append(capturedCursors, params.PageCursor)
			return rolesTestResponse(t, rolesTestPage(rolesTestRole("role-page1", "Owner", false), "cursor-page2")), nil
		})
	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListRolesParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedCursors = append(capturedCursors, params.PageCursor)
			return rolesTestResponse(t, rolesTestPage(rolesTestRole("role-page2", "Viewer", false), "")), nil
		})

	data := rolesTestRead(t, rolesTestState(t, "", "ALL"), mockClient)
	roles := rolesFromState(t, data)

	r.Len(roles, 2)
	r.Equal("role-page1", roles[0]["id"])
	r.Equal("role-page2", roles[1]["id"])

	r.Len(capturedCursors, 2)
	r.Nil(capturedCursors[0], "first request should not carry a cursor")
	r.NotNil(capturedCursors[1])
	r.Equal("cursor-page2", *capturedCursors[1])
}

func TestRolesDataSource_RepeatingCursorStopsWithoutDuplicates(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListRolesParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return rolesTestResponse(t, rolesTestPage(
				rolesTestRole("role-1", "Owner", false)+","+rolesTestRole("role-2", "Viewer", false),
				"0",
			)), nil
		}).
		Times(2)

	data := rolesTestRead(t, rolesTestState(t, "", "ALL"), mockClient)
	roles := rolesFromState(t, data)

	r.Len(roles, 2)
	r.Equal("role-1", roles[0]["id"])
	r.Equal("role-2", roles[1]["id"])
}

func TestRolesDataSource_TooManyPages(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	// Every page advertises a next cursor and returns a fresh role, so the only
	// thing that can stop the loop is the page cap. Exactly rolesListMaxPages
	// calls may happen; a 501st call would fail the gomock expectation.
	page := 0
	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListRolesParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			page++
			return rolesTestResponse(t, rolesTestPage(
				rolesTestRole(fmt.Sprintf("role-%d", page), fmt.Sprintf("Role %d", page), false),
				"cursor",
			)), nil
		}).
		Times(rolesListMaxPages)

	resource := dataSourceRoles()
	data := resource.Data(rolesTestState(t, "", "ALL"))
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	r.True(result.HasError())
	r.Contains(result[0].Summary, fmt.Sprintf("exceeded maximum of %d pages", rolesListMaxPages))
}

func TestRolesDataSource_APIError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListRoles(gomock.Any(), rolesTestOrganizationID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"message": "Internal server error"}`))),
			Header:     map[string][]string{"Content-Type": {"json"}},
		}, nil)

	resource := dataSourceRoles()
	data := resource.Data(rolesTestState(t, "", "ALL"))
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	r.True(result.HasError())
	r.Contains(result[0].Summary, "listing roles")
}
