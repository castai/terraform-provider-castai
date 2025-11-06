package castai

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

var (
	_ resource.Resource                = (*edgeLocationResource)(nil)
	_ resource.ResourceWithConfigure   = (*edgeLocationResource)(nil)
	_ resource.ResourceWithImportState = (*edgeLocationResource)(nil)
)

type edgeLocationResource struct {
	client *ProviderConfig
}

type edgeLocationModel struct {
	ID                  types.String `tfsdk:"id"`
	OrganizationID      types.String `tfsdk:"organization_id"`
	ClusterID           types.String `tfsdk:"cluster_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	CredentialsRevision types.String `tfsdk:"credentials_revision"`
	Region              types.String `tfsdk:"region"`
	Zones               []zoneModel  `tfsdk:"zones"`
	AWS                 types.List   `tfsdk:"aws"`
	GCP                 types.List   `tfsdk:"gcp"`
	OCI                 types.List   `tfsdk:"oci"`
}

type zoneModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type awsModel struct {
	AccountID       types.String `tfsdk:"account_id"`
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
	VpcID           types.String `tfsdk:"vpc_id"`
	SecurityGroupID types.String `tfsdk:"security_group_id"`
	SubnetIDs       types.Map    `tfsdk:"subnet_ids"`
	NameTag         types.String `tfsdk:"name_tag"`
}

type gcpModel struct {
	ProjectID                types.String `tfsdk:"project_id"`
	ClientServiceAccountJSON types.String `tfsdk:"client_service_account_json"`
	NetworkName              types.String `tfsdk:"network_name"`
	SubnetName               types.String `tfsdk:"subnet_name"`
	NetworkTags              types.Set    `tfsdk:"network_tags"`
}

type ociModel struct {
	TenancyID     types.String `tfsdk:"tenancy_id"`
	CompartmentID types.String `tfsdk:"compartment_id"`
	UserID        types.String `tfsdk:"user_id"`
	Fingerprint   types.String `tfsdk:"fingerprint"`
	PrivateKey    types.String `tfsdk:"private_key"`
	VcnID         types.String `tfsdk:"vcn_id"`
	SubnetID      types.String `tfsdk:"subnet_id"`
}

func NewEdgeLocationResource() resource.Resource {
	return &edgeLocationResource{}
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
			"credentials_revision": schema.StringAttribute{
				Computed:    true,
				Description: "Hash of credentials used to detect credential changes",
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
		},
		Blocks: map[string]schema.Block{
			"aws": schema.ListNestedBlock{
				Description: "AWS configuration for the edge location",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("gcp"),
						path.MatchRoot("oci"),
					),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"account_id": schema.StringAttribute{
							Required:    true,
							Description: "AWS account ID",
						},
						"access_key_id": schema.StringAttribute{
							Required:    true,
							WriteOnly:   true,
							Sensitive:   true,
							Description: "AWS access key ID",
						},
						"secret_access_key": schema.StringAttribute{
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
			},
			"gcp": schema.ListNestedBlock{
				Description: "GCP configuration for the edge location",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("aws"),
						path.MatchRoot("oci"),
					),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"project_id": schema.StringAttribute{
							Required:    true,
							Description: "GCP project ID where edges run",
						},
						"client_service_account_json": schema.StringAttribute{
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
			},
			"oci": schema.ListNestedBlock{
				Description: "OCI configuration for the edge location",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
					listvalidator.ExactlyOneOf(
						path.MatchRoot("aws"),
						path.MatchRoot("gcp"),
					),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"tenancy_id": schema.StringAttribute{
							Required:    true,
							Description: "OCI tenancy ID of the account",
						},
						"compartment_id": schema.StringAttribute{
							Required:    true,
							Description: "OCI compartment ID of edge location",
						},
						"user_id": schema.StringAttribute{
							Required:    true,
							Description: "User ID used to authenticate OCI",
							WriteOnly:   true,
						},
						"fingerprint": schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "API key fingerprint",
						},
						"private_key": schema.StringAttribute{
							WriteOnly:   true,
							Required:    true,
							Sensitive:   true,
							Description: "Base64 encoded API private key",
						},
						"vcn_id": schema.StringAttribute{
							Required:    true,
							WriteOnly:   true,
							Description: "OCI virtual cloud network ID",
						},
						"subnet_id": schema.StringAttribute{
							Required:    true,
							Description: "OCI subnet ID of edge location",
						},
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
		plan   edgeLocationModel
		config edgeLocationModel
	)

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
	if !plan.AWS.IsNull() && len(plan.AWS.Elements()) > 0 {
		awsConfig, diags := r.expandAWSConfig(ctx, plan.AWS, config.AWS)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Aws = awsConfig
	}

	if !plan.GCP.IsNull() && len(plan.GCP.Elements()) > 0 {
		gcpConfig, diags := r.expandGCPConfig(ctx, plan.GCP, config.GCP)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Gcp = gcpConfig
	}

	if !plan.OCI.IsNull() && len(plan.OCI.Elements()) > 0 {
		ociConfig, diags := r.expandOCIConfig(ctx, plan.OCI, config.OCI)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Oci = ociConfig
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

	// Compute credentials revision hash
	credentialsHash := r.computeCredentialsHash(ctx, &plan)
	plan.CredentialsRevision = types.StringValue(credentialsHash)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *edgeLocationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var (
		state edgeLocationModel
	)
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

	state.Zones = nil
	if edgeLocation.Zones != nil {
		state.Zones = r.toZoneModel(edgeLocation.Zones)
	}

	// Flatten cloud provider configs (preserving write-only fields from state)
	if edgeLocation.Aws != nil {
		awsList, diags := r.flattenAWSConfig(ctx, edgeLocation.Aws, state.AWS)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.AWS = awsList
	}

	if edgeLocation.Gcp != nil {
		gcpList, diags := r.flattenGCPConfig(ctx, edgeLocation.Gcp, state.GCP)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.GCP = gcpList
	}

	if edgeLocation.Oci != nil {
		ociList, diags := r.flattenOCIConfig(ctx, edgeLocation.Oci, state.OCI)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.OCI = ociList
	}

	// Compute credentials revision hash
	credentialsHash := r.computeCredentialsHash(ctx, &state)
	state.CredentialsRevision = types.StringValue(credentialsHash)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *edgeLocationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var (
		plan   edgeLocationModel
		config edgeLocationModel
	)

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	// Map cloud provider specific configurations for update
	// API requires complete objects with all fields including credentials
	if !plan.AWS.IsNull() && len(plan.AWS.Elements()) > 0 {
		awsConfig, diags := r.expandAWSConfig(ctx, plan.AWS, config.AWS)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Aws = awsConfig
	}

	if !plan.GCP.IsNull() && len(plan.GCP.Elements()) > 0 {
		gcpConfig, diags := r.expandGCPConfig(ctx, plan.GCP, config.GCP)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Gcp = gcpConfig
	}

	if !plan.OCI.IsNull() && len(plan.OCI.Elements()) > 0 {
		ociConfig, diags := r.expandOCIConfig(ctx, plan.OCI, config.OCI)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.Oci = ociConfig
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

	// Compute credentials revision hash
	credentialsHash := r.computeCredentialsHash(ctx, &plan)
	plan.CredentialsRevision = types.StringValue(credentialsHash)

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

func (r *edgeLocationResource) expandAWSConfig(ctx context.Context, planList, configList types.List) (*omni.AWSParam, diag.Diagnostics) {
	var (
		diags       diag.Diagnostics
		configModel []awsModel
		planModel   []awsModel
	)

	diags.Append(planList.ElementsAs(ctx, &planModel, false)...)
	if diags.HasError() {
		return nil, diags
	}
	diags.Append(configList.ElementsAs(ctx, &configModel, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if len(planModel) == 0 || len(configModel) == 0 {
		return nil, diags
	}
	plan, config := planModel[0], configModel[0]

	var subnetMap map[string]string
	diags.Append(plan.SubnetIDs.ElementsAs(ctx, &subnetMap, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := &omni.AWSParam{
		AccountId: toPtr(plan.AccountID.ValueString()),
		Credentials: &omni.AWSParamCredentials{
			AccessKeyId:     config.AccessKeyID.ValueString(),
			SecretAccessKey: config.SecretAccessKey.ValueString(),
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

func (r *edgeLocationResource) flattenAWSConfig(ctx context.Context, config *omni.AWSParam, stateList types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if config == nil {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"account_id":        types.StringType,
				"access_key_id":     types.StringType,
				"secret_access_key": types.StringType,
				"vpc_id":            types.StringType,
				"security_group_id": types.StringType,
				"subnet_ids":        types.MapType{ElemType: types.StringType},
				"name_tag":          types.StringType,
			},
		}), diags
	}

	aws := awsModel{}

	if config.AccountId != nil {
		aws.AccountID = types.StringValue(*config.AccountId)
	} else {
		aws.AccountID = types.StringNull()
	}

	// Preserve write-only credentials from state
	if !stateList.IsNull() && len(stateList.Elements()) > 0 {
		var stateConfigs []awsModel
		diags.Append(stateList.ElementsAs(ctx, &stateConfigs, false)...)
		if !diags.HasError() && len(stateConfigs) > 0 {
			aws.AccessKeyID = stateConfigs[0].AccessKeyID
			aws.SecretAccessKey = stateConfigs[0].SecretAccessKey
		}
	} else {
		aws.AccessKeyID = types.StringNull()
		aws.SecretAccessKey = types.StringNull()
	}

	if config.Networking != nil {
		aws.VpcID = types.StringValue(config.Networking.VpcId)
		aws.SecurityGroupID = types.StringValue(config.Networking.SecurityGroupId)
		aws.NameTag = types.StringValue(config.Networking.NameTag)

		if config.Networking.SubnetIds != nil {
			subnetMap, d := types.MapValueFrom(ctx, types.StringType, config.Networking.SubnetIds)
			diags.Append(d...)
			aws.SubnetIDs = subnetMap
		} else {
			aws.SubnetIDs = types.MapNull(types.StringType)
		}
	} else {
		aws.VpcID = types.StringNull()
		aws.SecurityGroupID = types.StringNull()
		aws.SubnetIDs = types.MapNull(types.StringType)
		aws.NameTag = types.StringNull()
	}

	listValue, d := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"account_id":        types.StringType,
			"access_key_id":     types.StringType,
			"secret_access_key": types.StringType,
			"vpc_id":            types.StringType,
			"security_group_id": types.StringType,
			"subnet_ids":        types.MapType{ElemType: types.StringType},
			"name_tag":          types.StringType,
		},
	}, []awsModel{aws})
	diags.Append(d...)
	return listValue, diags
}

func (r *edgeLocationResource) expandGCPConfig(ctx context.Context, planList, configList types.List) (*omni.GCPParam, diag.Diagnostics) {
	var (
		diags       diag.Diagnostics
		configModel []gcpModel
		planModel   []gcpModel
	)

	diags.Append(planList.ElementsAs(ctx, &planModel, false)...)
	if diags.HasError() {
		return nil, diags
	}
	diags.Append(configList.ElementsAs(ctx, &configModel, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if len(planModel) == 0 || len(configModel) == 0 {
		return nil, diags
	}
	plan, config := planModel[0], configModel[0]

	var tags []string
	diags.Append(plan.NetworkTags.ElementsAs(ctx, &tags, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := &omni.GCPParam{
		ProjectId: plan.ProjectID.ValueString(),
		Credentials: &omni.GCPParamCredentials{
			ClientServiceAccountJsonBase64: config.ClientServiceAccountJSON.ValueString(),
		},
		Networking: &omni.GCPParamGCPNetworking{
			NetworkName: plan.NetworkName.ValueString(),
			SubnetName:  plan.SubnetName.ValueString(),
			Tags:        tags,
		},
	}

	return out, diags
}

func (r *edgeLocationResource) flattenGCPConfig(ctx context.Context, config *omni.GCPParam, stateList types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if config == nil {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"project_id":                  types.StringType,
				"client_service_account_json": types.StringType,
				"network_name":                types.StringType,
				"subnet_name":                 types.StringType,
				"network_tags":                types.SetType{ElemType: types.StringType},
			},
		}), diags
	}

	gcp := gcpModel{
		ProjectID: types.StringValue(config.ProjectId),
	}

	// Preserve write-only credentials from state
	if !stateList.IsNull() && len(stateList.Elements()) > 0 {
		var stateConfigs []gcpModel
		diags.Append(stateList.ElementsAs(ctx, &stateConfigs, false)...)
		if !diags.HasError() && len(stateConfigs) > 0 {
			gcp.ClientServiceAccountJSON = stateConfigs[0].ClientServiceAccountJSON
		}
	} else {
		gcp.ClientServiceAccountJSON = types.StringNull()
	}

	if config.Networking != nil {
		gcp.NetworkName = types.StringValue(config.Networking.NetworkName)
		gcp.SubnetName = types.StringValue(config.Networking.SubnetName)

		if config.Networking.Tags != nil {
			tagsSet, d := types.SetValueFrom(ctx, types.StringType, config.Networking.Tags)
			diags.Append(d...)
			gcp.NetworkTags = tagsSet
		} else {
			gcp.NetworkTags = types.SetNull(types.StringType)
		}
	} else {
		gcp.NetworkName = types.StringNull()
		gcp.SubnetName = types.StringNull()
		gcp.NetworkTags = types.SetNull(types.StringType)
	}

	listValue, d := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"project_id":                  types.StringType,
			"client_service_account_json": types.StringType,
			"network_name":                types.StringType,
			"subnet_name":                 types.StringType,
			"network_tags":                types.SetType{ElemType: types.StringType},
		},
	}, []gcpModel{gcp})
	diags.Append(d...)
	return listValue, diags
}

func (r *edgeLocationResource) expandOCIConfig(ctx context.Context, planList, configList types.List) (*omni.OCIParam, diag.Diagnostics) {
	var (
		diags       diag.Diagnostics
		configModel []ociModel
		planModel   []ociModel
	)

	diags.Append(planList.ElementsAs(ctx, &planModel, false)...)
	if diags.HasError() {
		return nil, diags
	}
	diags.Append(configList.ElementsAs(ctx, &configModel, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if len(planModel) == 0 || len(configModel) == 0 {
		return nil, diags
	}
	plan, config := planModel[0], configModel[0]

	out := &omni.OCIParam{
		TenancyId:     toPtr(plan.TenancyID.ValueString()),
		CompartmentId: toPtr(plan.CompartmentID.ValueString()),
		Credentials: &omni.OCIParamCredentials{
			UserId:           config.UserID.ValueString(),
			Fingerprint:      config.Fingerprint.ValueString(),
			PrivateKeyBase64: config.PrivateKey.ValueString(),
		},
		Networking: &omni.OCIParamNetworking{
			VcnId:    config.VcnID.ValueString(),
			SubnetId: plan.SubnetID.ValueString(),
		},
	}

	return out, diags
}

func (r *edgeLocationResource) flattenOCIConfig(ctx context.Context, config *omni.OCIParam, stateList types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if config == nil {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"tenancy_id":     types.StringType,
				"compartment_id": types.StringType,
				"user_id":        types.StringType,
				"fingerprint":    types.StringType,
				"private_key":    types.StringType,
				"vcn_id":         types.StringType,
				"subnet_id":      types.StringType,
			},
		}), diags
	}

	oci := ociModel{}

	if config.TenancyId != nil {
		oci.TenancyID = types.StringValue(*config.TenancyId)
	} else {
		oci.TenancyID = types.StringNull()
	}

	if config.CompartmentId != nil {
		oci.CompartmentID = types.StringValue(*config.CompartmentId)
	} else {
		oci.CompartmentID = types.StringNull()
	}

	// Preserve write-only credentials from state
	if !stateList.IsNull() && len(stateList.Elements()) > 0 {
		var stateConfigs []ociModel
		diags.Append(stateList.ElementsAs(ctx, &stateConfigs, false)...)
		if !diags.HasError() && len(stateConfigs) > 0 {
			oci.UserID = stateConfigs[0].UserID
			oci.Fingerprint = stateConfigs[0].Fingerprint
			oci.PrivateKey = stateConfigs[0].PrivateKey
			oci.VcnID = stateConfigs[0].VcnID
		}
	} else {
		oci.UserID = types.StringNull()
		oci.Fingerprint = types.StringNull()
		oci.PrivateKey = types.StringNull()
		oci.VcnID = types.StringNull()
	}

	if config.Networking != nil {
		oci.SubnetID = types.StringValue(config.Networking.SubnetId)
	} else {
		oci.SubnetID = types.StringNull()
	}

	listValue, d := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"tenancy_id":     types.StringType,
			"compartment_id": types.StringType,
			"user_id":        types.StringType,
			"fingerprint":    types.StringType,
			"private_key":    types.StringType,
			"vcn_id":         types.StringType,
			"subnet_id":      types.StringType,
		},
	}, []ociModel{oci})
	diags.Append(d...)
	return listValue, diags
}

func (r *edgeLocationResource) computeCredentialsHash(ctx context.Context, model *edgeLocationModel) string {
	hasher := sha256.New()

	// Hash AWS credentials
	if !model.AWS.IsNull() && len(model.AWS.Elements()) > 0 {
		var awsConfigs []awsModel
		model.AWS.ElementsAs(ctx, &awsConfigs, false)
		if len(awsConfigs) > 0 {
			aws := awsConfigs[0]
			if !aws.AccessKeyID.IsNull() {
				hasher.Write([]byte("aws_access_key:"))
				hasher.Write([]byte(aws.AccessKeyID.ValueString()))
			}
			if !aws.SecretAccessKey.IsNull() {
				hasher.Write([]byte("aws_secret_key:"))
				hasher.Write([]byte(aws.SecretAccessKey.ValueString()))
			}
		}
	}

	// Hash GCP credentials
	if !model.GCP.IsNull() && len(model.GCP.Elements()) > 0 {
		var gcpConfigs []gcpModel
		model.GCP.ElementsAs(ctx, &gcpConfigs, false)
		if len(gcpConfigs) > 0 {
			gcp := gcpConfigs[0]
			if !gcp.ClientServiceAccountJSON.IsNull() {
				hasher.Write([]byte("gcp_service_account:"))
				hasher.Write([]byte(gcp.ClientServiceAccountJSON.ValueString()))
			}
		}
	}

	// Hash OCI credentials
	if !model.OCI.IsNull() && len(model.OCI.Elements()) > 0 {
		var ociConfigs []ociModel
		model.OCI.ElementsAs(ctx, &ociConfigs, false)
		if len(ociConfigs) > 0 {
			oci := ociConfigs[0]
			if !oci.UserID.IsNull() {
				hasher.Write([]byte("oci_user_id:"))
				hasher.Write([]byte(oci.UserID.ValueString()))
			}
			if !oci.Fingerprint.IsNull() {
				hasher.Write([]byte("oci_fingerprint:"))
				hasher.Write([]byte(oci.Fingerprint.ValueString()))
			}
			if !oci.PrivateKey.IsNull() {
				hasher.Write([]byte("oci_private_key:"))
				hasher.Write([]byte(oci.PrivateKey.ValueString()))
			}
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}
