package castai

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	mock_sdk "github.com/castai/terraform-provider-castai/castai/sdk/mock"
)

// Both create and update use the same "upsert" handler under the hood so we test them together
func TestCommitmentsResourceCreateAndUpdate(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)

	orgID, clusterID, commitmentID := uuid.New(), uuid.New(), uuid.New()

	type test struct {
		resource               map[string]any
		commitmentImport       any // CastaiInventoryV1beta1GCPCommitmentImport | CastaiInventoryV1beta1AzureReservationImport
		expectCommitmentUpdate sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody
		mockImportedCommitment sdk.CastaiInventoryV1beta1Commitment
	}

	gcpImport := sdk.CastaiInventoryV1beta1GCPCommitmentImport{
		AutoRenew:         lo.ToPtr(true),
		Category:          lo.ToPtr("MACHINE"),
		CreationTimestamp: lo.ToPtr("2023-01-01T00:00:00Z"),
		Description:       lo.ToPtr("some description"),
		EndTimestamp:      lo.ToPtr("2024-01-01T00:00:00Z"),
		Id:                lo.ToPtr("123456789"),
		Kind:              lo.ToPtr("compute#commitment"),
		Name:              lo.ToPtr("test"),
		Plan:              lo.ToPtr("TWELVE_MONTH"),
		Region:            lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1"),
		Resources: &[]sdk.CastaiInventoryV1beta1GCPResource{
			{Amount: lo.ToPtr("10"), Type: lo.ToPtr("VCPU")},
			{Amount: lo.ToPtr("20480"), Type: lo.ToPtr("MEMORY")},
		},
		SelfLink:       lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1/commitments/test"),
		StartTimestamp: lo.ToPtr("2023-01-01T00:00:00Z"),
		Status:         lo.ToPtr("ACTIVE"),
		StatusMessage:  lo.ToPtr("The commitment is active, and so will apply to current resource usage."),
		Type:           lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
	}

	// Note that the import endpoints are called in "overwrite" mode. This means that we don't need to test scenarios
	// where certain commitments already exist and some of them need to be deleted because they don't exist anymore
	// in the new plan.
	tests := map[string]test{
		"should create a gcp commitment": func() test {
			return test{
				resource: map[string]any{
					fieldCommitmentsGCPCUDsJSON: toJSONString(r, []sdk.CastaiInventoryV1beta1GCPCommitmentImport{gcpImport}),
					fieldCommitmentsConfigs: []any{
						map[string]any{
							"matcher": []any{
								map[string]any{
									"name":   "test",
									"type":   "COMPUTE_OPTIMIZED_C2D",
									"region": "us-central1",
								},
							},
							"assignments": []any{
								map[string]any{
									"cluster_id": clusterID.String(),
									"priority":   1,
								},
							},
							"prioritization":   true,
							"status":           "Active",
							"allowed_usage":    0.6,
							"scaling_strategy": "CPUBased",
						},
					},
				},
				commitmentImport: gcpImport,
				expectCommitmentUpdate: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
					AllowedUsage:    lo.ToPtr[float32](0.6),
					Prioritization:  lo.ToPtr(true),
					ScalingStrategy: lo.ToPtr(sdk.CPUBased),
					Status:          lo.ToPtr(sdk.CastaiInventoryV1beta1CommitmentStatusActive),
				},
				mockImportedCommitment: sdk.CastaiInventoryV1beta1Commitment{
					EndDate: lo.ToPtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
					Id:      lo.ToPtr(commitmentID.String()),
					Name:    lo.ToPtr("test"),
					GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{
						Cpu:      lo.ToPtr("10"),
						CudId:    lo.ToPtr("123456789"),
						MemoryMb: lo.ToPtr("20480"),
						Plan:     lo.ToPtr(sdk.CastaiInventoryV1beta1GCPResourceCUDCUDPlanTWELVEMONTH),
						Status:   lo.ToPtr("Active"),
						Type:     lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
					},
					Region:       lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1"),
					StartDate:    lo.ToPtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
					Status:       lo.ToPtr(sdk.CastaiInventoryV1beta1CommitmentStatusActive),
					AllowedUsage: lo.ToPtr[float32](1),
				},
			}
		}(),
		"should create an azure commitment": func() test {
			commitmentImport := sdk.CastaiInventoryV1beta1AzureReservationImport{
				ExpirationDate:     lo.ToPtr("2050-01-01T00:00:00Z"),
				Name:               lo.ToPtr("test"),
				ProductName:        lo.ToPtr("Standard_D32as_v4"),
				PurchaseDate:       lo.ToPtr("2023-01-11T00:00:00Z"),
				Quantity:           lo.ToPtr[int32](3),
				Region:             lo.ToPtr("eastus"),
				ReservationId:      lo.ToPtr("3b3de39c-bc44-4d69-be2d-69527dfe9958"),
				Scope:              lo.ToPtr("Single subscription"),
				ScopeResourceGroup: lo.ToPtr("All resource groups"),
				ScopeSubscription:  lo.ToPtr("8faa0959-093b-4612-8686-a996ac19db00"),
				Status:             lo.ToPtr("Succeeded"),
				Term:               lo.ToPtr("P3Y"),
				Type:               lo.ToPtr("VirtualMachines"),
			}

			return test{
				resource: map[string]any{
					fieldCommitmentsAzureReservationsCSV: `Name,Reservation Id,Reservation order Id,Status,Expiration date,Purchase date,Term,Scope,Scope subscription,Scope resource group,Type,Product name,Region,Quantity,Utilization % 1 Day,Utilization % 7 Day,Utilization % 30 Day,Deep link to reservation
test,3b3de39c-bc44-4d69-be2d-69527dfe9958,630226bb-5170-4b95-90b0-f222757130c1,Succeeded,2050-01-01T00:00:00Z,2023-01-11T00:00:00Z,P3Y,Single subscription,8faa0959-093b-4612-8686-a996ac19db00,All resource groups,VirtualMachines,Standard_D32as_v4,eastus,3,100,100,100,https://portal.azure.com#resource/providers/microsoft.capacity/reservationOrders/59791a62-264b-4b9f-aa3a-5eeb761e4583/reservations/883afd52-54c8-4bc6-a0f2-ccbaf7b84bda/overview`,
					fieldCommitmentsConfigs: []any{
						map[string]any{
							"matcher": []any{
								map[string]any{
									"name":   "test",
									"type":   "Standard_D32as_v4",
									"region": "eastus",
								},
							},
							"assignments": []any{
								map[string]any{
									"cluster_id": clusterID.String(),
									"priority":   1,
								},
							},
							"prioritization":   true,
							"status":           "Active",
							"allowed_usage":    0.7,
							"scaling_strategy": "Default",
						},
					},
				},
				commitmentImport: commitmentImport,
				expectCommitmentUpdate: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
					AllowedUsage:    lo.ToPtr[float32](0.7),
					Prioritization:  lo.ToPtr(true),
					ScalingStrategy: lo.ToPtr(sdk.Default),
					Status:          lo.ToPtr(sdk.CastaiInventoryV1beta1CommitmentStatusActive),
				},
				mockImportedCommitment: sdk.CastaiInventoryV1beta1Commitment{
					EndDate: lo.ToPtr(time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC)),
					Id:      lo.ToPtr(commitmentID.String()),
					Name:    lo.ToPtr("test"),
					Region:  lo.ToPtr("eastus"),
					AzureReservationContext: &sdk.CastaiInventoryV1beta1AzureReservation{
						Count:                 lo.ToPtr[int32](3),
						Id:                    lo.ToPtr("3b3de39c-bc44-4d69-be2d-69527dfe9958"),
						InstanceType:          lo.ToPtr("Standard_D32as_v4"),
						InstanceTypeCpu:       lo.ToPtr("32"),
						InstanceTypeMemoryMib: lo.ToPtr("131072"),
						Plan:                  lo.ToPtr(sdk.THREEYEAR),
						Scope:                 lo.ToPtr("Single subscription"),
						ScopeResourceGroup:    lo.ToPtr("All resource groups"),
						ScopeSubscription:     lo.ToPtr("8faa0959-093b-4612-8686-a996ac19db00"),
						Status:                lo.ToPtr("Succeeded"),
					},
				},
			}
		}(),
	}

	type testedFn string
	const (
		testedFnCreate testedFn = "create"
		testedFnUpdate testedFn = "update"
	)
	for _, testedFn := range []testedFn{testedFnCreate, testedFnUpdate} {
		t.Run(string(testedFn), func(t *testing.T) {
			for name, tt := range tests {
				t.Run(name, func(t *testing.T) {
					r := require.New(t)

					ctrl := gomock.NewController(t)
					defer ctrl.Finish()

					resource := resourceCommitments()
					mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
					provider := &ProviderConfig{api: mockClient}

					// Fetches the default organization ID
					mockClient.EXPECT().UsersAPIListOrganizationsWithResponse(gomock.Any()).Return(&sdk.UsersAPIListOrganizationsResponse{
						JSON200: &sdk.CastaiUsersV1beta1ListOrganizationsResponse{
							Organizations: []sdk.CastaiUsersV1beta1UserOrganization{
								{Id: lo.ToPtr(orgID.String())}, // the first org is the default one so everything else should be ignored
								{Id: lo.ToPtr(uuid.New().String())},
							},
						},
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil).Times(1)

					data := schema.TestResourceDataRaw(t, resource.Schema, tt.resource)

					// Actual commitments import call
					switch v := tt.commitmentImport.(type) {
					case sdk.CastaiInventoryV1beta1GCPCommitmentImport:
						mockClient.EXPECT().CommitmentsAPIImportGCPCommitmentsWithResponse(
							gomock.Any(),
							&sdk.CommitmentsAPIImportGCPCommitmentsParams{
								Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportGCPCommitmentsParamsBehaviour]("OVERWRITE"),
							},
							[]sdk.CastaiInventoryV1beta1GCPCommitmentImport{v},
						).Return(&sdk.CommitmentsAPIImportGCPCommitmentsResponse{
							HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						}, nil).Times(1)
					case sdk.CastaiInventoryV1beta1AzureReservationImport:
						mockClient.EXPECT().CommitmentsAPIImportAzureReservationsWithResponse(
							gomock.Any(),
							&sdk.CommitmentsAPIImportAzureReservationsParams{
								Behaviour: lo.ToPtr[sdk.CommitmentsAPIImportAzureReservationsParamsBehaviour]("OVERWRITE"),
							},
							[]sdk.CastaiInventoryV1beta1AzureReservationImport{v},
						).Return(&sdk.CommitmentsAPIImportAzureReservationsResponse{
							HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						}, nil).Times(1)
					}

					// There are 2 get commitments calls: one during the creation and one by the state importer
					mockClient.EXPECT().CommitmentsAPIGetCommitmentsWithResponse(
						gomock.Any(), &sdk.CommitmentsAPIGetCommitmentsParams{},
					).Return(&sdk.CommitmentsAPIGetCommitmentsResponse{
						JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsResponse{
							Commitments: &[]sdk.CastaiInventoryV1beta1Commitment{tt.mockImportedCommitment},
						},
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil).Times(2)

					// Update is called after importing the commitments to set fields such as status,
					// allowed usage, etc. specified in the config
					mockClient.EXPECT().CommitmentsAPIUpdateCommitmentWithResponse(
						gomock.Any(), commitmentID.String(), tt.expectCommitmentUpdate,
					).Return(&sdk.CommitmentsAPIUpdateCommitmentResponse{
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200:      &sdk.CastaiInventoryV1beta1UpdateCommitmentResponse{},
					}, nil).Times(1)

					// Assignments replace is called to assign the commitment to clusters specified in the config
					mockClient.EXPECT().CommitmentsAPIReplaceCommitmentAssignmentsWithResponse(
						gomock.Any(),
						commitmentID.String(),
						sdk.CommitmentsAPIReplaceCommitmentAssignmentsJSONRequestBody{clusterID.String()},
					).Return(&sdk.CommitmentsAPIReplaceCommitmentAssignmentsResponse{
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						JSON200:      &sdk.CastaiInventoryV1beta1ReplaceCommitmentAssignmentsResponse{},
					}, nil).Times(1)

					// Commitment assignments are fetched by the state importer
					mockClient.EXPECT().CommitmentsAPIGetCommitmentsAssignmentsWithResponse(gomock.Any()).
						Return(&sdk.CommitmentsAPIGetCommitmentsAssignmentsResponse{
							JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsAssignmentsResponse{
								CommitmentsAssignments: &[]sdk.CastaiInventoryV1beta1CommitmentAssignment{},
							},
							HTTPResponse: &http.Response{StatusCode: http.StatusOK},
						}, nil).Times(1)

					var fn func(context.Context, *schema.ResourceData, any) diag.Diagnostics
					switch testedFn {
					case testedFnCreate:
						fn = resource.CreateContext
					case testedFnUpdate:
						fn = resource.UpdateContext
					default:
						r.Failf("unexpected tested function: %s", string(testedFn))
					}

					diag := fn(ctx, data, provider)
					noErrInDiagnostics(r, diag)
				})
			}
		})
	}
}

func TestCommitmentsResourceRead(t *testing.T) {
	ctx := context.Background()
	orgID, clusterID, commitment1ID, commitment2ID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	tests := map[string]struct {
		stateKey    string
		id          string
		commitments []sdk.CastaiInventoryV1beta1Commitment
		assignments []sdk.CastaiInventoryV1beta1CommitmentAssignment
		checkState  func(*require.Assertions, any)
	}{
		"should import gcp commitments with assignments": {
			stateKey: fieldCommitmentsGCPCUDs,
			id:       orgID.String() + ":gcp",
			// Mapper functions are tested in their own unit tests, hence we only do basic checks here
			commitments: []sdk.CastaiInventoryV1beta1Commitment{
				{
					Id:                    lo.ToPtr(commitment1ID.String()),
					GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{},
				},
				{
					Id:                    lo.ToPtr(commitment2ID.String()),
					GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{},
				},
			},
			assignments: []sdk.CastaiInventoryV1beta1CommitmentAssignment{
				{
					ClusterId:    lo.ToPtr(clusterID.String()),
					CommitmentId: lo.ToPtr(commitment1ID.String()),
				},
				{
					ClusterId:    lo.ToPtr(clusterID.String()),
					CommitmentId: lo.ToPtr(commitment2ID.String()),
				},
			},
			checkState: func(r *require.Assertions, v any) {
				var parsed []gcpCUDResource
				r.NoError(mapstructure.Decode(v, &parsed))
				r.Len(parsed, 2)

				r.Equal(commitment1ID.String(), parsed[0].getCommitmentID())
				r.Len(parsed[0].Assignments, 1)
				r.Equal(clusterID.String(), parsed[0].Assignments[0].ClusterID)

				r.Equal(commitment2ID.String(), parsed[1].getCommitmentID())
				r.Len(parsed[1].Assignments, 1)
				r.Equal(clusterID.String(), parsed[1].Assignments[0].ClusterID)
			},
		},
		"should import azure commitments with no assignments": {
			stateKey: fieldCommitmentsAzureReservations,
			id:       orgID.String() + ":azure",
			commitments: []sdk.CastaiInventoryV1beta1Commitment{
				{
					Id:                      lo.ToPtr(commitment1ID.String()),
					AzureReservationContext: &sdk.CastaiInventoryV1beta1AzureReservation{},
				},
			},
			checkState: func(r *require.Assertions, v any) {
				var parsed []azureReservationResource
				r.NoError(mapstructure.Decode(v, &parsed))
				r.Len(parsed, 1)
				r.Equal(commitment1ID.String(), parsed[0].getCommitmentID())
				r.Len(parsed[0].Assignments, 0)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := resourceCommitments()

			mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
			provider := &ProviderConfig{api: mockClient}

			data := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{tt.stateKey: []any{}})
			data.SetId(tt.id)

			mockClient.EXPECT().
				CommitmentsAPIGetCommitmentsWithResponse(gomock.Any(), &sdk.CommitmentsAPIGetCommitmentsParams{}).
				Return(&sdk.CommitmentsAPIGetCommitmentsResponse{
					JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsResponse{
						Commitments: &tt.commitments,
					},
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				}, nil).
				Times(1)

			mockClient.EXPECT().
				CommitmentsAPIGetCommitmentsAssignmentsWithResponse(gomock.Any()).
				Return(&sdk.CommitmentsAPIGetCommitmentsAssignmentsResponse{
					JSON200: &sdk.CastaiInventoryV1beta1GetCommitmentsAssignmentsResponse{
						CommitmentsAssignments: &tt.assignments,
					},
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				}, nil).
				Times(1)

			diag := resource.ReadContext(ctx, data, provider)
			noErrInDiagnostics(r, diag)

			v := data.Get(tt.stateKey)
			if tt.checkState != nil {
				tt.checkState(r, v)
			}
		})
	}
}

func TestCommitmentsResourceDelete(t *testing.T) {
	ctx := context.Background()
	orgID, commitmentID := uuid.New(), uuid.New()

	tests := map[string]struct {
		resource     map[string]any
		expectDelete bool
	}{
		"should be a no-op when no commitments are present": {
			resource: map[string]any{},
		},
		"should delete gcp commitments resource": {
			resource: map[string]any{
				fieldCommitmentsGCPCUDs: []any{
					map[string]any{
						"id": commitmentID.String(),
					},
				},
			},
			expectDelete: true,
		},
		"should delete azure commitments resource": {
			resource: map[string]any{
				fieldCommitmentsAzureReservations: []any{
					map[string]any{
						"id": commitmentID.String(),
					},
				},
			},
			expectDelete: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := resourceCommitments()
			mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
			provider := &ProviderConfig{api: mockClient}

			data := schema.TestResourceDataRaw(t, resource.Schema, tt.resource)

			// Fetches the default organization ID to get commitments import ID
			mockClient.EXPECT().UsersAPIListOrganizationsWithResponse(gomock.Any()).Return(&sdk.UsersAPIListOrganizationsResponse{
				JSON200: &sdk.CastaiUsersV1beta1ListOrganizationsResponse{
					Organizations: []sdk.CastaiUsersV1beta1UserOrganization{
						{Id: lo.ToPtr(orgID.String())}, // the first org is the default one so everything else should be ignored
						{Id: lo.ToPtr(uuid.New().String())},
					},
				},
				HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			}, nil).Times(1)

			if tt.expectDelete {
				mockClient.EXPECT().
					CommitmentsAPIDeleteCommitmentWithResponse(gomock.Any(), commitmentID.String()).
					Return(&sdk.CommitmentsAPIDeleteCommitmentResponse{
						JSON200:      &map[string]any{},
						HTTPResponse: &http.Response{StatusCode: http.StatusOK},
					}, nil).
					Times(1)
			}

			diag := resource.DeleteContext(ctx, data, provider)
			noErrInDiagnostics(r, diag)
		})
	}
}

func TestGetCommitmentsImportID(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	tests := map[string]struct {
		resourceData                 map[string]any
		expectOrganizationsAPICalled bool
		expectedCloud                string
	}{
		"should use provided organization_id for Azure": {
			resourceData: map[string]any{
				fieldCommitmentsOrganizationId:       orgID.String(),
				fieldCommitmentsAzureReservationsCSV: "dummy-csv",
			},
			expectOrganizationsAPICalled: false,
			expectedCloud:                "azure",
		},
		"should use provided organization_id for GCP": {
			resourceData: map[string]any{
				fieldCommitmentsOrganizationId: orgID.String(),
				fieldCommitmentsGCPCUDsJSON:    "dummy-json",
			},
			expectOrganizationsAPICalled: false,
			expectedCloud:                "gcp",
		},
		"should fetch organization_id when not provided": {
			resourceData: map[string]any{
				fieldCommitmentsAzureReservationsCSV: "dummy-csv",
			},
			expectOrganizationsAPICalled: true,
			expectedCloud:                "azure",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resource := resourceCommitments()
			mockClient := mock_sdk.NewMockClientWithResponsesInterface(ctrl)
			provider := &ProviderConfig{api: mockClient}

			data := schema.TestResourceDataRaw(t, resource.Schema, tt.resourceData)

			// Set expectation for /v1/organizations call
			if tt.expectOrganizationsAPICalled {
				mockClient.EXPECT().UsersAPIListOrganizationsWithResponse(gomock.Any()).Return(&sdk.UsersAPIListOrganizationsResponse{
					JSON200: &sdk.CastaiUsersV1beta1ListOrganizationsResponse{
						Organizations: []sdk.CastaiUsersV1beta1UserOrganization{
							{Id: lo.ToPtr(orgID.String())},
						},
					},
					HTTPResponse: &http.Response{StatusCode: http.StatusOK},
				}, nil).Times(1)
			} else {
				// Verify it's NOT called
				mockClient.EXPECT().UsersAPIListOrganizationsWithResponse(gomock.Any()).Times(0)
			}

			// Call the function under test
			importID, err := getCommitmentsImportID(ctx, data, provider)

			// Verify results
			r.NoError(err)
			expectedID := orgID.String() + ":" + tt.expectedCloud
			r.Equal(expectedID, importID)
		})
	}
}

func noErrInDiagnostics(r *require.Assertions, diags diag.Diagnostics) {
	for _, d := range diags {
		if d.Severity == diag.Error {
			r.Failf("unexpected error: %s", d.Summary)
		}
	}
}

func toJSONString(r *require.Assertions, v any) string {
	raw, err := json.Marshal(v)
	r.NoError(err)
	return string(raw)
}
