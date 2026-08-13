package castai

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
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
	resourceName := "castai_commitment.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with auto_assignment not set — server decides.
				Config: testAccCommitmentConfig(rName, cudID, "INACTIVE", 1.0, false, false),
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
				Config: testAccCommitmentConfig(rName, cudID, "ACTIVE", 0.75, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "autoscaling_status", "ACTIVE"),
					resource.TestCheckResourceAttr(resourceName, "allowed_usage", "0.75"),
				),
			},
			{
				// Upsert-path update: rename. ID must be preserved.
				Config: testAccCommitmentConfig(rName+"-renamed", cudID, "ACTIVE", 0.75, false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", rName+"-renamed"),
				),
			},
			{
				// Switch to explicit auto_assignment=false.
				Config: testAccCommitmentConfig(rName+"-renamed", cudID, "ACTIVE", 0.75, true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "auto_assignment", "false"),
				),
			},
			{
				// Upsert-path update with explicit false: auto_assignment must stay false.
				Config: testAccCommitmentConfig(rName+"-renamed-2", cudID, "ACTIVE", 0.75, true, false),
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

func testAccCommitmentConfig(name, cudID, autoscalingStatus string, allowedUsage float64, setAutoAssignment bool, autoAssignment bool) string {
	autoAssignmentLine := ""
	if setAutoAssignment {
		autoAssignmentLine = fmt.Sprintf("auto_assignment = %v", autoAssignment)
	}
	return fmt.Sprintf(`
resource "castai_commitment" "test" {
  name               = %[1]q
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
`, name, cudID, autoscalingStatus, allowedUsage, autoAssignmentLine)
}

// ---------------------------------------------------------------------------
// Retry unit tests — mock-based tests for getCommitmentWithRetry,
// createCommitmentWithRetry, and deleteCommitmentWithRetry.
// ---------------------------------------------------------------------------

// newCommitmentResourceWithMock creates a genericCommitmentResource whose
// pricingClient is the supplied mock. All other ProviderConfig fields are
// left zero — the retry helpers only touch pricingClient.
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

// ---------------------------------------------------------------------------
// getCommitmentWithRetry
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// createCommitmentWithRetry
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// deleteCommitmentWithRetry
// ---------------------------------------------------------------------------

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


