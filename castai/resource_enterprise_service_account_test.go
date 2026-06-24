package castai

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
	mockOrganizationManagement "github.com/castai/terraform-provider-castai/castai/sdk/organization_management/mock"
)

func TestEnterpriseServiceAccountReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when API call returns error then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()
		saID := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIListEnterpriseServiceAccountsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(nil, errors.New("network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal(""),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = saID

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Equal("list enterprise service accounts failed: network error", result[0].Summary)
	})

	t.Run("when API returns matching service account then update state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()
		saID := uuid.NewString()
		otherID := uuid.NewString()

		apiResponse := &organization_management.ListEnterpriseServiceAccountsResponse{
			Items: &[]organization_management.ListEnterpriseServiceAccountsResponseServiceAccount{
				{
					Id:             lo.ToPtr(otherID), // should be ignored
					Name:           lo.ToPtr("other-sa"),
					Email:          lo.ToPtr("other@example.com"),
					OrganizationId: lo.ToPtr(orgID),
				},
				{
					Id:             lo.ToPtr(saID),
					Name:           lo.ToPtr("my-sa"),
					Description:    lo.ToPtr("a test SA"),
					Email:          lo.ToPtr("my-sa@cast.ai"),
					OrganizationId: lo.ToPtr(orgID),
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIListEnterpriseServiceAccountsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListEnterpriseServiceAccountsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal(""),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = saID

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(saID, data.Id())
		r.Equal("my-sa", data.Get(FieldEnterpriseServiceAccountName).(string))
		r.Equal("a test SA", data.Get(FieldEnterpriseServiceAccountDescription).(string))
		r.Equal("my-sa@cast.ai", data.Get(FieldEnterpriseServiceAccountEmail).(string))
	})

	t.Run("when service account is not found then clear state ID", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()
		saID := uuid.NewString()

		apiResponse := &organization_management.ListEnterpriseServiceAccountsResponse{
			Items: &[]organization_management.ListEnterpriseServiceAccountsResponseServiceAccount{
				{
					Id:             lo.ToPtr(uuid.NewString()), // different ID — not our SA
					Name:           lo.ToPtr("some-other-sa"),
					OrganizationId: lo.ToPtr(orgID),
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIListEnterpriseServiceAccountsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListEnterpriseServiceAccountsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal(""),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = saID

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id())
	})

	t.Run("when API returns nil items then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()
		saID := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIListEnterpriseServiceAccountsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(&organization_management.EnterpriseAPIListEnterpriseServiceAccountsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      &organization_management.ListEnterpriseServiceAccountsResponse{Items: nil},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal(""),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = saID

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Equal("unexpected empty response from list enterprise service accounts", result[0].Summary)
	})
}

func TestEnterpriseServiceAccountCreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when API call returns error then return error", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseServiceAccountsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(nil, errors.New("network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal("a description"),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.CreateContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Equal("batch create enterprise service accounts failed: network error", result[0].Summary)
		r.Empty(data.Id())
	})

	t.Run("when API successfully creates service account then set state", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()
		saID := uuid.NewString()

		apiResponse := &organization_management.BatchCreateEnterpriseServiceAccountsResponse{
			Items: &[]organization_management.ListEnterpriseServiceAccountsResponseServiceAccount{
				{
					Id:             lo.ToPtr(saID),
					Name:           lo.ToPtr("my-sa"),
					Description:    lo.ToPtr("a description"),
					Email:          lo.ToPtr("my-sa@cast.ai"),
					OrganizationId: lo.ToPtr(orgID),
				},
			},
		}

		mockClient.EXPECT().
			EnterpriseAPIBatchCreateEnterpriseServiceAccountsWithResponse(
				gomock.Any(),
				enterpriseID,
				organization_management.BatchCreateEnterpriseServiceAccountsRequest{
					EnterpriseId: enterpriseID,
					Requests: []organization_management.BatchCreateEnterpriseServiceAccountsRequestServiceAccountRequest{
						{Name: "my-sa", Description: lo.ToPtr("a description"), OrganizationId: orgID},
					},
				},
			).
			Return(&organization_management.EnterpriseAPIBatchCreateEnterpriseServiceAccountsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				JSON200:      apiResponse,
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal("a description"),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(saID, data.Id())
		r.Equal("my-sa@cast.ai", data.Get(FieldEnterpriseServiceAccountEmail).(string))
	})
}

func TestEnterpriseServiceAccountDeleteContext(t *testing.T) {
	t.Parallel()

	t.Run("when API call returns error then return error and preserve ID", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()
		saID := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseServiceAccountsWithResponse(gomock.Any(), enterpriseID, gomock.Any()).
			Return(nil, errors.New("network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal(""),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = saID

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Equal("batch delete enterprise service accounts failed: network error", result[0].Summary)
		r.NotEmpty(data.Id(), "ID must not be cleared when delete fails")
	})

	t.Run("when API successfully deletes service account then clear state ID", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		mockClient := mockOrganizationManagement.NewMockClientWithResponsesInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			organizationManagementClient: mockClient,
		}

		enterpriseID := uuid.NewString()
		orgID := uuid.NewString()
		saID := uuid.NewString()

		mockClient.EXPECT().
			EnterpriseAPIBatchDeleteEnterpriseServiceAccountsWithResponse(
				gomock.Any(),
				enterpriseID,
				organization_management.BatchDeleteEnterpriseServiceAccountsRequest{
					EnterpriseId: enterpriseID,
					Requests: []organization_management.BatchDeleteEnterpriseServiceAccountsRequestDeleteServiceAccountRequest{
						{Id: saID, OrganizationId: orgID},
					},
				},
			).
			Return(&organization_management.EnterpriseAPIBatchDeleteEnterpriseServiceAccountsResponse{
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldEnterpriseServiceAccountEnterpriseID:   cty.StringVal(enterpriseID),
			FieldEnterpriseServiceAccountOrganizationID: cty.StringVal(orgID),
			FieldEnterpriseServiceAccountName:           cty.StringVal("my-sa"),
			FieldEnterpriseServiceAccountDescription:    cty.StringVal(""),
			FieldEnterpriseServiceAccountEmail:          cty.StringVal(""),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = saID

		res := resourceEnterpriseServiceAccount()
		data := res.Data(state)

		result := res.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id())
	})
}
