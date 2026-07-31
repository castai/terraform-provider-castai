package castai

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/pricing"
)

func TestCommitmentModel_ToCreateInput_GCPResourceCUD(t *testing.T) {
	r := require.New(t)

	m := &commitmentModel{
		Name:              types.StringValue("prod-cud-us-central1"),
		Cloud:             types.StringValue("GCP"),
		Region:            types.StringValue("us-central1"),
		Type:              types.StringValue("RESOURCE_CUD"),
		StartTime:         types.StringValue("2026-01-01T00:00:00Z"),
		EndTime:           types.StringValue("2027-01-01T00:00:00Z"),
		AutoscalingStatus: types.StringValue("ACTIVE"),
		AllowedUsage:      types.Float64Value(0.9),
		Prioritization:    types.BoolValue(true),
		ScalingStrategy:   types.StringValue("CPU_BASED"),
		AutoAssignment:    types.BoolValue(true),
		GCPResourceCUDDetails: &gcpResourceCUDDetailsModel{
			CUDID:    types.StringValue("123456789"),
			Plan:     types.StringValue("TWELVE_MONTH"),
			Type:     types.StringValue("COMPUTE_OPTIMIZED_C2D"),
			MemoryMb: types.Int64Value(131072),
			CPU:      types.Int64Value(32),
			Status:   types.StringValue("ACTIVE"),
		},
	}

	input, diags := m.toCreateInput(context.Background())
	r.False(diags.HasError(), "unexpected diagnostics: %v", diags)

	r.Equal("prod-cud-us-central1", input.Name)
	r.Equal(pricing.CreateCommitmentInputCloudGCP, input.Cloud)
	r.Equal("us-central1", input.Region)
	r.Equal(pricing.CreateCommitmentInputTypeRESOURCECUD, input.Type)
	r.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), input.StartTime.UTC())
	r.NotNil(input.EndTime)
	r.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), input.EndTime.UTC())
	r.NotNil(input.AutoscalingStatus)
	r.Equal(pricing.CreateCommitmentInputAutoscalingStatusACTIVE, *input.AutoscalingStatus)
	r.NotNil(input.AllowedUsage)
	r.InDelta(0.9, float64(*input.AllowedUsage), 1e-6)
	r.NotNil(input.Prioritization)
	r.True(*input.Prioritization)
	r.NotNil(input.ScalingStrategy)
	r.Equal(pricing.CreateCommitmentInputScalingStrategyCPUBASED, *input.ScalingStrategy)
	r.NotNil(input.AutoAssignment)
	r.True(*input.AutoAssignment)

	r.NotNil(input.GcpResourceCudDetails)
	d := input.GcpResourceCudDetails
	r.Equal("123456789", *d.CudId)
	r.Equal(pricing.GCPResourceCUDDetailsPlanTWELVEMONTH, *d.Plan)
	r.Equal("COMPUTE_OPTIMIZED_C2D", *d.Type)
	r.Equal("131072", *d.MemoryMb) // proto int64 → string on the wire
	r.Equal("32", *d.Cpu)
	r.Equal("ACTIVE", *d.Status)

	// No other details set.
	r.Nil(input.AwsReservedInstancesDetails)
	r.Nil(input.AzureReservationDetails)
	r.Nil(input.AwsSavingsPlanDetails)
	r.Nil(input.AwsCapacityBlockDetails)
	r.Nil(input.AwsOdcrDetails)
	r.Nil(input.GcpFlexCudDetails)
	r.Nil(input.AzureSavingsPlanDetails)
	r.Nil(input.GcpCapacityReservationDetails)
}

func TestCommitmentModel_ToCreateInput_AWSReservedInstances(t *testing.T) {
	r := require.New(t)

	m := &commitmentModel{
		Name:      types.StringValue("prod-ri"),
		Cloud:     types.StringValue("AWS"),
		Region:    types.StringValue("us-east-1"),
		Type:      types.StringValue("RESERVED_INSTANCE"),
		StartTime: types.StringValue("2026-01-01T00:00:00Z"),
		AWSReservedInstancesDetails: &awsReservedInstancesDetailsModel{
			ID:            types.StringValue("ri-0abc"),
			Scope:         types.StringValue("Region"),
			InstanceType:  types.StringValue("m5.xlarge"),
			InstanceCount: types.Int64Value(10),
			State:         types.StringValue("active"),
		},
	}

	input, diags := m.toCreateInput(context.Background())
	r.False(diags.HasError(), "unexpected diagnostics: %v", diags)

	r.Nil(input.EndTime)
	r.Nil(input.AutoscalingStatus)
	r.NotNil(input.AwsReservedInstancesDetails)
	d := input.AwsReservedInstancesDetails
	r.Equal("ri-0abc", *d.Id)
	r.Equal("Region", *d.Scope)
	r.Equal("m5.xlarge", *d.InstanceType)
	r.Equal("10", *d.InstanceCount)
	r.Nil(d.AvailabilityZoneName)
	r.Nil(d.AvailabilityZoneId)
}

func TestCommitmentModel_ToCreateInput_GCPCapacityReservation(t *testing.T) {
	r := require.New(t)

	projectIDs, d0 := types.ListValueFrom(context.Background(), types.StringType, []string{"proj-a", "proj-b"})
	r.False(d0.HasError())

	m := &commitmentModel{
		Name:      types.StringValue("gcp-odcr"),
		Cloud:     types.StringValue("GCP"),
		Region:    types.StringValue("us-central1"),
		Type:      types.StringValue("ON_DEMAND_CAPACITY_RESERVATION"),
		StartTime: types.StringValue("2026-01-01T00:00:00Z"),
		GCPCapacityReservationDetails: &gcpCapacityReservationDetailsModel{
			ID:                          types.StringValue("res-1"),
			ProjectID:                   types.StringValue("my-project"),
			Zone:                        types.StringValue("us-central1-a"),
			InstanceType:                types.StringValue("a2-highgpu-1g"),
			TotalInstanceCount:          types.Int64Value(4),
			InUseInstanceCount:          types.Int64Value(1),
			State:                       types.StringValue("READY"),
			SpecificReservationRequired: types.BoolValue(true),
			MinCPUPlatform:              types.StringValue("Intel Cascade Lake"),
			ShareSettings: &gcpReservationShareSettingsModel{
				ShareType:  types.StringValue("GCP_RESERVATION_SHARE_TYPE_SPECIFIC_PROJECTS"),
				ProjectIDs: projectIDs,
			},
			Accelerators: []gcpReservationAcceleratorModel{
				{
					AcceleratorType:  types.StringValue("https://www.googleapis.com/compute/v1/projects/p/zones/z/acceleratorTypes/nvidia-tesla-t4"),
					AcceleratorCount: types.Int64Value(1),
				},
			},
			LocalSsds: []gcpReservationLocalSsdModel{
				{
					Interface:  types.StringValue("GCP_RESERVATION_LOCAL_SSD_INTERFACE_NVME"),
					DiskSizeGb: types.Int64Value(375),
				},
			},
		},
	}

	input, diags := m.toCreateInput(context.Background())
	r.False(diags.HasError(), "unexpected diagnostics: %v", diags)

	d := input.GcpCapacityReservationDetails
	r.NotNil(d)
	r.Equal("res-1", *d.Id)
	r.Equal("my-project", *d.ProjectId)
	r.Equal("us-central1-a", *d.Zone)
	r.Equal("a2-highgpu-1g", *d.InstanceType)
	r.Equal("4", *d.TotalInstanceCount)
	r.Equal("1", *d.InUseInstanceCount)
	r.True(*d.SpecificReservationRequired)
	r.Equal("Intel Cascade Lake", *d.MinCpuPlatform)

	r.NotNil(d.ShareSettings)
	r.Equal(pricing.GCPRESERVATIONSHARETYPESPECIFICPROJECTS, *d.ShareSettings.ShareType)
	r.Equal([]string{"proj-a", "proj-b"}, *d.ShareSettings.ProjectIds)

	r.NotNil(d.Accelerators)
	r.Len(*d.Accelerators, 1)
	r.Equal("1", *(*d.Accelerators)[0].AcceleratorCount)

	r.NotNil(d.LocalSsds)
	r.Len(*d.LocalSsds, 1)
	r.Equal(pricing.GCPRESERVATIONLOCALSSDINTERFACENVME, *(*d.LocalSsds)[0].Interface)
	r.Equal("375", *(*d.LocalSsds)[0].DiskSizeGb)
}

func TestCommitmentModel_ToCreateInput_InvalidStartTime(t *testing.T) {
	r := require.New(t)

	m := &commitmentModel{
		Name:      types.StringValue("x"),
		Cloud:     types.StringValue("GCP"),
		Region:    types.StringValue("us-central1"),
		Type:      types.StringValue("RESOURCE_CUD"),
		StartTime: types.StringValue("not-a-timestamp"),
	}

	_, diags := m.toCreateInput(context.Background())
	r.True(diags.HasError())
}

func TestCommitmentModel_ToUpdateInput(t *testing.T) {
	r := require.New(t)

	m := &commitmentModel{
		AutoscalingStatus: types.StringValue("ACTIVE"),
		AllowedUsage:      types.Float64Value(0.5),
		Prioritization:    types.BoolValue(true),
		ScalingStrategy:   types.StringValue("MEMORY_BASED"),
		AutoAssignment:    types.BoolValue(false),
	}

	input := m.toUpdateInput()
	r.Equal(pricing.ACTIVE, *input.AutoscalingStatus)
	r.InDelta(0.5, float64(*input.AllowedUsage), 1e-6)
	r.True(*input.Prioritization)
	r.Equal(pricing.MEMORYBASED, *input.ScalingStrategy)
	r.False(*input.AutoAssignment)
}

func TestCommitmentModel_ToUpdateInput_NullsOmitted(t *testing.T) {
	r := require.New(t)

	m := &commitmentModel{
		AutoscalingStatus: types.StringNull(),
		AllowedUsage:      types.Float64Null(),
		Prioritization:    types.BoolNull(),
		ScalingStrategy:   types.StringNull(),
		AutoAssignment:    types.BoolNull(),
	}

	input := m.toUpdateInput()
	r.Nil(input.AutoscalingStatus)
	r.Nil(input.AllowedUsage)
	r.Nil(input.Prioritization)
	r.Nil(input.ScalingStrategy)
	r.Nil(input.AutoAssignment)
}

func TestCommitmentModel_ApplyCommitment_RoundTrip(t *testing.T) {
	r := require.New(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

	c := &pricing.Commitment{
		Id:                ptr("11111111-2222-3333-4444-555555555555"),
		Name:              ptr("prod-cud"),
		Cloud:             ptr(pricing.CommitmentCloudGCP),
		Region:            ptr("us-central1"),
		Type:              ptr(pricing.CommitmentTypeRESOURCECUD),
		StartTime:         &start,
		EndTime:           &end,
		AutoscalingStatus: ptr(pricing.CommitmentAutoscalingStatusACTIVE),
		AllowedUsage:      ptr(float32(0.9)),
		Prioritization:    ptr(true),
		ScalingStrategy:   ptr(pricing.CommitmentScalingStrategyDEFAULT),
		AutoAssignment:    ptr(false),
		State:             ptr(pricing.CommitmentStateSTATEACTIVE),
		CreateTime:        &created,
		UpdateTime:        &created,
		GcpResourceCudDetails: &pricing.GCPResourceCUDDetails{
			CudId:    ptr("123456789"),
			Plan:     ptr(pricing.GCPResourceCUDDetailsPlanTWELVEMONTH),
			Type:     ptr("COMPUTE_OPTIMIZED_C2D"),
			MemoryMb: ptr("131072"),
			Cpu:      ptr("32"),
			Status:   ptr("ACTIVE"),
		},
	}

	m := &commitmentModel{
		// Pre-populate state values that should be preserved when semantically equal.
		StartTime:    types.StringValue("2026-01-01T00:00:00Z"),
		AllowedUsage: types.Float64Value(0.9),
	}
	m.applyCommitment(context.Background(), c)

	r.Equal("11111111-2222-3333-4444-555555555555", m.ID.ValueString())
	r.Equal("prod-cud", m.Name.ValueString())
	r.Equal("GCP", m.Cloud.ValueString())
	r.Equal("us-central1", m.Region.ValueString())
	r.Equal("RESOURCE_CUD", m.Type.ValueString())
	// State string representation preserved (not reformatted).
	r.Equal("2026-01-01T00:00:00Z", m.StartTime.ValueString())
	r.Equal("2027-01-01T00:00:00Z", m.EndTime.ValueString())
	r.Equal("ACTIVE", m.AutoscalingStatus.ValueString())
	// Float32 round-trip must preserve the state value 0.9 exactly.
	r.Equal(0.9, m.AllowedUsage.ValueFloat64())
	r.True(m.Prioritization.ValueBool())
	r.Equal("DEFAULT", m.ScalingStrategy.ValueString())
	r.False(m.AutoAssignment.ValueBool())
	r.Equal("STATE_ACTIVE", m.State.ValueString())
	r.Equal("2026-07-30T12:00:00Z", m.CreateTime.ValueString())

	r.NotNil(m.GCPResourceCUDDetails)
	r.Equal("123456789", m.GCPResourceCUDDetails.CUDID.ValueString())
	r.Equal("TWELVE_MONTH", m.GCPResourceCUDDetails.Plan.ValueString())
	r.Equal(int64(131072), m.GCPResourceCUDDetails.MemoryMb.ValueInt64())
	r.Equal(int64(32), m.GCPResourceCUDDetails.CPU.ValueInt64())
}

func TestCommitmentModel_ApplyCommitment_AWSSavingsPlan(t *testing.T) {
	r := require.New(t)

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := &pricing.Commitment{
		Id:        ptr("id-1"),
		Name:      ptr("sp"),
		Cloud:     ptr(pricing.CommitmentCloudAWS),
		Region:    ptr("us-east-1"),
		Type:      ptr(pricing.CommitmentTypeSAVINGSPLAN),
		StartTime: &start,
		AwsSavingsPlanDetails: &pricing.AWSSavingsPlanDetails{
			Id:               ptr("sp-1"),
			Type:             ptr("Compute"),
			CommitmentAmount: ptr(float32(1.5)),
			CommitmentTerm:   ptr(pricing.COMMITMENTTERMUNITONEYEAR),
			PaymentOption:    ptr(pricing.NOUPFRONT),
		},
	}

	m := &commitmentModel{
		AWSSavingsPlanDetails: &awsSavingsPlanDetailsModel{
			CommitmentAmount: types.Float64Value(1.5),
		},
	}
	m.applyCommitment(context.Background(), c)

	r.NotNil(m.AWSSavingsPlanDetails)
	r.Equal("sp-1", m.AWSSavingsPlanDetails.ID.ValueString())
	r.Equal("Compute", m.AWSSavingsPlanDetails.Type.ValueString())
	// State value preserved through float32 round-trip.
	r.Equal(1.5, m.AWSSavingsPlanDetails.CommitmentAmount.ValueFloat64())
	r.Equal("COMMITMENT_TERM_UNIT_ONE_YEAR", m.AWSSavingsPlanDetails.CommitmentTerm.ValueString())
	r.Equal("NO_UPFRONT", m.AWSSavingsPlanDetails.PaymentOption.ValueString())
	// end_time stays null when API returns none.
	r.True(m.EndTime.IsNull())
}

func TestUpsertAndPatchChangeDetection(t *testing.T) {
	r := require.New(t)

	base := func() commitmentModel {
		return commitmentModel{
			Name:              types.StringValue("n"),
			Region:            types.StringValue("r"),
			StartTime:         types.StringValue("2026-01-01T00:00:00Z"),
			EndTime:           types.StringNull(),
			AutoscalingStatus: types.StringValue("INACTIVE"),
			AllowedUsage:      types.Float64Value(1.0),
			Prioritization:    types.BoolValue(false),
			ScalingStrategy:   types.StringValue("DEFAULT"),
			AutoAssignment:    types.BoolValue(false),
			GCPResourceCUDDetails: &gcpResourceCUDDetailsModel{
				CUDID: types.StringValue("1"),
				CPU:   types.Int64Value(32),
			},
		}
	}

	// No changes.
	a, b := base(), base()
	r.False(upsertPathChanged(&a, &b))
	r.False(patchPathChanged(&a, &b))

	// Name change → upsert only.
	a, b = base(), base()
	a.Name = types.StringValue("renamed")
	r.True(upsertPathChanged(&a, &b))
	r.False(patchPathChanged(&a, &b))

	// Details content change → upsert only.
	a, b = base(), base()
	a.GCPResourceCUDDetails.CPU = types.Int64Value(64)
	r.True(upsertPathChanged(&a, &b))
	r.False(patchPathChanged(&a, &b))

	// allowed_usage change → patch only.
	a, b = base(), base()
	a.AllowedUsage = types.Float64Value(0.5)
	r.False(upsertPathChanged(&a, &b))
	r.True(patchPathChanged(&a, &b))

	// autoscaling_status change → patch only.
	a, b = base(), base()
	a.AutoscalingStatus = types.StringValue("ACTIVE")
	r.False(upsertPathChanged(&a, &b))
	r.True(patchPathChanged(&a, &b))

	// Both paths.
	a, b = base(), base()
	a.Name = types.StringValue("renamed")
	a.ScalingStrategy = types.StringValue("CPU_BASED")
	r.True(upsertPathChanged(&a, &b))
	r.True(patchPathChanged(&a, &b))
}
