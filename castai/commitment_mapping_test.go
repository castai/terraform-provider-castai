package castai

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/castai/terraform-provider-castai/castai/sdk/pricing"
)

// ---------------------------------------------------------------------------
// toCreateInput: table-driven tests for all 9 detail-block types.
// ---------------------------------------------------------------------------

func TestCommitmentModel_ToCreateInput_AllDetailBlocks(t *testing.T) {
	tests := []struct {
		name      string
		model     *commitmentModel
		checkFunc func(t *testing.T, input pricing.CreateCommitmentInput)
	}{
		{
			name: "GCP Resource CUD",
			model: &commitmentModel{
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
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				assert.Equal(t, "prod-cud-us-central1", input.Name)
				assert.Equal(t, pricing.CreateCommitmentInputCloudGCP, input.Cloud)
				assert.Equal(t, "us-central1", input.Region)
				assert.Equal(t, pricing.CreateCommitmentInputTypeRESOURCECUD, input.Type)
				assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), input.StartTime.UTC())
				require.NotNil(t, input.EndTime)
				assert.Equal(t, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), input.EndTime.UTC())
				require.NotNil(t, input.AutoscalingStatus)
				assert.Equal(t, pricing.CreateCommitmentInputAutoscalingStatusACTIVE, *input.AutoscalingStatus)
				require.NotNil(t, input.AllowedUsage)
				assert.InDelta(t, 0.9, float64(*input.AllowedUsage), 1e-6)
				require.NotNil(t, input.Prioritization)
				assert.True(t, *input.Prioritization)
				require.NotNil(t, input.ScalingStrategy)
				assert.Equal(t, pricing.CreateCommitmentInputScalingStrategyCPUBASED, *input.ScalingStrategy)
				require.NotNil(t, input.AutoAssignment)
				assert.True(t, *input.AutoAssignment)

				require.NotNil(t, input.GcpResourceCudDetails)
				d := input.GcpResourceCudDetails
				assert.Equal(t, "123456789", *d.CudId)
				assert.Equal(t, pricing.GCPResourceCUDDetailsPlanTWELVEMONTH, *d.Plan)
				assert.Equal(t, "COMPUTE_OPTIMIZED_C2D", *d.Type)
				assert.Equal(t, "131072", *d.MemoryMb)
				assert.Equal(t, "32", *d.Cpu)
				assert.Equal(t, "ACTIVE", *d.Status)

				// No other details set.
				assert.Nil(t, input.AwsReservedInstancesDetails)
				assert.Nil(t, input.AzureReservationDetails)
				assert.Nil(t, input.AwsSavingsPlanDetails)
				assert.Nil(t, input.AwsCapacityBlockDetails)
				assert.Nil(t, input.AwsOdcrDetails)
				assert.Nil(t, input.GcpFlexCudDetails)
				assert.Nil(t, input.AzureSavingsPlanDetails)
				assert.Nil(t, input.GcpCapacityReservationDetails)
			},
		},
		{
			name: "AWS Reserved Instances",
			model: &commitmentModel{
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
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				assert.Nil(t, input.EndTime)
				assert.Nil(t, input.AutoscalingStatus)
				require.NotNil(t, input.AwsReservedInstancesDetails)
				d := input.AwsReservedInstancesDetails
				assert.Equal(t, "ri-0abc", *d.Id)
				assert.Equal(t, "Region", *d.Scope)
				assert.Equal(t, "m5.xlarge", *d.InstanceType)
				assert.Equal(t, "10", *d.InstanceCount)
				assert.Nil(t, d.AvailabilityZoneName)
				assert.Nil(t, d.AvailabilityZoneId)
			},
		},
		{
			name: "Azure Reservation",
			model: &commitmentModel{
				Name:      types.StringValue("prod-reservation"),
				Cloud:     types.StringValue("AZURE"),
				Region:    types.StringValue("eastus"),
				Type:      types.StringValue("RESERVED_INSTANCE"),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
				EndTime:   types.StringValue("2029-01-01T00:00:00Z"),
				AzureReservationDetails: &azureReservationDetailsModel{
					ID:                   types.StringValue("res-1"),
					Plan:                 types.StringValue("THREE_YEAR"),
					Status:               types.StringValue("Succeeded"),
					Scope:                types.StringValue("Shared"),
					ScopeSubscription:    types.StringValue("sub-1"),
					ScopeResourceGroup:   types.StringValue("rg-1"),
					ScopeManagementGroup: types.StringValue("mg-1"),
					ScopeTenant:          types.StringValue("tenant-1"),
					InstanceType:         types.StringValue("Standard_D4s_v3"),
					Count:                types.Int64Value(5),
					InstanceFlexibility:  types.StringValue("ON"),
					PurchaseTime:         types.StringValue("2026-01-15T10:00:00Z"),
				},
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				require.NotNil(t, input.AzureReservationDetails)
				d := input.AzureReservationDetails
				assert.Equal(t, "res-1", *d.Id)
				require.NotNil(t, d.Plan)
				assert.Equal(t, pricing.AzureReservationDetailsPlanTHREEYEAR, *d.Plan)
				assert.Equal(t, "Succeeded", *d.Status)
				assert.Equal(t, "Shared", *d.Scope)
				assert.Equal(t, "sub-1", *d.ScopeSubscription)
				assert.Equal(t, "rg-1", *d.ScopeResourceGroup)
				assert.Equal(t, "mg-1", *d.ScopeManagementGroup)
				assert.Equal(t, "tenant-1", *d.ScopeTenant)
				assert.Equal(t, "Standard_D4s_v3", *d.InstanceType)
				require.NotNil(t, d.Count)
				assert.Equal(t, int32(5), *d.Count)
				require.NotNil(t, d.InstanceFlexibility)
				assert.Equal(t, pricing.ON, *d.InstanceFlexibility)
				require.NotNil(t, d.PurchaseTime)
				assert.Equal(t, time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), d.PurchaseTime.UTC())
			},
		},
		{
			name: "AWS Savings Plan",
			model: &commitmentModel{
				Name:      types.StringValue("prod-sp"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringValue("us-east-1"),
				Type:      types.StringValue("SAVINGS_PLAN"),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
				AWSSavingsPlanDetails: &awsSavingsPlanDetailsModel{
					ID:                 types.StringValue("sp-1"),
					OfferingID:         types.StringValue("off-1"),
					Type:               types.StringValue("Compute"),
					State:              types.StringValue("active"),
					Region:             types.StringValue("us-east-1"),
					InstanceTypeFamily: types.StringValue("m5"),
					CommitmentAmount:   types.Float64Value(1.5),
					CommitmentTerm:     types.StringValue("COMMITMENT_TERM_UNIT_ONE_YEAR"),
					PaymentOption:      types.StringValue("NO_UPFRONT"),
				},
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				require.NotNil(t, input.AwsSavingsPlanDetails)
				d := input.AwsSavingsPlanDetails
				assert.Equal(t, "sp-1", *d.Id)
				assert.Equal(t, "off-1", *d.OfferingId)
				assert.Equal(t, "Compute", *d.Type)
				assert.Equal(t, "active", *d.State)
				assert.Equal(t, "us-east-1", *d.Region)
				assert.Equal(t, "m5", *d.InstanceTypeFamily)
				require.NotNil(t, d.CommitmentAmount)
				assert.InDelta(t, 1.5, float64(*d.CommitmentAmount), 1e-6)
				require.NotNil(t, d.CommitmentTerm)
				assert.Equal(t, pricing.COMMITMENTTERMUNITONEYEAR, *d.CommitmentTerm)
				require.NotNil(t, d.PaymentOption)
				assert.Equal(t, pricing.NOUPFRONT, *d.PaymentOption)
			},
		},
		{
			name: "AWS Capacity Block",
			model: &commitmentModel{
				Name:      types.StringValue("prod-cb"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringValue("us-east-1"),
				Type:      types.StringValue("CAPACITY_BLOCK"),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
				AWSCapacityBlockDetails: &awsCapacityBlockDetailsModel{
					ID:                     types.StringValue("cb-1"),
					AvailabilityZone:       types.StringValue("us-east-1a"),
					AvailabilityZoneID:     types.StringValue("use1-az1"),
					InstanceType:           types.StringValue("p4d.24xlarge"),
					InstancePlatform:       types.StringValue("Linux/UNIX"),
					TotalInstanceCount:     types.Int64Value(2),
					AvailableInstanceCount:  types.Int64Value(1),
					State:                   types.StringValue("active"),
				},
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				require.NotNil(t, input.AwsCapacityBlockDetails)
				d := input.AwsCapacityBlockDetails
				assert.Equal(t, "cb-1", *d.Id)
				assert.Equal(t, "us-east-1a", *d.AvailabilityZone)
				assert.Equal(t, "use1-az1", *d.AvailabilityZoneId)
				assert.Equal(t, "p4d.24xlarge", *d.InstanceType)
				assert.Equal(t, "Linux/UNIX", *d.InstancePlatform)
				assert.Equal(t, "2", *d.TotalInstanceCount)
				assert.Equal(t, "1", *d.AvailableInstanceCount)
				assert.Equal(t, "active", *d.State)
			},
		},
		{
			name: "AWS ODCR",
			model: &commitmentModel{
				Name:      types.StringValue("prod-odcr"),
				Cloud:     types.StringValue("AWS"),
				Region:    types.StringValue("us-east-1"),
				Type:      types.StringValue("ON_DEMAND_CAPACITY_RESERVATION"),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
				AWSODCRDetails: &awsODCRDetailsModel{
					ID:                     types.StringValue("odcr-1"),
					AvailabilityZone:       types.StringValue("us-east-1a"),
					AvailabilityZoneID:     types.StringValue("use1-az1"),
					InstanceType:           types.StringValue("m5.xlarge"),
					InstancePlatform:       types.StringValue("Linux/UNIX"),
					Tenancy:                types.StringValue("default"),
					TotalInstanceCount:     types.Int64Value(4),
					AvailableInstanceCount:  types.Int64Value(2),
					State:                   types.StringValue("active"),
					EndDateType:             types.StringValue("unlimited"),
					InstanceMatchCriteria:   types.StringValue("open"),
					Interruptible:           types.BoolValue(false),
				},
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				require.NotNil(t, input.AwsOdcrDetails)
				d := input.AwsOdcrDetails
				assert.Equal(t, "odcr-1", *d.Id)
				assert.Equal(t, "us-east-1a", *d.AvailabilityZone)
				assert.Equal(t, "use1-az1", *d.AvailabilityZoneId)
				assert.Equal(t, "m5.xlarge", *d.InstanceType)
				assert.Equal(t, "Linux/UNIX", *d.InstancePlatform)
				assert.Equal(t, "default", *d.Tenancy)
				assert.Equal(t, "4", *d.TotalInstanceCount)
				assert.Equal(t, "2", *d.AvailableInstanceCount)
				assert.Equal(t, "active", *d.State)
				assert.Equal(t, "unlimited", *d.EndDateType)
				assert.Equal(t, "open", *d.InstanceMatchCriteria)
				require.NotNil(t, d.Interruptible)
				assert.False(t, *d.Interruptible)
			},
		},
		{
			name: "GCP Flex CUD",
			model: &commitmentModel{
				Name:      types.StringValue("prod-flex-cud"),
				Cloud:     types.StringValue("GCP"),
				Region:    types.StringValue("us-central1"),
				Type:      types.StringValue("FLEX_CUD"),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
				GCPFlexCUDDetails: &gcpFlexCUDDetailsModel{
					OrderName:        types.StringValue("order-1"),
					DisplayName:      types.StringValue("my-flex-cud"),
					LineItemID:       types.StringValue("li-1"),
					Offer:            types.StringValue("offer-1"),
					Region:           types.StringValue("us-central1"),
					CommitmentAmount: types.Float64Value(5.0),
					State:            types.StringValue("ACTIVE"),
					Plan:             types.StringValue("THIRTY_SIX_MONTH"),
				},
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				require.NotNil(t, input.GcpFlexCudDetails)
				d := input.GcpFlexCudDetails
				assert.Equal(t, "order-1", *d.OrderName)
				assert.Equal(t, "my-flex-cud", *d.DisplayName)
				assert.Equal(t, "li-1", *d.LineItemId)
				assert.Equal(t, "offer-1", *d.Offer)
				assert.Equal(t, "us-central1", *d.Region)
				require.NotNil(t, d.CommitmentAmount)
				assert.Equal(t, 5.0, *d.CommitmentAmount)
				assert.Equal(t, "ACTIVE", *d.State)
				require.NotNil(t, d.Plan)
				assert.Equal(t, pricing.GCPFlexCUDDetailsPlanTHIRTYSIXMONTH, *d.Plan)
			},
		},
		{
			name: "Azure Savings Plan",
			model: &commitmentModel{
				Name:      types.StringValue("prod-asp"),
				Cloud:     types.StringValue("AZURE"),
				Region:    types.StringValue("eastus"),
				Type:      types.StringValue("SAVINGS_PLAN"),
				StartTime: types.StringValue("2026-01-01T00:00:00Z"),
				AzureSavingsPlanDetails: &azureSavingsPlanDetailsModel{
					ID:                   types.StringValue("asp-1"),
					Term:                 types.StringValue("THREE_YEARS"),
					ProvisioningState:    types.StringValue("Succeeded"),
					Scope:                types.StringValue("SHARED"),
					ScopeSubscription:    types.StringValue("sub-1"),
					ScopeResourceGroup:   types.StringValue("rg-1"),
					ScopeManagementGroup: types.StringValue("mg-1"),
					ScopeTenant:          types.StringValue("tenant-1"),
					CommitmentAmount:     types.Float64Value(10.0),
				},
			},
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				require.NotNil(t, input.AzureSavingsPlanDetails)
				d := input.AzureSavingsPlanDetails
				assert.Equal(t, "asp-1", *d.Id)
				require.NotNil(t, d.Term)
				assert.Equal(t, pricing.AzureSavingsPlanDetailsTermTHREEYEARS, *d.Term)
				assert.Equal(t, "Succeeded", *d.ProvisioningState)
				require.NotNil(t, d.Scope)
				assert.Equal(t, pricing.SHARED, *d.Scope)
				assert.Equal(t, "sub-1", *d.ScopeSubscription)
				assert.Equal(t, "rg-1", *d.ScopeResourceGroup)
				assert.Equal(t, "mg-1", *d.ScopeManagementGroup)
				assert.Equal(t, "tenant-1", *d.ScopeTenant)
				require.NotNil(t, d.CommitmentAmount)
				assert.Equal(t, 10.0, *d.CommitmentAmount)
			},
		},
		{
			name: "GCP Capacity Reservation",
			model: func() *commitmentModel {
				projectIDs, d0 := types.ListValueFrom(context.Background(), types.StringType, []string{"proj-a", "proj-b"})
				if d0.HasError() {
					panic(d0)
				}
				return &commitmentModel{
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
						AssuredInstanceCount:        types.Int64Value(2),
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
			}(),
			checkFunc: func(t *testing.T, input pricing.CreateCommitmentInput) {
				d := input.GcpCapacityReservationDetails
				require.NotNil(t, d)
				assert.Equal(t, "res-1", *d.Id)
				assert.Equal(t, "my-project", *d.ProjectId)
				assert.Equal(t, "us-central1-a", *d.Zone)
				assert.Equal(t, "a2-highgpu-1g", *d.InstanceType)
				assert.Equal(t, "4", *d.TotalInstanceCount)
				assert.Equal(t, "1", *d.InUseInstanceCount)
				assert.Equal(t, "2", *d.AssuredInstanceCount)
				assert.Equal(t, "READY", *d.State)
				require.NotNil(t, d.SpecificReservationRequired)
				assert.True(t, *d.SpecificReservationRequired)
				assert.Equal(t, "Intel Cascade Lake", *d.MinCpuPlatform)

				require.NotNil(t, d.ShareSettings)
				assert.Equal(t, pricing.GCPRESERVATIONSHARETYPESPECIFICPROJECTS, *d.ShareSettings.ShareType)
				assert.Equal(t, []string{"proj-a", "proj-b"}, *d.ShareSettings.ProjectIds)

				require.NotNil(t, d.Accelerators)
				assert.Len(t, *d.Accelerators, 1)
				assert.Equal(t, "1", *(*d.Accelerators)[0].AcceleratorCount)

				require.NotNil(t, d.LocalSsds)
				assert.Len(t, *d.LocalSsds, 1)
				assert.Equal(t, pricing.GCPRESERVATIONLOCALSSDINTERFACENVME, *(*d.LocalSsds)[0].Interface)
				assert.Equal(t, "375", *(*d.LocalSsds)[0].DiskSizeGb)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, diags := tt.model.toCreateInput(context.Background())
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
			tt.checkFunc(t, input)
		})
	}
}

func TestCommitmentModel_ToCreateInput_InvalidStartTime(t *testing.T) {
	m := &commitmentModel{
		Name:      types.StringValue("x"),
		Cloud:     types.StringValue("GCP"),
		Region:    types.StringValue("us-central1"),
		Type:      types.StringValue("RESOURCE_CUD"),
		StartTime: types.StringValue("not-a-timestamp"),
	}

	_, diags := m.toCreateInput(context.Background())
	require.True(t, diags.HasError())
}

// ---------------------------------------------------------------------------
// toUpdateInput
// ---------------------------------------------------------------------------

func TestCommitmentModel_ToUpdateInput(t *testing.T) {
	m := &commitmentModel{
		AutoscalingStatus: types.StringValue("ACTIVE"),
		AllowedUsage:      types.Float64Value(0.5),
		Prioritization:    types.BoolValue(true),
		ScalingStrategy:   types.StringValue("MEMORY_BASED"),
		AutoAssignment:    types.BoolValue(false),
	}

	input := m.toUpdateInput()
	require.NotNil(t, input.AutoscalingStatus)
	assert.Equal(t, pricing.ACTIVE, *input.AutoscalingStatus)
	require.NotNil(t, input.AllowedUsage)
	assert.InDelta(t, 0.5, float64(*input.AllowedUsage), 1e-6)
	require.NotNil(t, input.Prioritization)
	assert.True(t, *input.Prioritization)
	require.NotNil(t, input.ScalingStrategy)
	assert.Equal(t, pricing.MEMORYBASED, *input.ScalingStrategy)
	require.NotNil(t, input.AutoAssignment)
	assert.False(t, *input.AutoAssignment)
}

func TestCommitmentModel_ToUpdateInput_NullsOmitted(t *testing.T) {
	m := &commitmentModel{
		AutoscalingStatus: types.StringNull(),
		AllowedUsage:      types.Float64Null(),
		Prioritization:    types.BoolNull(),
		ScalingStrategy:   types.StringNull(),
		AutoAssignment:    types.BoolNull(),
	}

	input := m.toUpdateInput()
	assert.Nil(t, input.AutoscalingStatus)
	assert.Nil(t, input.AllowedUsage)
	assert.Nil(t, input.Prioritization)
	assert.Nil(t, input.ScalingStrategy)
	assert.Nil(t, input.AutoAssignment)
}

// ---------------------------------------------------------------------------
// applyCommitment: table-driven round-trip tests for all 9 detail-block types.
// ---------------------------------------------------------------------------

func TestCommitmentModel_ApplyDetails_AllDetailBlocks(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		commitment  *pricing.Commitment
		initialState *commitmentModel // pre-populated state to test sync preservation
		checkFunc   func(t *testing.T, m *commitmentModel)
	}{
		{
			name: "GCP Resource CUD",
			commitment: &pricing.Commitment{
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
				CreateTime:        &start,
				UpdateTime:        &start,
				GcpResourceCudDetails: &pricing.GCPResourceCUDDetails{
					CudId:    ptr("123456789"),
					Plan:     ptr(pricing.GCPResourceCUDDetailsPlanTWELVEMONTH),
					Type:     ptr("COMPUTE_OPTIMIZED_C2D"),
					MemoryMb: ptr("131072"),
					Cpu:      ptr("32"),
					Status:   ptr("ACTIVE"),
				},
			},
			initialState: &commitmentModel{
				StartTime:    types.StringValue("2026-01-01T00:00:00Z"),
				AllowedUsage: types.Float64Value(0.9),
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				assert.Equal(t, "11111111-2222-3333-4444-555555555555", m.ID.ValueString())
				assert.Equal(t, "prod-cud", m.Name.ValueString())
				assert.Equal(t, "GCP", m.Cloud.ValueString())
				assert.Equal(t, "us-central1", m.Region.ValueString())
				assert.Equal(t, "RESOURCE_CUD", m.Type.ValueString())
				// State string representation preserved (not reformatted).
				assert.Equal(t, "2026-01-01T00:00:00Z", m.StartTime.ValueString())
				assert.Equal(t, "2027-01-01T00:00:00Z", m.EndTime.ValueString())
				assert.Equal(t, "ACTIVE", m.AutoscalingStatus.ValueString())
				// Float32 round-trip must preserve the state value 0.9 exactly.
				assert.Equal(t, 0.9, m.AllowedUsage.ValueFloat64())
				assert.True(t, m.Prioritization.ValueBool())
				assert.Equal(t, "DEFAULT", m.ScalingStrategy.ValueString())
				assert.False(t, m.AutoAssignment.ValueBool())
				assert.Equal(t, "STATE_ACTIVE", m.State.ValueString())
				assert.Equal(t, "2026-01-01T00:00:00Z", m.CreateTime.ValueString())

				require.NotNil(t, m.GCPResourceCUDDetails)
				assert.Equal(t, "123456789", m.GCPResourceCUDDetails.CUDID.ValueString())
				assert.Equal(t, "TWELVE_MONTH", m.GCPResourceCUDDetails.Plan.ValueString())
				assert.Equal(t, int64(131072), m.GCPResourceCUDDetails.MemoryMb.ValueInt64())
				assert.Equal(t, int64(32), m.GCPResourceCUDDetails.CPU.ValueInt64())
			},
		},
		{
			name: "AWS Reserved Instances",
			commitment: &pricing.Commitment{
				Id:        ptr("ri-id"),
				Name:      ptr("prod-ri"),
				Cloud:     ptr(pricing.CommitmentCloudAWS),
				Region:    ptr("us-east-1"),
				Type:      ptr(pricing.CommitmentTypeRESERVEDINSTANCE),
				StartTime: &start,
				AwsReservedInstancesDetails: &pricing.AWSReservedInstancesDetails{
					Id:                   ptr("ri-0abc"),
					Scope:                ptr("Region"),
					AvailabilityZoneName: ptr("us-east-1a"),
					AvailabilityZoneId:   ptr("use1-az1"),
					InstanceType:         ptr("m5.xlarge"),
					InstanceCount:        ptr("10"),
					State:                ptr("active"),
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.AWSReservedInstancesDetails)
				d := m.AWSReservedInstancesDetails
				assert.Equal(t, "ri-0abc", d.ID.ValueString())
				assert.Equal(t, "Region", d.Scope.ValueString())
				assert.Equal(t, "us-east-1a", d.AvailabilityZoneName.ValueString())
				assert.Equal(t, "use1-az1", d.AvailabilityZoneID.ValueString())
				assert.Equal(t, "m5.xlarge", d.InstanceType.ValueString())
				assert.Equal(t, int64(10), d.InstanceCount.ValueInt64())
				assert.Equal(t, "active", d.State.ValueString())
			},
		},
		{
			name: "Azure Reservation",
			commitment: &pricing.Commitment{
				Id:        ptr("az-res-id"),
				Name:      ptr("prod-res"),
				Cloud:     ptr(pricing.CommitmentCloudAZURE),
				Region:    ptr("eastus"),
				Type:      ptr(pricing.CommitmentTypeRESERVEDINSTANCE),
				StartTime: &start,
				AzureReservationDetails: &pricing.AzureReservationDetails{
					Id:                   ptr("res-1"),
					Plan:                 ptr(pricing.AzureReservationDetailsPlanTHREEYEAR),
					Status:               ptr("Succeeded"),
					Scope:                ptr("Shared"),
					ScopeSubscription:    ptr("sub-1"),
					ScopeResourceGroup:   ptr("rg-1"),
					ScopeManagementGroup: ptr("mg-1"),
					ScopeTenant:          ptr("tenant-1"),
					InstanceType:         ptr("Standard_D4s_v3"),
					Count:                ptr(int32(5)),
					InstanceFlexibility:  ptr(pricing.ON),
					PurchaseTime:         &start,
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.AzureReservationDetails)
				d := m.AzureReservationDetails
				assert.Equal(t, "res-1", d.ID.ValueString())
				assert.Equal(t, "THREE_YEAR", d.Plan.ValueString())
				assert.Equal(t, "Succeeded", d.Status.ValueString())
				assert.Equal(t, "Shared", d.Scope.ValueString())
				assert.Equal(t, "sub-1", d.ScopeSubscription.ValueString())
				assert.Equal(t, "rg-1", d.ScopeResourceGroup.ValueString())
				assert.Equal(t, "mg-1", d.ScopeManagementGroup.ValueString())
				assert.Equal(t, "tenant-1", d.ScopeTenant.ValueString())
				assert.Equal(t, "Standard_D4s_v3", d.InstanceType.ValueString())
				assert.Equal(t, int64(5), d.Count.ValueInt64())
				assert.Equal(t, "ON", d.InstanceFlexibility.ValueString())
				assert.Equal(t, "2026-01-01T00:00:00Z", d.PurchaseTime.ValueString())
			},
		},
		{
			name: "AWS Savings Plan",
			commitment: &pricing.Commitment{
				Id:        ptr("sp-id"),
				Name:      ptr("sp"),
				Cloud:     ptr(pricing.CommitmentCloudAWS),
				Region:    ptr("us-east-1"),
				Type:      ptr(pricing.CommitmentTypeSAVINGSPLAN),
				StartTime: &start,
				AwsSavingsPlanDetails: &pricing.AWSSavingsPlanDetails{
					Id:                 ptr("sp-1"),
					OfferingId:         ptr("off-1"),
					Type:               ptr("Compute"),
					State:              ptr("active"),
					Region:             ptr("us-east-1"),
					InstanceTypeFamily: ptr("m5"),
					CommitmentAmount:   ptr(float32(1.5)),
					CommitmentTerm:     ptr(pricing.COMMITMENTTERMUNITONEYEAR),
					PaymentOption:      ptr(pricing.NOUPFRONT),
				},
			},
			initialState: &commitmentModel{
				AWSSavingsPlanDetails: &awsSavingsPlanDetailsModel{
					CommitmentAmount: types.Float64Value(1.5),
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.AWSSavingsPlanDetails)
				d := m.AWSSavingsPlanDetails
				assert.Equal(t, "sp-1", d.ID.ValueString())
				assert.Equal(t, "off-1", d.OfferingID.ValueString())
				assert.Equal(t, "Compute", d.Type.ValueString())
				assert.Equal(t, "active", d.State.ValueString())
				assert.Equal(t, "us-east-1", d.Region.ValueString())
				assert.Equal(t, "m5", d.InstanceTypeFamily.ValueString())
				// State value preserved through float32 round-trip.
				assert.Equal(t, 1.5, d.CommitmentAmount.ValueFloat64())
				assert.Equal(t, "COMMITMENT_TERM_UNIT_ONE_YEAR", d.CommitmentTerm.ValueString())
				assert.Equal(t, "NO_UPFRONT", d.PaymentOption.ValueString())
				// end_time stays null when API returns none.
				assert.True(t, m.EndTime.IsNull())
			},
		},
		{
			name: "AWS Capacity Block",
			commitment: &pricing.Commitment{
				Id:        ptr("cb-id"),
				Name:      ptr("cb"),
				Cloud:     ptr(pricing.CommitmentCloudAWS),
				Region:    ptr("us-east-1"),
				Type:      ptr(pricing.CommitmentTypeCAPACITYBLOCK),
				StartTime: &start,
				AwsCapacityBlockDetails: &pricing.AWSCapacityBlockDetails{
					Id:                     ptr("cb-1"),
					AvailabilityZone:       ptr("us-east-1a"),
					AvailabilityZoneId:     ptr("use1-az1"),
					InstanceType:           ptr("p4d.24xlarge"),
					InstancePlatform:       ptr("Linux/UNIX"),
					TotalInstanceCount:     ptr("2"),
					AvailableInstanceCount: ptr("1"),
					State:                   ptr("active"),
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.AWSCapacityBlockDetails)
				d := m.AWSCapacityBlockDetails
				assert.Equal(t, "cb-1", d.ID.ValueString())
				assert.Equal(t, "us-east-1a", d.AvailabilityZone.ValueString())
				assert.Equal(t, "use1-az1", d.AvailabilityZoneID.ValueString())
				assert.Equal(t, "p4d.24xlarge", d.InstanceType.ValueString())
				assert.Equal(t, "Linux/UNIX", d.InstancePlatform.ValueString())
				assert.Equal(t, int64(2), d.TotalInstanceCount.ValueInt64())
				assert.Equal(t, int64(1), d.AvailableInstanceCount.ValueInt64())
				assert.Equal(t, "active", d.State.ValueString())
			},
		},
		{
			name: "AWS ODCR",
			commitment: &pricing.Commitment{
				Id:        ptr("odcr-id"),
				Name:      ptr("odcr"),
				Cloud:     ptr(pricing.CommitmentCloudAWS),
				Region:    ptr("us-east-1"),
				Type:      ptr(pricing.CommitmentTypeONDEMANDCAPACITYRESERVATION),
				StartTime: &start,
				AwsOdcrDetails: &pricing.AWSODCRDetails{
					Id:                     ptr("odcr-1"),
					AvailabilityZone:       ptr("us-east-1a"),
					AvailabilityZoneId:     ptr("use1-az1"),
					InstanceType:           ptr("m5.xlarge"),
					InstancePlatform:       ptr("Linux/UNIX"),
					Tenancy:                ptr("default"),
					TotalInstanceCount:     ptr("4"),
					AvailableInstanceCount: ptr("2"),
					State:                   ptr("active"),
					EndDateType:             ptr("unlimited"),
					InstanceMatchCriteria:   ptr("open"),
					Interruptible:           ptr(false),
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.AWSODCRDetails)
				d := m.AWSODCRDetails
				assert.Equal(t, "odcr-1", d.ID.ValueString())
				assert.Equal(t, "us-east-1a", d.AvailabilityZone.ValueString())
				assert.Equal(t, "use1-az1", d.AvailabilityZoneID.ValueString())
				assert.Equal(t, "m5.xlarge", d.InstanceType.ValueString())
				assert.Equal(t, "Linux/UNIX", d.InstancePlatform.ValueString())
				assert.Equal(t, "default", d.Tenancy.ValueString())
				assert.Equal(t, int64(4), d.TotalInstanceCount.ValueInt64())
				assert.Equal(t, int64(2), d.AvailableInstanceCount.ValueInt64())
				assert.Equal(t, "active", d.State.ValueString())
				assert.Equal(t, "unlimited", d.EndDateType.ValueString())
				assert.Equal(t, "open", d.InstanceMatchCriteria.ValueString())
				assert.False(t, d.Interruptible.ValueBool())
			},
		},
		{
			name: "GCP Flex CUD",
			commitment: &pricing.Commitment{
				Id:        ptr("flex-id"),
				Name:      ptr("flex"),
				Cloud:     ptr(pricing.CommitmentCloudGCP),
				Region:    ptr("us-central1"),
				Type:      ptr(pricing.CommitmentTypeFLEXCUD),
				StartTime: &start,
				GcpFlexCudDetails: &pricing.GCPFlexCUDDetails{
					OrderName:        ptr("order-1"),
					DisplayName:      ptr("my-flex-cud"),
					LineItemId:       ptr("li-1"),
					Offer:            ptr("offer-1"),
					Region:           ptr("us-central1"),
					CommitmentAmount: ptr(5.0),
					State:            ptr("ACTIVE"),
					Plan:             ptr(pricing.GCPFlexCUDDetailsPlanTHIRTYSIXMONTH),
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.GCPFlexCUDDetails)
				d := m.GCPFlexCUDDetails
				assert.Equal(t, "order-1", d.OrderName.ValueString())
				assert.Equal(t, "my-flex-cud", d.DisplayName.ValueString())
				assert.Equal(t, "li-1", d.LineItemID.ValueString())
				assert.Equal(t, "offer-1", d.Offer.ValueString())
				assert.Equal(t, "us-central1", d.Region.ValueString())
				assert.Equal(t, 5.0, d.CommitmentAmount.ValueFloat64())
				assert.Equal(t, "ACTIVE", d.State.ValueString())
				assert.Equal(t, "THIRTY_SIX_MONTH", d.Plan.ValueString())
			},
		},
		{
			name: "Azure Savings Plan",
			commitment: &pricing.Commitment{
				Id:        ptr("asp-id"),
				Name:      ptr("asp"),
				Cloud:     ptr(pricing.CommitmentCloudAZURE),
				Region:    ptr("eastus"),
				Type:      ptr(pricing.CommitmentTypeSAVINGSPLAN),
				StartTime: &start,
				AzureSavingsPlanDetails: &pricing.AzureSavingsPlanDetails{
					Id:                   ptr("asp-1"),
					Term:                 ptr(pricing.AzureSavingsPlanDetailsTermTHREEYEARS),
					ProvisioningState:    ptr("Succeeded"),
					Scope:                ptr(pricing.SHARED),
					ScopeSubscription:    ptr("sub-1"),
					ScopeResourceGroup:   ptr("rg-1"),
					ScopeManagementGroup: ptr("mg-1"),
					ScopeTenant:          ptr("tenant-1"),
					CommitmentAmount:     ptr(10.0),
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.AzureSavingsPlanDetails)
				d := m.AzureSavingsPlanDetails
				assert.Equal(t, "asp-1", d.ID.ValueString())
				assert.Equal(t, "THREE_YEARS", d.Term.ValueString())
				assert.Equal(t, "Succeeded", d.ProvisioningState.ValueString())
				assert.Equal(t, "SHARED", d.Scope.ValueString())
				assert.Equal(t, "sub-1", d.ScopeSubscription.ValueString())
				assert.Equal(t, "rg-1", d.ScopeResourceGroup.ValueString())
				assert.Equal(t, "mg-1", d.ScopeManagementGroup.ValueString())
				assert.Equal(t, "tenant-1", d.ScopeTenant.ValueString())
				assert.Equal(t, 10.0, d.CommitmentAmount.ValueFloat64())
			},
		},
		{
			name: "GCP Capacity Reservation",
			commitment: &pricing.Commitment{
				Id:        ptr("gcr-id"),
				Name:      ptr("gcr"),
				Cloud:     ptr(pricing.CommitmentCloudGCP),
				Region:    ptr("us-central1"),
				Type:      ptr(pricing.CommitmentTypeONDEMANDCAPACITYRESERVATION),
				StartTime: &start,
				GcpCapacityReservationDetails: &pricing.GCPCapacityReservationDetails{
					Id:                          ptr("res-1"),
					ProjectId:                   ptr("my-project"),
					Zone:                        ptr("us-central1-a"),
					InstanceType:                ptr("a2-highgpu-1g"),
					TotalInstanceCount:          ptr("4"),
					InUseInstanceCount:          ptr("1"),
					AssuredInstanceCount:        ptr("2"),
					State:                       ptr("READY"),
					SpecificReservationRequired: ptr(true),
					MinCpuPlatform:              ptr("Intel Cascade Lake"),
					ShareSettings: &pricing.GCPReservationShareSettings{
						ShareType:  ptr(pricing.GCPRESERVATIONSHARETYPESPECIFICPROJECTS),
						ProjectIds: ptr([]string{"proj-a", "proj-b"}),
					},
					Accelerators: ptr([]pricing.GCPReservationAccelerator{
						{
							AcceleratorType:  ptr("https://www.googleapis.com/compute/v1/projects/p/zones/z/acceleratorTypes/nvidia-tesla-t4"),
							AcceleratorCount: ptr("1"),
						},
					}),
					LocalSsds: ptr([]pricing.GCPReservationLocalSsd{
						{
							Interface:  ptr(pricing.GCPRESERVATIONLOCALSSDINTERFACENVME),
							DiskSizeGb: ptr("375"),
						},
					}),
				},
			},
			checkFunc: func(t *testing.T, m *commitmentModel) {
				require.NotNil(t, m.GCPCapacityReservationDetails)
				d := m.GCPCapacityReservationDetails
				assert.Equal(t, "res-1", d.ID.ValueString())
				assert.Equal(t, "my-project", d.ProjectID.ValueString())
				assert.Equal(t, "us-central1-a", d.Zone.ValueString())
				assert.Equal(t, "a2-highgpu-1g", d.InstanceType.ValueString())
				assert.Equal(t, int64(4), d.TotalInstanceCount.ValueInt64())
				assert.Equal(t, int64(1), d.InUseInstanceCount.ValueInt64())
				assert.Equal(t, int64(2), d.AssuredInstanceCount.ValueInt64())
				assert.Equal(t, "READY", d.State.ValueString())
				assert.True(t, d.SpecificReservationRequired.ValueBool())
				assert.Equal(t, "Intel Cascade Lake", d.MinCPUPlatform.ValueString())

				require.NotNil(t, d.ShareSettings)
				assert.Equal(t, "GCP_RESERVATION_SHARE_TYPE_SPECIFIC_PROJECTS", d.ShareSettings.ShareType.ValueString())
				assert.Equal(t, 2, len(d.ShareSettings.ProjectIDs.Elements()))

				require.Len(t, d.Accelerators, 1)
				assert.Equal(t, "https://www.googleapis.com/compute/v1/projects/p/zones/z/acceleratorTypes/nvidia-tesla-t4", d.Accelerators[0].AcceleratorType.ValueString())
				assert.Equal(t, int64(1), d.Accelerators[0].AcceleratorCount.ValueInt64())

				require.Len(t, d.LocalSsds, 1)
				assert.Equal(t, "GCP_RESERVATION_LOCAL_SSD_INTERFACE_NVME", d.LocalSsds[0].Interface.ValueString())
				assert.Equal(t, int64(375), d.LocalSsds[0].DiskSizeGb.ValueInt64())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.initialState
			if m == nil {
				m = &commitmentModel{}
			}
			diags := m.applyCommitment(context.Background(), tt.commitment)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)
			tt.checkFunc(t, m)
		})
	}
}

// ---------------------------------------------------------------------------
// Change detection (upsert vs patch path).
// ---------------------------------------------------------------------------

func TestUpsertAndPatchChangeDetection(t *testing.T) {
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
	assert.False(t, upsertPathChanged(&a, &b))
	assert.False(t, patchPathChanged(&a, &b))

	// Name change → upsert only.
	a, b = base(), base()
	a.Name = types.StringValue("renamed")
	assert.True(t, upsertPathChanged(&a, &b))
	assert.False(t, patchPathChanged(&a, &b))

	// Details content change → upsert only.
	a, b = base(), base()
	a.GCPResourceCUDDetails.CPU = types.Int64Value(64)
	assert.True(t, upsertPathChanged(&a, &b))
	assert.False(t, patchPathChanged(&a, &b))

	// allowed_usage change → patch only.
	a, b = base(), base()
	a.AllowedUsage = types.Float64Value(0.5)
	assert.False(t, upsertPathChanged(&a, &b))
	assert.True(t, patchPathChanged(&a, &b))

	// autoscaling_status change → patch only.
	a, b = base(), base()
	a.AutoscalingStatus = types.StringValue("ACTIVE")
	assert.False(t, upsertPathChanged(&a, &b))
	assert.True(t, patchPathChanged(&a, &b))

	// Both paths.
	a, b = base(), base()
	a.Name = types.StringValue("renamed")
	a.ScalingStrategy = types.StringValue("CPU_BASED")
	assert.True(t, upsertPathChanged(&a, &b))
	assert.True(t, patchPathChanged(&a, &b))
}

func TestDetailsEqual_NestedListChanges(t *testing.T) {
	base := func() commitmentModel {
		return commitmentModel{
			Name:      types.StringValue("n"),
			Region:    types.StringValue("r"),
			StartTime: types.StringValue("2026-01-01T00:00:00Z"),
			GCPCapacityReservationDetails: &gcpCapacityReservationDetailsModel{
				ID:                 types.StringValue("res-1"),
				TotalInstanceCount: types.Int64Value(4),
			},
		}
	}

	// No changes.
	a, b := base(), base()
	assert.False(t, upsertPathChanged(&a, &b))

	// Accelerator count change detected.
	a, b = base(), base()
	a.GCPCapacityReservationDetails.Accelerators = []gcpReservationAcceleratorModel{
		{AcceleratorType: types.StringValue("t4"), AcceleratorCount: types.Int64Value(1)},
	}
	b.GCPCapacityReservationDetails.Accelerators = []gcpReservationAcceleratorModel{
		{AcceleratorType: types.StringValue("t4"), AcceleratorCount: types.Int64Value(2)},
	}
	assert.True(t, upsertPathChanged(&a, &b))

	// Accelerator added vs empty detected.
	a, b = base(), base()
	a.GCPCapacityReservationDetails.Accelerators = []gcpReservationAcceleratorModel{
		{AcceleratorType: types.StringValue("t4"), AcceleratorCount: types.Int64Value(1)},
	}
	b.GCPCapacityReservationDetails.Accelerators = nil
	assert.True(t, upsertPathChanged(&a, &b))

	// Local SSD interface change detected.
	a, b = base(), base()
	a.GCPCapacityReservationDetails.LocalSsds = []gcpReservationLocalSsdModel{
		{Interface: types.StringValue("GCP_RESERVATION_LOCAL_SSD_INTERFACE_NVME"), DiskSizeGb: types.Int64Value(375)},
	}
	b.GCPCapacityReservationDetails.LocalSsds = []gcpReservationLocalSsdModel{
		{Interface: types.StringValue("GCP_RESERVATION_LOCAL_SSD_INTERFACE_SCSI"), DiskSizeGb: types.Int64Value(375)},
	}
	assert.True(t, upsertPathChanged(&a, &b))

	// Share settings share_type change detected.
	a, b = base(), base()
	a.GCPCapacityReservationDetails.ShareSettings = &gcpReservationShareSettingsModel{
		ShareType: types.StringValue("GCP_RESERVATION_SHARE_TYPE_LOCAL"),
	}
	b.GCPCapacityReservationDetails.ShareSettings = &gcpReservationShareSettingsModel{
		ShareType: types.StringValue("GCP_RESERVATION_SHARE_TYPE_ORGANIZATION"),
	}
	assert.True(t, upsertPathChanged(&a, &b))
}
