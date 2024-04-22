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
		input  sdk.CastaiInventoryV1beta1Commitment
		output *GCPCUDResource
		err    error
	}{
		"should succeed as all the fields are set": {
			input: makeCommitment(),
			output: &GCPCUDResource{
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
			output: &GCPCUDResource{
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
			output: &GCPCUDResource{
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
				r.Equal(tt.output, actual)
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
				CastaiInventoryV1beta1GCPCommitmentImport: sdk.CastaiInventoryV1beta1GCPCommitmentImport{
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
				},
			},
		}
	}

	tests := map[string]struct {
		input  *cudWithConfig[CastaiGCPCommitmentImport]
		output *GCPCUDResource
		err    error
	}{
		"should succeed, no config passed": {
			input: makeInput(),
			output: &GCPCUDResource{
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
			output: &GCPCUDResource{
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
			output: &GCPCUDResource{
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
				(*(c.CUD.Resources))[0].Amount = lo.ToPtr("invalid")
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
		"should fail as memory amount is invalid": {
			input: func() *cudWithConfig[CastaiGCPCommitmentImport] {
				c := makeInput()
				(*(c.CUD.Resources))[1].Amount = lo.ToPtr("invalid")
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
			output: &GCPCUDResource{
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
			output: &GCPCUDResource{
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
				r.Equal(tt.output, actual)
			} else {
				r.Nil(actual)
				r.Error(err)
				r.Equal(tt.err.Error(), err.Error())
			}
		})
	}
}
