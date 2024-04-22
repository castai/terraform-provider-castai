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
		"should fail as cpu is a float": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext.Cpu = lo.ToPtr("1.5")
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"1.5\": invalid syntax"),
		},
		"should fail as memory is an invalid string": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext.MemoryMb = lo.ToPtr("invalid")
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"invalid\": invalid syntax"),
		},
		"should fail as memory is a float": {
			input: func() sdk.CastaiInventoryV1beta1Commitment {
				c := makeCommitment()
				c.GcpResourceCudContext.MemoryMb = lo.ToPtr("1.5")
				return c
			}(),
			err: errors.New("strconv.Atoi: parsing \"1.5\": invalid syntax"),
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
