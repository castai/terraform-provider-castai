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

	"github.com/castai/terraform-provider-castai/castai/sdk/pricing"
	mock_pricing "github.com/castai/terraform-provider-castai/castai/sdk/pricing/mock"
)

func TestAccCloudAgnostic_Commitment(t *testing.T) {
	rName := fmt.Sprintf("%v-commitment-%v", ResourcePrefix, acctest.RandString(8))
	cudID := acctest.RandStringFromCharSet(12, "0123456789")
	organizationID := os.Getenv("ACCEPTANCE_TEST_ORGANIZATION_ID")
	resourceName := "castai_commitment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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
			{
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
			},
		},
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
	return &genericCommitmentResource{
		client: &ProviderConfig{
			pricingClient: mockClient,
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
