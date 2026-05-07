package castai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
	"github.com/castai/terraform-provider-castai/castai/store"
	"github.com/castai/terraform-provider-castai/castai/validators"
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
	ID               types.String       `tfsdk:"id"`
	OrganizationID   types.String       `tfsdk:"organization_id"`
	ClusterID        types.String       `tfsdk:"cluster_id"`
	Name             types.String       `tfsdk:"name"`
	Description      types.String       `tfsdk:"description"`
	Region           types.String       `tfsdk:"region"`
	ControlPlaneMode types.String       `tfsdk:"control_plane_mode"`
	ControlPlane     *controlPlaneModel `tfsdk:"control_plane"`
	Networking       *networkingModel   `tfsdk:"networking"`
	Zones            []zoneModel        `tfsdk:"zones"`
	AWS              *awsModel          `tfsdk:"aws"`
	GCP              *gcpModel          `tfsdk:"gcp"`
	OCI              *ociModel          `tfsdk:"oci"`
	// Computed revision number incremented each time credentials have changed.
	CredentialsRevision types.Int64 `tfsdk:"credentials_revision"`
}

type controlPlaneModel struct {
	Ha types.Bool `tfsdk:"ha"`
}

type networkingModel struct {
	TunneledCIDRs types.List `tfsdk:"tunneled_cidrs"`
}

type zoneModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type awsModel struct {
	AccountID         types.String `tfsdk:"account_id"`
	InstanceProfile   types.String `tfsdk:"instance_profile"`
	RoleArn           types.String `tfsdk:"role_arn"`
	AccessKeyIDWO     types.String `tfsdk:"access_key_id_wo"`
	SecretAccessKeyWO types.String `tfsdk:"secret_access_key_wo"`
	VpcID             types.String `tfsdk:"vpc_id"`
	VpcCidr           types.String `tfsdk:"vpc_cidr"`
	VpcPeered         types.Bool   `tfsdk:"vpc_peered"`
	SecurityGroupID   types.String `tfsdk:"security_group_id"`
	SubnetIDs         types.Map    `tfsdk:"subnet_ids"`
	// Deprecated. Should be removed with name_tag attribute removal.
	NameTag types.String `tfsdk:"name_tag"`
}

type gcpModel struct {
	ProjectID                        types.String `tfsdk:"project_id"`
	InstanceServiceAccount           types.String `tfsdk:"instance_service_account"`
	TargetServiceAccountEmail        types.String `tfsdk:"target_service_account_email"`
	ClientServiceAccountJSONBase64WO types.String `tfsdk:"client_service_account_json_base64_wo"`
	NetworkName                      types.String `tfsdk:"network_name"`
	SubnetName                       types.String `tfsdk:"subnet_name"`
	SubnetCidr                       types.String `tfsdk:"subnet_cidr"`
	NetworkTags                      types.Set    `tfsdk:"network_tags"`
}

type ociModel struct {
	TenancyID          types.String `tfsdk:"tenancy_id"`
	CompartmentID      types.String `tfsdk:"compartment_id"`
	UserIDWO           types.String `tfsdk:"user_id_wo"`
	FingerprintWO      types.String `tfsdk:"fingerprint_wo"`
	PrivateKeyBase64WO types.String `tfsdk:"private_key_base64_wo"`
	VcnID              types.String `tfsdk:"vcn_id"`
	VcnCidr            types.String `tfsdk:"vcn_cidr"`
	SubnetID           types.String `tfsdk:"subnet_id"`
	SecurityGroupID    types.String `tfsdk:"security_group_id"`
}

func (m awsModel) credentials() types.String {
	if m.AccessKeyIDWO.IsNull() && m.SecretAccessKeyWO.IsNull() {
		return types.StringNull()
	}
	return types.StringValue(m.SecretAccessKeyWO.String() + m.AccessKeyIDWO.String())
}

func (m awsModel) Equal(other *awsModel) bool {
	if other == nil {
		return false
	}
	return m.AccountID.Equal(other.AccountID) &&
		m.RoleArn.Equal(other.RoleArn) &&
		m.VpcID.Equal(other.VpcID) &&
		m.VpcCidr.Equal(other.VpcCidr) &&
		m.VpcPeered.Equal(other.VpcPeered) &&
		m.SecurityGroupID.Equal(other.SecurityGroupID) &&
		m.SubnetIDs.Equal(other.SubnetIDs) &&
		m.InstanceProfile.Equal(other.InstanceProfile)
}

func (m gcpModel) credentials() types.String {
	if m.ClientServiceAccountJSONBase64WO.IsNull() {
		return types.StringNull()
	}
	return m.ClientServiceAccountJSONBase64WO
}

func (m gcpModel) Equal(other *gcpModel) bool {
	if other == nil {
		return false
	}
	return m.ProjectID.Equal(other.ProjectID) &&
		m.TargetServiceAccountEmail.Equal(other.TargetServiceAccountEmail) &&
		m.NetworkName.Equal(other.NetworkName) &&
		m.SubnetName.Equal(other.SubnetName) &&
		m.SubnetCidr.Equal(other.SubnetCidr) &&
		m.NetworkTags.Equal(other.NetworkTags) &&
		m.InstanceServiceAccount.Equal(other.InstanceServiceAccount)
}

func (m ociModel) credentials() types.String {
	return types.StringValue(m.UserIDWO.String() + m.PrivateKeyBase64WO.String() + m.FingerprintWO.String())
}

func (m ociModel) Equal(other *ociModel) bool {
	if other == nil {
		return false
	}
	return m.TenancyID.Equal(other.TenancyID) &&
		m.CompartmentID.Equal(other.CompartmentID) &&
		m.VcnID.Equal(other.VcnID) &&
		m.VcnCidr.Equal(other.VcnCidr) &&
		m.SubnetID.Equal(other.SubnetID) &&
		m.SecurityGroupID.Equal(other.SecurityGroupID)
}

type ModelWithCredentials interface {
	credentials() types.String
}

func newEdgeLocationResource() resource.Resource {
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
			"control_plane_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(string(omni.DEDICATED)),
				Description: "The mode of control plane inside edge location. Valid values: DEDICATED, SHARED.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(string(omni.DEDICATED), string(omni.SHARED)),
				},
			},
			"control_plane": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Control plane configuration. Only valid when control_plane_mode is SHARED.",
				Attributes: map[string]schema.Attribute{
					"ha": schema.BoolAttribute{
						Required:    true,
						Description: "Whether to use HA mode for control plane. If not set, default is HA.",
					},
				},
			},
			"networking": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Edge cluster networking configuration.",
				Attributes: map[string]schema.Attribute{
					"tunneled_cidrs": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "List of destination CIDR blocks whose traffic should be routed through the main cluster instead of directly from the edge cluster.",
					},
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
					"instance_profile": schema.StringAttribute{
						Optional:    true,
						Description: "AWS IAM instance profile ARN to be attached to edge instances. It can be used to grant permissions to access other AWS resources such as ECR.",
					},
					"role_arn": schema.StringAttribute{
						Optional:    true,
						Description: "AWS IAM role ARN used for Google OIDC federation impersonation",
					},
					"access_key_id_wo": schema.StringAttribute{
						Optional:    true,
						WriteOnly:   true,
						Sensitive:   true,
						Description: "AWS access key ID",
					},
					"secret_access_key_wo": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						WriteOnly:   true,
						Description: "AWS secret access key",
					},
					"vpc_id": schema.StringAttribute{
						Required:    true,
						Description: "VPC ID to be used in the selected region",
					},
					"vpc_cidr": schema.StringAttribute{
						Optional:    true,
						Description: "VPC IPv4 CIDR block",
					},
					"vpc_peered": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether existing VPC is peered with main cluster's VPC. Field is ignored if vpc_id is not provided or main cluster is not EKS",
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
						Optional:           true,
						Description:        "The value of a 'Name' tag applied to VPC resources",
						DeprecationMessage: "Deprecated. Can be omitted",
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
					"instance_service_account": schema.StringAttribute{
						Optional:    true,
						Description: "GCP service account email to be attached to edge instances. It can be used to grant permissions to access other GCP resources.",
					},
					"target_service_account_email": schema.StringAttribute{
						Optional:    true,
						Description: "Target service account email to be used for impersonation",
					},
					"client_service_account_json_base64_wo": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						WriteOnly:   true,
						Description: "Base64 encoded service account JSON for provisioning edge resources",
						Validators: []validator.String{
							validators.ValidBase64(),
						},
					},
					"network_name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the network to be used in the selected region",
					},
					"subnet_name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the subnetwork to be used in the selected region",
					},
					"subnet_cidr": schema.StringAttribute{
						Optional:    true,
						Description: "VPC Subnet IPv4 CIDR block",
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
					"private_key_base64_wo": schema.StringAttribute{
						WriteOnly:   true,
						Required:    true,
						Sensitive:   true,
						Description: "Base64 encoded API private key",
						Validators: []validator.String{
							validators.ValidBase64(),
						},
					},
					"vcn_id": schema.StringAttribute{
						Required:    true,
						Description: "OCI virtual cloud network ID",
					},
					"vcn_cidr": schema.StringAttribute{
						Optional:    true,
						Description: "OCI VCN IPv4 CIDR block",
					},
					"subnet_id": schema.StringAttribute{
						Required:    true,
						Description: "OCI subnet ID of edge location",
					},
					"security_group_id": schema.StringAttribute{
						Optional:    true,
						Description: "OCI network security group ID",
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

	client := r.client.omniAPI
	organizationID := plan.OrganizationID.ValueString()
	clusterID := plan.ClusterID.ValueString()

	createReq := omni.EdgeLocationsAPICreateEdgeLocationJSONRequestBody{
		Name:             plan.Name.ValueString(),
		Region:           plan.Region.ValueStringPointer(),
		Zones:            lo.ToPtr(r.toZones(plan.Zones)),
		ControlPlaneMode: lo.ToPtr(omni.EdgeLocationControlPlaneMode(plan.ControlPlaneMode.ValueString())),
	}

	if !plan.Description.IsNull() {
		createReq.Description = lo.ToPtr(plan.Description.ValueString())
	}

	createReq.EdgeClusterSpec, diags = r.toEdgeClusterSpec(ctx, plan.ControlPlane, plan.Networking)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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
	// Store credential hash in private state (skip when credentials are missing e.g. OIDC flow with AWS and GCP)
	if mc != nil && !mc.credentials().IsNull() {
		resp.Diagnostics.Append(r.woCredentialsStore(resp.Private).Set(ctx, mc.credentials())...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *edgeLocationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state edgeLocationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.omniAPI
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

	if edgeLocation.Region != nil {
		state.Region = types.StringValue(*edgeLocation.Region)
	}
	state.Name = types.StringValue(edgeLocation.Name)
	state.Description = types.StringNull()
	if edgeLocation.Description != nil {
		state.Description = types.StringValue(*edgeLocation.Description)
	}
	if edgeLocation.ControlPlaneMode != nil {
		state.ControlPlaneMode = types.StringValue(string(*edgeLocation.ControlPlaneMode))
	}
	if edgeLocation.EdgeClusterSpec != nil {
		if edgeLocation.EdgeClusterSpec.ControlPlane != nil {
			state.ControlPlane = &controlPlaneModel{
				Ha: types.BoolPointerValue(edgeLocation.EdgeClusterSpec.ControlPlane.Ha),
			}
		}
		if edgeLocation.EdgeClusterSpec.Networking != nil && edgeLocation.EdgeClusterSpec.Networking.TunneledCidrs != nil {
			cidrs, d := types.ListValueFrom(ctx, types.StringType, *edgeLocation.EdgeClusterSpec.Networking.TunneledCidrs)
			resp.Diagnostics.Append(d...)
			state.Networking = &networkingModel{TunneledCIDRs: cidrs}
		}
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

	client := r.client.omniAPI
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

	updateReq.EdgeClusterSpec, diags = r.toEdgeClusterSpec(ctx, plan.ControlPlane, plan.Networking)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Include cloud provider config if it or credentials has changed.
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

	// Update stored credentials hash if credentials changed (skip when using impersonation flow)
	if credentialsChanged && mc != nil && !mc.credentials().IsNull() {
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

	client := r.client.omniAPI
	organizationID := state.OrganizationID.ValueString()
	clusterID := state.ClusterID.ValueString()
	edgeLocationID := state.ID.ValueString()

	apiResp, err := client.EdgeLocationsAPIDeleteEdgeLocationWithResponse(ctx, organizationID, clusterID, edgeLocationID)
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

	// Edge location deletion is async. Wait until it's fully removed
	// so that dependent resources (e.g. omni cluster) can be deleted.
	if err := r.waitForDeletion(ctx, client, organizationID, clusterID, edgeLocationID); err != nil {
		resp.Diagnostics.AddError("Failed waiting for edge location deletion", err.Error())
	}
}

func (r *edgeLocationResource) waitForDeletion(ctx context.Context, client omni.ClientWithResponsesInterface, organizationID, clusterID, edgeLocationID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp, err := client.EdgeLocationsAPIGetEdgeLocationWithResponse(ctx, organizationID, clusterID, edgeLocationID)
			if err != nil {
				return fmt.Errorf("polling edge location status: %w", err)
			}
			if resp.StatusCode() == http.StatusNotFound {
				return nil
			}
		}
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

	var mc ModelWithCredentials
	switch {
	case config.AWS != nil:
		mc = config.AWS
	case config.GCP != nil:
		mc = config.GCP
	case config.OCI != nil:
		mc = config.OCI
	}

	// Skip credentials comparison when no credentials are provided (e.g. impersonation flow)
	if mc != nil && !mc.credentials().IsNull() {
		credentialsEqual, diags := r.woCredentialsStore(req.Private).Equal(ctx, mc.credentials())
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

func (r *edgeLocationResource) toEdgeClusterSpec(ctx context.Context, cp *controlPlaneModel, net *networkingModel) (*omni.EdgeClusterSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	if cp == nil && net == nil {
		return nil, diags
	}

	spec := &omni.EdgeClusterSpec{}
	if cp != nil {
		spec.ControlPlane = &omni.EdgeClusterControlPlane{
			Ha: cp.Ha.ValueBoolPointer(),
		}
	}

	if net != nil {
		var items []string
		diags.Append(net.TunneledCIDRs.ElementsAs(ctx, &items, true)...)
		if diags.HasError() {
			return nil, diags
		}
		spec.Networking = &omni.EdgeClusterNetworking{
			TunneledCidrs: &items,
		}
	}
	return spec, diags
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

	var instanceProfile *string
	if !plan.InstanceProfile.IsNull() && plan.InstanceProfile.ValueString() != "" {
		instanceProfile = lo.ToPtr(plan.InstanceProfile.ValueString())
	}

	out := &omni.AWSParam{
		AccountId:       toPtr(plan.AccountID.ValueString()),
		InstanceProfile: instanceProfile,
		RoleArn:         plan.RoleArn.ValueStringPointer(),
		Networking: &omni.AWSParamAWSNetworking{
			VpcId:           plan.VpcID.ValueString(),
			VpcPeered:       plan.VpcPeered.ValueBoolPointer(),
			VpcCidr:         plan.VpcCidr.ValueStringPointer(),
			SecurityGroupId: plan.SecurityGroupID.ValueString(),
			SubnetIds:       subnetMap,
		},
	}
	if !config.AccessKeyIDWO.IsNull() || !config.SecretAccessKeyWO.IsNull() {
		out.Credentials = &omni.AWSParamCredentials{
			AccessKeyId:     config.AccessKeyIDWO.ValueStringPointer(),
			SecretAccessKey: config.SecretAccessKeyWO.ValueStringPointer(),
		}
	}
	if !plan.NameTag.IsNull() {
		out.Networking.NameTag = lo.ToPtr(plan.NameTag.ValueString())
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
		InstanceProfile:   types.StringNull(),
		RoleArn:           types.StringNull(),
		VpcID:             types.StringNull(),
		VpcCidr:           types.StringNull(),
		SecurityGroupID:   types.StringNull(),
		VpcPeered:         types.BoolNull(),
		SubnetIDs:         types.MapNull(types.StringType),
		NameTag:           types.StringNull(),
		AccessKeyIDWO:     types.StringNull(),
		SecretAccessKeyWO: types.StringNull(),
	}

	if config.InstanceProfile != nil && *config.InstanceProfile != "" {
		aws.InstanceProfile = types.StringValue(*config.InstanceProfile)
	}

	if config.RoleArn != nil && *config.RoleArn != "" {
		aws.RoleArn = types.StringValue(*config.RoleArn)
	}

	if config.Networking != nil {
		aws.VpcID = types.StringValue(config.Networking.VpcId)
		aws.VpcPeered = types.BoolPointerValue(config.Networking.VpcPeered)
		aws.SecurityGroupID = types.StringValue(config.Networking.SecurityGroupId)
		if config.Networking.VpcCidr != nil && *config.Networking.VpcCidr != "" {
			aws.VpcCidr = types.StringValue(*config.Networking.VpcCidr)
		}
		if config.Networking.NameTag != nil && *config.Networking.NameTag != "" {
			aws.NameTag = types.StringValue(*config.Networking.NameTag)
		}

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
		ProjectId:                 plan.ProjectID.ValueString(),
		InstanceServiceAccount:    plan.InstanceServiceAccount.ValueStringPointer(),
		TargetServiceAccountEmail: plan.TargetServiceAccountEmail.ValueStringPointer(),
		Networking: &omni.GCPParamGCPNetworking{
			NetworkName: plan.NetworkName.ValueString(),
			SubnetName:  plan.SubnetName.ValueString(),
			SubnetCidr:  plan.SubnetCidr.ValueStringPointer(),
			Tags:        tags,
		},
	}
	if !config.ClientServiceAccountJSONBase64WO.IsNull() {
		out.Credentials = &omni.GCPParamCredentials{
			ClientServiceAccountJsonBase64: config.ClientServiceAccountJSONBase64WO.ValueStringPointer(),
		}
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
		InstanceServiceAccount:           types.StringNull(),
		TargetServiceAccountEmail:        types.StringNull(),
		ClientServiceAccountJSONBase64WO: types.StringNull(),
		NetworkName:                      types.StringNull(),
		SubnetName:                       types.StringNull(),
		SubnetCidr:                       types.StringNull(),
		NetworkTags:                      types.SetNull(types.StringType),
	}

	if config.InstanceServiceAccount != nil {
		gcp.InstanceServiceAccount = types.StringValue(*config.InstanceServiceAccount)
	}

	if config.TargetServiceAccountEmail != nil && *config.TargetServiceAccountEmail != "" {
		gcp.TargetServiceAccountEmail = types.StringValue(*config.TargetServiceAccountEmail)
	}

	if config.Networking != nil {
		gcp.NetworkName = types.StringValue(config.Networking.NetworkName)
		gcp.SubnetName = types.StringValue(config.Networking.SubnetName)
		if config.Networking.SubnetCidr != nil && *config.Networking.SubnetCidr != "" {
			gcp.SubnetCidr = types.StringValue(*config.Networking.SubnetCidr)
		}
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
		Networking: &omni.OCIParamNetworking{
			VcnId:           plan.VcnID.ValueString(),
			SubnetId:        plan.SubnetID.ValueString(),
			VcnCidr:         plan.VcnCidr.ValueStringPointer(),
			SecurityGroupId: plan.SecurityGroupID.ValueStringPointer(),
		},
		Credentials: &omni.OCIParamCredentials{
			UserId:           config.UserIDWO.ValueString(),
			Fingerprint:      config.FingerprintWO.ValueString(),
			PrivateKeyBase64: config.PrivateKeyBase64WO.ValueString(),
		},
	}

	return out
}

func (r *edgeLocationResource) toOCIModel(config *omni.OCIParam) *ociModel {
	if config == nil {
		return nil
	}

	oci := &ociModel{
		TenancyID:          types.StringValue(lo.FromPtr(config.TenancyId)),
		CompartmentID:      types.StringValue(lo.FromPtr(config.CompartmentId)),
		VcnID:              types.StringNull(),
		VcnCidr:            types.StringNull(),
		SubnetID:           types.StringNull(),
		SecurityGroupID:    types.StringNull(),
		UserIDWO:           types.StringNull(),
		FingerprintWO:      types.StringNull(),
		PrivateKeyBase64WO: types.StringNull(),
	}

	if config.Networking != nil {
		oci.VcnID = types.StringValue(config.Networking.VcnId)
		oci.SubnetID = types.StringValue(config.Networking.SubnetId)
		if config.Networking.VcnCidr != nil && *config.Networking.VcnCidr != "" {
			oci.VcnCidr = types.StringValue(*config.Networking.VcnCidr)
		}
		if config.Networking.SecurityGroupId != nil && *config.Networking.SecurityGroupId != "" {
			oci.SecurityGroupID = types.StringValue(*config.Networking.SecurityGroupId)
		}
	}

	return oci
}

func (r *edgeLocationResource) woCredentialsStore(private store.PrivateState) *store.WriteOnlyStore {
	return store.NewWriteOnlyStore(private, "credentials")
}
