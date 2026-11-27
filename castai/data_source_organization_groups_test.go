package castai

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

const groupsTestOrganizationID = "d09389f9-6a95-46a1-9ded-5aa76e7cbfa5"

func groupsTestSingularState(t *testing.T, name string) *terraform.InstanceState {
	t.Helper()

	return terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		FieldOrgGroupOrganizationID: cty.StringVal(groupsTestOrganizationID),
		FieldOrgGroupName:           cty.StringVal(name),
	}), 0)
}

func groupsTestPluralState(t *testing.T, managedBy []string) *terraform.InstanceState {
	t.Helper()

	managedByValues := make([]cty.Value, 0, len(managedBy))
	for _, v := range managedBy {
		managedByValues = append(managedByValues, cty.StringVal(v))
	}
	managedByList := cty.ListValEmpty(cty.String)
	if len(managedByValues) > 0 {
		managedByList = cty.ListVal(managedByValues)
	}

	return terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		FieldOrgGroupsOrganizationID: cty.StringVal(groupsTestOrganizationID),
		FieldOrgGroupsManagedBy:      managedByList,
	}), 0)
}

func groupsTestResponse(t *testing.T, groups string, nextPageCursor string) *http.Response {
	t.Helper()

	nextPage := ""
	if nextPageCursor != "" {
		nextPage = fmt.Sprintf(`,"nextPage":{"cursor":%q,"limit":"100"}`, nextPageCursor)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"groups":[%s]%s}`, groups, nextPage)))),
		Header:     map[string][]string{"Content-Type": {"json"}},
	}
}

func groupsTestGroup(id, name string, extra ...string) string {
	description := fmt.Sprintf("description of %s", name)
	managedBy := "console"
	ssoProvider := ""
	if len(extra) > 0 {
		description = extra[0]
	}
	if len(extra) > 1 {
		managedBy = extra[1]
	}
	if len(extra) > 2 {
		ssoProvider = extra[2]
	}
	return fmt.Sprintf(`{
		"id": %q,
		"name": %q,
		"description": %q,
		"managedBy": %q,
		"ssoProvider": %q
	}`, id, name, description, managedBy, ssoProvider)
}

func groupsTestRead(t *testing.T, resource *schema.Resource, state *terraform.InstanceState, mockClient *mock_sdk.MockClientInterface) (*schema.ResourceData, diag.Diagnostics) {
	t.Helper()

	data := resource.Data(state)
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	return data, result
}

func groupsTestFromState(t *testing.T, data *schema.ResourceData) []map[string]any {
	t.Helper()

	raw, ok := data.Get(FieldOrgGroupsGroups).([]interface{})
	require.True(t, ok, "groups should be a list")

	groups := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		group, ok := item.(map[string]any)
		require.True(t, ok, "group should be a map")
		groups = append(groups, group)
	}
	return groups
}

func TestOrganizationGroupsDataSource_NoFilterReturnsAll(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return groupsTestResponse(t, strings.Join([]string{
				groupsTestGroup("group-a", "platform-engineers", "", "idp-sync", "okta"),
				groupsTestGroup("group-b", "tf-managed", "", "terraform", ""),
				groupsTestGroup("group-c", "legacy", "", "", ""),
			}, ","), ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroups(), groupsTestPluralState(t, nil), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	groups := groupsTestFromState(t, data)
	r.Len(groups, 3)
	r.Equal("group-a", groups[0]["id"])
	r.Equal("idp-sync", groups[0]["managed_by"])
	r.Equal("okta", groups[0]["sso_provider"])
	r.Equal("terraform", groups[1]["managed_by"])
	// Legacy groups have an empty managed_by.
	r.Equal("", groups[2]["managed_by"])
}

func TestOrganizationGroupsDataSource_NoResultsIsEmptyNotError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return groupsTestResponse(t, "", ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroups(), groupsTestPluralState(t, []string{"sso-default"}), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	groups := groupsTestFromState(t, data)
	r.Empty(groups)
}

func TestOrganizationGroupsDataSource_ManagedByReachesParams(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedParams *sdk.RbacServiceAPIListGroupsParams
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedParams = params
			return groupsTestResponse(t, groupsTestGroup("group-a", "platform-engineers", "", "idp-sync", "okta"), ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroups(), groupsTestPluralState(t, []string{"idp-sync", "console"}), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	r.NotNil(capturedParams)
	r.NotNil(capturedParams.ManagedBy)
	r.Equal([]string{"idp-sync", "console"}, *capturedParams.ManagedBy)

	groups := groupsTestFromState(t, data)
	r.Len(groups, 1)
}

func TestOrganizationGroupsDataSource_NoManagedBySendsNoFilter(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedParams *sdk.RbacServiceAPIListGroupsParams
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedParams = params
			return groupsTestResponse(t, "", ""), nil
		})

	_, result := groupsTestRead(t, dataSourceOrganizationGroups(), groupsTestPluralState(t, nil), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	r.NotNil(capturedParams)
	r.Nil(capturedParams.ManagedBy, "no managedBy filter should be sent when managed_by is unset")
}

func TestOrganizationGroupsDataSource_CursorPagination(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedCursors []*string
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedCursors = append(capturedCursors, params.PageCursor)
			return groupsTestResponse(t, groupsTestGroup("group-page1", "platform-engineers"), "cursor-page2"), nil
		})
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedCursors = append(capturedCursors, params.PageCursor)
			return groupsTestResponse(t, groupsTestGroup("group-page2", "data-engineers"), ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroups(), groupsTestPluralState(t, nil), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	groups := groupsTestFromState(t, data)
	r.Len(groups, 2)
	r.Equal("group-page1", groups[0]["id"])
	r.Equal("group-page2", groups[1]["id"])

	r.Len(capturedCursors, 2)
	r.Nil(capturedCursors[0], "first request should not carry a cursor")
	r.NotNil(capturedCursors[1])
	r.Equal("cursor-page2", *capturedCursors[1])
}

// TestOrganizationGroupsDataSource_RepeatedCursorTerminates covers a
// misbehaving server that keeps returning the same next cursor: the
// seen-cursors guard must stop the loop without duplicates.
func TestOrganizationGroupsDataSource_RepeatedCursorTerminates(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	page := 0
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			page++
			// Fresh response per call: the parser consumes and closes the body.
			return groupsTestResponse(t, groupsTestGroup("group-1", "platform-engineers"), "cursor-1"), nil
		}).
		Times(2) // initial request + one retry with the repeated cursor

	data, result := groupsTestRead(t, dataSourceOrganizationGroups(), groupsTestPluralState(t, nil), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	groups := groupsTestFromState(t, data)
	r.Len(groups, 1)
}

// TestOrganizationGroupsDataSource_OverlappingPagesNotDropped covers a server
// that returns a page with only already-seen groups while a next cursor
// remains: pagination must continue instead of stopping early.
func TestOrganizationGroupsDataSource_OverlappingPagesNotDropped(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedCursors []*string
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedCursors = append(capturedCursors, params.PageCursor)
			return groupsTestResponse(t, groupsTestGroup("group-1", "platform-engineers"), "cursor-page2"), nil
		})
	// Second page repeats group-1 and adds group-2; the loop must not stop
	// just because nothing new was added so far.
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedCursors = append(capturedCursors, params.PageCursor)
			return groupsTestResponse(t, strings.Join([]string{
				groupsTestGroup("group-1", "platform-engineers"),
				groupsTestGroup("group-2", "data-engineers"),
			}, ","), "cursor-page3"), nil
		})
	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedCursors = append(capturedCursors, params.PageCursor)
			return groupsTestResponse(t, groupsTestGroup("group-3", "sre"), ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroups(), groupsTestPluralState(t, nil), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	groups := groupsTestFromState(t, data)
	r.Len(groups, 3)
	r.Equal("group-3", groups[2]["id"])
	r.Len(capturedCursors, 3)
}

func TestOrganizationGroupsDataSource_APIError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"message": "Internal server error"}`))),
			Header:     map[string][]string{"Content-Type": {"json"}},
		}, nil)

	resource := dataSourceOrganizationGroups()
	data := resource.Data(groupsTestPluralState(t, nil))
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	r.True(result.HasError())
	r.Contains(result[0].Summary, "listing groups")
}

// TestOrganizationGroupsDataSource_TransportError covers a request that fails
// before a response exists: the mock returns a nil response together with an
// error, and the read must surface the error without panicking.
func TestOrganizationGroupsDataSource_TransportError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		Return(nil, errors.New("connection reset"))

	resource := dataSourceOrganizationGroups()
	data := resource.Data(groupsTestPluralState(t, nil))
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	r.True(result.HasError())
	r.Contains(result[0].Summary, "connection reset")
}
