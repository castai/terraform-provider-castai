package castai

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/external_connections"
)

const (
	FieldExtConnPrincipalsCloudProvider    = "cloud_provider"
	FieldExtConnPrincipalsConnectionScope  = "connection_scope"
	FieldExtConnPrincipalsOrganizationID   = "organization_id"
	FieldExtConnPrincipalsScopeKey         = "scope_key"
	FieldExtConnPrincipalsFeatures         = "features"
	FieldExtConnPrincipalsResourceSuffix   = "resource_suffix"
	FieldExtConnPrincipalsProvisionedRes   = "provisioned_resources"
	FieldExtConnPrincipalsPermissions      = "permissions"
	FieldExtConnPrincipalsConnectionID     = "connection_id"

	// features block (reuses FeatureSelection shape)
	FieldExtConnPrincipalsFeatFeature          = "feature"
	FieldExtConnPrincipalsFeatRegistryVersion  = "registry_version"
	FieldExtConnPrincipalsFeatSubFeatures      = "sub_features"

	// provisioned_resources block
	FieldExtConnPrincipalsPRFeature          = "feature"
	FieldExtConnPrincipalsPRGcpSAEmail       = "gcp_service_account_email"
	FieldExtConnPrincipalsPRGcpSAID          = "gcp_service_account_id"

	// permissions block
	FieldExtConnPrincipalsPermName          = "name"
	FieldExtConnPrincipalsPermAccessType    = "access_type"
	FieldExtConnPrincipalsPermResourceType   = "resource_type"
	FieldExtConnPrincipalsPermScope          = "scope"
	FieldExtConnPrincipalsPermFeatureID      = "feature_id"
	FieldExtConnPrincipalsPermSubFeatureID   = "sub_feature_id"
	FieldExtConnPrincipalsPermJustification  = "justification"
	FieldExtConnPrincipalsPermResourceID     = "resource_id"
)

func resourceExternalConnectionPrincipals() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceExternalConnectionPrincipalsCreate,
		ReadContext:   resourceExternalConnectionPrincipalsRead,
		UpdateContext: resourceExternalConnectionPrincipalsUpdate,
		DeleteContext: resourceExternalConnectionPrincipalsDelete,
		Description:   "Provisions Cast-side IAM principals (service accounts, roles) for a cloud-account external connection. The Delete operation is a no-op — Cast-side principals persist in CAST AI and are only removed from Terraform state. The resource_suffix output should be passed to the castai_external_connection resource.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldExtConnPrincipalsCloudProvider: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The cloud provider to provision resources for. Valid values: AWS, AZURE, GCP, ORACLE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS", "AZURE", "GCP", "ORACLE"}, false)),
			},
			FieldExtConnPrincipalsConnectionScope: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The connection scope. Valid values: AWS_ACCOUNT, AWS_ORGANIZATION, AZURE_SUBSCRIPTION, GCP_ORGANIZATION, GCP_PROJECT.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS_ACCOUNT", "AWS_ORGANIZATION", "AZURE_SUBSCRIPTION", "GCP_ORGANIZATION", "GCP_PROJECT"}, false)),
			},
			FieldExtConnPrincipalsOrganizationID: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The organization requesting the connection. If not provided, will attempt to infer it using the CAST AI API client.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldExtConnPrincipalsScopeKey: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional scope key for multi-connection-per-account scenarios. When provided, uniqueness becomes (organization_id, cloud_provider, scope_key) instead of (organization_id, cloud_provider). Each scoped connection gets its own Cast-side principals and resource suffix.",
			},
			FieldExtConnPrincipalsFeatures: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "The features to provision resources for.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldExtConnPrincipalsFeatFeature: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "The feature type. Valid values: CLOUD_CONNECT, COST_MONITORING, NODE_AUTOSCALING, WORKLOAD_AUTOSCALING, KARPENTER_ENTERPRISE, STORAGE_OPTIMIZATION, POD_MUTATIONS, APA, DBO, OMNI, KIMCHI.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"CLOUD_CONNECT", "COST_MONITORING", "NODE_AUTOSCALING", "WORKLOAD_AUTOSCALING", "KARPENTER_ENTERPRISE", "STORAGE_OPTIMIZATION", "POD_MUTATIONS", "APA", "DBO", "OMNI", "KIMCHI"}, false)),
						},
						FieldExtConnPrincipalsFeatRegistryVersion: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Registry version to pin for this feature. When set, the connection is stamped with that version's permissions snapshot. When unset, the latest version is used.",
						},
						FieldExtConnPrincipalsFeatSubFeatures: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Sub-features to include. Empty means base permissions only.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			FieldExtConnPrincipalsResourceSuffix: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Deterministic suffix appended to all IAM resource names for this connection (e.g. c39cca43). Stable across repeated calls with the same input. Pass this to the castai_external_connection resource.",
			},
			FieldExtConnPrincipalsProvisionedRes: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Cast-side resources that were created (or already existed).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldExtConnPrincipalsPRFeature: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The feature this resource serves.",
						},
						FieldExtConnPrincipalsPRGcpSAEmail: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "GCP service account email provisioned in Cast's identity project (GCP only).",
						},
						FieldExtConnPrincipalsPRGcpSAID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "GCP service account unique ID (GCP only).",
						},
					},
				},
			},
			FieldExtConnPrincipalsPermissions: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "IAM permissions the customer must set up in their cloud account.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldExtConnPrincipalsPermName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The permission name — an IAM action, role name, policy name, or policy statement depending on resource_type.",
						},
						FieldExtConnPrincipalsPermAccessType: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Whether this permission is read-only or allows write/mutation. Values: READ, WRITE.",
						},
						FieldExtConnPrincipalsPermResourceType: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The kind of IAM resource this permission belongs to.",
						},
						FieldExtConnPrincipalsPermScope: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Scope at which this permission must be granted.",
						},
						FieldExtConnPrincipalsPermFeatureID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The feature this permission belongs to.",
						},
						FieldExtConnPrincipalsPermSubFeatureID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The sub-feature this permission belongs to. Empty if from base permissions.",
						},
						FieldExtConnPrincipalsPermJustification: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Human-readable explanation of why this permission is needed.",
						},
						FieldExtConnPrincipalsPermResourceID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the resource (policy ID, role ID, etc.). Only set for AWS_CUSTOM_POLICY, GCP_CUSTOM_ROLE, and AZURE_CUSTOM_ROLE.",
						},
					},
				},
			},
			FieldExtConnPrincipalsConnectionID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Set if a cloud connection already exists for this (organization, cloud, scope_key). Empty when no connection has been confirmed yet via UpsertConnection.",
			},
		},
	}
}

func resourceExternalConnectionPrincipalsCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).externalConnectionsClient
	orgID, diags := getExtConnPrincipalsOrgID(ctx, data, meta)
	if diags.HasError() {
		return diags
	}

	req := buildProvisionRequest(data, orgID)

	resp, err := client.ExternalConnectionsAPIProvisionCastPrincipalsWithResponse(ctx, orgID, req)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.Errorf("provisioning cast principals: %v", e)
	}

	if resp.JSON200 == nil {
		return diag.Errorf("provisioning cast principals: empty response body")
	}

	data.SetId(buildPrincipalsID(orgID, data))

	return setPrincipalsState(data, resp.JSON200)
}

func resourceExternalConnectionPrincipalsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ProvisionCastPrincipals is idempotent — re-provisioning with the same inputs returns
	// the same (already-existing) resources. There is no separate Get endpoint.
	client := meta.(*ProviderConfig).externalConnectionsClient
	orgID, diags := getExtConnPrincipalsOrgID(ctx, data, meta)
	if diags.HasError() {
		return diags
	}

	req := buildProvisionRequest(data, orgID)

	resp, err := client.ExternalConnectionsAPIProvisionCastPrincipalsWithResponse(ctx, orgID, req)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.Errorf("reading cast principals: %v", e)
	}

	if resp.JSON200 == nil {
		data.SetId("")
		return nil
	}

	return setPrincipalsState(data, resp.JSON200)
}

func resourceExternalConnectionPrincipalsUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Re-provision with updated inputs — idempotent, creates any new principals
	// and returns the full set.
	client := meta.(*ProviderConfig).externalConnectionsClient
	orgID, diags := getExtConnPrincipalsOrgID(ctx, data, meta)
	if diags.HasError() {
		return diags
	}

	req := buildProvisionRequest(data, orgID)

	resp, err := client.ExternalConnectionsAPIProvisionCastPrincipalsWithResponse(ctx, orgID, req)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.Errorf("updating cast principals: %v", e)
	}

	if resp.JSON200 == nil {
		return diag.Errorf("updating cast principals: empty response body")
	}

	data.SetId(buildPrincipalsID(orgID, data))

	return setPrincipalsState(data, resp.JSON200)
}

func resourceExternalConnectionPrincipalsDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Delete is a no-op: the ExternalConnectionsAPI exposes no endpoint to remove
	// Cast-side IAM principals. They persist in CAST AI and are only removed from
	// Terraform state. Emit a warning so users are aware.
	return diag.Diagnostics{
		{
			Severity: diag.Warning,
			Summary:  "Cast-side IAM principals are not deleted",
			Detail:   "The ExternalConnectionsAPI does not provide a delete endpoint for Cast-side IAM principals. The principals provisioned by this resource persist in CAST AI and are only removed from Terraform state. To clean up the actual cloud resources, use the CAST AI console or castctl.",
		},
	}
}

func getExtConnPrincipalsOrgID(ctx context.Context, data *schema.ResourceData, meta interface{}) (string, diag.Diagnostics) {
	if v, ok := data.GetOk(FieldExtConnPrincipalsOrganizationID); ok {
		return v.(string), nil
	}
	id, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return "", diag.Errorf("resolving organization id: %v", err)
	}
	if err := data.Set(FieldExtConnPrincipalsOrganizationID, id); err != nil {
		return "", diag.Errorf("setting organization_id: %v", err)
	}
	return id, nil
}

func buildProvisionRequest(data *schema.ResourceData, orgID string) external_connections.ProvisionCastPrincipalsRequest {
	req := external_connections.ProvisionCastPrincipalsRequest{
		CloudProvider:    external_connections.ProvisionCastPrincipalsRequestCloudProvider(data.Get(FieldExtConnPrincipalsCloudProvider).(string)),
		ConnectionScope:  external_connections.ProvisionCastPrincipalsRequestConnectionScope(data.Get(FieldExtConnPrincipalsConnectionScope).(string)),
		Features:         buildPrincipalsFeatureSelections(data),
		OrganizationId:   orgID,
	}

	if sk, ok := data.GetOk(FieldExtConnPrincipalsScopeKey); ok {
		s := sk.(string)
		if s != "" {
			req.ScopeKey = toPtr(s)
		}
	}

	return req
}

func buildPrincipalsFeatureSelections(data *schema.ResourceData) []external_connections.FeatureSelection {
	features := data.Get(FieldExtConnPrincipalsFeatures).([]any)
	out := make([]external_connections.FeatureSelection, 0, len(features))
	for _, f := range features {
		m := f.(map[string]any)
		fs := external_connections.FeatureSelection{
			Feature: external_connections.FeatureSelectionFeature(m[FieldExtConnPrincipalsFeatFeature].(string)),
		}
		if rv, ok := m[FieldExtConnPrincipalsFeatRegistryVersion].(string); ok && rv != "" {
			fs.RegistryVersion = toPtr(rv)
		}
		if sfs, ok := m[FieldExtConnPrincipalsFeatSubFeatures].([]any); ok && len(sfs) > 0 {
			subs := make([]external_connections.FeatureSelectionSubFeatures, 0, len(sfs))
			for _, sf := range sfs {
				if s, ok := sf.(string); ok && s != "" {
					subs = append(subs, external_connections.FeatureSelectionSubFeatures(s))
				}
			}
			if len(subs) > 0 {
				fs.SubFeatures = &subs
			}
		}
		out = append(out, fs)
	}
	return out
}

// buildPrincipalsID derives a deterministic resource ID from the inputs since
// ProvisionCastPrincipals returns no id. Format: orgID-cloudProvider-scopeKey
// (or orgID-cloudProvider when scope_key is unset).
func buildPrincipalsID(orgID string, data *schema.ResourceData) string {
	cloud := data.Get(FieldExtConnPrincipalsCloudProvider).(string)
	scopeKey := data.Get(FieldExtConnPrincipalsScopeKey).(string)
	if scopeKey == "" {
		return fmt.Sprintf("%s-%s", orgID, cloud)
	}
	return fmt.Sprintf("%s-%s-%s", orgID, cloud, scopeKey)
}

func setPrincipalsState(data *schema.ResourceData, resp *external_connections.ProvisionCastPrincipalsResponse) diag.Diagnostics {
	if err := data.Set(FieldExtConnPrincipalsResourceSuffix, resp.ResourceSuffix); err != nil {
		return diag.Errorf("setting resource_suffix: %v", err)
	}

	if err := data.Set(FieldExtConnPrincipalsProvisionedRes, flattenProvisionedResources(resp.ProvisionedResources)); err != nil {
		return diag.Errorf("setting provisioned_resources: %v", err)
	}

	if err := data.Set(FieldExtConnPrincipalsPermissions, flattenPrincipalsPermissions(resp.Permissions)); err != nil {
		return diag.Errorf("setting permissions: %v", err)
	}

	connID := ""
	if resp.ConnectionId != nil {
		connID = *resp.ConnectionId
	}
	if err := data.Set(FieldExtConnPrincipalsConnectionID, connID); err != nil {
		return diag.Errorf("setting connection_id: %v", err)
	}

	return nil
}

func flattenProvisionedResources(resources []external_connections.ProvisionedResource) []map[string]any {
	out := make([]map[string]any, 0, len(resources))
	for _, r := range resources {
		m := map[string]any{
			FieldExtConnPrincipalsPRFeature: string(r.Feature),
		}
		if r.GcpServiceAccount != nil {
			m[FieldExtConnPrincipalsPRGcpSAEmail] = r.GcpServiceAccount.Email
			m[FieldExtConnPrincipalsPRGcpSAID] = r.GcpServiceAccount.Id
		}
		out = append(out, m)
	}
	return out
}

func flattenPrincipalsPermissions(perms []external_connections.Permission) []map[string]any {
	out := make([]map[string]any, 0, len(perms))
	for _, p := range perms {
		m := map[string]any{
			FieldExtConnPrincipalsPermName:         p.Name,
			FieldExtConnPrincipalsPermAccessType:   string(p.AccessType),
			FieldExtConnPrincipalsPermResourceType: string(p.ResourceType),
			FieldExtConnPrincipalsPermScope:        string(p.Scope),
			FieldExtConnPrincipalsPermFeatureID:    p.FeatureId,
			FieldExtConnPrincipalsPermJustification: p.Justification,
		}
		if p.SubFeatureId != nil {
			m[FieldExtConnPrincipalsPermSubFeatureID] = *p.SubFeatureId
		}
		if p.ResourceId != nil {
			m[FieldExtConnPrincipalsPermResourceID] = *p.ResourceId
		}
		out = append(out, m)
	}
	return out
}
