package castai

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestOrganizationGroupDataSource_MatchingNameReturnsGroup(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return groupsTestResponse(t, strings.Join([]string{
				groupsTestGroup("group-a", "platform-engineers", "Platform team", "idp-sync", "okta"),
				groupsTestGroup("group-b", "other-team", "", "console", ""),
			}, ","), ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroup(), groupsTestSingularState(t, "platform-engineers"), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	// The resolved id must be exposed through the SDK-managed resource id:
	// schema declares no top-level "id" attribute, d.SetId is the single
	// source of truth for data.castai_organization_group.<alias>.id.
	r.Equal("group-a", data.Id())
	state := data.State()
	r.NotNil(state)
	r.Equal("group-a", state.ID)
	r.Equal("group-a", state.Attributes["id"])

	r.Equal("platform-engineers", data.Get(FieldOrgGroupName).(string))
	r.Equal("Platform team", data.Get("description").(string))
	r.Equal("idp-sync", data.Get("managed_by").(string))
	r.Equal("okta", data.Get("sso_provider").(string))
}

func TestOrganizationGroupDataSource_NoMatchReturnsNotFound(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return groupsTestResponse(t, groupsTestGroup("group-b", "other-team"), ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroup(), groupsTestSingularState(t, "platform-engineers"), mockClient)
	r.True(result.HasError())
	r.Contains(result[0].Summary, `no group with name "platform-engineers" found`)
	r.Empty(data.Id(), "no id should be set when the group is not found")
}

func TestOrganizationGroupDataSource_MoreThanOneMatchReturnsError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		RbacServiceAPIListGroups(gomock.Any(), groupsTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.RbacServiceAPIListGroupsParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return groupsTestResponse(t, strings.Join([]string{
				groupsTestGroup("group-a", "platform-engineers"),
				groupsTestGroup("group-b", "platform-engineers"),
			}, ","), ""), nil
		})

	data, result := groupsTestRead(t, dataSourceOrganizationGroup(), groupsTestSingularState(t, "platform-engineers"), mockClient)
	r.True(result.HasError())
	r.Contains(result[0].Summary, `expected a single group with name "platform-engineers"`)
	r.Empty(data.Id(), "no id should be set when the lookup is ambiguous")
}
