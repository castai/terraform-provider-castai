package castai

import (
	"bytes"
	"context"
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
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

const organizationUsersTestOrganizationID = "d09389f9-6a95-46a1-9ded-5aa76e7cbfa5"

func organizationUsersTestState(t *testing.T, mutate func(attrs map[string]cty.Value)) *terraform.InstanceState {
	t.Helper()

	attrs := map[string]cty.Value{
		FieldOrgUsersOrganizationID: cty.StringVal(organizationUsersTestOrganizationID),
		FieldOrgUsersEmail:          cty.StringVal(""),
		FieldOrgUsersAuthProvider:   cty.StringVal(""),
		FieldOrgUsersScimActive:     cty.BoolVal(false),
		FieldOrgUsersIncludeGroups:  cty.BoolVal(true),
		FieldOrgUsersRoleIDs:        cty.ListValEmpty(cty.String),
	}
	if mutate != nil {
		mutate(attrs)
	}
	return terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(attrs), 0)
}

func organizationUsersTestResponse(t *testing.T, memberships string) *http.Response {
	t.Helper()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(fmt.Sprintf(`{"users":[%s]}`, memberships)))),
		Header:     map[string][]string{"Content-Type": {"json"}},
	}
}

func organizationUsersTestMembership(id, email, username, name, authProvider string, scimActive bool, groupIDs ...string) string {
	groups := ""
	if len(groupIDs) > 0 {
		refs := make([]string, 0, len(groupIDs))
		for _, groupID := range groupIDs {
			refs = append(refs, fmt.Sprintf(`{"id":%q,"name":"group-%s"}`, groupID, groupID))
		}
		groups = fmt.Sprintf(`,"groups":[%s]`, strings.Join(refs, ","))
	}
	return fmt.Sprintf(`{
		"user": {
			"id": %q,
			"email": %q,
			"username": %q,
			"name": %q,
			"authProvider": %q,
			"scimActive": %t
		}%s
	}`, id, email, username, name, authProvider, scimActive, groups)
}

func organizationUsersTestRead(t *testing.T, state *terraform.InstanceState, mockClient *mock_sdk.MockClientInterface) (*schema.ResourceData, diag.Diagnostics) {
	t.Helper()

	resource := dataSourceOrganizationUsers()
	data := resource.Data(state)
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	return data, result
}

func organizationUsersFromState(t *testing.T, data *schema.ResourceData) []map[string]any {
	t.Helper()

	raw, ok := data.Get(FieldOrgUsersUsers).([]interface{})
	require.True(t, ok, "users should be a list")

	users := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		user, ok := item.(map[string]any)
		require.True(t, ok, "user should be a map")
		users = append(users, user)
	}
	return users
}

// TestOrganizationUsersDataSource_DuplicateEmailsBothReturned covers the core
// lookup: the API has no email filter and one email can map to several
// identities, so all matching users must come back.
func TestOrganizationUsersDataSource_DuplicateEmailsBothReturned(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.UsersAPIListOrganizationUsersParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return organizationUsersTestResponse(t, strings.Join([]string{
				organizationUsersTestMembership("user-sso", "Alice@Corp.com", "auth0|111", "Alice SSO", "google", true, "group-a"),
				organizationUsersTestMembership("user-pwd", "alice@corp.com", "auth0|222", "Alice password", "", false),
			}, ",")), nil
		})

	state := organizationUsersTestState(t, func(attrs map[string]cty.Value) {
		attrs[FieldOrgUsersEmail] = cty.StringVal("alice@corp.com")
	})
	data, result := organizationUsersTestRead(t, state, mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)
	r.Equal(organizationUsersTestOrganizationID, data.Id())

	// Results are sorted by user ID, so user-pwd comes before user-sso.
	users := organizationUsersFromState(t, data)
	r.Len(users, 2)

	r.Equal("user-pwd", users[0]["id"])
	r.Equal("alice@corp.com", users[0]["email"])
	r.Equal("auth0|222", users[0]["username"])
	r.Equal("Alice password", users[0]["name"])
	r.Equal("", users[0]["auth_provider"])
	r.Equal(false, users[0]["scim_active"])
	r.Equal([]any{}, users[0]["group_ids"])

	r.Equal("user-sso", users[1]["id"])
	r.Equal("Alice@Corp.com", users[1]["email"])
	r.Equal("auth0|111", users[1]["username"])
	r.Equal("Alice SSO", users[1]["name"])
	r.Equal("google", users[1]["auth_provider"])
	r.Equal(true, users[1]["scim_active"])
	r.Equal([]any{"group-a"}, users[1]["group_ids"])
}

// TestOrganizationUsersDataSource_SortedByID covers the deterministic ordering:
// the API does not promise an order, so users are sorted by ID to keep ids and
// indexes stable across reads.
func TestOrganizationUsersDataSource_SortedByID(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.UsersAPIListOrganizationUsersParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return organizationUsersTestResponse(t, strings.Join([]string{
				organizationUsersTestMembership("user-c", "c@corp.com", "auth0|3", "C", "", false),
				organizationUsersTestMembership("user-a", "a@corp.com", "auth0|1", "A", "", false),
				organizationUsersTestMembership("user-b", "b@corp.com", "auth0|2", "B", "", false),
			}, ",")), nil
		})

	data, result := organizationUsersTestRead(t, organizationUsersTestState(t, nil), mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	users := organizationUsersFromState(t, data)
	r.Len(users, 3)
	r.Equal([]any{"user-a", "user-b", "user-c"}, []any{users[0]["id"], users[1]["id"], users[2]["id"]})
}

// TestOrganizationUsersDataSource_EmptyBodyIsError covers the defensive branch
// for a 200 response that carries no JSON body.
func TestOrganizationUsersDataSource_EmptyBodyIsError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			// A non-JSON content type keeps the generated parser from unmarshalling,
			// so JSON200 stays nil without an error.
			Body:   io.NopCloser(bytes.NewReader(nil)),
			Header: map[string][]string{"Content-Type": {"text/plain"}},
		}, nil)

	_, result := organizationUsersTestRead(t, organizationUsersTestState(t, nil), mockClient)
	r.True(result.HasError())
	r.Contains(result[0].Summary, "unexpected empty response")
}

func TestOrganizationUsersDataSource_NarrowedByAuthProvider(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.UsersAPIListOrganizationUsersParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return organizationUsersTestResponse(t, strings.Join([]string{
				organizationUsersTestMembership("user-sso", "alice@corp.com", "auth0|111", "Alice SSO", "google", false),
				organizationUsersTestMembership("user-pwd", "alice@corp.com", "auth0|222", "Alice password", "", false),
			}, ",")), nil
		})

	state := organizationUsersTestState(t, func(attrs map[string]cty.Value) {
		attrs[FieldOrgUsersAuthProvider] = cty.StringVal("google")
	})
	data, result := organizationUsersTestRead(t, state, mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	users := organizationUsersFromState(t, data)
	r.Len(users, 1)
	r.Equal("user-sso", users[0]["id"])
}

func TestOrganizationUsersDataSource_CaseInsensitiveEmail(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.UsersAPIListOrganizationUsersParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return organizationUsersTestResponse(t,
				organizationUsersTestMembership("user-1", "alice@corp.com", "auth0|1", "Alice", "", false),
			), nil
		})

	state := organizationUsersTestState(t, func(attrs map[string]cty.Value) {
		attrs[FieldOrgUsersEmail] = cty.StringVal("ALICE@CORP.COM")
	})
	data, result := organizationUsersTestRead(t, state, mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	users := organizationUsersFromState(t, data)
	r.Len(users, 1)
	r.Equal("user-1", users[0]["id"])
}

func TestOrganizationUsersDataSource_NoMatchIsEmptyNotError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ *sdk.UsersAPIListOrganizationUsersParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			return organizationUsersTestResponse(t,
				organizationUsersTestMembership("user-1", "bob@corp.com", "auth0|1", "Bob", "", false),
			), nil
		})

	state := organizationUsersTestState(t, func(attrs map[string]cty.Value) {
		attrs[FieldOrgUsersEmail] = cty.StringVal("alice@corp.com")
	})
	data, result := organizationUsersTestRead(t, state, mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	users := organizationUsersFromState(t, data)
	r.Empty(users)
}

func TestOrganizationUsersDataSource_RoleIDsReachParams(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedParams *sdk.UsersAPIListOrganizationUsersParams
	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.UsersAPIListOrganizationUsersParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedParams = params
			return organizationUsersTestResponse(t,
				organizationUsersTestMembership("user-1", "alice@corp.com", "auth0|1", "Alice", "", false),
			), nil
		})

	state := organizationUsersTestState(t, func(attrs map[string]cty.Value) {
		attrs[FieldOrgUsersRoleIDs] = cty.ListVal([]cty.Value{cty.StringVal("role-a"), cty.StringVal("role-b")})
	})
	_, result := organizationUsersTestRead(t, state, mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	r.NotNil(capturedParams)
	r.NotNil(capturedParams.RoleId)
	r.Equal([]string{"role-a", "role-b"}, *capturedParams.RoleId)
	// include_groups defaults to true.
	r.NotNil(capturedParams.IncludeGroups)
	r.True(*capturedParams.IncludeGroups)
}

func TestOrganizationUsersDataSource_IncludeGroupsFalseAndNoRoleIDsReachParams(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	var capturedParams *sdk.UsersAPIListOrganizationUsersParams
	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params *sdk.UsersAPIListOrganizationUsersParams, _ ...sdk.RequestEditorFn) (*http.Response, error) {
			capturedParams = params
			return organizationUsersTestResponse(t,
				organizationUsersTestMembership("user-1", "alice@corp.com", "auth0|1", "Alice", "", false),
			), nil
		})

	state := organizationUsersTestState(t, func(attrs map[string]cty.Value) {
		attrs[FieldOrgUsersIncludeGroups] = cty.BoolVal(false)
	})
	data, result := organizationUsersTestRead(t, state, mockClient)
	r.False(result.HasError(), "unexpected diagnostics: %v", result)

	r.NotNil(capturedParams)
	r.NotNil(capturedParams.IncludeGroups)
	r.False(*capturedParams.IncludeGroups)
	r.Nil(capturedParams.RoleId, "no role filter should be sent when role_ids is unset")

	// Without includeGroups the API returns no groups.
	users := organizationUsersFromState(t, data)
	r.Len(users, 1)
	r.Equal([]any{}, users[0]["group_ids"])
}

func TestOrganizationUsersDataSource_APIError(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_sdk.NewMockClientInterface(ctrl)

	mockClient.EXPECT().
		UsersAPIListOrganizationUsers(gomock.Any(), organizationUsersTestOrganizationID, gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"message": "Internal server error"}`))),
			Header:     map[string][]string{"Content-Type": {"json"}},
		}, nil)

	resource := dataSourceOrganizationUsers()
	data := resource.Data(organizationUsersTestState(t, nil))
	result := resource.ReadContext(context.Background(), data, &ProviderConfig{
		api: &sdk.ClientWithResponses{ClientInterface: mockClient},
	})

	r.True(result.HasError())
	r.Contains(result[0].Summary, "listing organization users")
}

func TestOrganizationUsersScimActiveFilter_FromRawConfig(t *testing.T) {
	t.Parallel()

	r := require.New(t)

	// Attribute present with true.
	raw := cty.ObjectVal(map[string]cty.Value{FieldOrgUsersScimActive: cty.BoolVal(true)})
	filter := organizationUsersScimActiveFilter(raw)
	r.NotNil(filter)
	r.True(*filter)

	// Attribute present with false: must be distinguishable from "not set".
	raw = cty.ObjectVal(map[string]cty.Value{FieldOrgUsersScimActive: cty.BoolVal(false)})
	filter = organizationUsersScimActiveFilter(raw)
	r.NotNil(filter)
	r.False(*filter)

	// Attribute present but null.
	raw = cty.ObjectVal(map[string]cty.Value{FieldOrgUsersScimActive: cty.NullVal(cty.Bool)})
	r.Nil(organizationUsersScimActiveFilter(raw))

	// Attribute present but unknown: reading it would panic.
	raw = cty.ObjectVal(map[string]cty.Value{FieldOrgUsersScimActive: cty.UnknownVal(cty.Bool)})
	r.Nil(organizationUsersScimActiveFilter(raw))

	// Attribute missing from the object.
	r.Nil(organizationUsersScimActiveFilter(cty.EmptyObjectVal))

	// Raw config unavailable (cty.NilVal).
	r.Nil(organizationUsersScimActiveFilter(cty.NilVal))
}

func TestOrganizationUsersMatchFilters(t *testing.T) {
	t.Parallel()

	ssoUser := func() sdk.CastaiUsersV1beta1User {
		return sdk.CastaiUsersV1beta1User{
			Id:           lo.ToPtr("user-1"),
			Email:        "Alice@Corp.com",
			Username:     "auth0|1",
			Name:         "Alice",
			AuthProvider: lo.ToPtr("google"),
			ScimActive:   lo.ToPtr(true),
		}
	}
	passwordUser := func() sdk.CastaiUsersV1beta1User {
		user := ssoUser()
		user.AuthProvider = nil
		user.ScimActive = nil
		return user
	}

	scimTrue := lo.ToPtr(true)
	scimFalse := lo.ToPtr(false)

	t.Run("case-insensitive email", func(t *testing.T) {
		r := require.New(t)
		r.True(organizationUsersMatchFilters(ssoUser(), "alice@corp.com", "", nil))
		r.True(organizationUsersMatchFilters(ssoUser(), "ALICE@CORP.COM", "", nil))
		r.False(organizationUsersMatchFilters(ssoUser(), "bob@corp.com", "", nil))
	})

	t.Run("auth provider", func(t *testing.T) {
		r := require.New(t)
		r.True(organizationUsersMatchFilters(ssoUser(), "", "google", nil))
		r.False(organizationUsersMatchFilters(ssoUser(), "", "azure", nil))
		// Identity without an auth provider never matches a set filter.
		r.False(organizationUsersMatchFilters(passwordUser(), "", "google", nil))
	})

	t.Run("scim active", func(t *testing.T) {
		r := require.New(t)
		r.True(organizationUsersMatchFilters(ssoUser(), "", "", scimTrue))
		r.False(organizationUsersMatchFilters(ssoUser(), "", "", scimFalse))
		// A user without scimActive is treated as not SCIM-active.
		r.True(organizationUsersMatchFilters(passwordUser(), "", "", scimFalse))
		r.False(organizationUsersMatchFilters(passwordUser(), "", "", scimTrue))
		// nil filter means no scim filtering.
		r.True(organizationUsersMatchFilters(passwordUser(), "", "", nil))
	})

	t.Run("combined", func(t *testing.T) {
		r := require.New(t)
		r.True(organizationUsersMatchFilters(ssoUser(), "alice@corp.com", "google", scimTrue))
		r.False(organizationUsersMatchFilters(ssoUser(), "alice@corp.com", "google", scimFalse))
	})
}
