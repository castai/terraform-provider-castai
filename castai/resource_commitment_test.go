package castai

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
	"github.com/castai/terraform-provider-castai/castai/sdk/pricing"
	mock_pricing "github.com/castai/terraform-provider-castai/castai/sdk/pricing/mock"
)

func TestAccCloudAgnostic_Commitment(t *testing.T) {
	rName := fmt.Sprintf("%v-commitment-%v", ResourcePrefix, acctest.RandString(8))
	cudID := acctest.RandStringFromCharSet(12, "0123456789")
	organizationID := os.Getenv("ACCEPTANCE_TEST_ORGANIZATION_ID")
	resourceName := "castai_commitment.test"

	steps := []resource.TestStep{
		{
			// Create with auto_assignment not set — server decides.
			Config: testAccCommitmentConfig(rName, cudID, organizationID, "INACTIVE", 1.0, false, false),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet(resourceName, "id"),
				resource.TestCheckResourceAttrSet(resourceName, "organization_id"),
				resource.TestCheckResourceAttr(resourceName, "name", rName),
				resource.TestCheckResourceAttr(resourceName, "cloud", "GCP"),
				resource.TestCheckResourceAttr(resourceName, "region", "us-central1"),
				resource.TestCheckResourceAttr(resourceName, "type", "RESOURCE_CUD"),
				resource.TestCheckResourceAttr(resourceName, "autoscaling_status", "INACTIVE"),
				resource.TestCheckResourceAttr(resourceName, "allowed_usage", "1"),
				resource.TestCheckResourceAttr(resourceName, "scaling_strategy", "DEFAULT"),
				resource.TestCheckResourceAttr(resourceName, "auto_assignment", "true"),
				resource.TestCheckResourceAttr(resourceName, "gcp_resource_cud_details.cud_id", cudID),
				resource.TestCheckResourceAttr(resourceName, "gcp_resource_cud_details.plan", "TWELVE_MONTH"),
				resource.TestCheckResourceAttr(resourceName, "gcp_resource_cud_details.cpu", "32"),
				resource.TestCheckResourceAttrSet(resourceName, "state"),
				resource.TestCheckResourceAttrSet(resourceName, "create_time"),
			),
		},
		{
			// Patch-path update: operational settings only.
			Config: testAccCommitmentConfig(rName, cudID, organizationID, "ACTIVE", 0.75, false, false),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "autoscaling_status", "ACTIVE"),
				resource.TestCheckResourceAttr(resourceName, "allowed_usage", "0.75"),
			),
		},
		{
			// Upsert-path update: rename. ID must be preserved.
			Config: testAccCommitmentConfig(rName+"-renamed", cudID, organizationID, "ACTIVE", 0.75, false, false),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "name", rName+"-renamed"),
			),
		},
		{
			// Switch to explicit auto_assignment=false.
			Config: testAccCommitmentConfig(rName+"-renamed", cudID, organizationID, "ACTIVE", 0.75, true, false),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "auto_assignment", "false"),
			),
		},
		{
			// Upsert-path update with explicit false: auto_assignment must stay false.
			Config: testAccCommitmentConfig(rName+"-renamed-2", cudID, organizationID, "ACTIVE", 0.75, true, false),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "name", rName+"-renamed-2"),
				resource.TestCheckResourceAttr(resourceName, "auto_assignment", "false"),
			),
		},
	}

	// The assignments step requires a real cluster in the commitment's region
	// (us-central1); it is skipped when no cluster ID is provided.
	if clusterID := os.Getenv("ACCEPTANCE_TEST_CLUSTER_ID"); clusterID != "" {
		steps = append(steps, resource.TestStep{
			Config: testAccCommitmentConfigWithAssignments(rName+"-renamed-2", cudID, organizationID, clusterID),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "auto_assignment", "false"),
				resource.TestCheckResourceAttr(resourceName, "assignments.0.cluster_id", clusterID),
				resource.TestCheckResourceAttr(resourceName, "assignments.0.priority", "1"),
			),
		})
	}

	steps = append(steps, resource.TestStep{
		ResourceName: resourceName,
		ImportStateIdFunc: func(s *terraform.State) (string, error) {
			rs, ok := s.RootModule().Resources[resourceName]
			if !ok {
				return "", fmt.Errorf("resource %s not found in state", resourceName)
			}
			return fmt.Sprintf("%s/%s", rs.Primary.Attributes["organization_id"], rs.Primary.ID), nil
		},
		ImportState:       true,
		ImportStateVerify: true,
		// Imported state does not manage assignments (opt-in attribute): they
		// are only synced from the API when present in state.
		ImportStateVerifyIgnore: []string{"assignments"},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

func testAccCommitmentConfig(name, cudID, organizationID, autoscalingStatus string, allowedUsage float64, setAutoAssignment bool, autoAssignment bool) string {
	autoAssignmentLine := ""
	if setAutoAssignment {
		autoAssignmentLine = fmt.Sprintf("auto_assignment = %v", autoAssignment)
	}
	return fmt.Sprintf(`
resource "castai_commitment" "test" {
  name               = %[1]q
  organization_id    = %[6]q
  cloud              = "GCP"
  region             = "us-central1"
  type               = "RESOURCE_CUD"
  start_time         = "2026-01-01T00:00:00Z"
  end_time           = "2027-01-01T00:00:00Z"
  autoscaling_status = %[3]q
  allowed_usage      = %[4]v
  %[5]s

  gcp_resource_cud_details = {
    cud_id    = %[2]q
    plan      = "TWELVE_MONTH"
    type      = "GENERAL_PURPOSE_E2"
    cpu       = 32
    memory_mb = 131072
    status    = "ACTIVE"
  }
}
`, name, cudID, autoscalingStatus, allowedUsage, autoAssignmentLine, organizationID)
}

// ---------------------------------------------------------------------------
// Retry unit tests
// ---------------------------------------------------------------------------
func newCommitmentResourceWithMock(mockClient *mock_pricing.MockClientWithResponsesInterface) *genericCommitmentResource {
	return newCommitmentResourceWithMocks(mockClient, nil)
}

// newCommitmentResourceWithMocks builds the resource with both API clients
// mocked: the pricing client (commitment CRUD) and the inventory client
// (assignment mutations).
func newCommitmentResourceWithMocks(
	pricingClient pricing.ClientWithResponsesInterface,
	apiClient sdk.ClientWithResponsesInterface,
) *genericCommitmentResource {
	return &genericCommitmentResource{
		client: &ProviderConfig{
			pricingClient: pricingClient,
			api:           apiClient,
		},
	}
}

func httpResp(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
	}
}

func TestGetCommitmentWithRetry_Retries5xxThenSucceeds(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"
	commitment := &pricing.Commitment{Id: &commitmentID}

	gomock.InOrder(
		mockClient.EXPECT().
			CommitmentsAPIGetCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
			Return(&pricing.CommitmentsAPIGetCommitmentResponse{
				HTTPResponse: httpResp(http.StatusInternalServerError),
				Body:         []byte("server error"),
			}, nil),
		mockClient.EXPECT().
			CommitmentsAPIGetCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
			Return(&pricing.CommitmentsAPIGetCommitmentResponse{
				HTTPResponse: httpResp(http.StatusOK),
				JSON200:      commitment,
			}, nil),
	)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.getCommitmentWithRetry(context.Background(), orgID, commitmentID)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusOK, apiResp.StatusCode())
	r.Equal(commitment, apiResp.JSON200)
}

func TestGetCommitmentWithRetry_4xxStopsImmediately(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"

	// 404 must NOT be retried — exactly one call.
	mockClient.EXPECT().
		CommitmentsAPIGetCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
		Return(&pricing.CommitmentsAPIGetCommitmentResponse{
			HTTPResponse: httpResp(http.StatusNotFound),
			Body:         []byte("not found"),
		}, nil)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.getCommitmentWithRetry(context.Background(), orgID, commitmentID)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusNotFound, apiResp.StatusCode())
}

func TestGetCommitmentWithRetry_ContextCancellationStopsRetries(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"

	// Return 500 on every call so the retry loop would be infinite without
	// context cancellation.
	mockClient.EXPECT().
		CommitmentsAPIGetCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
		Return(&pricing.CommitmentsAPIGetCommitmentResponse{
			HTTPResponse: httpResp(http.StatusInternalServerError),
			Body:         []byte("server error"),
		}, nil).
		AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so backoff exits on the first check

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.getCommitmentWithRetry(ctx, orgID, commitmentID)

	// The error should be the context cancellation or the last 500 error
	// wrapped by backoff — either way, the function must return.
	r.Error(err)
	// apiResp may be nil (if the retry function never ran) or the last 500
	// response — the key invariant is that we returned and did not hang.
	_ = apiResp
}

func TestCreateCommitmentWithRetry_Retries5xxThenSucceeds(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID := "org-1"
	input := pricing.CreateCommitmentInput{}
	commitmentID := "commit-1"
	commitment := &pricing.Commitment{Id: &commitmentID}

	gomock.InOrder(
		mockClient.EXPECT().
			CommitmentsAPICreateCommitmentWithResponse(gomock.Any(), orgID, input).
			Return(&pricing.CommitmentsAPICreateCommitmentResponse{
				HTTPResponse: httpResp(http.StatusBadGateway),
				Body:         []byte("bad gateway"),
			}, nil),
		mockClient.EXPECT().
			CommitmentsAPICreateCommitmentWithResponse(gomock.Any(), orgID, input).
			Return(&pricing.CommitmentsAPICreateCommitmentResponse{
				HTTPResponse: httpResp(http.StatusOK),
				JSON200:      commitment,
			}, nil),
	)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.createCommitmentWithRetry(context.Background(), orgID, input)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusOK, apiResp.StatusCode())
	r.Equal(commitment, apiResp.JSON200)
}

func TestCreateCommitmentWithRetry_4xxStopsImmediately(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID := "org-1"
	input := pricing.CreateCommitmentInput{}

	// 400 must NOT be retried.
	mockClient.EXPECT().
		CommitmentsAPICreateCommitmentWithResponse(gomock.Any(), orgID, input).
		Return(&pricing.CommitmentsAPICreateCommitmentResponse{
			HTTPResponse: httpResp(http.StatusBadRequest),
			Body:         []byte("bad request"),
		}, nil)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.createCommitmentWithRetry(context.Background(), orgID, input)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusBadRequest, apiResp.StatusCode())
}

func TestCreateCommitmentWithRetry_ContextCancellationStopsRetries(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID := "org-1"
	input := pricing.CreateCommitmentInput{}

	mockClient.EXPECT().
		CommitmentsAPICreateCommitmentWithResponse(gomock.Any(), orgID, input).
		Return(&pricing.CommitmentsAPICreateCommitmentResponse{
			HTTPResponse: httpResp(http.StatusServiceUnavailable),
			Body:         []byte("unavailable"),
		}, nil).
		AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := newCommitmentResourceWithMock(mockClient)
	_, err := res.createCommitmentWithRetry(ctx, orgID, input)

	r.Error(err)
}

func TestDeleteCommitmentWithRetry_Retries5xxThenSucceeds(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"

	gomock.InOrder(
		mockClient.EXPECT().
			CommitmentsAPIDeleteCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
			Return(&pricing.CommitmentsAPIDeleteCommitmentResponse{
				HTTPResponse: httpResp(http.StatusServiceUnavailable),
				Body:         []byte("unavailable"),
			}, nil),
		mockClient.EXPECT().
			CommitmentsAPIDeleteCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
			Return(&pricing.CommitmentsAPIDeleteCommitmentResponse{
				HTTPResponse: httpResp(http.StatusOK),
			}, nil),
	)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.deleteCommitmentWithRetry(context.Background(), orgID, commitmentID)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusOK, apiResp.StatusCode())
}

func TestDeleteCommitmentWithRetry_4xxStopsImmediately(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"

	// 404 must NOT be retried — exactly one call.
	mockClient.EXPECT().
		CommitmentsAPIDeleteCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
		Return(&pricing.CommitmentsAPIDeleteCommitmentResponse{
			HTTPResponse: httpResp(http.StatusNotFound),
			Body:         []byte("not found"),
		}, nil)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.deleteCommitmentWithRetry(context.Background(), orgID, commitmentID)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusNotFound, apiResp.StatusCode())
}

func TestDeleteCommitmentWithRetry_ContextCancellationStopsRetries(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"

	mockClient.EXPECT().
		CommitmentsAPIDeleteCommitmentWithResponse(gomock.Any(), orgID, commitmentID).
		Return(&pricing.CommitmentsAPIDeleteCommitmentResponse{
			HTTPResponse: httpResp(http.StatusGatewayTimeout),
			Body:         []byte("gateway timeout"),
		}, nil).
		AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := newCommitmentResourceWithMock(mockClient)
	_, err := res.deleteCommitmentWithRetry(ctx, orgID, commitmentID)

	r.Error(err)
}

// ---------------------------------------------------------------------------
// updateCommitmentWithRetry
// ---------------------------------------------------------------------------

func TestUpdateCommitmentWithRetry_Retries5xxThenSucceeds(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"
	input := pricing.UpdateCommitmentInput{}
	commitment := &pricing.Commitment{Id: &commitmentID}

	gomock.InOrder(
		mockClient.EXPECT().
			CommitmentsAPIUpdateCommitmentWithResponse(gomock.Any(), orgID, commitmentID, input).
			Return(&pricing.CommitmentsAPIUpdateCommitmentResponse{
				HTTPResponse: httpResp(http.StatusInternalServerError),
				Body:         []byte("server error"),
			}, nil),
		mockClient.EXPECT().
			CommitmentsAPIUpdateCommitmentWithResponse(gomock.Any(), orgID, commitmentID, input).
			Return(&pricing.CommitmentsAPIUpdateCommitmentResponse{
				HTTPResponse: httpResp(http.StatusOK),
				JSON200:      commitment,
			}, nil),
	)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.updateCommitmentWithRetry(context.Background(), orgID, commitmentID, input)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusOK, apiResp.StatusCode())
	r.Equal(commitment, apiResp.JSON200)
}

func TestUpdateCommitmentWithRetry_4xxStopsImmediately(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"
	input := pricing.UpdateCommitmentInput{}

	mockClient.EXPECT().
		CommitmentsAPIUpdateCommitmentWithResponse(gomock.Any(), orgID, commitmentID, input).
		Return(&pricing.CommitmentsAPIUpdateCommitmentResponse{
			HTTPResponse: httpResp(http.StatusBadRequest),
			Body:         []byte("bad request"),
		}, nil)

	res := newCommitmentResourceWithMock(mockClient)
	apiResp, err := res.updateCommitmentWithRetry(context.Background(), orgID, commitmentID, input)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusBadRequest, apiResp.StatusCode())
}

func TestUpdateCommitmentWithRetry_ContextCancellationStopsRetries(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	mockClient := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"
	input := pricing.UpdateCommitmentInput{}

	mockClient.EXPECT().
		CommitmentsAPIUpdateCommitmentWithResponse(gomock.Any(), orgID, commitmentID, input).
		Return(&pricing.CommitmentsAPIUpdateCommitmentResponse{
			HTTPResponse: httpResp(http.StatusServiceUnavailable),
			Body:         []byte("unavailable"),
		}, nil).
		AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := newCommitmentResourceWithMock(mockClient)
	_, err := res.updateCommitmentWithRetry(ctx, orgID, commitmentID, input)

	r.Error(err)
}

func TestRegionRequiredCommitmentTypes(t *testing.T) {
	// Types that must always have a region set.
	required := []string{
		string(pricing.CommitmentTypeRESERVEDINSTANCE),
		string(pricing.CommitmentTypeRESOURCECUD),
		string(pricing.CommitmentTypeCAPACITYBLOCK),
		string(pricing.CommitmentTypeONDEMANDCAPACITYRESERVATION),
	}
	for _, ct := range required {
		require.True(t, regionRequiredCommitmentTypes[ct], "type %q should require region", ct)
	}

	// Types that may be region-agnostic.
	optional := []string{
		string(pricing.CommitmentTypeSAVINGSPLAN),
		string(pricing.CommitmentTypeFLEXCUD),
	}
	for _, ct := range optional {
		require.False(t, regionRequiredCommitmentTypes[ct], "type %q should NOT require region", ct)
	}
}

// validateConfigForTest builds a resource.ValidateConfigRequest from the
// given commitmentModel and invokes ValidateConfig, returning whether the
// diagnostics contain an error.
func validateConfigForTest(t *testing.T, ctx context.Context, model commitmentModel) bool {
	t.Helper()
	r := newCommitmentResource().(*genericCommitmentResource)

	schemaResp := fwresource.SchemaResponse{}
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	objType := schemaResp.Schema.Type().(basetypes.ObjectType)
	attrTypes := objType.AttrTypes

	objVal, diags := types.ObjectValueFrom(ctx, attrTypes, &model)
	require.False(t, diags.HasError(), "failed to build ObjectValueFrom from model")

	rawVal, err := objVal.ToTerraformValue(ctx)
	require.NoError(t, err)

	req := fwresource.ValidateConfigRequest{
		Config: tfsdk.Config{
			Raw:    rawVal,
			Schema: schemaResp.Schema,
		},
	}
	resp := &fwresource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, req, resp)
	return resp.Diagnostics.HasError()
}

func TestValidateConfig_RegionRequiredForRegionScopedTypes(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		commitment  commitmentModel
		expectError bool
	}{
		{
			name: "RESERVED_INSTANCE without region → error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-ri"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringNull(),
				Type:      types.StringValue(string(pricing.CommitmentTypeRESERVEDINSTANCE)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: true,
		},
		{
			name: "RESERVED_INSTANCE with region → no error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-ri"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringValue("us-east-1"),
				Type:      types.StringValue(string(pricing.CommitmentTypeRESERVEDINSTANCE)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: false,
		},
		{
			name: "RESOURCE_CUD without region → error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-cud"),
				Cloud:     types.StringValue("GCP"),
				Region:    types.StringNull(),
				Type:      types.StringValue(string(pricing.CommitmentTypeRESOURCECUD)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: true,
		},
		{
			name: "RESOURCE_CUD with region → no error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-cud"),
				Cloud:     types.StringValue("GCP"),
				Region:    types.StringValue("us-central1"),
				Type:      types.StringValue(string(pricing.CommitmentTypeRESOURCECUD)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: false,
		},
		{
			name: "CAPACITY_BLOCK without region → error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-cb"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringNull(),
				Type:      types.StringValue(string(pricing.CommitmentTypeCAPACITYBLOCK)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: true,
		},
		{
			name: "ON_DEMAND_CAPACITY_RESERVATION without region → error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-odcr"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringNull(),
				Type:      types.StringValue(string(pricing.CommitmentTypeONDEMANDCAPACITYRESERVATION)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: true,
		},
		{
			name: "SAVINGS_PLAN without region → no error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-sp"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringNull(),
				Type:      types.StringValue(string(pricing.CommitmentTypeSAVINGSPLAN)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: false,
		},
		{
			name: "FLEX_CUD without region → no error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-flex"),
				Cloud:     types.StringValue("GCP"),
				Region:    types.StringNull(),
				Type:      types.StringValue(string(pricing.CommitmentTypeFLEXCUD)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: false,
		},
		{
			name: "SAVINGS_PLAN with region → no error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-sp"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringValue("us-east-1"),
				Type:      types.StringValue(string(pricing.CommitmentTypeSAVINGSPLAN)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: false,
		},
		{
			name: "RESERVED_INSTANCE with unknown region → error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-ri"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringUnknown(),
				Type:      types.StringValue(string(pricing.CommitmentTypeRESERVEDINSTANCE)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: true,
		},
		{
			name: "RESERVED_INSTANCE with empty string region → error",
			commitment: commitmentModel{
				Name:      types.StringValue("test-ri"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringValue(""),
				Type:      types.StringValue(string(pricing.CommitmentTypeRESERVEDINSTANCE)),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := validateConfigForTest(t, ctx, tt.commitment)
			require.Equal(t, tt.expectError, hasError)
		})
	}
}

// ---------------------------------------------------------------------------
// Assignments unit tests.
// ---------------------------------------------------------------------------

func TestValidateConfig_Assignments(t *testing.T) {
	ctx := context.Background()

	clusterA := "11111111-1111-1111-1111-111111111111"
	clusterB := "22222222-2222-2222-2222-222222222222"

	assignments := []commitmentAssignmentModel{
		{ClusterID: types.StringValue(clusterA)},
		{ClusterID: types.StringValue(clusterB)},
	}
	base := func(autoAssignment types.Bool, assignments []commitmentAssignmentModel) commitmentModel {
		return commitmentModel{
			Name:           types.StringValue("test-sp"),
			Cloud:          types.StringValue("AWS"),
			Type:           types.StringValue(string(pricing.CommitmentTypeSAVINGSPLAN)),
			StartTime:      types.StringValue("2026-01-01T00:00:00Z"),
			AutoAssignment: autoAssignment,
			Assignments:    assignments,
		}
	}

	tests := []struct {
		name        string
		commitment  commitmentModel
		expectError bool
	}{
		{
			name:        "assignments without auto_assignment → error",
			commitment:  base(types.BoolNull(), assignments),
			expectError: true,
		},
		{
			name:        "assignments with auto_assignment=true → error",
			commitment:  base(types.BoolValue(true), assignments),
			expectError: true,
		},
		{
			name:        "assignments with auto_assignment=false → ok",
			commitment:  base(types.BoolValue(false), assignments),
			expectError: false,
		},
		{
			name:        "assignments with unknown auto_assignment → skipped",
			commitment:  base(types.BoolUnknown(), assignments),
			expectError: false,
		},
		{
			name: "duplicate cluster_id → error",
			commitment: base(types.BoolValue(false), []commitmentAssignmentModel{
				{ClusterID: types.StringValue(clusterA)},
				{ClusterID: types.StringValue(clusterA)},
			}),
			expectError: true,
		},
		{
			name: "unknown cluster_ids skipped in duplicate check → ok",
			commitment: base(types.BoolValue(false), []commitmentAssignmentModel{
				{ClusterID: types.StringUnknown()},
				{ClusterID: types.StringUnknown()},
				{ClusterID: types.StringValue(clusterA)},
			}),
			expectError: false,
		},
		{
			name:        "no assignments, auto_assignment unset → ok",
			commitment:  base(types.BoolNull(), nil),
			expectError: false,
		},
		{
			name:        "empty assignments list, auto_assignment unset → ok",
			commitment:  base(types.BoolNull(), []commitmentAssignmentModel{}),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := validateConfigForTest(t, ctx, tt.commitment)
			require.Equal(t, tt.expectError, hasError)
		})
	}
}

func TestAssignmentsEqual(t *testing.T) {
	a := types.StringValue("cluster-a")
	b := types.StringValue("cluster-b")

	tests := []struct {
		name string
		x, y []commitmentAssignmentModel
		want bool
	}{
		{name: "nil vs nil", want: true},
		{name: "nil vs empty", y: []commitmentAssignmentModel{}, want: false},
		{name: "empty vs nil", x: []commitmentAssignmentModel{}, want: false},
		{name: "empty vs empty", x: []commitmentAssignmentModel{}, y: []commitmentAssignmentModel{}, want: true},
		{
			name: "same members and order (priorities ignored)",
			x:    []commitmentAssignmentModel{{ClusterID: a}, {ClusterID: b}},
			y:    []commitmentAssignmentModel{{ClusterID: a, Priority: types.Int64Value(1)}, {ClusterID: b, Priority: types.Int64Value(2)}},
			want: true,
		},
		{
			name: "different order",
			x:    []commitmentAssignmentModel{{ClusterID: a}, {ClusterID: b}},
			y:    []commitmentAssignmentModel{{ClusterID: b}, {ClusterID: a}},
			want: false,
		},
		{
			name: "different members",
			x:    []commitmentAssignmentModel{{ClusterID: a}},
			y:    []commitmentAssignmentModel{{ClusterID: b}},
			want: false,
		},
		{
			name: "different length",
			x:    []commitmentAssignmentModel{{ClusterID: a}},
			y:    []commitmentAssignmentModel{{ClusterID: a}, {ClusterID: b}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, assignmentsEqual(tt.x, tt.y))
		})
	}
}

func TestAssignmentsChanged(t *testing.T) {
	plan := &commitmentModel{Assignments: []commitmentAssignmentModel{{ClusterID: types.StringValue("cluster-a")}}}

	// state without assignments → replace
	require.True(t, assignmentsChanged(plan, &commitmentModel{}))

	// state with same assignments (computed priorities differ) → no replace
	require.False(t, assignmentsChanged(plan, &commitmentModel{
		Assignments: []commitmentAssignmentModel{{ClusterID: types.StringValue("cluster-a"), Priority: types.Int64Value(1)}},
	}))

	// removal (plan nil, state set) → replace
	require.True(t, assignmentsChanged(&commitmentModel{}, plan))
}

func TestAssignmentsFromReplaceResponse(t *testing.T) {
	resp := &sdk.CommitmentsAPIReplaceCommitmentAssignmentsResponse{
		HTTPResponse: httpResp(http.StatusOK),
		JSON200: &sdk.CastaiInventoryV1beta1ReplaceCommitmentAssignmentsResponse{
			Assignments: &[]sdk.CastaiInventoryV1beta1CommitmentAssignment{
				{ClusterId: ptr("cluster-b"), Priority: ptr(int32(2))},
				{ClusterId: ptr("cluster-a"), Priority: ptr(int32(1))},
				{ClusterId: ptr("cluster-c")}, // nil priority sorts first
			},
		},
	}

	got := assignmentsFromReplaceResponse(resp)
	require.Len(t, got, 3)
	// sorted by server-assigned priority (null = 0 sorts first)
	require.Equal(t, "cluster-c", got[0].ClusterID.ValueString())
	require.Equal(t, "cluster-a", got[1].ClusterID.ValueString())
	require.Equal(t, int64(1), got[1].Priority.ValueInt64())
	require.Equal(t, "cluster-b", got[2].ClusterID.ValueString())
	require.Equal(t, int64(2), got[2].Priority.ValueInt64())

	require.Nil(t, assignmentsFromReplaceResponse(nil))
}

func TestAssignmentsFromPricing_SortsByPriority(t *testing.T) {
	items := []pricing.CommitmentAssignment{
		{ClusterId: ptr("cluster-b"), Priority: ptr(int32(2))},
		{ClusterId: ptr("cluster-a"), Priority: ptr(int32(1))},
		{ClusterId: ptr("cluster-c")}, // nil priority
	}

	got := assignmentsFromPricing(items)
	require.Len(t, got, 3)
	require.Equal(t, "cluster-c", got[0].ClusterID.ValueString())
	require.Equal(t, "cluster-a", got[1].ClusterID.ValueString())
	require.Equal(t, "cluster-b", got[2].ClusterID.ValueString())

	// empty result is a non-nil empty list (managed, zero assignments)
	got = assignmentsFromPricing(nil)
	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestReplaceCommitmentAssignmentsWithRetry_Retries5xxThenSucceeds(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	pricingMock := mock_pricing.NewMockClientWithResponsesInterface(ctrl)
	apiMock := mock_sdk.NewMockClientWithResponsesInterface(ctrl)

	commitmentID := "commit-1"
	clusterIDs := []string{"cluster-a", "cluster-b"}

	gomock.InOrder(
		apiMock.EXPECT().
			CommitmentsAPIReplaceCommitmentAssignmentsWithResponse(gomock.Any(), commitmentID, clusterIDs).
			Return(&sdk.CommitmentsAPIReplaceCommitmentAssignmentsResponse{
				HTTPResponse: httpResp(http.StatusInternalServerError),
				Body:         []byte("server error"),
			}, nil),
		apiMock.EXPECT().
			CommitmentsAPIReplaceCommitmentAssignmentsWithResponse(gomock.Any(), commitmentID, clusterIDs).
			Return(&sdk.CommitmentsAPIReplaceCommitmentAssignmentsResponse{
				HTTPResponse: httpResp(http.StatusOK),
				JSON200: &sdk.CastaiInventoryV1beta1ReplaceCommitmentAssignmentsResponse{
					Assignments: &[]sdk.CastaiInventoryV1beta1CommitmentAssignment{
						{ClusterId: ptr("cluster-a"), Priority: ptr(int32(1))},
						{ClusterId: ptr("cluster-b"), Priority: ptr(int32(2))},
					},
				},
			}, nil),
	)

	res := newCommitmentResourceWithMocks(pricingMock, apiMock)
	apiResp, err := res.replaceCommitmentAssignmentsWithRetry(context.Background(), commitmentID, clusterIDs)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusOK, apiResp.StatusCode())
	r.Len(*apiResp.JSON200.Assignments, 2)
}

func TestReplaceCommitmentAssignmentsWithRetry_4xxStopsImmediately(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	pricingMock := mock_pricing.NewMockClientWithResponsesInterface(ctrl)
	apiMock := mock_sdk.NewMockClientWithResponsesInterface(ctrl)

	commitmentID := "commit-1"
	clusterIDs := []string{"cluster-a"}

	apiMock.EXPECT().
		CommitmentsAPIReplaceCommitmentAssignmentsWithResponse(gomock.Any(), commitmentID, clusterIDs).
		Return(&sdk.CommitmentsAPIReplaceCommitmentAssignmentsResponse{
			HTTPResponse: httpResp(http.StatusBadRequest),
			Body:         []byte(`cannot assign commitment in region "us-east-1" to cluster in region "europe-west1"`),
		}, nil)

	res := newCommitmentResourceWithMocks(pricingMock, apiMock)
	apiResp, err := res.replaceCommitmentAssignmentsWithRetry(context.Background(), commitmentID, clusterIDs)

	r.NoError(err)
	r.NotNil(apiResp)
	r.Equal(http.StatusBadRequest, apiResp.StatusCode())
}

func TestListCommitmentAssignments_Pagination(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	ctrl := gomock.NewController(t)
	pricingMock := mock_pricing.NewMockClientWithResponsesInterface(ctrl)

	orgID, commitmentID := "org-1", "commit-1"

	page := func(nextToken string, items ...pricing.CommitmentAssignment) *pricing.CommitmentsAPIListCommitmentAssignmentsResponse {
		var np *string
		if nextToken != "" {
			np = ptr(nextToken)
		}
		return &pricing.CommitmentsAPIListCommitmentAssignmentsResponse{
			HTTPResponse: httpResp(http.StatusOK),
			JSON200: &pricing.ListCommitmentAssignmentsResponse{
				CommitmentAssignments: &items,
				NextPageToken:         np,
			},
		}
	}

	gomock.InOrder(
		pricingMock.EXPECT().
			CommitmentsAPIListCommitmentAssignmentsWithResponse(gomock.Any(), orgID, commitmentID, gomock.Any()).
			Return(page("page-2",
				pricing.CommitmentAssignment{ClusterId: ptr("cluster-a"), Priority: ptr(int32(1))},
				pricing.CommitmentAssignment{ClusterId: ptr("cluster-b"), Priority: ptr(int32(2))},
			), nil),
		pricingMock.EXPECT().
			CommitmentsAPIListCommitmentAssignmentsWithResponse(gomock.Any(), orgID, commitmentID, gomock.Any()).
			Return(page("",
				pricing.CommitmentAssignment{ClusterId: ptr("cluster-c"), Priority: ptr(int32(3))},
			), nil),
	)

	res := newCommitmentResourceWithMocks(pricingMock, nil)
	items, err := res.listCommitmentAssignments(context.Background(), orgID, commitmentID)

	r.NoError(err)
	r.Len(items, 3)
	r.Equal("cluster-a", *items[0].ClusterId)
	r.Equal("cluster-b", *items[1].ClusterId)
	r.Equal("cluster-c", *items[2].ClusterId)
}

func testAccCommitmentConfigWithAssignments(name, cudID, organizationID, clusterID string) string {
	return fmt.Sprintf(`
resource "castai_commitment" "test" {
  name               = %[1]q
  organization_id    = %[3]q
  cloud              = "GCP"
  region             = "us-central1"
  type               = "RESOURCE_CUD"
  start_time         = "2026-01-01T00:00:00Z"
  end_time           = "2027-01-01T00:00:00Z"
  autoscaling_status = "ACTIVE"
  allowed_usage      = 0.75
  auto_assignment    = false

  assignments = [
    {
      cluster_id = %[4]q
    },
  ]

  gcp_resource_cud_details = {
    cud_id    = %[2]q
    plan      = "TWELVE_MONTH"
    type      = "GENERAL_PURPOSE_E2"
    cpu       = 32
    memory_mb = 131072
    status    = "ACTIVE"
  }
}
`, name, cudID, organizationID, clusterID)
}
