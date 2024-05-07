package castai

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func TestMapCommitmentToCUDResource(t *testing.T) {
	id1 := uuid.New()
	now := time.Now()
	startTs, endTs := now.Add(365*24*time.Hour), now.Add(-24*time.Hour)

	makeCommitment := func() sdk.CastaiInventoryV1beta1Commitment {
		return sdk.CastaiInventoryV1beta1Commitment{
			AllowedUsage: lo.ToPtr[float32](0.5),
			EndDate:      lo.ToPtr(startTs),
			GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{
				Cpu:      lo.ToPtr("8"),
				CudId:    lo.ToPtr("123456"),
				MemoryMb: lo.ToPtr("1024"),
				Plan:     lo.ToPtr[sdk.CastaiInventoryV1beta1GCPResourceCUDCUDPlan]("TWELVE_MONTHS"),
				Status:   lo.ToPtr("ACTIVE"),
				Type:     lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
			},
			Id:             lo.ToPtr(id1.String()),
			Name:           lo.ToPtr("test-cud"),
			Prioritization: lo.ToPtr(true),
			Region:         lo.ToPtr("us-central1"),
			StartDate:      lo.ToPtr(endTs),
			Status:         lo.ToPtr[sdk.CastaiInventoryV1beta1CommitmentStatus]("ACTIVE"),
			UpdatedAt:      lo.ToPtr(now),
		}
	}

	tests := map[string]struct {
		input    sdk.CastaiInventoryV1beta1Commitment
		expected *GCPCUDResource
		err      error
	}{
		"should succeed as all the fields are set": {
			input: makeCommitment(),
			expected: &GCPCUDResource{
				CASTCommitmentFields: CASTCommitmentFields{
					ID:             lo.ToPtr(id1.String()),
					AllowedUsage:   lo.ToPtr[float32](0.5),
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
				},
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: endTs.Format(time.RFC3339),
				EndTimestamp:   startTs.Format(time.RFC3339),
				Name:           "test-cud",
				Region:         "us-central1",
				CPU:            8,
				MemoryMb:       1024,
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
			},
		},
		"should fail as gcp cud context is nil": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext = nil
				return c
			}(),
			err: errors.New("missing GCP resource CUD context"),
		},
		"should succeed with nil cpu": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext.Cpu = nil
				return c
			}(),
			expected: &GCPCUDResource{
				CASTCommitmentFields: CASTCommitmentFields{
					ID:             lo.ToPtr(id1.String()),
					AllowedUsage:   lo.ToPtr[float32](0.5),
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
				},
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: endTs.Format(time.RFC3339),
				EndTimestamp:   startTs.Format(time.RFC3339),
				Name:           "test-cud",
				Region:         "us-central1",
				MemoryMb:       1024,
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
			},
		},
		"should succeed with nil memory": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext.MemoryMb = nil
				return c
			}(),
			expected: &GCPCUDResource{
				CASTCommitmentFields: CASTCommitmentFields{
					ID:             lo.ToPtr(id1.String()),
					AllowedUsage:   lo.ToPtr[float32](0.5),
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
				},
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: endTs.Format(time.RFC3339),
				EndTimestamp:   startTs.Format(time.RFC3339),
				Name:           "test-cud",
				Region:         "us-central1",
				CPU:            8,
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
			},
		},
		"should fail as cpu is an invalid string": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext.Cpu = lo.ToPtr("invalid")
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
		"should fail as memory is an invalid string": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext.MemoryMb = lo.ToPtr("invalid")
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapCommitmentToCUDResource(tt.input)
			if tt.err == nil {
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)
			} else {
				r.Nil(actual)
				r.Error(err)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}

func TestMapCommitmentToReservationResource(t *testing.T) {
	id1 := uuid.New()
	now := time.Now()
	startTs, endTs := now.Add(365*24*time.Hour), now.Add(-24*time.Hour)

	reservationID, scopeSubscription := uuid.New(), uuid.New()

	makeCommitment := func() sdk.CastaiInventoryV1beta1Commitment {
		return sdk.CastaiInventoryV1beta1Commitment{
			AllowedUsage: lo.ToPtr[float32](0.5),
			EndDate:      lo.ToPtr(startTs),
			AzureReservationContext: &sdk.CastaiInventoryV1beta1AzureReservation{
				Count:              lo.ToPtr[int32](2),
				Id:                 lo.ToPtr(reservationID.String()),
				InstanceType:       lo.ToPtr("Standard_D32as_v4"),
				Plan:               lo.ToPtr[sdk.CastaiInventoryV1beta1AzureReservationReservationPlan]("THREE_YEAR"),
				Scope:              lo.ToPtr("Single subscription"),
				ScopeResourceGroup: lo.ToPtr("All resource groups"),
				ScopeSubscription:  lo.ToPtr(scopeSubscription.String()),
				Status:             lo.ToPtr("Succeeded"),
			},
			Id:             lo.ToPtr(id1.String()),
			Name:           lo.ToPtr("test-reservation"),
			Prioritization: lo.ToPtr(true),
			Region:         lo.ToPtr("eastus"),
			StartDate:      lo.ToPtr(endTs),
			Status:         lo.ToPtr[sdk.CastaiInventoryV1beta1CommitmentStatus]("ACTIVE"),
			UpdatedAt:      lo.ToPtr(now),
		}
	}

	tests := map[string]struct {
		input    sdk.CastaiInventoryV1beta1Commitment
		expected *AzureReservationResource
		err      error
	}{
		"should succeed as all the fields are set": {
			input: makeCommitment(),
			expected: &AzureReservationResource{
				CASTCommitmentFields: CASTCommitmentFields{
					ID:             lo.ToPtr(id1.String()),
					AllowedUsage:   lo.ToPtr[float32](0.5),
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
				},
				Count:              2,
				ReservationID:      reservationID.String(),
				ReservationStatus:  "Succeeded",
				StartTimestamp:     endTs.Format(time.RFC3339),
				EndTimestamp:       startTs.Format(time.RFC3339),
				Name:               "test-reservation",
				Region:             "eastus",
				InstanceType:       "Standard_D32as_v4",
				Plan:               "THREE_YEAR",
				Scope:              "Single subscription",
				ScopeResourceGroup: "All resource groups",
				ScopeSubscription:  scopeSubscription.String(),
			},
		},
		"should fail as azure reservation context is nil": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.AzureReservationContext = nil
				return c
			}(),
			err: errors.New("missing azure resource reservation context"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapCommitmentToReservationResource(tt.input)
			if tt.err == nil {
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)
			} else {
				r.Nil(actual)
				r.Error(err)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}

func TestMapCUDImportToResource(t *testing.T) {
	makeInput := func() *commitmentWithConfig[CastaiGCPCommitmentImport] {
		return &commitmentWithConfig[CastaiGCPCommitmentImport]{
			Commitment: CastaiGCPCommitmentImport{
				CastaiInventoryV1beta1GCPCommitmentImport: testGCPCommitmentImport,
			},
		}
	}

	tests := map[string]struct {
		input    *commitmentWithConfig[CastaiGCPCommitmentImport]
		expected *GCPCUDResource
		err      error
	}{
		"should succeed, no config passed": {
			input: makeInput(),
			expected: &GCPCUDResource{
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: "2023-01-01T00:00:00.000-07:00",
				EndTimestamp:   "2024-01-01T00:00:00.000-07:00",
				Name:           "test-cud",
				Region:         "us-central1",
				CPU:            10,
				MemoryMb:       20480,
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
			},
		},
		"should succeed, nil cud resources": {
			input: func() *commitmentWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.Commitment.Resources = nil
				return c
			}(),
			expected: &GCPCUDResource{
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: "2023-01-01T00:00:00.000-07:00",
				EndTimestamp:   "2024-01-01T00:00:00.000-07:00",
				Name:           "test-cud",
				Region:         "us-central1",
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
			},
		},
		"should succeed, with a config passed": {
			input: func() *commitmentWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.Config = &CommitmentConfigResource{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name:   "test-cud",
							Type:   lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
							Region: lo.ToPtr("us-central1"),
						},
					},
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.7),
				}
				return c
			}(),
			expected: &GCPCUDResource{
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: "2023-01-01T00:00:00.000-07:00",
				EndTimestamp:   "2024-01-01T00:00:00.000-07:00",
				Name:           "test-cud",
				Region:         "us-central1",
				CPU:            10,
				MemoryMb:       20480,
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
				// Configured fields
				CASTCommitmentFields: CASTCommitmentFields{
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.7),
				},
			},
		},
		"should fail as cpu amount is invalid": {
			input: func() *commitmentWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				inv := (*c.Commitment.Resources)[0]
				inv.Amount = lo.ToPtr("invalid")
				c.Commitment.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{inv, (*c.Commitment.Resources)[1]}
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
		"should fail as memory amount is invalid": {
			input: func() *commitmentWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				inv := (*c.Commitment.Resources)[1]
				inv.Amount = lo.ToPtr("invalid")
				c.Commitment.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{(*c.Commitment.Resources)[0], inv}
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
		"should succeed with zeroed out cpu as its resource is not contained by the resources": {
			input: func() *commitmentWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.Commitment.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{(*c.Commitment.Resources)[1]}
				return c
			}(),
			expected: &GCPCUDResource{
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: "2023-01-01T00:00:00.000-07:00",
				EndTimestamp:   "2024-01-01T00:00:00.000-07:00",
				Name:           "test-cud",
				Region:         "us-central1",
				MemoryMb:       20480,
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
			},
		},
		"should succeed with zeroed out memory as its resource is not contained by the resources": {
			input: func() *commitmentWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.Commitment.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{(*c.Commitment.Resources)[0]}
				return c
			}(),
			expected: &GCPCUDResource{
				CUDID:          "123456",
				CUDStatus:      "ACTIVE",
				StartTimestamp: "2023-01-01T00:00:00.000-07:00",
				EndTimestamp:   "2024-01-01T00:00:00.000-07:00",
				Name:           "test-cud",
				Region:         "us-central1",
				CPU:            10,
				Plan:           "TWELVE_MONTHS",
				Type:           "COMPUTE_OPTIMIZED_C2D",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapCUDImportToResource(tt.input)
			if tt.err == nil {
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)
			} else {
				r.Nil(actual)
				r.Error(err)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}

func TestMapReservationImportToResource(t *testing.T) {
	makeInput := func() *commitmentWithConfig[CastaiAzureReservationImport] {
		return &commitmentWithConfig[CastaiAzureReservationImport]{
			Commitment: CastaiAzureReservationImport{
				CastaiInventoryV1beta1AzureReservationImport: testAzureCommitmentImport,
			},
		}
	}

	tests := map[string]struct {
		input    *commitmentWithConfig[CastaiAzureReservationImport]
		expected *AzureReservationResource
		err      error
	}{
		"should succeed, no config passed": {
			input: makeInput(),
			expected: &AzureReservationResource{
				Count:              2,
				ReservationID:      lo.FromPtr(testAzureCommitmentImport.ReservationId),
				ReservationStatus:  lo.FromPtr(testAzureCommitmentImport.Status),
				StartTimestamp:     lo.FromPtr(testAzureCommitmentImport.PurchaseDate),
				EndTimestamp:       lo.FromPtr(testAzureCommitmentImport.ExpirationDate),
				Name:               lo.FromPtr(testAzureCommitmentImport.Name),
				Region:             lo.FromPtr(testAzureCommitmentImport.Region),
				InstanceType:       lo.FromPtr(testAzureCommitmentImport.ProductName),
				Plan:               "THREE_YEAR",
				Scope:              lo.FromPtr(testAzureCommitmentImport.Scope),
				ScopeResourceGroup: lo.FromPtr(testAzureCommitmentImport.ScopeResourceGroup),
				ScopeSubscription:  lo.FromPtr(testAzureCommitmentImport.ScopeSubscription),
			},
		},
		"should succeed, with a config passed": {
			input: func() *commitmentWithConfig[CastaiAzureReservationImport] {
				c := makeInput()
				c.Config = &CommitmentConfigResource{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name:   lo.FromPtr(testAzureCommitmentImport.Name),
							Type:   testAzureCommitmentImport.ProductName,
							Region: testAzureCommitmentImport.Region,
						},
					},
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.7),
				}
				return c
			}(),
			expected: &AzureReservationResource{
				CASTCommitmentFields: CASTCommitmentFields{
					AllowedUsage:   lo.ToPtr[float32](0.7),
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
				},
				Count:              2,
				ReservationID:      lo.FromPtr(testAzureCommitmentImport.ReservationId),
				ReservationStatus:  lo.FromPtr(testAzureCommitmentImport.Status),
				StartTimestamp:     lo.FromPtr(testAzureCommitmentImport.PurchaseDate),
				EndTimestamp:       lo.FromPtr(testAzureCommitmentImport.ExpirationDate),
				Name:               lo.FromPtr(testAzureCommitmentImport.Name),
				Region:             lo.FromPtr(testAzureCommitmentImport.Region),
				InstanceType:       lo.FromPtr(testAzureCommitmentImport.ProductName),
				Plan:               "THREE_YEAR",
				Scope:              lo.FromPtr(testAzureCommitmentImport.Scope),
				ScopeResourceGroup: lo.FromPtr(testAzureCommitmentImport.ScopeResourceGroup),
				ScopeSubscription:  lo.FromPtr(testAzureCommitmentImport.ScopeSubscription),
			},
		},
		"should map P1Y term to ONE_YEAR plan": {
			input: &commitmentWithConfig[CastaiAzureReservationImport]{
				Commitment: CastaiAzureReservationImport{
					CastaiInventoryV1beta1AzureReservationImport: sdk.CastaiInventoryV1beta1AzureReservationImport{
						Term: lo.ToPtr("P1Y"),
					},
				},
			},
			expected: &AzureReservationResource{
				Plan: "ONE_YEAR",
			},
		},
		"should map P3Y term to THREE_YEAR plan": {
			input: &commitmentWithConfig[CastaiAzureReservationImport]{
				Commitment: CastaiAzureReservationImport{
					CastaiInventoryV1beta1AzureReservationImport: sdk.CastaiInventoryV1beta1AzureReservationImport{
						Term: lo.ToPtr("P3Y"),
					},
				},
			},
			expected: &AzureReservationResource{
				Plan: "THREE_YEAR",
			},
		},
		"should map ONE_YEAR term to ONE_YEAR plan": {
			input: &commitmentWithConfig[CastaiAzureReservationImport]{
				Commitment: CastaiAzureReservationImport{
					CastaiInventoryV1beta1AzureReservationImport: sdk.CastaiInventoryV1beta1AzureReservationImport{
						Term: lo.ToPtr("ONE_YEAR"),
					},
				},
			},
			expected: &AzureReservationResource{
				Plan: "ONE_YEAR",
			},
		},
		"should map ONE_YEAR term to THREE_YEAR plan": {
			input: &commitmentWithConfig[CastaiAzureReservationImport]{
				Commitment: CastaiAzureReservationImport{
					CastaiInventoryV1beta1AzureReservationImport: sdk.CastaiInventoryV1beta1AzureReservationImport{
						Term: lo.ToPtr("THREE_YEAR"),
					},
				},
			},
			expected: &AzureReservationResource{
				Plan: "THREE_YEAR",
			},
		},
		"should fail when invalid term is passed": {
			input: &commitmentWithConfig[CastaiAzureReservationImport]{
				Commitment: CastaiAzureReservationImport{
					CastaiInventoryV1beta1AzureReservationImport: sdk.CastaiInventoryV1beta1AzureReservationImport{
						Term: lo.ToPtr("invalid"),
					},
				},
			},
			err: errors.New("invalid plan value: invalid"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapReservationImportToResource(tt.input)
			if tt.err == nil {
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)
			} else {
				r.Nil(actual)
				r.Error(err)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}

func TestMapConfigsToCommitments(t *testing.T) {
	var (
		cudImport1 = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
			Id:     lo.ToPtr("1"),
			Name:   lo.ToPtr("test-cud-1"),
			Type:   lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
			Region: lo.ToPtr("us-central1"),
		}
		cudImport2 = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
			Id:     lo.ToPtr("2"),
			Name:   lo.ToPtr("test-cud-2"),
			Type:   lo.ToPtr("COMPUTE_OPTIMIZED_N2D"),
			Region: lo.ToPtr("us-central1"),
		}
		cudImport3 = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
			Id:     lo.ToPtr("3"),
			Name:   lo.ToPtr("test-cud-3"),
			Type:   lo.ToPtr("COMPUTE_OPTIMIZED_E2"),
			Region: lo.ToPtr("eu-central1"),
		}

		cudCfg1 = &CommitmentConfigResource{
			Matcher: []*CommitmentConfigMatcherResource{
				{
					Name:   "test-cud-1",
					Type:   lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
					Region: lo.ToPtr("us-central1"),
				},
			},
			Prioritization: lo.ToPtr(true),
			Status:         lo.ToPtr("ACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](0.5),
		}
		cudCfg2 = &CommitmentConfigResource{
			Matcher: []*CommitmentConfigMatcherResource{
				{
					Name:   "test-cud-2",
					Type:   lo.ToPtr("COMPUTE_OPTIMIZED_N2D"),
					Region: lo.ToPtr("us-central1"),
				},
			},
			Prioritization: lo.ToPtr(false),
			Status:         lo.ToPtr("INACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](0.7),
		}
		cudCfg3 = &CommitmentConfigResource{
			Matcher: []*CommitmentConfigMatcherResource{
				{
					Name:   "test-cud-3",
					Type:   lo.ToPtr("COMPUTE_OPTIMIZED_E2"),
					Region: lo.ToPtr("eu-central1"),
				},
			},
			Prioritization: lo.ToPtr(true),
			Status:         lo.ToPtr("ACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](1),
		}

		reservationImport1 = sdk.CastaiInventoryV1beta1AzureReservationImport{
			ReservationId: lo.ToPtr("1"),
			Name:          lo.ToPtr("test-reservation-1"),
			ProductName:   lo.ToPtr("Standard_D32as_v4"),
			Region:        lo.ToPtr("eastus"),
		}
		reservationImport2 = sdk.CastaiInventoryV1beta1AzureReservationImport{
			ReservationId: lo.ToPtr("2"),
			Name:          lo.ToPtr("test-reservation-2"),
			ProductName:   lo.ToPtr("Standard_B1s"),
			Region:        lo.ToPtr("eastus"),
		}
		reservationImport3 = sdk.CastaiInventoryV1beta1AzureReservationImport{
			ReservationId: lo.ToPtr("3"),
			Name:          lo.ToPtr("test-reservation-3"),
			ProductName:   lo.ToPtr("Standard_A2_v2"),
			Region:        lo.ToPtr("westus"),
		}

		reservationCfg1 = &CommitmentConfigResource{
			Matcher: []*CommitmentConfigMatcherResource{
				{
					Name:   "test-reservation-1",
					Type:   lo.ToPtr("Standard_D32as_v4"),
					Region: lo.ToPtr("eastus"),
				},
			},
			Prioritization: lo.ToPtr(true),
			Status:         lo.ToPtr("ACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](0.5),
		}
		reservationCfg2 = &CommitmentConfigResource{
			Matcher: []*CommitmentConfigMatcherResource{
				{
					Name:   "test-reservation-2",
					Type:   lo.ToPtr("Standard_B1s"),
					Region: lo.ToPtr("eastus"),
				},
			},
			Prioritization: lo.ToPtr(false),
			Status:         lo.ToPtr("INACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](0.7),
		}
		reservationCfg3 = &CommitmentConfigResource{
			Matcher: []*CommitmentConfigMatcherResource{
				{
					Name:   "test-reservation-3",
					Type:   lo.ToPtr("Standard_A2_v2"),
					Region: lo.ToPtr("westus"),
				},
			},
			Prioritization: lo.ToPtr(true),
			Status:         lo.ToPtr("ACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](1),
		}
	)

	tests := map[string]struct {
		cuds     []commitment
		configs  []*CommitmentConfigResource
		expected []*commitmentWithConfig[commitment]
		err      error
	}{
		"should successfully map all the configs to cuds": {
			cuds: []commitment{
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
			},
			configs: []*CommitmentConfigResource{cudCfg1, cudCfg2, cudCfg3}, // make sure the order doesn't match the CUDs
			expected: []*commitmentWithConfig[commitment]{
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
					Config:     cudCfg2,
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
					Config:     cudCfg3,
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
					Config:     cudCfg1,
				},
			},
		},
		"should successfully map all the configs to reservations": {
			cuds: []commitment{
				CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport2},
				CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport3},
				CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport1},
			},
			configs: []*CommitmentConfigResource{reservationCfg1, reservationCfg2, reservationCfg3}, // make sure the order doesn't match the CUDs
			expected: []*commitmentWithConfig[commitment]{
				{
					Commitment: CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport2},
					Config:     reservationCfg2,
				},
				{
					Commitment: CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport3},
					Config:     reservationCfg3,
				},
				{
					Commitment: CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport1},
					Config:     reservationCfg1,
				},
			},
		},
		"should successfully map all the configs to cud imports with url-based regions": {
			cuds: func() []commitment {
				cudImport2.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cudImport2.Region)
				cudImport3.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cudImport3.Region)
				cudImport1.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cudImport1.Region)
				return []commitment{
					CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
					CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
					CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
				}
			}(),
			configs: []*CommitmentConfigResource{cudCfg1, cudCfg2, cudCfg3}, // make sure the order doesn't match the CUDs
			expected: []*commitmentWithConfig[commitment]{
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
					Config:     cudCfg2,
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
					Config:     cudCfg3,
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
					Config:     cudCfg1,
				},
			},
		},
		"should successfully map all the configs with url-based regions to cud imports": {
			cuds: []commitment{
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
			},
			configs: func() []*CommitmentConfigResource {
				cudCfg1.GetMatcher().Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cudCfg1.GetMatcher().Region)
				cudCfg2.GetMatcher().Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cudCfg2.GetMatcher().Region)
				cudCfg3.GetMatcher().Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cudCfg3.GetMatcher().Region)
				return []*CommitmentConfigResource{cudCfg1, cudCfg2, cudCfg3} // make sure the order doesn't match the CUDs
			}(),
			expected: []*commitmentWithConfig[commitment]{
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
					Config:     cudCfg2,
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
					Config:     cudCfg3,
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
					Config:     cudCfg1,
				},
			},
		},
		"should fail as there's one additional config that doesn't match any cud": {
			cuds: []commitment{
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
			},
			configs: []*CommitmentConfigResource{cudCfg1, cudCfg2},
			err:     errors.New("not all commitment configurations were mapped"),
		},
		"should fail as one of the configs cannot be mapped": {
			cuds: []commitment{
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
			},
			configs: []*CommitmentConfigResource{cudCfg1, cudCfg3},
			err:     errors.New("not all commitment configurations were mapped"),
		},
		"should fail as one config can be mapped to multiple cud imports": {
			cuds: []commitment{
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
			},
			configs: []*CommitmentConfigResource{cudCfg1, cudCfg3},
			err:     errors.New("duplicate import for test-cud-1-us-central1-COMPUTE_OPTIMIZED_C2D"),
		},
		"should fail as one config can be mapped to multiple reservation imports": {
			cuds: []commitment{
				CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport1},
				CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: reservationImport1},
			},
			configs: []*CommitmentConfigResource{reservationCfg1, reservationCfg3},
			err:     errors.New("duplicate import for test-reservation-1-eastus-Standard_D32as_v4"),
		},
		"should fail as duplicate configs are passed": {
			cuds: []commitment{
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
			},
			configs: []*CommitmentConfigResource{cudCfg1, cudCfg1},
			err:     errors.New("duplicate configuration for test-cud-1-us-central1-COMPUTE_OPTIMIZED_C2D"),
		},
		"should successfully map a config when more commitments are passed": {
			cuds: []commitment{
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
				CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
			},
			configs: []*CommitmentConfigResource{cudCfg2},
			expected: []*commitmentWithConfig[commitment]{
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport2},
					Config:     cudCfg2,
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport3},
				},
				{
					Commitment: CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cudImport1},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapConfigsToCommitments(tt.cuds, tt.configs)
			if tt.err == nil {
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)
			} else {
				r.Error(err)
				r.Nil(actual)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}

func TestMapConfiguredCUDImportsToResources(t *testing.T) {
	tests := map[string]struct {
		cuds     []sdk.CastaiInventoryV1beta1GCPCommitmentImport
		configs  []*CommitmentConfigResource
		expected []*GCPCUDResource
		err      error
	}{
		"should fail as there are more configs than cuds": {
			configs: []*CommitmentConfigResource{
				{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name: "test-cud",
						},
					},
					Prioritization: lo.ToPtr(true),
				},
				{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name: "test-cud-2",
						},
					},
					AllowedUsage: lo.ToPtr[float32](0.45),
				},
			},
			cuds: []sdk.CastaiInventoryV1beta1GCPCommitmentImport{
				{
					Name: lo.ToPtr("test-cud"),
				},
			},
			err: errors.New("more configurations than CUDs"),
		},
		"should successfully map cuds with configs to resources": {
			cuds: []sdk.CastaiInventoryV1beta1GCPCommitmentImport{testGCPCommitmentImport},
			configs: []*CommitmentConfigResource{
				{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name:   lo.FromPtr(testGCPCommitmentImport.Name),
							Type:   testGCPCommitmentImport.Type,
							Region: testGCPCommitmentImport.Region,
						},
					},
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.5),
				},
			},
			expected: []*GCPCUDResource{
				{
					CASTCommitmentFields: CASTCommitmentFields{
						AllowedUsage:   lo.ToPtr[float32](0.5),
						Prioritization: lo.ToPtr(true),
						Status:         lo.ToPtr("ACTIVE"),
					},
					CUDID:          lo.FromPtr(testGCPCommitmentImport.Id),
					CUDStatus:      lo.FromPtr(testGCPCommitmentImport.Status),
					StartTimestamp: lo.FromPtr(testGCPCommitmentImport.StartTimestamp),
					EndTimestamp:   lo.FromPtr(testGCPCommitmentImport.EndTimestamp),
					Name:           lo.FromPtr(testGCPCommitmentImport.Name),
					Region:         "us-central1",
					CPU:            10,
					MemoryMb:       20480,
					Plan:           lo.FromPtr(testGCPCommitmentImport.Plan),
					Type:           lo.FromPtr(testGCPCommitmentImport.Type),
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapConfiguredCUDImportsToResources(tt.cuds, tt.configs)
			if tt.err == nil {
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)

				// Do the same test but for wrapped CUDs
				wrappedCUDs := make([]CastaiGCPCommitmentImport, len(tt.cuds))
				for i, cud := range tt.cuds {
					wrappedCUDs[i] = CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: cud}
				}
				actual, err = MapConfiguredCUDImportsToResources(wrappedCUDs, tt.configs)
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)
			} else {
				r.Error(err)
				r.Nil(actual)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}

func TestMapConfiguredReservationImportsToResources(t *testing.T) {
	tests := map[string]struct {
		cuds     []sdk.CastaiInventoryV1beta1AzureReservationImport
		configs  []*CommitmentConfigResource
		expected []*AzureReservationResource
		err      error
	}{
		"should fail as there are more configs than reservations": {
			configs: []*CommitmentConfigResource{
				{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name: "test-reservation",
						},
					},
					Prioritization: lo.ToPtr(true),
				},
				{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name: "test-reservation-2",
						},
					},
					AllowedUsage: lo.ToPtr[float32](0.45),
				},
			},
			cuds: []sdk.CastaiInventoryV1beta1AzureReservationImport{
				{
					Name: lo.ToPtr("test-reservation"),
				},
			},
			err: errors.New("more configurations than reservations"),
		},
		"should successfully map reservations with configs to resources": {
			cuds: []sdk.CastaiInventoryV1beta1AzureReservationImport{testAzureCommitmentImport},
			configs: []*CommitmentConfigResource{
				{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name:   lo.FromPtr(testAzureCommitmentImport.Name),
							Type:   testAzureCommitmentImport.ProductName,
							Region: testAzureCommitmentImport.Region,
						},
					},
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.5),
				},
			},
			expected: []*AzureReservationResource{
				{
					CASTCommitmentFields: CASTCommitmentFields{
						AllowedUsage:   lo.ToPtr[float32](0.5),
						Prioritization: lo.ToPtr(true),
						Status:         lo.ToPtr("ACTIVE"),
					},
					Count:              int(lo.FromPtr(testAzureCommitmentImport.Quantity)),
					ReservationID:      lo.FromPtr(testAzureCommitmentImport.ReservationId),
					ReservationStatus:  lo.FromPtr(testAzureCommitmentImport.Status),
					StartTimestamp:     lo.FromPtr(testAzureCommitmentImport.PurchaseDate),
					EndTimestamp:       lo.FromPtr(testAzureCommitmentImport.ExpirationDate),
					Name:               lo.FromPtr(testAzureCommitmentImport.Name),
					Region:             lo.FromPtr(testAzureCommitmentImport.Region),
					InstanceType:       lo.FromPtr(testAzureCommitmentImport.ProductName),
					Plan:               "THREE_YEAR",
					Scope:              lo.FromPtr(testAzureCommitmentImport.Scope),
					ScopeResourceGroup: lo.FromPtr(testAzureCommitmentImport.ScopeResourceGroup),
					ScopeSubscription:  lo.FromPtr(testAzureCommitmentImport.ScopeSubscription),
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapConfiguredReservationImportsToResources(tt.cuds, tt.configs)
			if tt.err == nil {
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)

				// Do the same test but for wrapped reservations
				wrappedReservations := make([]CastaiAzureReservationImport, len(tt.cuds))
				for i, cud := range tt.cuds {
					wrappedReservations[i] = CastaiAzureReservationImport{CastaiInventoryV1beta1AzureReservationImport: cud}
				}
				actual, err = MapConfiguredReservationImportsToResources(wrappedReservations, tt.configs)
				r.NoError(err)
				r.NotNil(actual)
				r.Equal(tt.expected, actual)
			} else {
				r.Error(err)
				r.Nil(actual)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}

func TestMapCommitmentImportWithConfigToUpdateRequest(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	tests := map[string]struct {
		input    *commitmentWithConfig[CastaiCommitment]
		expected sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody
	}{
		"should map gcp cud import with config": {
			input: &commitmentWithConfig[CastaiCommitment]{
				Commitment: CastaiCommitment{
					CastaiInventoryV1beta1Commitment: sdk.CastaiInventoryV1beta1Commitment{
						AllowedUsage:          lo.ToPtr[float32](0.75),
						EndDate:               lo.ToPtr(now.Add(365 * 24 * time.Hour)),
						GcpResourceCudContext: testGCPCUDContext,
						Id:                    lo.ToPtr(id.String()),
						Name:                  lo.ToPtr("test-cud-1"),
						Prioritization:        lo.ToPtr(true),
						Region:                lo.ToPtr("us-central1"),
						StartDate:             lo.ToPtr(now.Add(-24 * time.Hour)),
						Status:                lo.ToPtr[sdk.CastaiInventoryV1beta1CommitmentStatus]("ACTIVE"),
					},
				},
				Config: &CommitmentConfigResource{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name:   "test-cud-1",
							Type:   lo.ToPtr("COMPUTE_OPTIMIZED_N2D"),
							Region: lo.ToPtr("us-central1"),
						},
					},
					Prioritization: lo.ToPtr(false),
					Status:         lo.ToPtr("INACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.7),
				},
			},
			expected: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
				AllowedUsage: lo.ToPtr[float32](0.7),
				EndDate:      lo.ToPtr(now.Add(365 * 24 * time.Hour)),
				GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{
					Cpu:      lo.ToPtr("8"),
					CudId:    lo.ToPtr("123456"),
					MemoryMb: lo.ToPtr("1024"),
					Plan:     lo.ToPtr[sdk.CastaiInventoryV1beta1GCPResourceCUDCUDPlan]("TWELVE_MONTHS"),
					Status:   lo.ToPtr("ACTIVE"),
					Type:     lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
				},
				Id:             lo.ToPtr(id.String()),
				Name:           lo.ToPtr("test-cud-1"),
				Prioritization: lo.ToPtr(false),
				Region:         lo.ToPtr("us-central1"),
				StartDate:      lo.ToPtr(now.Add(-24 * time.Hour)),
				Status:         lo.ToPtr[sdk.CastaiInventoryV1beta1CommitmentStatus]("INACTIVE"),
			},
		},
		"should map gcp cud import without config": {
			input: &commitmentWithConfig[CastaiCommitment]{
				Commitment: CastaiCommitment{
					CastaiInventoryV1beta1Commitment: sdk.CastaiInventoryV1beta1Commitment{
						EndDate:               lo.ToPtr(now.Add(365 * 24 * time.Hour)),
						GcpResourceCudContext: testGCPCUDContext,
						Id:                    lo.ToPtr(id.String()),
						Name:                  lo.ToPtr("test-cud-1"),
						Region:                lo.ToPtr("us-central1"),
						StartDate:             lo.ToPtr(now.Add(-24 * time.Hour)),
					},
				},
			},
			expected: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
				EndDate: lo.ToPtr(now.Add(365 * 24 * time.Hour)),
				GcpResourceCudContext: &sdk.CastaiInventoryV1beta1GCPResourceCUD{
					Cpu:      lo.ToPtr("8"),
					CudId:    lo.ToPtr("123456"),
					MemoryMb: lo.ToPtr("1024"),
					Plan:     lo.ToPtr[sdk.CastaiInventoryV1beta1GCPResourceCUDCUDPlan]("TWELVE_MONTHS"),
					Status:   lo.ToPtr("ACTIVE"),
					Type:     lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
				},
				Id:        lo.ToPtr(id.String()),
				Name:      lo.ToPtr("test-cud-1"),
				Region:    lo.ToPtr("us-central1"),
				StartDate: lo.ToPtr(now.Add(-24 * time.Hour)),
			},
		},
		"should map azure reservation import with config": {
			input: &commitmentWithConfig[CastaiCommitment]{
				Commitment: CastaiCommitment{
					CastaiInventoryV1beta1Commitment: sdk.CastaiInventoryV1beta1Commitment{
						AllowedUsage:            lo.ToPtr[float32](0.75),
						EndDate:                 lo.ToPtr(now.Add(365 * 24 * time.Hour)),
						AzureReservationContext: testAzureReservationContext,
						Id:                      lo.ToPtr(id.String()),
						Name:                    lo.ToPtr("test-reservation-1"),
						Prioritization:          lo.ToPtr(true),
						Region:                  lo.ToPtr("eastus"),
						StartDate:               lo.ToPtr(now.Add(-24 * time.Hour)),
						Status:                  lo.ToPtr[sdk.CastaiInventoryV1beta1CommitmentStatus]("ACTIVE"),
					},
				},
				Config: &CommitmentConfigResource{
					Matcher: []*CommitmentConfigMatcherResource{
						{
							Name:   "test-reservation-1",
							Type:   testAzureReservationContext.InstanceType,
							Region: lo.ToPtr("eastus"),
						},
					},
					Prioritization: lo.ToPtr(false),
					Status:         lo.ToPtr("INACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.7),
				},
			},
			expected: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
				AllowedUsage:            lo.ToPtr[float32](0.7),
				EndDate:                 lo.ToPtr(now.Add(365 * 24 * time.Hour)),
				AzureReservationContext: testAzureReservationContext,
				Id:                      lo.ToPtr(id.String()),
				Name:                    lo.ToPtr("test-reservation-1"),
				Prioritization:          lo.ToPtr(false),
				Region:                  lo.ToPtr("eastus"),
				StartDate:               lo.ToPtr(now.Add(-24 * time.Hour)),
				Status:                  lo.ToPtr[sdk.CastaiInventoryV1beta1CommitmentStatus]("INACTIVE"),
			},
		},
		"should map azure reservation import without config": {
			input: &commitmentWithConfig[CastaiCommitment]{
				Commitment: CastaiCommitment{
					CastaiInventoryV1beta1Commitment: sdk.CastaiInventoryV1beta1Commitment{
						EndDate:                 lo.ToPtr(now.Add(365 * 24 * time.Hour)),
						AzureReservationContext: testAzureReservationContext,
						Id:                      lo.ToPtr(id.String()),
						Name:                    lo.ToPtr("test-reservation-1"),
						Region:                  lo.ToPtr("eastus"),
						StartDate:               lo.ToPtr(now.Add(-24 * time.Hour)),
					},
				},
			},
			expected: sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody{
				EndDate:                 lo.ToPtr(now.Add(365 * 24 * time.Hour)),
				AzureReservationContext: testAzureReservationContext,
				Id:                      lo.ToPtr(id.String()),
				Name:                    lo.ToPtr("test-reservation-1"),
				Region:                  lo.ToPtr("eastus"),
				StartDate:               lo.ToPtr(now.Add(-24 * time.Hour)),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual := MapCommitmentImportWithConfigToUpdateRequest(tt.input)
			r.Equal(tt.expected, actual)
		})
	}
}

func TestSortResources(t *testing.T) {
	tests := map[string]struct {
		toSort, targetOrder []CommitmentResource
	}{
		"should sort gcp cud resources": {
			toSort: []CommitmentResource{
				&GCPCUDResource{CUDID: "1"},
				&GCPCUDResource{CUDID: "2"},
				&GCPCUDResource{CUDID: "3"},
				&GCPCUDResource{CUDID: "4"},
				&GCPCUDResource{CUDID: "5"},
			},
			targetOrder: []CommitmentResource{
				&GCPCUDResource{CUDID: "3"},
				&GCPCUDResource{CUDID: "1"},
				&GCPCUDResource{CUDID: "4"},
				&GCPCUDResource{CUDID: "2"},
				&GCPCUDResource{CUDID: "5"},
			},
		},
		"should sort azure reservation resources": {
			toSort: []CommitmentResource{
				&AzureReservationResource{ReservationID: "a"},
				&AzureReservationResource{ReservationID: "b"},
				&AzureReservationResource{ReservationID: "c"},
				&AzureReservationResource{ReservationID: "d"},
				&AzureReservationResource{ReservationID: "e"},
			},
			targetOrder: []CommitmentResource{
				&AzureReservationResource{ReservationID: "e"},
				&AzureReservationResource{ReservationID: "a"},
				&AzureReservationResource{ReservationID: "c"},
				&AzureReservationResource{ReservationID: "d"},
				&AzureReservationResource{ReservationID: "b"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			SortCommitmentResources(tt.toSort, tt.targetOrder)
			require.Equal(t, tt.targetOrder, tt.toSort)
		})
	}
}

var (
	testGCPCommitmentImport = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
		AutoRenew:         lo.ToPtr(true),
		Category:          lo.ToPtr("MACHINE"),
		CreationTimestamp: lo.ToPtr("2023-01-01T00:00:00.000-07:00"),
		EndTimestamp:      lo.ToPtr("2024-01-01T00:00:00.000-07:00"),
		Id:                lo.ToPtr("123456"),
		Kind:              lo.ToPtr("compute#commitment"),
		Name:              lo.ToPtr("test-cud"),
		Plan:              lo.ToPtr("TWELVE_MONTHS"),
		// Remember to pass the region as a URL!
		Region: lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1"),
		Resources: &[]sdk.CastaiInventoryV1beta1GCPResource{
			{
				Amount: lo.ToPtr("10"),
				Type:   lo.ToPtr("VCPU"),
			},
			{
				Amount: lo.ToPtr("20480"),
				Type:   lo.ToPtr("MEMORY"),
			},
		},
		SelfLink:       lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1/commitments/123456"),
		StartTimestamp: lo.ToPtr("2023-01-01T00:00:00.000-07:00"),
		Status:         lo.ToPtr("ACTIVE"),
		StatusMessage:  lo.ToPtr("The commitment is active, and so will apply to current resource usage."),
		Type:           lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
	}

	testAzureCommitmentImport = sdk.CastaiInventoryV1beta1AzureReservationImport{
		ExpirationDate:     lo.ToPtr("2024-01-01T00:00:00.000-07:00"),
		Name:               lo.ToPtr("test-reservation"),
		ProductName:        lo.ToPtr("Standard_D32as_v4"),
		PurchaseDate:       lo.ToPtr("2023-01-01T00:00:00.000-07:00"),
		Quantity:           lo.ToPtr[int32](2),
		Region:             lo.ToPtr("eastus"),
		ReservationId:      lo.ToPtr(uuid.New().String()),
		Scope:              lo.ToPtr("Single subscription"),
		ScopeResourceGroup: lo.ToPtr("All resource groups"),
		ScopeSubscription:  lo.ToPtr(uuid.New().String()),
		Status:             lo.ToPtr("Succeeded"),
		Term:               lo.ToPtr("P3Y"),
		Type:               lo.ToPtr("VirtualMachines"),
	}

	testGCPCUDContext = &sdk.CastaiInventoryV1beta1GCPResourceCUD{
		Cpu:      lo.ToPtr("8"),
		CudId:    lo.ToPtr("123456"),
		MemoryMb: lo.ToPtr("1024"),
		Plan:     lo.ToPtr[sdk.CastaiInventoryV1beta1GCPResourceCUDCUDPlan]("TWELVE_MONTHS"),
		Status:   lo.ToPtr("ACTIVE"),
		Type:     lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
	}

	testAzureReservationContext = &sdk.CastaiInventoryV1beta1AzureReservation{
		Count:              lo.ToPtr[int32](2),
		Id:                 lo.ToPtr("123456"),
		InstanceType:       lo.ToPtr("Standard_D32as_v4"),
		Plan:               lo.ToPtr[sdk.CastaiInventoryV1beta1AzureReservationReservationPlan]("THREE_YEAR"),
		Scope:              lo.ToPtr("Single subscription"),
		ScopeResourceGroup: lo.ToPtr("All resource groups"),
		ScopeSubscription:  lo.ToPtr("scope-subscription"),
		Status:             lo.ToPtr("Succeeded"),
	}
)
