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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

func TestOrganizationResourceReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when only role bindings present then populate the state with owners viewers and members based on role bindings", func(t *testing.T) {
		t.Parallel()

		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "b6bfc024-a267-400f-b8f1-db0850c369b1"

		listInvitationsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "nextCursor": "",
  "invitations": []
}`)))

		listRoleBindingsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "roleBindings": [
    {
      "id": "1",
      "organizationId": "` + organizationID + `",
      "name": "owner-binding",
      "description": "owner binding",
      "createdAt": "2025-09-05T09:34:45.624Z",
      "updatedAt": "2025-09-05T09:34:45.624Z",
      "definition": {
        "roleId": "` + OwnerRoleID + `",
        "subjects": [
          {
            "user": {
              "id": "1",
              "name": "user-1",
              "email": "user-1@cast.ai"
            }
          }
        ],
        "scopes": [
          {
            "organization": {
              "id": "` + organizationID + `"
            }
          }
        ]
      }
    },
	{
      "id": "2",
      "organizationId": "` + organizationID + `",
      "name": "viewer-binding",
      "description": "viewer binding",
      "createdAt": "2025-09-05T09:34:45.624Z",
      "updatedAt": "2025-09-05T09:34:45.624Z",
      "definition": {
        "roleId": "` + ViewerRoleID + `",
        "subjects": [
          {
            "user": {
              "id": "2",
              "name": "user-2",
              "email": "user-2@cast.ai"
            }
          }
        ],
        "scopes": [
          {
            "organization": {
              "id": "` + organizationID + `"
            }
          }
        ]
      }
    },
	{
      "id": "3",
      "organizationId": "` + organizationID + `",
      "name": "member-binding",
      "description": "member binding",
      "createdAt": "2025-09-05T09:34:45.624Z",
      "updatedAt": "2025-09-05T09:34:45.624Z",
      "definition": {
        "roleId": "` + MemberRoleID + `",
        "subjects": [
          {
            "user": {
              "id": "3",
              "name": "user-3",
              "email": "user-3@cast.ai"
            }
          }
        ],
        "scopes": [
          {
            "organization": {
              "id": "` + organizationID + `"
            }
          }
        ]
      }
    }
  ],
  "nextPage": {
    "limit": "100",
    "cursor": ""
  },
  "totalCount": "3"
	}`)))
		mockClient.EXPECT().
			RbacServiceAPIListRoleBindings(gomock.Any(), organizationID, &sdk.RbacServiceAPIListRoleBindingsParams{
				SubjectType: &[]sdk.RbacServiceAPIListRoleBindingsParamsSubjectType{sdk.SUBJECTUSER},
				ScopeType:   &[]sdk.RbacServiceAPIListRoleBindingsParamsScopeType{sdk.RbacServiceAPIListRoleBindingsParamsScopeTypeORGANIZATION},
			}).
			Return(&http.Response{StatusCode: 200, Body: listRoleBindingsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			UsersAPIListInvitations(gomock.Any(), gomock.Any()).Return(&http.Response{StatusCode: 200, Body: listInvitationsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
		state.ID = organizationID

		resource := resourceOrganizationMembers()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = b6bfc024-a267-400f-b8f1-db0850c369b1
members.# = 1
members.0 = user-3@cast.ai
organization_id = b6bfc024-a267-400f-b8f1-db0850c369b1
owners.# = 1
owners.0 = user-1@cast.ai
viewers.# = 1
viewers.0 = user-2@cast.ai
Tainted = false
`, data.State().String())
	})

	t.Run("when no role bindings present then populate the state with empty lists", func(t *testing.T) {
		t.Parallel()

		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "b6bfc024-a267-400f-b8f1-db0850c369b1"

		listRoleBindingsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "roleBindings": [],
  "nextPage": {
    "limit": "100",
    "cursor": ""
  },
  "totalCount": "0"
}`)))

		listInvitationsBody := io.NopCloser(bytes.NewReader([]byte(`{
			"nextCursor": "",
			"invitations": []
		}`)))

		mockClient.EXPECT().
			RbacServiceAPIListRoleBindings(gomock.Any(), organizationID, &sdk.RbacServiceAPIListRoleBindingsParams{
				SubjectType: &[]sdk.RbacServiceAPIListRoleBindingsParamsSubjectType{sdk.SUBJECTUSER},
				ScopeType:   &[]sdk.RbacServiceAPIListRoleBindingsParamsScopeType{sdk.RbacServiceAPIListRoleBindingsParamsScopeTypeORGANIZATION},
			}).
			Return(&http.Response{StatusCode: 200, Body: listRoleBindingsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			UsersAPIListInvitations(gomock.Any(), gomock.Any()).Return(&http.Response{StatusCode: 200, Body: listInvitationsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
		state.ID = organizationID

		resource := resourceOrganizationMembers()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = b6bfc024-a267-400f-b8f1-db0850c369b1
members.# = 0
organization_id = b6bfc024-a267-400f-b8f1-db0850c369b1
owners.# = 0
viewers.# = 0
Tainted = false
`, data.State().String())
	})

	t.Run("when pending invitations present then populate the state with pending invitations", func(t *testing.T) {
		t.Parallel()

		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "b6bfc024-a267-400f-b8f1-db0850c369b1"
		listInvitationsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "nextCursor": "",
  "invitations": [
    {
      "id": "1",
      "inviteEmail": "user-1@cast.ai",
      "roleBindings": [
        {
          "roleId": "` + OwnerRoleID + `",
          "scopes": [
            {
              "id": "` + organizationID + `",
              "type": "ORGANIZATION"
            }
          ]
        }
      ],
      "validUntil": "2225-09-05T09:48:48.106Z"
    },
    {
      "id": "2",
      "inviteEmail": "user-2@cast.ai",
      "roleBindings": [
        {
          "roleId": "` + ViewerRoleID + `",
          "scopes": [
            {
              "id": "` + organizationID + `",
              "type": "ORGANIZATION"
            }
          ]
        }
      ],
      "validUntil": "2225-09-05T09:48:48.106Z"
    },
    {
      "id": "3",
      "inviteEmail": "user-3@cast.ai",
      "roleBindings": [
        {
          "roleId": "` + MemberRoleID + `",
          "scopes": [
            {
              "id": "` + organizationID + `",
              "type": "ORGANIZATION"
            }
          ]
        }
      ],
      "validUntil": "2225-09-05T09:48:48.106Z"
    }
  ]
}`)))

		listRoleBindingsBody := io.NopCloser(bytes.NewReader([]byte(`{
			"roleBindings": [],
			"nextPage": {
				"limit": "100",
				"cursor": ""
			},
			"totalCount": "0"
		}`)))
		mockClient.EXPECT().
			RbacServiceAPIListRoleBindings(gomock.Any(), organizationID, &sdk.RbacServiceAPIListRoleBindingsParams{
				SubjectType: &[]sdk.RbacServiceAPIListRoleBindingsParamsSubjectType{sdk.SUBJECTUSER},
				ScopeType:   &[]sdk.RbacServiceAPIListRoleBindingsParamsScopeType{sdk.RbacServiceAPIListRoleBindingsParamsScopeTypeORGANIZATION},
			}).
			Return(&http.Response{StatusCode: 200, Body: listRoleBindingsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			UsersAPIListInvitations(gomock.Any(), gomock.Any()).Return(&http.Response{StatusCode: 200, Body: listInvitationsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
		state.ID = organizationID

		resource := resourceOrganizationMembers()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = b6bfc024-a267-400f-b8f1-db0850c369b1
members.# = 1
members.0 = user-3@cast.ai
organization_id = b6bfc024-a267-400f-b8f1-db0850c369b1
owners.# = 1
owners.0 = user-1@cast.ai
viewers.# = 1
viewers.0 = user-2@cast.ai
Tainted = false
`, data.State().String())
	})

	t.Run("when both role bindings and pending invitations present then populate the state with both", func(t *testing.T) {
		t.Parallel()

		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		organizationID := "b6bfc024-a267-400f-b8f1-db0850c369b1"
		listInvitationsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "nextCursor": "",
  "invitations": [
    {
      "id": "1",
      "inviteEmail": "user-1@cast.ai",
      "roleBindings": [
        {
          "roleId": "` + OwnerRoleID + `",
          "scopes": [
            {
              "id": "` + organizationID + `",
              "type": "ORGANIZATION"
            }
          ]
        }
      ],
      "validUntil": "2225-09-05T09:48:48.106Z"
    }
  ]
}`)))

		listRoleBindingsBody := io.NopCloser(bytes.NewReader([]byte(`{
  "roleBindings": [
    {
      "id": "3",
      "organizationId": "` + organizationID + `",
      "name": "member-binding",
      "description": "member binding",
      "createdAt": "2025-09-05T09:34:45.624Z",
      "updatedAt": "2025-09-05T09:34:45.624Z",
      "definition": {
        "roleId": "` + MemberRoleID + `",
        "subjects": [
          {
            "user": {
              "id": "3",
              "name": "user-3",
              "email": "user-3@cast.ai"
            }
          }
        ],
        "scopes": [
          {
            "organization": {
              "id": "` + organizationID + `"
            }
          }
        ]
      }
    }
  ],
  "nextPage": {
    "limit": "100",
    "cursor": ""
  },
  "totalCount": "0"
}`)))
		mockClient.EXPECT().
			RbacServiceAPIListRoleBindings(gomock.Any(), organizationID, &sdk.RbacServiceAPIListRoleBindingsParams{
				SubjectType: &[]sdk.RbacServiceAPIListRoleBindingsParamsSubjectType{sdk.SUBJECTUSER},
				ScopeType:   &[]sdk.RbacServiceAPIListRoleBindingsParamsScopeType{sdk.RbacServiceAPIListRoleBindingsParamsScopeTypeORGANIZATION},
			}).
			Return(&http.Response{StatusCode: 200, Body: listRoleBindingsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		mockClient.EXPECT().
			UsersAPIListInvitations(gomock.Any(), gomock.Any()).Return(&http.Response{StatusCode: 200, Body: listInvitationsBody, Header: map[string][]string{"Content-Type": {"json"}}}, nil)

		state := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0)
		state.ID = organizationID

		resource := resourceOrganizationMembers()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(`ID = b6bfc024-a267-400f-b8f1-db0850c369b1
members.# = 1
members.0 = user-3@cast.ai
organization_id = b6bfc024-a267-400f-b8f1-db0850c369b1
owners.# = 1
owners.0 = user-1@cast.ai
viewers.# = 0
Tainted = false
`, data.State().String())
	})
}

func TestOrganizationResourceSchemaDeprecation(t *testing.T) {
	t.Parallel()

	r := require.New(t)
	resource := resourceOrganizationMembers()
	schema := resource.Schema

	// Test that deprecated fields have deprecation warnings
	r.NotEmpty(schema[FieldOrganizationMembersOwners].Deprecated)
	r.NotEmpty(schema[FieldOrganizationMembersViewers].Deprecated)
	r.NotEmpty(schema[FieldOrganizationMembersMembers].Deprecated)

	// Test that deprecation messages mention role bindings
	r.Contains(schema[FieldOrganizationMembersOwners].Deprecated, "castai_role_bindings")
	r.Contains(schema[FieldOrganizationMembersViewers].Deprecated, "castai_role_bindings")
	r.Contains(schema[FieldOrganizationMembersMembers].Deprecated, "castai_role_bindings")
}
func TestCompareRoleMembers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		before           []string
		after            []string
		exptectedDeleted []string
		expectedAdded    []string
	}{
		{
			before:           []string{"1", "2", "3"},
			after:            []string{"2", "3", "4"},
			expectedAdded:    []string{"4"},
			exptectedDeleted: []string{"1"},
		},
		{
			before:           []string{},
			after:            []string{"1", "2", "3"},
			expectedAdded:    []string{"1", "2", "3"},
			exptectedDeleted: nil,
		},
		{
			before:           []string{"1", "2", "3"},
			after:            []string{},
			expectedAdded:    nil,
			exptectedDeleted: []string{"1", "2", "3"},
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("test-case-%d", i), func(t *testing.T) {
			t.Parallel()
			toAdd, toDelete := compareRoleMembers(tc.before, tc.after)

			r := require.New(t)
			r.Equal(tc.exptectedDeleted, toDelete)
			r.Equal(tc.expectedAdded, toAdd)
		})
	}
}

func TestGetMembersDiff(t *testing.T) {
	t.Parallel()

	type input struct {
		owners  roleChange
		viewers roleChange
		members roleChange
	}

	testCases := []struct {
		input                   input
		expectedMembersToAdd    userRoleByEmail
		expectedMembersToMove   userRoleByEmail
		expectedMembersToDelete map[string]struct{}
	}{
		{
			input: input{
				owners: roleChange{addedMembers: []string{"user-1"}},
				viewers: roleChange{
					addedMembers:   []string{"user-2"},
					deletedMembers: []string{"user-3"},
				},
				members: roleChange{
					addedMembers:   []string{"user-3"},
					deletedMembers: []string{"user-4"},
				},
			},
			expectedMembersToAdd:    map[string]string{"user-1": OwnerRoleID, "user-2": ViewerRoleID},
			expectedMembersToMove:   map[string]string{"user-3": MemberRoleID},
			expectedMembersToDelete: map[string]struct{}{"user-4": {}},
		},
		{
			input: input{
				owners:  roleChange{addedMembers: []string{"user-1"}},
				viewers: roleChange{addedMembers: []string{"user-2"}},
				members: roleChange{addedMembers: []string{"user-3"}},
			},
			expectedMembersToAdd:    map[string]string{"user-1": OwnerRoleID, "user-2": ViewerRoleID, "user-3": MemberRoleID},
			expectedMembersToMove:   map[string]string{},
			expectedMembersToDelete: map[string]struct{}{},
		},
		{
			input: input{
				owners:  roleChange{addedMembers: []string{"user-1"}},
				viewers: roleChange{addedMembers: []string{"user-2"}},
				members: roleChange{addedMembers: []string{"user-3"}},
			},
			expectedMembersToAdd:    map[string]string{"user-1": OwnerRoleID, "user-2": ViewerRoleID, "user-3": MemberRoleID},
			expectedMembersToMove:   map[string]string{},
			expectedMembersToDelete: map[string]struct{}{},
		},
		{
			input: input{
				owners:  roleChange{addedMembers: []string{"user-1"}, deletedMembers: []string{"user-2"}},
				viewers: roleChange{addedMembers: []string{"user-3"}, deletedMembers: []string{"user-4"}},
				members: roleChange{addedMembers: []string{"user-5"}, deletedMembers: []string{"user-6"}},
			},
			expectedMembersToAdd:    map[string]string{"user-1": OwnerRoleID, "user-3": ViewerRoleID, "user-5": MemberRoleID},
			expectedMembersToMove:   map[string]string{},
			expectedMembersToDelete: map[string]struct{}{"user-2": {}, "user-4": {}, "user-6": {}},
		},
		{
			input: input{
				owners:  roleChange{addedMembers: []string{"user-1"}, deletedMembers: []string{"user-2"}},
				viewers: roleChange{addedMembers: []string{"user-2"}, deletedMembers: []string{"user-3"}},
				members: roleChange{addedMembers: []string{"user-3"}, deletedMembers: []string{"user-1"}},
			},
			expectedMembersToAdd:    map[string]string{},
			expectedMembersToMove:   map[string]string{"user-1": OwnerRoleID, "user-2": ViewerRoleID, "user-3": MemberRoleID},
			expectedMembersToDelete: map[string]struct{}{},
		},
		{
			input: input{
				owners:  roleChange{addedMembers: []string{"user-1", "user-2"}, deletedMembers: []string{"user-9", "user-4"}},
				viewers: roleChange{addedMembers: []string{"user-3", "user-5"}, deletedMembers: []string{"user-1", "user-6"}},
				members: roleChange{addedMembers: []string{"user-6", "user-7"}, deletedMembers: []string{"user-8", "user-3"}},
			},
			expectedMembersToAdd:    map[string]string{"user-2": OwnerRoleID, "user-5": ViewerRoleID, "user-7": MemberRoleID},
			expectedMembersToMove:   map[string]string{"user-1": OwnerRoleID, "user-3": ViewerRoleID, "user-6": MemberRoleID},
			expectedMembersToDelete: map[string]struct{}{"user-4": {}, "user-8": {}, "user-9": {}},
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("test-case-%d", i), func(t *testing.T) {
			t.Parallel()
			diff := getMembersDiff(tc.input.owners, tc.input.viewers, tc.input.members)

			r := require.New(t)
			r.Equal(tc.expectedMembersToAdd, diff.membersToAdd)
			r.Equal(tc.expectedMembersToDelete, diff.membersToDelete)
			r.Equal(tc.expectedMembersToMove, diff.membersToMove)
		})
	}
}

func TestPendingManipulations(t *testing.T) {
	t.Parallel()

	type input struct {
		diff                  membersDiff
		existingUserIDByEmail map[string]string
		invitationIDByEmail   map[string]string
	}

	testCases := []struct {
		input                   input
		expectedMembersToAdd    userRoleByEmail
		expectedMembersToUpdate userRoleByID
		expectedMembersToDelete []string
		invitationsToDelete     []string
	}{
		{
			input: input{
				diff: membersDiff{
					membersToDelete: map[string]struct{}{"user-1@cast.ai": {}, "user-2@cast.ai": {}},
				},
				existingUserIDByEmail: map[string]string{"user-1@cast.ai": "id-1"},
				invitationIDByEmail:   map[string]string{"user-2@cast.ai": "id-2"},
			},
			expectedMembersToDelete: []string{"id-1"},
			expectedMembersToAdd:    userRoleByEmail{},
			expectedMembersToUpdate: userRoleByID{},
			invitationsToDelete:     []string{"id-2"},
		},
		{
			input: input{
				diff: membersDiff{
					membersToMove: map[string]string{"user-1@cast.ai": "owner", "user-2@cast.ai": "member"},
					membersToAdd:  map[string]string{"user-3@cast.ai": "viewer"},
				},
				existingUserIDByEmail: map[string]string{"user-1@cast.ai": "id-1"},
				invitationIDByEmail:   map[string]string{"user-2@cast.ai": "id-2"},
			},
			expectedMembersToDelete: []string{},
			expectedMembersToAdd:    userRoleByEmail{"user-3@cast.ai": "viewer", "user-2@cast.ai": "member"},
			expectedMembersToUpdate: userRoleByID{"id-1": "owner"},
			invitationsToDelete:     []string{"id-2"},
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("test-case-%d", i), func(t *testing.T) {
			t.Parallel()
			pendingManipulations := getPendingManipulations(tc.input.diff, tc.input.existingUserIDByEmail, tc.input.invitationIDByEmail)

			r := require.New(t)
			r.Equal(tc.expectedMembersToAdd, pendingManipulations.membersToAdd)
			r.Equal(tc.expectedMembersToDelete, pendingManipulations.membersToDelete)
			r.Equal(tc.expectedMembersToUpdate, pendingManipulations.membersToUpdate)
			r.Equal(tc.invitationsToDelete, pendingManipulations.invitationsToDelete)
		})
	}
}
