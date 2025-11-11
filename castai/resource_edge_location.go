package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
	"github.com/castai/terraform-provider-castai/castai/store"
)

var (
	_ resource.Resource                = (*edgeLocationResource)(nil)
	_ resource.ResourceWithConfigure   = (*edgeLocationResource)(nil)
	_ resource.ResourceWithImportState = (*edgeLocationResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*edgeLocationResource)(nil)
)

type edgeLocationResource struct {
	client *ProviderConfig
}

type edgeLocationModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ClusterID      types.String `tfsdk:"cluster_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Region         types.String `tfsdk:"region"`
	Zones          []zoneModel  `tfsdk:"zones"`
	AWS            *awsModel    `tfsdk:"aws"`
	GCP            *gcpModel    `tfsdk:"gcp"`
	OCI            *ociModel    `tfsdk:"oci"`
	// Computed revision number incremented each time credentials have changed.
	CredentialsRevision types.Int64 `tfsdk:"credentials_revision"`
}

type zoneModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type awsModel struct {
	AccountID         types.String `tfsdk:"account_id"`
	AccessKeyIDWO     types.String `tfsdk:"access_key_id_wo"`
	SecretAccessKeyWO types.String `tfsdk:"secret_access_key_wo"`
	VpcID             types.String `tfsdk:"vpc_id"`
	SecurityGroupID   types.String `tfsdk:"security_group_id"`
	SubnetIDs         types.Map    `tfsdk:"subnet_ids"`
	NameTag           types.String `tfsdk:"name_tag"`
}

type gcpModel struct {
	ProjectID                        types.String `tfsdk:"project_id"`
	ClientServiceAccountJSONBase64WO types.String `tfsdk:"client_service_account_json_base64_wo"`
	NetworkName                      types.String `tfsdk:"network_name"`
	SubnetName                       types.String `tfsdk:"subnet_name"`
	NetworkTags                      types.Set    `tfsdk:"network_tags"`
}

type ociModel struct {
	TenancyID     types.String `tfsdk:"tenancy_id"`
	CompartmentID types.String `tfsdk:"compartment_id"`
	UserIDWO      types.String `tfsdk:"user_id_wo"`
	FingerprintWO types.String `tfsdk:"fingerprint_wo"`
	PrivateKeyWO  types.String `tfsdk:"private_key_wo"`
	VcnID         types.String `tfsdk:"vcn_id"`
	SubnetID      types.String `tfsdk:"subnet_id"`
}

func (m awsModel) credentials() types.String {
	return types.StringValue(m.SecretAccessKeyWO.String() + m.AccessKeyIDWO.String())
}

func (m awsModel) Equal(other *awsModel) bool {
	if other == nil {
		return false
	}
	return m.AccountID.Equal(other.AccountID) &&
		m.VpcID.Equal(other.VpcID) &&
		m.SecurityGroupID.Equal(other.SecurityGroupID) &&
		m.SubnetIDs.Equal(other.SubnetIDs) &&
		m.NameTag.Equal(other.NameTag)
}

func (m gcpModel) credentials() types.String {
	return m.ClientServiceAccountJSONBase64WO
}

func (m gcpModel) Equal(other *gcpModel) bool {
	if other == nil {
		return false
	}
	return m.ProjectID.Equal(other.ProjectID) &&
		m.NetworkName.Equal(other.NetworkName) &&
		m.SubnetName.Equal(other.SubnetName) &&
		m.NetworkTags.Equal(other.NetworkTags)
}

func (m ociModel) credentials() types.String {
	return types.StringValue(m.UserIDWO.String() + m.PrivateKeyWO.String() + m.FingerprintWO.String())
}

func (m ociModel) Equal(other *ociModel) bool {
	if other == nil {
		return false
	}
	return m.TenancyID.Equal(other.TenancyID) &&
		m.CompartmentID.Equal(other.CompartmentID) &&
		m.SubnetID.Equal(other.SubnetID) &&
		m.VcnID.Equal(other.VcnID)
}

type ModelWithCredentials interface {
	credentials() types.String
}

func NewEdgeLocationResource() resource.Resource {
	return &edgeLocationResource{}
}

func (r *edgeLocationResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("aws"),
			path.MatchRoot("gcp"),
			path.MatchRoot("oci"),
		),
	}
}

func (r *edgeLocationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edge_location"
}

func (r *edgeLocationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage CAST AI Edge Location for edge computing deployments",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Edge location ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI organization ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Required:    true,
				Description: "CAST AI cluster ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the edge location. Must be unique and relatively short as it's used for creating service accounts.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the edge location",
			},
			"region": schema.StringAttribute{
				Required:    true,
				Description: "The region where the edge location is deployed",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zones": schema.ListNestedAttribute{
				Optional:    true,
				Description: "List of availability zones for the edge location",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "The ID of the zone",
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the zone",
						},
					},
				},
			},
			"credentials_revision": schema.Int64Attribute{
				Computed:    true,
				Description: "Revision number incremented each time credentials change",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"aws": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "AWS configuration for the edge location",
				Attributes: map[string]schema.Attribute{
					"account_id": schema.StringAttribute{
						Required:    true,
						Description: "AWS account ID",
					},
					"access_key_id_wo": schema.StringAttribute{
						Required:    true,
						WriteOnly:   true,
						Sensitive:   true,
						Description: "AWS access key ID",
					},
					"secret_access_key_wo": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						WriteOnly:   true,
						Description: "AWS secret access key",
					},
					"vpc_id": schema.StringAttribute{
						Required:    true,
						Description: "VPC ID to be used in the selected region",
					},
					"security_group_id": schema.StringAttribute{
						Required:    true,
						Description: "Security group ID to be used in the selected region",
					},
					"subnet_ids": schema.MapAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "Map of zone names to subnet IDs to be used in the selected region",
					},
					"name_tag": schema.StringAttribute{
						Required:    true,
						Description: "The value of a 'Name' tag applied to VPC resources",
					},
				},
			},
			"gcp": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "GCP configuration for the edge location",
				Attributes: map[string]schema.Attribute{
					"project_id": schema.StringAttribute{
						Required:    true,
						Description: "GCP project ID where edges run",
					},
					"client_service_account_json_base64_wo": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						WriteOnly:   true,
						Description: "Base64 encoded service account JSON for provisioning edge resources",
					},
					"network_name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the network to be used in the selected region",
					},
					"subnet_name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the subnetwork to be used in the selected region",
					},
					"network_tags": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "Tags applied on the provisioned cloud resources and the firewall rule",
					},
				},
			},
			"oci": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "OCI configuration for the edge location",
				Attributes: map[string]schema.Attribute{
					"tenancy_id": schema.StringAttribute{
						Required:    true,
						Description: "OCI tenancy ID of the account",
					},
					"compartment_id": schema.StringAttribute{
						Required:    true,
						Description: "OCI compartment ID of edge location",
					},
					"user_id_wo": schema.StringAttribute{
						Required:    true,
						Description: "User ID used to authenticate OCI",
						WriteOnly:   true,
					},
					"fingerprint_wo": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						WriteOnly:   true,
						Description: "API key fingerprint",
					},
					"private_key_wo": schema.StringAttribute{
						WriteOnly:   true,
						Required:    true,
						Sensitive:   true,
						Description: "Base64 encoded API private key",
					},
					"vcn_id": schema.StringAttribute{
						Required:    true,
						Description: "OCI virtual cloud network ID",
					},
					"subnet_id": schema.StringAttribute{
						Required:    true,
						Description: "OCI subnet ID of edge location",
					},
				},
			},
		},
	}
}

func (r *edgeLocationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *edgeLocationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var (
		plan, config edgeLocationModel
		diags        diag.Diagnostics
	)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.OmniAPI
	organizationID := plan.OrganizationID.ValueString()
	clusterID := plan.ClusterID.ValueString()

	createReq := omni.EdgeLocationsAPICreateEdgeLocationJSONRequestBody{
		Name:   plan.Name.ValueString(),
		Region: plan.Region.ValueString(),
		Zones:  lo.ToPtr(r.toZones(plan.Zones)),
	}

	if !plan.Description.IsNull() {
		createReq.Description = lo.ToPtr(plan.Description.ValueString())
	}

	// Map cloud provider specific configurations.
	var mc ModelWithCredentials
	if plan.AWS != nil {
		createReq.Aws, diags = r.toAWS(ctx, plan.AWS, config.AWS)
		mc = config.AWS
	}
	if plan.GCP != nil {
		createReq.Gcp, diags = r.toGCP(ctx, plan.GCP, config.GCP)
		mc = config.GCP
	}
	if plan.OCI != nil {
		createReq.Oci = r.toOCI(plan.OCI, config.OCI)
		mc = config.OCI
	}
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.EdgeLocationsAPICreateEdgeLocationWithResponse(ctx, organizationID, clusterID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create edge location", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to create edge location",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	plan.ID = types.StringValue(*apiResp.JSON200.Id)
	plan.CredentialsRevision = types.Int64Value(1)
	// Store credential hash in private state
	resp.Diagnostics.Append(r.woCredentialsStore(resp.Private).Set(ctx, mc.credentials())...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *edgeLocationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state edgeLocationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.OmniAPI
	organizationID := state.OrganizationID.ValueString()
	clusterID := state.ClusterID.ValueString()

	apiResp, err := client.EdgeLocationsAPIGetEdgeLocationWithResponse(ctx, organizationID, clusterID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read edge location", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to read edge location",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	edgeLocation := apiResp.JSON200

	state.Region = types.StringValue(edgeLocation.Region)
	state.Name = types.StringValue(edgeLocation.Name)
	state.Description = types.StringNull()
	if edgeLocation.Description != nil {
		state.Description = types.StringValue(*edgeLocation.Description)
	}

	if edgeLocation.Zones != nil {
		state.Zones = r.toZoneModel(edgeLocation.Zones)
	}

	var diags diag.Diagnostics
	if edgeLocation.Aws != nil {
		state.AWS, diags = r.toAWSModel(ctx, edgeLocation.Aws)
	}
	if edgeLocation.Gcp != nil {
		state.GCP, diags = r.toGCPModel(ctx, edgeLocation.Gcp)
	}
	if edgeLocation.Oci != nil {
		state.OCI = r.toOCIModel(edgeLocation.Oci)
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Initialize credentials_revision to 1 if not set (e.g., during import)
	if state.CredentialsRevision.IsNull() {
		state.CredentialsRevision = types.Int64Value(1)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *edgeLocationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state, config edgeLocationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.OmniAPI
	organizationID := plan.OrganizationID.ValueString()
	clusterID := plan.ClusterID.ValueString()

	updateReq := omni.EdgeLocationsAPIUpdateEdgeLocationJSONRequestBody{}
	if !plan.Description.IsNull() {
		updateReq.Description = toPtr(plan.Description.ValueString())
	}

	if len(plan.Zones) > 0 {
		updateReq.Zones = toPtr(r.toZones(plan.Zones))
	}

	// Check if credentials have changed by comparing credentials revision
	credentialsChanged := !plan.CredentialsRevision.Equal(state.CredentialsRevision)

	// Include cloud provider config if it or credentials has changed.
	var (
		diags diag.Diagnostics
		mc    ModelWithCredentials
	)
	switch {
	case plan.AWS != nil && (!plan.AWS.Equal(state.AWS) || credentialsChanged):
		updateReq.Aws, diags = r.toAWS(ctx, plan.AWS, config.AWS)
		mc = config.AWS
	case plan.GCP != nil && (!plan.GCP.Equal(state.GCP) || credentialsChanged):
		updateReq.Gcp, diags = r.toGCP(ctx, plan.GCP, config.GCP)
		mc = config.GCP
	case plan.OCI != nil && (!plan.OCI.Equal(state.OCI) || credentialsChanged):
		updateReq.Oci = r.toOCI(plan.OCI, config.OCI)
		mc = config.OCI
	}
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := client.EdgeLocationsAPIUpdateEdgeLocationWithResponse(ctx, organizationID, clusterID, plan.ID.ValueString(), nil, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update edge location", err.Error())
		return
	}

	if apiResp.StatusCode() != http.StatusOK {
		resp.Diagnostics.AddError(
			"Failed to update edge location",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}

	// Update stored credentials hash if credentials changed
	if credentialsChanged {
		resp.Diagnostics.Append(r.woCredentialsStore(resp.Private).Set(ctx, mc.credentials())...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *edgeLocationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state edgeLocationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.OmniAPI
	organizationID := state.OrganizationID.ValueString()
	clusterID := state.ClusterID.ValueString()

	apiResp, err := client.EdgeLocationsAPIDeleteEdgeLocationWithResponse(ctx, organizationID, clusterID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete edge location", err.Error())
		return
	}

	if apiResp.StatusCode() == http.StatusNotFound {
		return
	}

	if apiResp.StatusCode() != http.StatusOK && apiResp.StatusCode() != http.StatusNoContent {
		resp.Diagnostics.AddError(
			"Failed to delete edge location",
			fmt.Sprintf("unexpected status code: %d, body: %s", apiResp.StatusCode(), string(apiResp.Body)),
		)
		return
	}
}

func (r *edgeLocationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip if resource is being created or deleted
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan, config edgeLocationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		credentialsEqual bool
		diags            diag.Diagnostics
	)
	switch {
	case config.AWS != nil:
		credentialsEqual, diags = r.woCredentialsStore(req.Private).Equal(ctx, config.AWS.credentials())
	case config.GCP != nil:
		credentialsEqual, diags = r.woCredentialsStore(req.Private).Equal(ctx, config.GCP.credentials())
	case config.OCI != nil:
		credentialsEqual, diags = r.woCredentialsStore(req.Private).Equal(ctx, config.OCI.credentials())
	}
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If credentials changed, update the planned credentials_revision
	if !credentialsEqual {
		plan.CredentialsRevision = types.Int64Value(state.CredentialsRevision.ValueInt64() + 1)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}
}

func (r *edgeLocationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: organization_id/cluster_id/edge_location_id
	ids := strings.Split(req.ID, "/")
	if len(ids) != 3 || ids[0] == "" || ids[1] == "" || ids[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("expected: organization_id/cluster_id/edge_location_id, got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), ids[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), ids[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ids[2])...)
}

func (r *edgeLocationResource) toZones(zones []zoneModel) []omni.Zone {
	if len(zones) == 0 {
		return nil
	}

	out := make([]omni.Zone, 0, len(zones))
	for _, z := range zones {
		zone := omni.Zone{
			Id:   lo.ToPtr(z.ID.ValueString()),
			Name: lo.ToPtr(z.Name.ValueString()),
		}
		out = append(out, zone)
	}
	return out
}

func (r *edgeLocationResource) toZoneModel(zones *[]omni.Zone) []zoneModel {
	if zones == nil || len(*zones) == 0 {
		return nil
	}
	out := make([]zoneModel, 0, len(*zones))
	for _, zone := range *zones {
		out = append(out, zoneModel{
			ID:   types.StringValue(lo.FromPtr(zone.Id)),
			Name: types.StringValue(lo.FromPtr(zone.Name)),
		})
	}
	return out
}

func (r *edgeLocationResource) toAWS(ctx context.Context, plan, config *awsModel) (*omni.AWSParam, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan == nil || config == nil {
		return nil, diags
	}

	var subnetMap map[string]string
	diags.Append(plan.SubnetIDs.ElementsAs(ctx, &subnetMap, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := &omni.AWSParam{
		AccountId: toPtr(plan.AccountID.ValueString()),
		Credentials: &omni.AWSParamCredentials{
			AccessKeyId:     config.AccessKeyIDWO.ValueString(),
			SecretAccessKey: config.SecretAccessKeyWO.ValueString(),
		},
		Networking: &omni.AWSParamAWSNetworking{
			VpcId:           plan.VpcID.ValueString(),
			SecurityGroupId: plan.SecurityGroupID.ValueString(),
			SubnetIds:       subnetMap,
			NameTag:         plan.NameTag.ValueString(),
		},
	}

	return out, diags
}

func (r *edgeLocationResource) toAWSModel(ctx context.Context, config *omni.AWSParam) (*awsModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if config == nil {
		return nil, diags
	}

	aws := &awsModel{
		AccountID:         types.StringValue(lo.FromPtr(config.AccountId)),
		VpcID:             types.StringNull(),
		SecurityGroupID:   types.StringNull(),
		SubnetIDs:         types.MapNull(types.StringType),
		NameTag:           types.StringNull(),
		AccessKeyIDWO:     types.StringNull(),
		SecretAccessKeyWO: types.StringNull(),
	}

	if config.Networking != nil {
		aws.VpcID = types.StringValue(config.Networking.VpcId)
		aws.SecurityGroupID = types.StringValue(config.Networking.SecurityGroupId)
		aws.NameTag = types.StringValue(config.Networking.NameTag)

		if config.Networking.SubnetIds != nil {
			subnetMap, d := types.MapValueFrom(ctx, types.StringType, config.Networking.SubnetIds)
			diags.Append(d...)
			aws.SubnetIDs = subnetMap
		}
	}

	return aws, diags
}

func (r *edgeLocationResource) toGCP(ctx context.Context, plan, config *gcpModel) (*omni.GCPParam, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan == nil || config == nil {
		return nil, diags
	}

	var tags []string
	diags.Append(plan.NetworkTags.ElementsAs(ctx, &tags, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := &omni.GCPParam{
		ProjectId: plan.ProjectID.ValueString(),
		Credentials: &omni.GCPParamCredentials{
			ClientServiceAccountJsonBase64: config.ClientServiceAccountJSONBase64WO.ValueString(),
		},
		Networking: &omni.GCPParamGCPNetworking{
			NetworkName: plan.NetworkName.ValueString(),
			SubnetName:  plan.SubnetName.ValueString(),
			Tags:        tags,
		},
	}

	return out, diags
}

func (r *edgeLocationResource) toGCPModel(ctx context.Context, config *omni.GCPParam) (*gcpModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if config == nil {
		return nil, diags
	}

	gcp := &gcpModel{
		ProjectID:                        types.StringValue(config.ProjectId),
		ClientServiceAccountJSONBase64WO: types.StringNull(),
		NetworkName:                      types.StringNull(),
		SubnetName:                       types.StringNull(),
		NetworkTags:                      types.SetNull(types.StringType),
	}

	if config.Networking != nil {
		gcp.NetworkName = types.StringValue(config.Networking.NetworkName)
		gcp.SubnetName = types.StringValue(config.Networking.SubnetName)
		if config.Networking.Tags != nil {
			tagsSet, d := types.SetValueFrom(ctx, types.StringType, config.Networking.Tags)
			diags.Append(d...)
			gcp.NetworkTags = tagsSet
		}
	}

	return gcp, diags
}

func (r *edgeLocationResource) toOCI(plan, config *ociModel) *omni.OCIParam {
	if plan == nil || config == nil {
		return nil
	}

	out := &omni.OCIParam{
		TenancyId:     toPtr(plan.TenancyID.ValueString()),
		CompartmentId: toPtr(plan.CompartmentID.ValueString()),
		Credentials: &omni.OCIParamCredentials{
			UserId:           config.UserIDWO.ValueString(),
			Fingerprint:      config.FingerprintWO.ValueString(),
			PrivateKeyBase64: config.PrivateKeyWO.ValueString(),
		},
		Networking: &omni.OCIParamNetworking{
			VcnId:    plan.VcnID.ValueString(),
			SubnetId: plan.SubnetID.ValueString(),
		},
	}

	return out
}

func (r *edgeLocationResource) toOCIModel(config *omni.OCIParam) *ociModel {
	if config == nil {
		return nil
	}

	oci := &ociModel{
		TenancyID:     types.StringValue(lo.FromPtr(config.TenancyId)),
		CompartmentID: types.StringValue(lo.FromPtr(config.CompartmentId)),
		VcnID:         types.StringNull(),
		SubnetID:      types.StringNull(),
		UserIDWO:      types.StringNull(),
		FingerprintWO: types.StringNull(),
		PrivateKeyWO:  types.StringNull(),
	}
	if config.Networking != nil {
		oci.SubnetID = types.StringValue(config.Networking.SubnetId)
		oci.VcnID = types.StringValue(config.Networking.VcnId)
	}

	return oci
}

func (r *edgeLocationResource) woCredentialsStore(private store.PrivateState) *store.WriteOnlyStore {
	return store.NewWriteOnlyStore(private, "credentials")
}
