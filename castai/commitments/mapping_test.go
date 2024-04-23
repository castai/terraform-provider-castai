package commitments

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
				ID:             lo.ToPtr(id1.String()),
				AllowedUsage:   lo.ToPtr[float32](0.5),
				Prioritization: lo.ToPtr(true),
				Status:         lo.ToPtr("ACTIVE"),
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
				ID:             lo.ToPtr(id1.String()),
				AllowedUsage:   lo.ToPtr[float32](0.5),
				Prioritization: lo.ToPtr(true),
				Status:         lo.ToPtr("ACTIVE"),
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
				ID:             lo.ToPtr(id1.String()),
				AllowedUsage:   lo.ToPtr[float32](0.5),
				Prioritization: lo.ToPtr(true),
				Status:         lo.ToPtr("ACTIVE"),
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

func TestMapCUDImportToResource(t *testing.T) {
	makeInput := func() *cudWithConfig[CastaiGCPCommitmentImport] {
		return &cudWithConfig[CastaiGCPCommitmentImport]{
			CUD: CastaiGCPCommitmentImport{
				CastaiInventoryV1beta1GCPCommitmentImport: testGCPCommitmentImport,
			},
		}
	}

	tests := map[string]struct {
		input    *cudWithConfig[CastaiGCPCommitmentImport]
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
			input: func() *cudWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.CUD.Resources = nil
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
			input: func() *cudWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.Config = &GCPCUDConfigResource{
					Matcher: GCPCUDConfigMatcherResource{
						Name:   "test-cud",
						Type:   lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
						Region: lo.ToPtr("us-central1"),
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
				Prioritization: lo.ToPtr(true),
				Status:         lo.ToPtr("ACTIVE"),
				AllowedUsage:   lo.ToPtr[float32](0.7),
			},
		},
		"should fail as cpu amount is invalid": {
			input: func() *cudWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				inv := (*c.CUD.Resources)[0]
				inv.Amount = lo.ToPtr("invalid")
				c.CUD.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{inv, (*c.CUD.Resources)[1]}
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
		"should fail as memory amount is invalid": {
			input: func() *cudWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				inv := (*c.CUD.Resources)[1]
				inv.Amount = lo.ToPtr("invalid")
				c.CUD.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{(*c.CUD.Resources)[0], inv}
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
		"should succeed with zeroed out cpu as its resource is not contained by the resources": {
			input: func() *cudWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.CUD.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{(*c.CUD.Resources)[1]}
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
			input: func() *cudWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				c.CUD.Resources = &[]sdk.CastaiInventoryV1beta1GCPResource{(*c.CUD.Resources)[0]}
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

func TestMapConfigsToCUDs(t *testing.T) {
	var (
		import1 = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
			Id:     lo.ToPtr("1"),
			Name:   lo.ToPtr("test-cud-1"),
			Type:   lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
			Region: lo.ToPtr("us-central1"),
		}
		import2 = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
			Id:     lo.ToPtr("2"),
			Name:   lo.ToPtr("test-cud-2"),
			Type:   lo.ToPtr("COMPUTE_OPTIMIZED_N2D"),
			Region: lo.ToPtr("us-central1"),
		}
		import3 = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
			Id:     lo.ToPtr("3"),
			Name:   lo.ToPtr("test-cud-3"),
			Type:   lo.ToPtr("COMPUTE_OPTIMIZED_E2"),
			Region: lo.ToPtr("eu-central1"),
		}

		cfg1 = &GCPCUDConfigResource{
			Matcher: GCPCUDConfigMatcherResource{
				Name:   "test-cud-1",
				Type:   lo.ToPtr("COMPUTE_OPTIMIZED_C2D"),
				Region: lo.ToPtr("us-central1"),
			},
			Prioritization: lo.ToPtr(true),
			Status:         lo.ToPtr("ACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](0.5),
		}
		cfg2 = &GCPCUDConfigResource{
			Matcher: GCPCUDConfigMatcherResource{
				Name:   "test-cud-2",
				Type:   lo.ToPtr("COMPUTE_OPTIMIZED_N2D"),
				Region: lo.ToPtr("us-central1"),
			},
			Prioritization: lo.ToPtr(false),
			Status:         lo.ToPtr("INACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](0.7),
		}
		cfg3 = &GCPCUDConfigResource{
			Matcher: GCPCUDConfigMatcherResource{
				Name:   "test-cud-3",
				Type:   lo.ToPtr("COMPUTE_OPTIMIZED_E2"),
				Region: lo.ToPtr("eu-central1"),
			},
			Prioritization: lo.ToPtr(true),
			Status:         lo.ToPtr("ACTIVE"),
			AllowedUsage:   lo.ToPtr[float32](1),
		}
	)

	tests := map[string]struct {
		cuds     []CastaiGCPCommitmentImport
		configs  []*GCPCUDConfigResource
		expected []*cudWithConfig[CastaiGCPCommitmentImport]
		err      error
	}{
		"should successfully map all the configs": {
			cuds: []CastaiGCPCommitmentImport{
				{CastaiInventoryV1beta1GCPCommitmentImport: import2},
				{CastaiInventoryV1beta1GCPCommitmentImport: import3},
				{CastaiInventoryV1beta1GCPCommitmentImport: import1},
			},
			configs: []*GCPCUDConfigResource{cfg1, cfg2, cfg3}, // make sure the order doesn't match the CUDs
			expected: []*cudWithConfig[CastaiGCPCommitmentImport]{
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import2},
					Config: cfg2,
				},
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import3},
					Config: cfg3,
				},
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import1},
					Config: cfg1,
				},
			},
		},
		"should successfully map all the configs to imports with url-based regions": {
			cuds: func() []CastaiGCPCommitmentImport {
				import2.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *import2.Region)
				import3.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *import3.Region)
				import1.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *import1.Region)
				return []CastaiGCPCommitmentImport{
					{CastaiInventoryV1beta1GCPCommitmentImport: import2},
					{CastaiInventoryV1beta1GCPCommitmentImport: import3},
					{CastaiInventoryV1beta1GCPCommitmentImport: import1},
				}
			}(),
			configs: []*GCPCUDConfigResource{cfg1, cfg2, cfg3}, // make sure the order doesn't match the CUDs
			expected: []*cudWithConfig[CastaiGCPCommitmentImport]{
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import2},
					Config: cfg2,
				},
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import3},
					Config: cfg3,
				},
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import1},
					Config: cfg1,
				},
			},
		},
		"should successfully map all the configs with url-based regions": {
			cuds: []CastaiGCPCommitmentImport{
				{CastaiInventoryV1beta1GCPCommitmentImport: import2},
				{CastaiInventoryV1beta1GCPCommitmentImport: import3},
				{CastaiInventoryV1beta1GCPCommitmentImport: import1},
			},
			configs: func() []*GCPCUDConfigResource {
				cfg1.Matcher.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cfg1.Matcher.Region)
				cfg2.Matcher.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cfg2.Matcher.Region)
				cfg3.Matcher.Region = lo.ToPtr("https://www.googleapis.com/compute/v1/projects/test-project/zones/" + *cfg3.Matcher.Region)
				return []*GCPCUDConfigResource{cfg1, cfg2, cfg3} // make sure the order doesn't match the CUDs
			}(),
			expected: []*cudWithConfig[CastaiGCPCommitmentImport]{
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import2},
					Config: cfg2,
				},
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import3},
					Config: cfg3,
				},
				{
					CUD:    CastaiGCPCommitmentImport{CastaiInventoryV1beta1GCPCommitmentImport: import1},
					Config: cfg1,
				},
			},
		},
		"should fail as there's one additional config that doesn't match any CUD": {
			cuds:    []CastaiGCPCommitmentImport{{CastaiInventoryV1beta1GCPCommitmentImport: import1}},
			configs: []*GCPCUDConfigResource{cfg1, cfg2},
			err:     errors.New("not all CUD configurations were mapped"),
		},
		"should fail as one of the configs cannot be mapped": {
			cuds: []CastaiGCPCommitmentImport{
				{CastaiInventoryV1beta1GCPCommitmentImport: import1},
				{CastaiInventoryV1beta1GCPCommitmentImport: import2},
			},
			configs: []*GCPCUDConfigResource{cfg1, cfg3},
			err:     errors.New("not all CUD configurations were mapped"),
		},
		"should fail as one import can be mapped to multiple configs": {
			cuds: []CastaiGCPCommitmentImport{
				{CastaiInventoryV1beta1GCPCommitmentImport: import1},
				{CastaiInventoryV1beta1GCPCommitmentImport: import1},
			},
			configs: []*GCPCUDConfigResource{cfg1, cfg3},
			err:     errors.New("duplicate CUD import for test-cud-1-us-central1-COMPUTE_OPTIMIZED_C2D"),
		},
		"should fail as duplicate configs are passed": {
			cuds: []CastaiGCPCommitmentImport{
				{CastaiInventoryV1beta1GCPCommitmentImport: import1},
			},
			configs: []*GCPCUDConfigResource{cfg1, cfg1},
			err:     errors.New("duplicate CUD configuration for test-cud-1-us-central1-COMPUTE_OPTIMIZED_C2D"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual, err := MapConfigsToCUDs(tt.cuds, tt.configs)
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
		configs  []*GCPCUDConfigResource
		expected []*GCPCUDResource
		err      error
	}{
		"should fail as there are more configs than cuds": {
			configs: []*GCPCUDConfigResource{
				{
					Matcher: GCPCUDConfigMatcherResource{
						Name: "test-cud",
					},
					Prioritization: lo.ToPtr(true),
				},
				{
					Matcher: GCPCUDConfigMatcherResource{
						Name: "test-cud-2",
					},
					AllowedUsage: lo.ToPtr[float32](0.45),
				},
			},
			cuds: []sdk.CastaiInventoryV1beta1GCPCommitmentImport{
				{
					Name: lo.ToPtr("test-cud"),
				},
			},
			err: errors.New("more CUD configurations than CUDs"),
		},
		"should successfully map cuds with configs to resources": {
			cuds: []sdk.CastaiInventoryV1beta1GCPCommitmentImport{testGCPCommitmentImport},
			configs: []*GCPCUDConfigResource{
				{
					Matcher: GCPCUDConfigMatcherResource{
						Name:   lo.FromPtr(testGCPCommitmentImport.Name),
						Type:   testGCPCommitmentImport.Type,
						Region: testGCPCommitmentImport.Region,
					},
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
					AllowedUsage:   lo.ToPtr[float32](0.5),
				},
			},
			expected: []*GCPCUDResource{
				{
					AllowedUsage:   lo.ToPtr[float32](0.5),
					Prioritization: lo.ToPtr(true),
					Status:         lo.ToPtr("ACTIVE"),
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

func TestMapCUDImportWithConfigToUpdateRequest(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	tests := map[string]struct {
		input    *cudWithConfig[CastaiCommitment]
		expected sdk.CommitmentsAPIUpdateCommitmentJSONRequestBody
	}{
		"should map gcp cud import with config": {
			input: &cudWithConfig[CastaiCommitment]{
				CUD: CastaiCommitment{
					CastaiInventoryV1beta1Commitment: sdk.CastaiInventoryV1beta1Commitment{
						AllowedUsage: lo.ToPtr[float32](0.75),
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
						Prioritization: lo.ToPtr(true),
						Region:         lo.ToPtr("us-central1"),
						StartDate:      lo.ToPtr(now.Add(-24 * time.Hour)),
						Status:         lo.ToPtr[sdk.CastaiInventoryV1beta1CommitmentStatus]("ACTIVE"),
					},
				},
				Config: &GCPCUDConfigResource{
					Matcher: GCPCUDConfigMatcherResource{
						Name:   "test-cud-1",
						Type:   lo.ToPtr("COMPUTE_OPTIMIZED_N2D"),
						Region: lo.ToPtr("us-central1"),
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
			input: &cudWithConfig[CastaiCommitment]{
				CUD: CastaiCommitment{
					CastaiInventoryV1beta1Commitment: sdk.CastaiInventoryV1beta1Commitment{
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
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r := require.New(t)
			actual := MapCUDImportWithConfigToUpdateRequest(tt.input)
			r.Equal(tt.expected, actual)
		})
	}
}

func TestSortResources(t *testing.T) {
	tests := map[string]struct {
		toSort, targetOrder []Resource
	}{
		"should sort gcp cud resources": {
			toSort: []Resource{
				&GCPCUDResource{CUDID: "1"},
				&GCPCUDResource{CUDID: "2"},
				&GCPCUDResource{CUDID: "3"},
				&GCPCUDResource{CUDID: "4"},
				&GCPCUDResource{CUDID: "5"},
			},
			targetOrder: []Resource{
				&GCPCUDResource{CUDID: "3"},
				&GCPCUDResource{CUDID: "1"},
				&GCPCUDResource{CUDID: "4"},
				&GCPCUDResource{CUDID: "2"},
				&GCPCUDResource{CUDID: "5"},
			},
		},
		"should sort azure reservation resources": {
			toSort: []Resource{
				&AzureReservationResource{ReservationID: "a"},
				&AzureReservationResource{ReservationID: "b"},
				&AzureReservationResource{ReservationID: "c"},
				&AzureReservationResource{ReservationID: "d"},
				&AzureReservationResource{ReservationID: "e"},
			},
			targetOrder: []Resource{
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
			SortResources(tt.toSort, tt.targetOrder)
			require.Equal(t, tt.targetOrder, tt.toSort)
		})
	}
}

var testGCPCommitmentImport = sdk.CastaiInventoryV1beta1GCPCommitmentImport{
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
