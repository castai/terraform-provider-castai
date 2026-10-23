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
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

const rolesTestOrganizationID = "d09389f9-6a95-46a1-9ded-5aa76e7cbfa5"

func rolesTestState(t *testing.T, name, roleType string) *terraform.InstanceState {
	t.Helper()

	attrs := map[string]cty.Value{
		FieldRolesOrganizationID: cty.StringVal(rolesTestOrganizationID),
		FieldRolesType:           cty.StringVal(roleType),
	}
	if name != "" {
		attrs[FieldRolesName] = cty.StringVal(name)
	}
	return terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(attrs), 0)
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

// rolesTestRead reads either roles data source (castai_roles or castai_role)
// against the mocked API and fails the test on unexpected diagnostics.
func rolesTestRead(t *testing.T, resource *schema.Resource, state *terraform.InstanceState, mockClient *mock_sdk.MockClientInterface) (*schema.ResourceData, diag.Diagnostics) {
	t.Helper()

	data := resource.Data(state)
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})
	require.False(t, result.HasError(), "unexpected diagnostics: %v", result)

	return data, result
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

	rolesTestRead(t, dataSourceRoles(), rolesTestState(t, "", "DEFAULT"), mockClient)

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

	data, _ := rolesTestRead(t, dataSourceRoles(), rolesTestState(t, "", "ALL"), mockClient)
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

	data, _ := rolesTestRead(t, dataSourceRoles(), rolesTestState(t, "", "ALL"), mockClient)
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
