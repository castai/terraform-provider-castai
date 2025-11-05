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

	"github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer"
	mock_ai_optimizer "github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer/mock"
)

func TestAIOptimizerAPIKeyResourceCreate(t *testing.T) {
	ctx := context.Background()
	organizationID := "test-org-123"
	apiKeyName := "test-api-key"
	apiToken := "test-token-abc123"

	t.Run("create should populate data correctly", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_ai_optimizer.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			aiOptimizerClient: &ai_optimizer.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		resource := resourceAIOptimizerAPIKey()

		val := cty.ObjectVal(map[string]cty.Value{
			FieldAIOptimizerAPIKeyOrganizationID: cty.StringVal(organizationID),
			FieldAIOptimizerAPIKeyName:           cty.StringVal(apiKeyName),
		})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		data := resource.Data(state)

		// Mock response with token
		body := io.NopCloser(bytes.NewReader([]byte(`{"token":"` + apiToken + `"}`)))
		mockClient.EXPECT().
			APIKeysAPICreateAPIKey(gomock.Any(), organizationID, gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Body:       body,
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		result := resource.CreateContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(apiKeyName, data.Id())
		r.Equal(apiToken, data.Get(FieldAIOptimizerAPIKeyToken).(string))
	})

	t.Run("create should fail when API returns error", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_ai_optimizer.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			aiOptimizerClient: &ai_optimizer.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		resource := resourceAIOptimizerAPIKey()

		val := cty.ObjectVal(map[string]cty.Value{
			FieldAIOptimizerAPIKeyOrganizationID: cty.StringVal(organizationID),
			FieldAIOptimizerAPIKeyName:           cty.StringVal(apiKeyName),
		})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		data := resource.Data(state)

		// Mock API error response
		body := io.NopCloser(bytes.NewReader([]byte(`{"message":"error creating API key"}`)))
		mockClient.EXPECT().
			APIKeysAPICreateAPIKey(gomock.Any(), organizationID, gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 400,
				Body:       body,
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		result := resource.CreateContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
	})

	t.Run("create should fail when response is missing token", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_ai_optimizer.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			aiOptimizerClient: &ai_optimizer.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		resource := resourceAIOptimizerAPIKey()

		val := cty.ObjectVal(map[string]cty.Value{
			FieldAIOptimizerAPIKeyOrganizationID: cty.StringVal(organizationID),
			FieldAIOptimizerAPIKeyName:           cty.StringVal(apiKeyName),
		})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		data := resource.Data(state)

		// Mock response without token field
		body := io.NopCloser(bytes.NewReader([]byte(`{}`)))
		mockClient.EXPECT().
			APIKeysAPICreateAPIKey(gomock.Any(), organizationID, gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Body:       body,
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		result := resource.CreateContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "unexpected response")
	})

	t.Run("create should fail when response JSON200 is nil", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_ai_optimizer.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			aiOptimizerClient: &ai_optimizer.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		resource := resourceAIOptimizerAPIKey()

		val := cty.ObjectVal(map[string]cty.Value{
			FieldAIOptimizerAPIKeyOrganizationID: cty.StringVal(organizationID),
			FieldAIOptimizerAPIKeyName:           cty.StringVal(apiKeyName),
		})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		data := resource.Data(state)

		// Mock response with invalid JSON (should result in nil JSON200)
		body := io.NopCloser(bytes.NewReader([]byte(`not valid json`)))
		mockClient.EXPECT().
			APIKeysAPICreateAPIKey(gomock.Any(), organizationID, gomock.Any(), gomock.Any()).
			Return(&http.Response{
				StatusCode: 200,
				Body:       body,
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		result := resource.CreateContext(ctx, data, provider)
		r.NotNil(result)
		r.True(result.HasError())
	})
}

func TestAIOptimizerAPIKeyResourceRead(t *testing.T) {
	ctx := context.Background()
	organizationID := "test-org-123"
	apiKeyName := "test-api-key"

	t.Run("read should succeed without making API calls", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_ai_optimizer.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			aiOptimizerClient: &ai_optimizer.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		// No API calls expected since Read is a no-op
		resource := resourceAIOptimizerAPIKey()

		val := cty.ObjectVal(map[string]cty.Value{
			FieldAIOptimizerAPIKeyOrganizationID: cty.StringVal(organizationID),
			FieldAIOptimizerAPIKeyName:           cty.StringVal(apiKeyName),
		})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = apiKeyName
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Equal(apiKeyName, data.Id())
	})
}

func TestAIOptimizerAPIKeyResourceDelete(t *testing.T) {
	ctx := context.Background()
	organizationID := "test-org-123"
	apiKeyName := "test-api-key"

	t.Run("delete should clear resource ID from state", func(t *testing.T) {
		r := require.New(t)
		mockctrl := gomock.NewController(t)
		mockClient := mock_ai_optimizer.NewMockClientInterface(mockctrl)

		provider := &ProviderConfig{
			aiOptimizerClient: &ai_optimizer.ClientWithResponses{
				ClientInterface: mockClient,
			},
		}

		// No API calls expected since Delete only removes from state
		resource := resourceAIOptimizerAPIKey()

		val := cty.ObjectVal(map[string]cty.Value{
			FieldAIOptimizerAPIKeyOrganizationID: cty.StringVal(organizationID),
			FieldAIOptimizerAPIKeyName:           cty.StringVal(apiKeyName),
		})
		state := terraform.NewInstanceStateShimmedFromValue(val, 0)
		state.ID = apiKeyName
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)
		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id())
	})
}

func TestAIOptimizerAPIKeyResourceSchema(t *testing.T) {
	t.Run("schema should have correct fields", func(t *testing.T) {
		r := require.New(t)
		resource := resourceAIOptimizerAPIKey()

		// Check organization_id field
		orgIDSchema := resource.Schema[FieldAIOptimizerAPIKeyOrganizationID]
		r.NotNil(orgIDSchema)
		r.True(orgIDSchema.Required)
		r.True(orgIDSchema.ForceNew)
		r.Equal("TypeString", orgIDSchema.Type.String())

		// Check name field
		nameSchema := resource.Schema[FieldAIOptimizerAPIKeyName]
		r.NotNil(nameSchema)
		r.True(nameSchema.Required)
		r.True(nameSchema.ForceNew)
		r.Equal("TypeString", nameSchema.Type.String())

		// Check token field
		tokenSchema := resource.Schema[FieldAIOptimizerAPIKeyToken]
		r.NotNil(tokenSchema)
		r.True(tokenSchema.Computed)
		r.True(tokenSchema.Sensitive)
		r.Equal("TypeString", tokenSchema.Type.String())
	})
}
