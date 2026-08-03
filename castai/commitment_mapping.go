package castai

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/castai/terraform-provider-castai/castai/sdk/pricing"
)

// commitmentModel is the Terraform state/plan model for the castai_commitment resource.
type commitmentModel struct {
	ID                types.String  `tfsdk:"id"`
	OrganizationID    types.String  `tfsdk:"organization_id"`
	Name              types.String  `tfsdk:"name"`
	Cloud             types.String  `tfsdk:"cloud"`
	Region            types.String  `tfsdk:"region"`
	Type              types.String  `tfsdk:"type"`
	StartTime         types.String  `tfsdk:"start_time"`
	EndTime           types.String  `tfsdk:"end_time"`
	AutoscalingStatus types.String  `tfsdk:"autoscaling_status"`
	AllowedUsage      types.Float64 `tfsdk:"allowed_usage"`
	Prioritization    types.Bool    `tfsdk:"prioritization"`
	ScalingStrategy   types.String  `tfsdk:"scaling_strategy"`
	AutoAssignment    types.Bool    `tfsdk:"auto_assignment"`
	State             types.String  `tfsdk:"state"`
	CreateTime        types.String  `tfsdk:"create_time"`
	UpdateTime        types.String  `tfsdk:"update_time"`

	AWSReservedInstancesDetails   *awsReservedInstancesDetailsModel   `tfsdk:"aws_reserved_instances_details"`
	AzureReservationDetails       *azureReservationDetailsModel       `tfsdk:"azure_reservation_details"`
	GCPResourceCUDDetails         *gcpResourceCUDDetailsModel         `tfsdk:"gcp_resource_cud_details"`
	AWSSavingsPlanDetails         *awsSavingsPlanDetailsModel         `tfsdk:"aws_savings_plan_details"`
	AWSCapacityBlockDetails       *awsCapacityBlockDetailsModel       `tfsdk:"aws_capacity_block_details"`
	AWSODCRDetails                *awsODCRDetailsModel                `tfsdk:"aws_odcr_details"`
	GCPFlexCUDDetails             *gcpFlexCUDDetailsModel             `tfsdk:"gcp_flex_cud_details"`
	AzureSavingsPlanDetails       *azureSavingsPlanDetailsModel       `tfsdk:"azure_savings_plan_details"`
	GCPCapacityReservationDetails *gcpCapacityReservationDetailsModel `tfsdk:"gcp_capacity_reservation_details"`
}

type awsReservedInstancesDetailsModel struct {
	ID                   types.String `tfsdk:"id"`
	Scope                types.String `tfsdk:"scope"`
	AvailabilityZoneName types.String `tfsdk:"availability_zone_name"`
	AvailabilityZoneID   types.String `tfsdk:"availability_zone_id"`
	InstanceType         types.String `tfsdk:"instance_type"`
	InstanceCount        types.Int64  `tfsdk:"instance_count"`
	State                types.String `tfsdk:"state"`
}

type azureReservationDetailsModel struct {
	ID                   types.String `tfsdk:"id"`
	Plan                 types.String `tfsdk:"plan"`
	Status               types.String `tfsdk:"status"`
	Scope                types.String `tfsdk:"scope"`
	ScopeSubscription    types.String `tfsdk:"scope_subscription"`
	ScopeResourceGroup   types.String `tfsdk:"scope_resource_group"`
	ScopeManagementGroup types.String `tfsdk:"scope_management_group"`
	ScopeTenant          types.String `tfsdk:"scope_tenant"`
	InstanceType         types.String `tfsdk:"instance_type"`
	Count                types.Int64  `tfsdk:"count"`
	InstanceFlexibility  types.String `tfsdk:"instance_flexibility"`
	PurchaseTime         types.String `tfsdk:"purchase_time"`
}

type gcpResourceCUDDetailsModel struct {
	CUDID    types.String `tfsdk:"cud_id"`
	Plan     types.String `tfsdk:"plan"`
	Type     types.String `tfsdk:"type"`
	MemoryMb types.Int64  `tfsdk:"memory_mb"`
	CPU      types.Int64  `tfsdk:"cpu"`
	Status   types.String `tfsdk:"status"`
}

type awsSavingsPlanDetailsModel struct {
	ID                 types.String  `tfsdk:"id"`
	OfferingID         types.String  `tfsdk:"offering_id"`
	Type               types.String  `tfsdk:"type"`
	State              types.String  `tfsdk:"state"`
	Region             types.String  `tfsdk:"region"`
	InstanceTypeFamily types.String  `tfsdk:"instance_type_family"`
	CommitmentAmount   types.Float64 `tfsdk:"commitment_amount"`
	CommitmentTerm     types.String  `tfsdk:"commitment_term"`
	PaymentOption      types.String  `tfsdk:"payment_option"`
}

type awsCapacityBlockDetailsModel struct {
	ID                     types.String `tfsdk:"id"`
	AvailabilityZone       types.String `tfsdk:"availability_zone"`
	AvailabilityZoneID     types.String `tfsdk:"availability_zone_id"`
	InstanceType           types.String `tfsdk:"instance_type"`
	InstancePlatform       types.String `tfsdk:"instance_platform"`
	TotalInstanceCount     types.Int64  `tfsdk:"total_instance_count"`
	AvailableInstanceCount types.Int64  `tfsdk:"available_instance_count"`
	State                  types.String `tfsdk:"state"`
}

type awsODCRDetailsModel struct {
	ID                     types.String `tfsdk:"id"`
	AvailabilityZone       types.String `tfsdk:"availability_zone"`
	AvailabilityZoneID     types.String `tfsdk:"availability_zone_id"`
	InstanceType           types.String `tfsdk:"instance_type"`
	InstancePlatform       types.String `tfsdk:"instance_platform"`
	Tenancy                types.String `tfsdk:"tenancy"`
	TotalInstanceCount     types.Int64  `tfsdk:"total_instance_count"`
	AvailableInstanceCount types.Int64  `tfsdk:"available_instance_count"`
	State                  types.String `tfsdk:"state"`
	EndDateType            types.String `tfsdk:"end_date_type"`
	InstanceMatchCriteria  types.String `tfsdk:"instance_match_criteria"`
	Interruptible          types.Bool   `tfsdk:"interruptible"`
}

type gcpFlexCUDDetailsModel struct {
	OrderName        types.String  `tfsdk:"order_name"`
	DisplayName      types.String  `tfsdk:"display_name"`
	LineItemID       types.String  `tfsdk:"line_item_id"`
	Offer            types.String  `tfsdk:"offer"`
	Region           types.String  `tfsdk:"region"`
	CommitmentAmount types.Float64 `tfsdk:"commitment_amount"`
	State            types.String  `tfsdk:"state"`
	Plan             types.String  `tfsdk:"plan"`
}

type azureSavingsPlanDetailsModel struct {
	ID                   types.String  `tfsdk:"id"`
	Term                 types.String  `tfsdk:"term"`
	ProvisioningState    types.String  `tfsdk:"provisioning_state"`
	Scope                types.String  `tfsdk:"scope"`
	ScopeSubscription    types.String  `tfsdk:"scope_subscription"`
	ScopeResourceGroup   types.String  `tfsdk:"scope_resource_group"`
	ScopeManagementGroup types.String  `tfsdk:"scope_management_group"`
	ScopeTenant          types.String  `tfsdk:"scope_tenant"`
	CommitmentAmount     types.Float64 `tfsdk:"commitment_amount"`
}

type gcpCapacityReservationDetailsModel struct {
	ID                          types.String                          `tfsdk:"id"`
	ProjectID                   types.String                          `tfsdk:"project_id"`
	Zone                        types.String                          `tfsdk:"zone"`
	InstanceType                types.String                          `tfsdk:"instance_type"`
	TotalInstanceCount          types.Int64                           `tfsdk:"total_instance_count"`
	InUseInstanceCount          types.Int64                           `tfsdk:"in_use_instance_count"`
	AssuredInstanceCount        types.Int64                           `tfsdk:"assured_instance_count"`
	State                       types.String                          `tfsdk:"state"`
	SpecificReservationRequired types.Bool                            `tfsdk:"specific_reservation_required"`
	MinCPUPlatform              types.String                          `tfsdk:"min_cpu_platform"`
	ShareSettings               *gcpReservationShareSettingsModel     `tfsdk:"share_settings"`
	Accelerators                []gcpReservationAcceleratorModel      `tfsdk:"accelerators"`
	LocalSsds                   []gcpReservationLocalSsdModel         `tfsdk:"local_ssds"`
}

type gcpReservationShareSettingsModel struct {
	ShareType  types.String `tfsdk:"share_type"`
	ProjectIDs types.List   `tfsdk:"project_ids"`
}

type gcpReservationAcceleratorModel struct {
	AcceleratorType  types.String `tfsdk:"accelerator_type"`
	AcceleratorCount types.Int64  `tfsdk:"accelerator_count"`
}

type gcpReservationLocalSsdModel struct {
	Interface  types.String `tfsdk:"interface"`
	DiskSizeGb types.Int64  `tfsdk:"disk_size_gb"`
}

// ---------------------------------------------------------------------------
// Conversion helpers.
// ---------------------------------------------------------------------------

// optStr converts a framework string to *string, returning nil for null/unknown.
func optStr(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	return s.ValueStringPointer()
}

// strOrNull converts an API string pointer to a framework string, treating
// nil and empty as null to avoid perpetual diffs on optional fields.
func strOrNull(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// syncStr preserves the state value when the API returns nil or empty string.
// This prevents perpetual diffs when the user configures an empty string ("")
// but the API omits the field (returns nil). Only used in Read/apply paths.
func syncStr(state types.String, s *string) types.String {
	if s == nil || *s == "" {
		if !state.IsNull() && !state.IsUnknown() {
			return state
		}
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// syncBool preserves the state value when the API returns nil.
func syncBool(state types.Bool, b *bool) types.Bool {
	if b == nil {
		if !state.IsNull() && !state.IsUnknown() {
			return state
		}
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

// syncEnumStr preserves the state value when the API returns nil or an
// UNSPECIFIED enum value, preventing perpetual diffs from enum defaults.
func syncEnumStr(state types.String, s *string, unspecified string) types.String {
	if s == nil || *s == unspecified || *s == "" {
		if !state.IsNull() && !state.IsUnknown() {
			return state
		}
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// optInt64Str converts a framework int64 to the SDK's string representation
// of proto int64 fields, returning nil for null/unknown.
func optInt64Str(v types.Int64) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return ptr(strconv.FormatInt(v.ValueInt64(), 10))
}

// int64FromStr converts the SDK's string-encoded int64 back to a framework int64.
func int64FromStr(s *string) types.Int64 {
	if s == nil || *s == "" {
		return types.Int64Null()
	}
	v, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return types.Int64Null()
	}
	return types.Int64Value(v)
}

// optInt32 converts a framework int64 to *int32 for SDK int32 fields.
func optInt32(v types.Int64) *int32 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return ptr(int32(v.ValueInt64()))
}

func int64FromInt32(v *int32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// optFloat32 converts a framework float64 to *float32.
func optFloat32(v types.Float64) *float32 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return ptr(float32(v.ValueFloat64()))
}

// syncFloat32 keeps the state value when it equals the API value at float32
// precision, avoiding perpetual diffs caused by float64→float32→float64
// round-trips (e.g. 0.9 → 0.8999999761581421).
func syncFloat32(state types.Float64, api *float32) types.Float64 {
	if api == nil {
		return types.Float64Null()
	}
	if !state.IsNull() && !state.IsUnknown() && float32(state.ValueFloat64()) == *api {
		return state
	}
	return types.Float64Value(float64(*api))
}

func syncFloat64(state types.Float64, api *float64) types.Float64 {
	if api == nil {
		return types.Float64Null()
	}
	if !state.IsNull() && !state.IsUnknown() && state.ValueFloat64() == *api {
		return state
	}
	return types.Float64Value(*api)
}

func optFloat64(v types.Float64) *float64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return ptr(v.ValueFloat64())
}

// optTime parses a framework RFC3339 string into *time.Time.
func optTime(s types.String) (*time.Time, error) {
	if s.IsNull() || s.IsUnknown() {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s.ValueString())
	if err != nil {
		return nil, fmt.Errorf("parsing %q as RFC3339 timestamp: %w", s.ValueString(), err)
	}
	return &t, nil
}

// syncTime keeps the state's string representation when it denotes the same
// instant as the API value, otherwise formats the API value as RFC3339.
func syncTime(state types.String, api *time.Time) types.String {
	if api == nil {
		return types.StringNull()
	}
	if !state.IsNull() && !state.IsUnknown() {
		if t, err := time.Parse(time.RFC3339, state.ValueString()); err == nil && t.Equal(*api) {
			return state
		}
	}
	return types.StringValue(api.UTC().Format(time.RFC3339))
}

func ptr[T any](v T) *T {
	return &v
}

// ---------------------------------------------------------------------------
// Equality helpers for change detection (used by detailsEqual).
// ---------------------------------------------------------------------------

func (m *awsReservedInstancesDetailsModel) equal(o *awsReservedInstancesDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.ID.Equal(o.ID) &&
		m.Scope.Equal(o.Scope) &&
		m.AvailabilityZoneName.Equal(o.AvailabilityZoneName) &&
		m.AvailabilityZoneID.Equal(o.AvailabilityZoneID) &&
		m.InstanceType.Equal(o.InstanceType) &&
		m.InstanceCount.Equal(o.InstanceCount) &&
		m.State.Equal(o.State)
}

func (m *azureReservationDetailsModel) equal(o *azureReservationDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.ID.Equal(o.ID) &&
		m.Plan.Equal(o.Plan) &&
		m.Status.Equal(o.Status) &&
		m.Scope.Equal(o.Scope) &&
		m.ScopeSubscription.Equal(o.ScopeSubscription) &&
		m.ScopeResourceGroup.Equal(o.ScopeResourceGroup) &&
		m.ScopeManagementGroup.Equal(o.ScopeManagementGroup) &&
		m.ScopeTenant.Equal(o.ScopeTenant) &&
		m.InstanceType.Equal(o.InstanceType) &&
		m.Count.Equal(o.Count) &&
		m.InstanceFlexibility.Equal(o.InstanceFlexibility) &&
		m.PurchaseTime.Equal(o.PurchaseTime)
}

func (m *gcpResourceCUDDetailsModel) equal(o *gcpResourceCUDDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.CUDID.Equal(o.CUDID) &&
		m.Plan.Equal(o.Plan) &&
		m.Type.Equal(o.Type) &&
		m.MemoryMb.Equal(o.MemoryMb) &&
		m.CPU.Equal(o.CPU) &&
		m.Status.Equal(o.Status)
}

func (m *awsSavingsPlanDetailsModel) equal(o *awsSavingsPlanDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.ID.Equal(o.ID) &&
		m.OfferingID.Equal(o.OfferingID) &&
		m.Type.Equal(o.Type) &&
		m.State.Equal(o.State) &&
		m.Region.Equal(o.Region) &&
		m.InstanceTypeFamily.Equal(o.InstanceTypeFamily) &&
		m.CommitmentAmount.Equal(o.CommitmentAmount) &&
		m.CommitmentTerm.Equal(o.CommitmentTerm) &&
		m.PaymentOption.Equal(o.PaymentOption)
}

func (m *awsCapacityBlockDetailsModel) equal(o *awsCapacityBlockDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.ID.Equal(o.ID) &&
		m.AvailabilityZone.Equal(o.AvailabilityZone) &&
		m.AvailabilityZoneID.Equal(o.AvailabilityZoneID) &&
		m.InstanceType.Equal(o.InstanceType) &&
		m.InstancePlatform.Equal(o.InstancePlatform) &&
		m.TotalInstanceCount.Equal(o.TotalInstanceCount) &&
		m.AvailableInstanceCount.Equal(o.AvailableInstanceCount) &&
		m.State.Equal(o.State)
}

func (m *awsODCRDetailsModel) equal(o *awsODCRDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.ID.Equal(o.ID) &&
		m.AvailabilityZone.Equal(o.AvailabilityZone) &&
		m.AvailabilityZoneID.Equal(o.AvailabilityZoneID) &&
		m.InstanceType.Equal(o.InstanceType) &&
		m.InstancePlatform.Equal(o.InstancePlatform) &&
		m.Tenancy.Equal(o.Tenancy) &&
		m.TotalInstanceCount.Equal(o.TotalInstanceCount) &&
		m.AvailableInstanceCount.Equal(o.AvailableInstanceCount) &&
		m.State.Equal(o.State) &&
		m.EndDateType.Equal(o.EndDateType) &&
		m.InstanceMatchCriteria.Equal(o.InstanceMatchCriteria) &&
		m.Interruptible.Equal(o.Interruptible)
}

func (m *gcpFlexCUDDetailsModel) equal(o *gcpFlexCUDDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.OrderName.Equal(o.OrderName) &&
		m.DisplayName.Equal(o.DisplayName) &&
		m.LineItemID.Equal(o.LineItemID) &&
		m.Offer.Equal(o.Offer) &&
		m.Region.Equal(o.Region) &&
		m.CommitmentAmount.Equal(o.CommitmentAmount) &&
		m.State.Equal(o.State) &&
		m.Plan.Equal(o.Plan)
}

func (m *azureSavingsPlanDetailsModel) equal(o *azureSavingsPlanDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.ID.Equal(o.ID) &&
		m.Term.Equal(o.Term) &&
		m.ProvisioningState.Equal(o.ProvisioningState) &&
		m.Scope.Equal(o.Scope) &&
		m.ScopeSubscription.Equal(o.ScopeSubscription) &&
		m.ScopeResourceGroup.Equal(o.ScopeResourceGroup) &&
		m.ScopeManagementGroup.Equal(o.ScopeManagementGroup) &&
		m.ScopeTenant.Equal(o.ScopeTenant) &&
		m.CommitmentAmount.Equal(o.CommitmentAmount)
}

func (m *gcpReservationShareSettingsModel) equal(o *gcpReservationShareSettingsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	return m.ShareType.Equal(o.ShareType) &&
		m.ProjectIDs.Equal(o.ProjectIDs)
}

func (m *gcpReservationAcceleratorModel) equal(o *gcpReservationAcceleratorModel) bool {
	return m.AcceleratorType.Equal(o.AcceleratorType) &&
		m.AcceleratorCount.Equal(o.AcceleratorCount)
}

func (m *gcpReservationLocalSsdModel) equal(o *gcpReservationLocalSsdModel) bool {
	return m.Interface.Equal(o.Interface) &&
		m.DiskSizeGb.Equal(o.DiskSizeGb)
}

func (m *gcpCapacityReservationDetailsModel) equal(o *gcpCapacityReservationDetailsModel) bool {
	if m == nil || o == nil {
		return m == nil && o == nil
	}
	if !m.ID.Equal(o.ID) ||
		!m.ProjectID.Equal(o.ProjectID) ||
		!m.Zone.Equal(o.Zone) ||
		!m.InstanceType.Equal(o.InstanceType) ||
		!m.TotalInstanceCount.Equal(o.TotalInstanceCount) ||
		!m.InUseInstanceCount.Equal(o.InUseInstanceCount) ||
		!m.AssuredInstanceCount.Equal(o.AssuredInstanceCount) ||
		!m.State.Equal(o.State) ||
		!m.SpecificReservationRequired.Equal(o.SpecificReservationRequired) ||
		!m.MinCPUPlatform.Equal(o.MinCPUPlatform) {
		return false
	}
	if !m.ShareSettings.equal(o.ShareSettings) {
		return false
	}
	// Compare slices element-by-element.
	if len(m.Accelerators) != len(o.Accelerators) {
		return false
	}
	for i := range m.Accelerators {
		if !m.Accelerators[i].equal(&o.Accelerators[i]) {
			return false
		}
	}
	if len(m.LocalSsds) != len(o.LocalSsds) {
		return false
	}
	for i := range m.LocalSsds {
		if !m.LocalSsds[i].equal(&o.LocalSsds[i]) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Model → SDK (create/upsert input).
// ---------------------------------------------------------------------------

func (m *commitmentModel) toCreateInput(ctx context.Context) (pricing.CreateCommitmentInput, diag.Diagnostics) {
	var diags diag.Diagnostics

	startTime, err := optTime(m.StartTime)
	if err != nil {
		diags.AddError("Invalid start_time", err.Error())
		return pricing.CreateCommitmentInput{}, diags
	}
	if startTime == nil {
		diags.AddError("Invalid start_time", "start_time is required")
		return pricing.CreateCommitmentInput{}, diags
	}
	endTime, err := optTime(m.EndTime)
	if err != nil {
		diags.AddError("Invalid end_time", err.Error())
		return pricing.CreateCommitmentInput{}, diags
	}

	input := pricing.CreateCommitmentInput{
		Name:      m.Name.ValueString(),
		Cloud:     pricing.CreateCommitmentInputCloud(m.Cloud.ValueString()),
		Region:    m.Region.ValueString(),
		Type:      pricing.CreateCommitmentInputType(m.Type.ValueString()),
		StartTime: *startTime,
		EndTime:   endTime,
	}

	if !m.AutoscalingStatus.IsNull() && !m.AutoscalingStatus.IsUnknown() {
		input.AutoscalingStatus = ptr(pricing.CreateCommitmentInputAutoscalingStatus(m.AutoscalingStatus.ValueString()))
	}
	input.AllowedUsage = optFloat32(m.AllowedUsage)
	if !m.Prioritization.IsNull() && !m.Prioritization.IsUnknown() {
		input.Prioritization = m.Prioritization.ValueBoolPointer()
	}
	if !m.ScalingStrategy.IsNull() && !m.ScalingStrategy.IsUnknown() {
		input.ScalingStrategy = ptr(pricing.CreateCommitmentInputScalingStrategy(m.ScalingStrategy.ValueString()))
	}
	if !m.AutoAssignment.IsNull() && !m.AutoAssignment.IsUnknown() {
		input.AutoAssignment = m.AutoAssignment.ValueBoolPointer()
	}

	input.AwsReservedInstancesDetails = m.AWSReservedInstancesDetails.toSDK()
	input.AzureReservationDetails, err = m.AzureReservationDetails.toSDK()
	if err != nil {
		diags.AddError("Invalid azure_reservation_details", err.Error())
		return pricing.CreateCommitmentInput{}, diags
	}
	input.GcpResourceCudDetails = m.GCPResourceCUDDetails.toSDK()
	input.AwsSavingsPlanDetails = m.AWSSavingsPlanDetails.toSDK()
	input.AwsCapacityBlockDetails = m.AWSCapacityBlockDetails.toSDK()
	input.AwsOdcrDetails = m.AWSODCRDetails.toSDK()
	input.GcpFlexCudDetails = m.GCPFlexCUDDetails.toSDK()
	input.AzureSavingsPlanDetails = m.AzureSavingsPlanDetails.toSDK()
	input.GcpCapacityReservationDetails, diags = m.GCPCapacityReservationDetails.toSDK(ctx, diags)

	return input, diags
}

func (m *commitmentModel) toUpdateInput() pricing.UpdateCommitmentInput {
	input := pricing.UpdateCommitmentInput{}
	if !m.AutoscalingStatus.IsNull() && !m.AutoscalingStatus.IsUnknown() {
		input.AutoscalingStatus = ptr(pricing.UpdateCommitmentInputAutoscalingStatus(m.AutoscalingStatus.ValueString()))
	}
	input.AllowedUsage = optFloat32(m.AllowedUsage)
	if !m.Prioritization.IsNull() && !m.Prioritization.IsUnknown() {
		input.Prioritization = m.Prioritization.ValueBoolPointer()
	}
	if !m.ScalingStrategy.IsNull() && !m.ScalingStrategy.IsUnknown() {
		input.ScalingStrategy = ptr(pricing.UpdateCommitmentInputScalingStrategy(m.ScalingStrategy.ValueString()))
	}
	if !m.AutoAssignment.IsNull() && !m.AutoAssignment.IsUnknown() {
		input.AutoAssignment = m.AutoAssignment.ValueBoolPointer()
	}
	return input
}

func (m *awsReservedInstancesDetailsModel) toSDK() *pricing.AWSReservedInstancesDetails {
	if m == nil {
		return nil
	}
	return &pricing.AWSReservedInstancesDetails{
		Id:                   optStr(m.ID),
		Scope:                optStr(m.Scope),
		AvailabilityZoneName: optStr(m.AvailabilityZoneName),
		AvailabilityZoneId:   optStr(m.AvailabilityZoneID),
		InstanceType:         optStr(m.InstanceType),
		InstanceCount:        optInt64Str(m.InstanceCount),
		State:                optStr(m.State),
	}
}

func (m *azureReservationDetailsModel) toSDK() (*pricing.AzureReservationDetails, error) {
	if m == nil {
		return nil, nil
	}
	purchaseTime, err := optTime(m.PurchaseTime)
	if err != nil {
		return nil, err
	}
	out := &pricing.AzureReservationDetails{
		Id:                   optStr(m.ID),
		Status:               optStr(m.Status),
		Scope:                optStr(m.Scope),
		ScopeSubscription:    optStr(m.ScopeSubscription),
		ScopeResourceGroup:   optStr(m.ScopeResourceGroup),
		ScopeManagementGroup: optStr(m.ScopeManagementGroup),
		ScopeTenant:          optStr(m.ScopeTenant),
		InstanceType:         optStr(m.InstanceType),
		Count:                optInt32(m.Count),
		PurchaseTime:         purchaseTime,
	}
	if !m.Plan.IsNull() && !m.Plan.IsUnknown() {
		out.Plan = ptr(pricing.AzureReservationDetailsPlan(m.Plan.ValueString()))
	}
	if !m.InstanceFlexibility.IsNull() && !m.InstanceFlexibility.IsUnknown() {
		out.InstanceFlexibility = ptr(pricing.AzureReservationDetailsInstanceFlexibility(m.InstanceFlexibility.ValueString()))
	}
	return out, nil
}

func (m *gcpResourceCUDDetailsModel) toSDK() *pricing.GCPResourceCUDDetails {
	if m == nil {
		return nil
	}
	out := &pricing.GCPResourceCUDDetails{
		CudId:    optStr(m.CUDID),
		Type:     optStr(m.Type),
		MemoryMb: optInt64Str(m.MemoryMb),
		Cpu:      optInt64Str(m.CPU),
		Status:   optStr(m.Status),
	}
	if !m.Plan.IsNull() && !m.Plan.IsUnknown() {
		out.Plan = ptr(pricing.GCPResourceCUDDetailsPlan(m.Plan.ValueString()))
	}
	return out
}

func (m *awsSavingsPlanDetailsModel) toSDK() *pricing.AWSSavingsPlanDetails {
	if m == nil {
		return nil
	}
	out := &pricing.AWSSavingsPlanDetails{
		Id:                 optStr(m.ID),
		OfferingId:         optStr(m.OfferingID),
		Type:               optStr(m.Type),
		State:              optStr(m.State),
		Region:             optStr(m.Region),
		InstanceTypeFamily: optStr(m.InstanceTypeFamily),
		CommitmentAmount:   optFloat32(m.CommitmentAmount),
	}
	if !m.CommitmentTerm.IsNull() && !m.CommitmentTerm.IsUnknown() {
		out.CommitmentTerm = ptr(pricing.AWSSavingsPlanDetailsCommitmentTerm(m.CommitmentTerm.ValueString()))
	}
	if !m.PaymentOption.IsNull() && !m.PaymentOption.IsUnknown() {
		out.PaymentOption = ptr(pricing.AWSSavingsPlanDetailsPaymentOption(m.PaymentOption.ValueString()))
	}
	return out
}

func (m *awsCapacityBlockDetailsModel) toSDK() *pricing.AWSCapacityBlockDetails {
	if m == nil {
		return nil
	}
	return &pricing.AWSCapacityBlockDetails{
		Id:                     optStr(m.ID),
		AvailabilityZone:       optStr(m.AvailabilityZone),
		AvailabilityZoneId:     optStr(m.AvailabilityZoneID),
		InstanceType:           optStr(m.InstanceType),
		InstancePlatform:       optStr(m.InstancePlatform),
		TotalInstanceCount:     optInt64Str(m.TotalInstanceCount),
		AvailableInstanceCount: optInt64Str(m.AvailableInstanceCount),
		State:                  optStr(m.State),
	}
}

func (m *awsODCRDetailsModel) toSDK() *pricing.AWSODCRDetails {
	if m == nil {
		return nil
	}
	out := &pricing.AWSODCRDetails{
		Id:                     optStr(m.ID),
		AvailabilityZone:       optStr(m.AvailabilityZone),
		AvailabilityZoneId:     optStr(m.AvailabilityZoneID),
		InstanceType:           optStr(m.InstanceType),
		InstancePlatform:       optStr(m.InstancePlatform),
		Tenancy:                optStr(m.Tenancy),
		TotalInstanceCount:     optInt64Str(m.TotalInstanceCount),
		AvailableInstanceCount: optInt64Str(m.AvailableInstanceCount),
		State:                  optStr(m.State),
		EndDateType:            optStr(m.EndDateType),
		InstanceMatchCriteria:  optStr(m.InstanceMatchCriteria),
	}
	if !m.Interruptible.IsNull() && !m.Interruptible.IsUnknown() {
		out.Interruptible = m.Interruptible.ValueBoolPointer()
	}
	return out
}

func (m *gcpFlexCUDDetailsModel) toSDK() *pricing.GCPFlexCUDDetails {
	if m == nil {
		return nil
	}
	out := &pricing.GCPFlexCUDDetails{
		OrderName:        optStr(m.OrderName),
		DisplayName:      optStr(m.DisplayName),
		LineItemId:       optStr(m.LineItemID),
		Offer:            optStr(m.Offer),
		Region:           optStr(m.Region),
		CommitmentAmount: optFloat64(m.CommitmentAmount),
		State:            optStr(m.State),
	}
	if !m.Plan.IsNull() && !m.Plan.IsUnknown() {
		out.Plan = ptr(pricing.GCPFlexCUDDetailsPlan(m.Plan.ValueString()))
	}
	return out
}

func (m *azureSavingsPlanDetailsModel) toSDK() *pricing.AzureSavingsPlanDetails {
	if m == nil {
		return nil
	}
	out := &pricing.AzureSavingsPlanDetails{
		Id:                   optStr(m.ID),
		ProvisioningState:    optStr(m.ProvisioningState),
		ScopeSubscription:    optStr(m.ScopeSubscription),
		ScopeResourceGroup:   optStr(m.ScopeResourceGroup),
		ScopeManagementGroup: optStr(m.ScopeManagementGroup),
		ScopeTenant:          optStr(m.ScopeTenant),
		CommitmentAmount:     optFloat64(m.CommitmentAmount),
	}
	if !m.Term.IsNull() && !m.Term.IsUnknown() {
		out.Term = ptr(pricing.AzureSavingsPlanDetailsTerm(m.Term.ValueString()))
	}
	if !m.Scope.IsNull() && !m.Scope.IsUnknown() {
		out.Scope = ptr(pricing.AzureSavingsPlanDetailsScope(m.Scope.ValueString()))
	}
	return out
}

func (m *gcpCapacityReservationDetailsModel) toSDK(ctx context.Context, diags diag.Diagnostics) (*pricing.GCPCapacityReservationDetails, diag.Diagnostics) {
	if m == nil {
		return nil, diags
	}
	out := &pricing.GCPCapacityReservationDetails{
		Id:                   optStr(m.ID),
		ProjectId:            optStr(m.ProjectID),
		Zone:                 optStr(m.Zone),
		InstanceType:         optStr(m.InstanceType),
		TotalInstanceCount:   optInt64Str(m.TotalInstanceCount),
		InUseInstanceCount:   optInt64Str(m.InUseInstanceCount),
		AssuredInstanceCount: optInt64Str(m.AssuredInstanceCount),
		State:                optStr(m.State),
		MinCpuPlatform:       optStr(m.MinCPUPlatform),
	}
	if !m.SpecificReservationRequired.IsNull() && !m.SpecificReservationRequired.IsUnknown() {
		out.SpecificReservationRequired = m.SpecificReservationRequired.ValueBoolPointer()
	}
	if m.ShareSettings != nil {
		ss := &pricing.GCPReservationShareSettings{}
		if !m.ShareSettings.ShareType.IsNull() && !m.ShareSettings.ShareType.IsUnknown() {
			ss.ShareType = ptr(pricing.GCPReservationShareSettingsShareType(m.ShareSettings.ShareType.ValueString()))
		}
		if !m.ShareSettings.ProjectIDs.IsNull() && !m.ShareSettings.ProjectIDs.IsUnknown() {
			var ids []string
			diags.Append(m.ShareSettings.ProjectIDs.ElementsAs(ctx, &ids, false)...)
			if diags.HasError() {
				return nil, diags
			}
			ss.ProjectIds = &ids
		}
		out.ShareSettings = ss
	}
	if len(m.Accelerators) > 0 {
		accs := make([]pricing.GCPReservationAccelerator, 0, len(m.Accelerators))
		for _, a := range m.Accelerators {
			accs = append(accs, pricing.GCPReservationAccelerator{
				AcceleratorType:  optStr(a.AcceleratorType),
				AcceleratorCount: optInt64Str(a.AcceleratorCount),
			})
		}
		out.Accelerators = &accs
	}
	if len(m.LocalSsds) > 0 {
		ssds := make([]pricing.GCPReservationLocalSsd, 0, len(m.LocalSsds))
		for _, s := range m.LocalSsds {
			ssd := pricing.GCPReservationLocalSsd{
				DiskSizeGb: optInt64Str(s.DiskSizeGb),
			}
			if !s.Interface.IsNull() && !s.Interface.IsUnknown() {
				ssd.Interface = ptr(pricing.GCPReservationLocalSsdInterface(s.Interface.ValueString()))
			}
			ssds = append(ssds, ssd)
		}
		out.LocalSsds = &ssds
	}
	return out, diags
}

// ---------------------------------------------------------------------------
// SDK → model (read-back).
// ---------------------------------------------------------------------------

// applyCommitment maps an API commitment onto the model, preserving state
// representations where they are semantically equal to the API values.
func (m *commitmentModel) applyCommitment(ctx context.Context, c *pricing.Commitment) diag.Diagnostics {
	var diags diag.Diagnostics
	m.ID = strOrNull(c.Id)
	m.Name = syncStr(m.Name, c.Name)
	if c.Cloud != nil {
		m.Cloud = types.StringValue(string(*c.Cloud))
	}
	m.Region = syncStr(m.Region, c.Region)
	if c.Type != nil {
		m.Type = types.StringValue(string(*c.Type))
	}
	m.StartTime = syncTime(m.StartTime, c.StartTime)
	m.EndTime = syncTime(m.EndTime, c.EndTime)
	m.AutoscalingStatus = syncEnumStr(m.AutoscalingStatus, (*string)(c.AutoscalingStatus), "AUTOSCALING_STATUS_UNSPECIFIED")
	m.AllowedUsage = syncFloat32(m.AllowedUsage, c.AllowedUsage)
	m.Prioritization = syncBool(m.Prioritization, c.Prioritization)
	m.ScalingStrategy = syncEnumStr(m.ScalingStrategy, (*string)(c.ScalingStrategy), "SCALING_STRATEGY_UNSPECIFIED")
	m.AutoAssignment = syncBool(m.AutoAssignment, c.AutoAssignment)
	if c.State != nil {
		m.State = types.StringValue(string(*c.State))
	} else {
		m.State = types.StringNull()
	}
	m.CreateTime = syncTime(m.CreateTime, c.CreateTime)
	m.UpdateTime = syncTime(m.UpdateTime, c.UpdateTime)

	diags.Append(m.applyDetails(ctx, c)...)
	return diags
}

func (m *commitmentModel) applyDetails(ctx context.Context, c *pricing.Commitment) diag.Diagnostics {
	var diags diag.Diagnostics
	if d := c.AwsReservedInstancesDetails; d != nil {
		prev := m.AWSReservedInstancesDetails
		m.AWSReservedInstancesDetails = &awsReservedInstancesDetailsModel{
			ID:                   strOrNull(d.Id),
			Scope:                syncStr(prev.Scope, d.Scope),
			AvailabilityZoneName: syncStr(prev.AvailabilityZoneName, d.AvailabilityZoneName),
			AvailabilityZoneID:   syncStr(prev.AvailabilityZoneID, d.AvailabilityZoneId),
			InstanceType:         strOrNull(d.InstanceType),
			InstanceCount:        int64FromStr(d.InstanceCount),
			State:                syncStr(prev.State, d.State),
		}
	}
	if d := c.AzureReservationDetails; d != nil {
		prev := m.AzureReservationDetails
		if prev == nil {
			prev = &azureReservationDetailsModel{}
		}
		out := &azureReservationDetailsModel{
			ID:                   strOrNull(d.Id),
			Status:               syncStr(prev.Status, d.Status),
			Scope:                syncStr(prev.Scope, d.Scope),
			ScopeSubscription:    syncStr(prev.ScopeSubscription, d.ScopeSubscription),
			ScopeResourceGroup:   syncStr(prev.ScopeResourceGroup, d.ScopeResourceGroup),
			ScopeManagementGroup: syncStr(prev.ScopeManagementGroup, d.ScopeManagementGroup),
			ScopeTenant:          syncStr(prev.ScopeTenant, d.ScopeTenant),
			InstanceType:         strOrNull(d.InstanceType),
			Count:                int64FromInt32(d.Count),
			PurchaseTime:         syncTime(prev.PurchaseTime, d.PurchaseTime),
		}
		out.Plan = syncEnumStr(prev.Plan, (*string)(d.Plan), "RESERVATION_PLAN_UNSPECIFIED")
		out.InstanceFlexibility = syncEnumStr(prev.InstanceFlexibility, (*string)(d.InstanceFlexibility), "INSTANCE_FLEXIBILITY_UNSPECIFIED")
		m.AzureReservationDetails = out
	}
	if d := c.GcpResourceCudDetails; d != nil {
		prev := m.GCPResourceCUDDetails
		if prev == nil {
			prev = &gcpResourceCUDDetailsModel{}
		}
		out := &gcpResourceCUDDetailsModel{
			CUDID:    strOrNull(d.CudId),
			Type:     syncStr(prev.Type, d.Type),
			MemoryMb: int64FromStr(d.MemoryMb),
			CPU:      int64FromStr(d.Cpu),
			Status:   syncStr(prev.Status, d.Status),
		}
		out.Plan = syncEnumStr(prev.Plan, (*string)(d.Plan), "CUD_PLAN_UNSPECIFIED")
		m.GCPResourceCUDDetails = out
	}
	if d := c.AwsSavingsPlanDetails; d != nil {
		prev := m.AWSSavingsPlanDetails
		if prev == nil {
			prev = &awsSavingsPlanDetailsModel{}
		}
		out := &awsSavingsPlanDetailsModel{
			ID:                 strOrNull(d.Id),
			OfferingID:         syncStr(prev.OfferingID, d.OfferingId),
			Type:               syncStr(prev.Type, d.Type),
			State:              syncStr(prev.State, d.State),
			Region:             syncStr(prev.Region, d.Region),
			InstanceTypeFamily: syncStr(prev.InstanceTypeFamily, d.InstanceTypeFamily),
			CommitmentAmount:   syncFloat32(prev.CommitmentAmount, d.CommitmentAmount),
		}
		out.CommitmentTerm = syncEnumStr(prev.CommitmentTerm, (*string)(d.CommitmentTerm), "COMMITMENT_TERM_UNIT_UNSPECIFIED")
		out.PaymentOption = syncEnumStr(prev.PaymentOption, (*string)(d.PaymentOption), "PAYMENT_OPTION_UNSPECIFIED")
		m.AWSSavingsPlanDetails = out
	}
	if d := c.AwsCapacityBlockDetails; d != nil {
		prev := m.AWSCapacityBlockDetails
		if prev == nil {
			prev = &awsCapacityBlockDetailsModel{}
		}
		m.AWSCapacityBlockDetails = &awsCapacityBlockDetailsModel{
			ID:                     strOrNull(d.Id),
			AvailabilityZone:       syncStr(prev.AvailabilityZone, d.AvailabilityZone),
			AvailabilityZoneID:     syncStr(prev.AvailabilityZoneID, d.AvailabilityZoneId),
			InstanceType:           strOrNull(d.InstanceType),
			InstancePlatform:       syncStr(prev.InstancePlatform, d.InstancePlatform),
			TotalInstanceCount:     int64FromStr(d.TotalInstanceCount),
			AvailableInstanceCount: int64FromStr(d.AvailableInstanceCount),
			State:                  syncStr(prev.State, d.State),
		}
	}
	if d := c.AwsOdcrDetails; d != nil {
		prev := m.AWSODCRDetails
		if prev == nil {
			prev = &awsODCRDetailsModel{}
		}
		out := &awsODCRDetailsModel{
			ID:                     strOrNull(d.Id),
			AvailabilityZone:       syncStr(prev.AvailabilityZone, d.AvailabilityZone),
			AvailabilityZoneID:     syncStr(prev.AvailabilityZoneID, d.AvailabilityZoneId),
			InstanceType:           strOrNull(d.InstanceType),
			InstancePlatform:       syncStr(prev.InstancePlatform, d.InstancePlatform),
			Tenancy:                syncStr(prev.Tenancy, d.Tenancy),
			TotalInstanceCount:     int64FromStr(d.TotalInstanceCount),
			AvailableInstanceCount: int64FromStr(d.AvailableInstanceCount),
			State:                  syncStr(prev.State, d.State),
			EndDateType:            syncStr(prev.EndDateType, d.EndDateType),
			InstanceMatchCriteria:  syncStr(prev.InstanceMatchCriteria, d.InstanceMatchCriteria),
			Interruptible:          syncBool(prev.Interruptible, d.Interruptible),
		}
		m.AWSODCRDetails = out
	}
	if d := c.GcpFlexCudDetails; d != nil {
		prev := m.GCPFlexCUDDetails
		if prev == nil {
			prev = &gcpFlexCUDDetailsModel{}
		}
		out := &gcpFlexCUDDetailsModel{
			OrderName:        syncStr(prev.OrderName, d.OrderName),
			DisplayName:      syncStr(prev.DisplayName, d.DisplayName),
			LineItemID:       strOrNull(d.LineItemId),
			Offer:            syncStr(prev.Offer, d.Offer),
			Region:           syncStr(prev.Region, d.Region),
			CommitmentAmount: syncFloat64(prev.CommitmentAmount, d.CommitmentAmount),
			State:            syncStr(prev.State, d.State),
		}
		out.Plan = syncEnumStr(prev.Plan, (*string)(d.Plan), "CUD_PLAN_UNSPECIFIED")
		m.GCPFlexCUDDetails = out
	}
	if d := c.AzureSavingsPlanDetails; d != nil {
		prev := m.AzureSavingsPlanDetails
		if prev == nil {
			prev = &azureSavingsPlanDetailsModel{}
		}
		out := &azureSavingsPlanDetailsModel{
			ID:                   strOrNull(d.Id),
			ProvisioningState:    syncStr(prev.ProvisioningState, d.ProvisioningState),
			ScopeSubscription:    syncStr(prev.ScopeSubscription, d.ScopeSubscription),
			ScopeResourceGroup:   syncStr(prev.ScopeResourceGroup, d.ScopeResourceGroup),
			ScopeManagementGroup: syncStr(prev.ScopeManagementGroup, d.ScopeManagementGroup),
			ScopeTenant:          syncStr(prev.ScopeTenant, d.ScopeTenant),
			CommitmentAmount:     syncFloat64(prev.CommitmentAmount, d.CommitmentAmount),
		}
		out.Term = syncEnumStr(prev.Term, (*string)(d.Term), "TERM_UNSPECIFIED")
		out.Scope = syncEnumStr(prev.Scope, (*string)(d.Scope), "SCOPE_UNSPECIFIED")
		m.AzureSavingsPlanDetails = out
	}
	if d := c.GcpCapacityReservationDetails; d != nil {
		prev := m.GCPCapacityReservationDetails
		if prev == nil {
			prev = &gcpCapacityReservationDetailsModel{}
		}
		out := &gcpCapacityReservationDetailsModel{
			ID:                          strOrNull(d.Id),
			ProjectID:                   syncStr(prev.ProjectID, d.ProjectId),
			Zone:                        syncStr(prev.Zone, d.Zone),
			InstanceType:                strOrNull(d.InstanceType),
			TotalInstanceCount:          int64FromStr(d.TotalInstanceCount),
			InUseInstanceCount:          int64FromStr(d.InUseInstanceCount),
			AssuredInstanceCount:        int64FromStr(d.AssuredInstanceCount),
			State:                       syncStr(prev.State, d.State),
			SpecificReservationRequired: syncBool(prev.SpecificReservationRequired, d.SpecificReservationRequired),
			MinCPUPlatform:              syncStr(prev.MinCPUPlatform, d.MinCpuPlatform),
		}
		if d.ShareSettings != nil || prev.ShareSettings != nil {
			ss := &gcpReservationShareSettingsModel{
				ProjectIDs: types.ListNull(types.StringType),
			}
			var prevShareType types.String
			var prevProjectIDs types.List
			if prev.ShareSettings != nil {
				prevShareType = prev.ShareSettings.ShareType
				prevProjectIDs = prev.ShareSettings.ProjectIDs
			}
			if d.ShareSettings != nil {
				ss.ShareType = syncEnumStr(prevShareType, (*string)(d.ShareSettings.ShareType), "GCP_RESERVATION_SHARE_TYPE_UNSPECIFIED")
				if d.ShareSettings.ProjectIds != nil && len(*d.ShareSettings.ProjectIds) > 0 {
					elems, listDiags := types.ListValueFrom(ctx, types.StringType, *d.ShareSettings.ProjectIds)
					diags.Append(listDiags...)
					if !listDiags.HasError() {
						ss.ProjectIDs = elems
					}
				} else {
					ss.ProjectIDs = prevProjectIDs
				}
			} else {
				ss.ShareType = prevShareType
				ss.ProjectIDs = prevProjectIDs
			}
			out.ShareSettings = ss
		}
		if d.Accelerators != nil && len(*d.Accelerators) > 0 {
			accs := make([]gcpReservationAcceleratorModel, 0, len(*d.Accelerators))
			for _, a := range *d.Accelerators {
				accs = append(accs, gcpReservationAcceleratorModel{
					AcceleratorType:  strOrNull(a.AcceleratorType),
					AcceleratorCount: int64FromStr(a.AcceleratorCount),
				})
			}
			out.Accelerators = accs
		} else {
			out.Accelerators = prev.Accelerators
		}
		if d.LocalSsds != nil && len(*d.LocalSsds) > 0 {
			ssds := make([]gcpReservationLocalSsdModel, 0, len(*d.LocalSsds))
			for _, s := range *d.LocalSsds {
				ssd := gcpReservationLocalSsdModel{
					DiskSizeGb: int64FromStr(s.DiskSizeGb),
				}
				ssd.Interface = syncEnumStr(types.StringNull(), (*string)(s.Interface), "GCP_RESERVATION_LOCAL_SSD_INTERFACE_UNSPECIFIED")
				ssds = append(ssds, ssd)
			}
			out.LocalSsds = ssds
		} else {
			out.LocalSsds = prev.LocalSsds
		}
		m.GCPCapacityReservationDetails = out
	}
	return diags
}
