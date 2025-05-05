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

func TestSecurityRuntimeRule_ReadContext(t *testing.T) {
	t.Parallel()

	t.Run("when rule is missing then remove from state", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		mockClient.EXPECT().
			RuntimeSecurityAPIGetRule(gomock.Any(), "uuid-123").
			Return(&http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader(nil))}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "uuid-123"

		resource := resourceSecurityRuntimeRule()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Empty(data.Id())
	})

	t.Run("when API returns error then surface error", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		mockClient.EXPECT().
			RuntimeSecurityAPIGetRule(gomock.Any(), "uuid-123").
			Return(nil, fmt.Errorf("mock network error"))

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "uuid-123"

		resource := resourceSecurityRuntimeRule()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "getting runtime rule")
	})

	t.Run("when rule is found then populate state including labels", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		body := `{
			"rule": {
				"name": "test-rule",
				"category": "test-category",
				"severity": "SEVERITY_HIGH",
				"ruleText": "event.type == 'exec'",
				"ruleEngineType": "RULE_ENGINE_TYPE_CEL",
				"enabled": true,
				"labels": {"env": "prod", "team": "security"},
				"anomaliesCount": 5,
				"isBuiltIn": true,
				"usedCustomLists": [{"name": "list1"}]
			}
		}`

		mockClient.EXPECT().
			RuntimeSecurityAPIGetRule(gomock.Any(), "uuid-123").
			Return(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(body))),
				Header:     map[string][]string{"Content-Type": {"application/json"}},
			}, nil)

		stateValue := cty.ObjectVal(map[string]cty.Value{})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "uuid-123"

		resource := resourceSecurityRuntimeRule()
		data := resource.Data(state)

		result := resource.ReadContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal("test-rule", data.Get(FieldRuntimeRuleName))
	})
}

func TestSecurityRuntimeRule_CreateContext(t *testing.T) {
	t.Parallel()

	t.Run("when API returns 200 then set ID", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		name := "test-rule"
		id := "uuid-123"

		mockClient.EXPECT().
			RuntimeSecurityAPICreateRule(gomock.Any(), gomock.Any()).
			Return(httpResponse(200, `{}`), nil)

		mockClient.EXPECT().
			RuntimeSecurityAPIGetRules(gomock.Any(), gomock.Any()).
			Return(httpResponse(200, fmt.Sprintf(`{
				"rules": [{"name": "%s", "id": "%s"}]
			}`, name, id)), nil)

		mockClient.EXPECT().
			RuntimeSecurityAPIGetRule(gomock.Any(), id).
			Return(httpResponse(200, fmt.Sprintf(`{
				"rule": {
					"name": "%s",
					"id": "%s",
					"category": "test-category",
					"severity": "SEVERITY_HIGH",
					"ruleText": "event.type == 'exec'",
					"ruleEngineType": "RULE_ENGINE_TYPE_CEL",
					"enabled": true
				}
			}`, name, id)), nil)

		resource := resourceSecurityRuntimeRule()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldRuntimeRuleName:           cty.StringVal(name),
			FieldRuntimeRuleCategory:       cty.StringVal("test-category"),
			FieldRuntimeRuleSeverity:       cty.StringVal("SEVERITY_HIGH"),
			FieldRuntimeRuleRuleText:       cty.StringVal("event.type == 'exec'"),
			FieldRuntimeRuleRuleEngineType: cty.StringVal("RULE_ENGINE_TYPE_CEL"),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		data := resource.Data(state)

		result := resource.CreateContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.Equal(id, data.Id())
	})
}

func TestSecurityRuntimeRule_DeleteContext(t *testing.T) {
	t.Parallel()

	t.Run("when rule is NOT built-in and deleted", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock_sdk.NewMockClientInterface(ctrl)
		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		// Expect DELETE with correct ID
		mockClient.EXPECT().
			RuntimeSecurityAPIDeleteRules(gomock.Any(), gomock.AssignableToTypeOf(sdk.RuntimeSecurityAPIDeleteRulesJSONRequestBody{})).
			DoAndReturn(func(ctx context.Context, req sdk.RuntimeSecurityAPIDeleteRulesJSONRequestBody) (*http.Response, error) {
				r.Equal([]string{"uuid-123"}, req.Ids)
				return httpResponse(200, `{}`), nil
			})

		resource := resourceSecurityRuntimeRule()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldRuntimeRuleIsBuiltIn: cty.BoolVal(false),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "uuid-123"
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.True(data.State().Empty())
	})

	t.Run("when rule is built-in and disabled via toggle", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock_sdk.NewMockClientInterface(ctrl)
		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		// Expect TOGGLE with correct ID
		mockClient.EXPECT().
			RuntimeSecurityAPIToggleRules(gomock.Any(), gomock.AssignableToTypeOf(sdk.RuntimeV1ToggleRulesRequest{})).
			DoAndReturn(func(ctx context.Context, req sdk.RuntimeV1ToggleRulesRequest) (*http.Response, error) {
				r.False(req.Enabled)
				r.Equal([]string{"uuid-456"}, req.Ids)
				return httpResponse(200, `{}`), nil
			})

		resource := resourceSecurityRuntimeRule()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldRuntimeRuleIsBuiltIn: cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "uuid-456"
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.Nil(result)
		r.False(result.HasError())
		r.True(data.State().Empty())
	})

	t.Run("when disabling built-in rule via toggle fails", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock_sdk.NewMockClientInterface(ctrl)
		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		mockClient.EXPECT().
			RuntimeSecurityAPIToggleRules(gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("mock toggle error"))

		resource := resourceSecurityRuntimeRule()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldRuntimeRuleIsBuiltIn: cty.BoolVal(true),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "uuid-999"
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "disabling built-in runtime rule")
	})

	t.Run("when delete API call fails", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock_sdk.NewMockClientInterface(ctrl)
		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		mockClient.EXPECT().
			RuntimeSecurityAPIDeleteRules(gomock.Any(), gomock.Any()).
			Return(nil, fmt.Errorf("mock delete error"))

		resource := resourceSecurityRuntimeRule()
		stateValue := cty.ObjectVal(map[string]cty.Value{
			FieldRuntimeRuleIsBuiltIn: cty.BoolVal(false),
		})
		state := terraform.NewInstanceStateShimmedFromValue(stateValue, 0)
		state.ID = "uuid-888"
		data := resource.Data(state)

		result := resource.DeleteContext(ctx, data, provider)

		r.NotNil(result)
		r.True(result.HasError())
		r.Contains(result[0].Summary, "deleting security runtime rule")
	})
}

func TestSecurityRuntimeRule_Importer(t *testing.T) {
	t.Parallel()

	t.Run("when rule found then import by ID", func(t *testing.T) {
		r := require.New(t)
		mockClient := mock_sdk.NewMockClientInterface(gomock.NewController(t))

		ctx := context.Background()
		provider := &ProviderConfig{
			api: &sdk.ClientWithResponses{ClientInterface: mockClient},
		}

		name := "import-rule"
		id := "uuid-import"

		mockClient.EXPECT().
			RuntimeSecurityAPIGetRules(gomock.Any(), gomock.Any()).
			Return(httpResponse(200, fmt.Sprintf(`{
				"rules": [{"name": "%s", "id": "%s"}]
			}`, name, id)), nil)

		resource := resourceSecurityRuntimeRule()
		d := resource.Data(terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{}), 0))
		d.SetId(name)

		res, err := resourceSecurityRuntimeRuleImporter(ctx, d, provider)
		r.NoError(err)
		r.Len(res, 1)
		r.Equal(id, d.Id())
	})
}

func TestFindRuntimeRuleByName_Pagination(t *testing.T) {
	t.Parallel()

	t.Run("when API returns empty rules with nextCursor, pagination stops", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock_sdk.NewMockClientInterface(ctrl)
		ctx := context.Background()

		// First call returns empty rules and a nextCursor
		mockClient.EXPECT().
			RuntimeSecurityAPIGetRules(gomock.Any(), gomock.Any()).
			Return(httpResponse(200, `{
				"rules": [],
				"nextCursor": "cursor-2"
			}`), nil)

		rule, err := findRuntimeRuleByName(ctx, &sdk.ClientWithResponses{ClientInterface: mockClient}, "target-rule")

		r.NoError(err)
		r.Nil(rule)
	})

	t.Run("when rule is found on the second page", func(t *testing.T) {
		r := require.New(t)
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockClient := mock_sdk.NewMockClientInterface(ctrl)
		ctx := context.Background()

		callCount := 0

		mockClient.EXPECT().
			RuntimeSecurityAPIGetRules(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, params *sdk.RuntimeSecurityAPIGetRulesParams) (*http.Response, error) {
				callCount++
				if params.PageCursor == nil {
					// Page 1: one unrelated rule
					return httpResponse(200, `{
						"rules": [{"name": "not-it", "id": "uuid-1"}],
						"nextCursor": "cursor-2"
					}`), nil
				}
				// Page 2: contains the matching rule
				return httpResponse(200, `{
					"rules": [{"name": "target-rule", "id": "uuid-456"}]
				}`), nil
			}).Times(2)

		rule, err := findRuntimeRuleByName(ctx, &sdk.ClientWithResponses{ClientInterface: mockClient}, "target-rule")

		r.NoError(err)
		r.NotNil(rule)
		r.Equal("target-rule", *rule.Name)
		r.Equal("uuid-456", *rule.Id)
		r.Equal(2, callCount)
	})
}

// Helpers
func httpResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     map[string][]string{"Content-Type": {"application/json"}},
	}
}
