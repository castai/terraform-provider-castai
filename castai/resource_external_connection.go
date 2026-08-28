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
	FieldExternalConnectionCloud            = "cloud"
	FieldExternalConnectionScope            = "connection_scope"
	FieldExternalConnectionScopeKey         = "scope_key"
	FieldExternalConnectionResourceSuffix   = "resource_suffix"
	FieldExternalConnectionEnabledFeatures  = "enabled_features"
	FieldExternalConnectionOrganizationID   = "organization_id"
	FieldExternalConnectionMetadata         = "metadata"
	FieldExternalConnectionCreateTime       = "create_time"
	FieldExternalConnectionUpdateTime       = "update_time"

	// enabled_feature block
	FieldExternalConnectionEFFeature          = "feature"
	FieldExternalConnectionEFRegistryVersion  = "registry_version"
	FieldExternalConnectionEFSubFeatures      = "sub_features"

	// metadata block
	FieldExternalConnectionMetadataAws    = "aws"
	FieldExternalConnectionMetadataAzure  = "azure"
	FieldExternalConnectionMetadataGcp    = "gcp"
	FieldExternalConnectionMetadataAwsRoleArn        = "role_arn"
	FieldExternalConnectionMetadataAzureSubscription = "subscription_id"
	FieldExternalConnectionMetadataAzureApps         = "apps"
	FieldExternalConnectionMetadataAzureAppClientID  = "client_id"
	FieldExternalConnectionMetadataAzureAppTenantID  = "tenant_id"
	FieldExternalConnectionMetadataGcpSAEmails       = "service_account_emails"
)

func resourceExternalConnection() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceExternalConnectionCreate,
		ReadContext:   resourceExternalConnectionRead,
		UpdateContext: resourceExternalConnectionUpdate,
		DeleteContext: resourceExternalConnectionDelete,
		Description:   "Manages a cloud-account external connection to CAST AI. A connection represents a customer's cloud account linked to CAST AI with a set of enabled features.",
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldExternalConnectionCloud: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The cloud provider for this connection. Valid values: AWS, AZURE, GCP, ORACLE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS", "AZURE", "GCP", "ORACLE"}, false)),
			},
			FieldExternalConnectionScope: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Whether this connection is at account or organization scope. Valid values: AWS_ACCOUNT, AWS_ORGANIZATION, AZURE_SUBSCRIPTION, GCP_ORGANIZATION, GCP_PROJECT.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS_ACCOUNT", "AWS_ORGANIZATION", "AZURE_SUBSCRIPTION", "GCP_ORGANIZATION", "GCP_PROJECT"}, false)),
			},
			FieldExternalConnectionScopeKey: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Unique identifier for the scope (e.g. AWS account ID, GCP project ID, Azure subscription ID).",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldExternalConnectionResourceSuffix: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Suffix for generated cloud resources. Typically obtained from the castai_external_connection_principals resource.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldExternalConnectionEnabledFeatures: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Features enabled for this connection with their selected sub-features.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldExternalConnectionEFFeature: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "The feature type. Valid values: CLOUD_CONNECT, COST_MONITORING, NODE_AUTOSCALING, WORKLOAD_AUTOSCALING, KARPENTER_ENTERPRISE, STORAGE_OPTIMIZATION, POD_MUTATIONS, APA, DBO, OMNI, KIMCHI.",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"CLOUD_CONNECT", "COST_MONITORING", "NODE_AUTOSCALING", "WORKLOAD_AUTOSCALING", "KARPENTER_ENTERPRISE", "STORAGE_OPTIMIZATION", "POD_MUTATIONS", "APA", "DBO", "OMNI", "KIMCHI"}, false)),
						},
						FieldExternalConnectionEFRegistryVersion: {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Registry version to pin for this feature. When set, the connection is stamped with that version's permissions snapshot rather than the latest. When unset, the latest version is used.",
						},
						FieldExternalConnectionEFSubFeatures: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Sub-features to include. Empty means base permissions only (no sub-features).",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			FieldExternalConnectionOrganizationID: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The organization ID to create/update the connection for. If not provided, will attempt to infer it using the CAST AI API client.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldExternalConnectionMetadata: {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Cloud-specific metadata for the connection. Includes the customer-side service accounts, Azure Entra app registrations, and AWS role ARN. Sent after provisioning and establishing trust on the customer side.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldExternalConnectionMetadataAws: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "AWS-specific metadata.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldExternalConnectionMetadataAwsRoleArn: {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The IAM role ARN in the customer's AWS account that Cast assumes.",
									},
								},
							},
						},
						FieldExternalConnectionMetadataAzure: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "Azure-specific metadata.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldExternalConnectionMetadataAzureSubscription: {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The Azure subscription scoped to this connection.",
									},
									FieldExternalConnectionMetadataAzureApps: {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Per-feature Azure Entra app registration details.",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"feature": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Feature ID this app registration belongs to.",
												},
												"client_id": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "The client (application) ID of the Entra app registration.",
												},
												"tenant_id": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "The tenant ID of the Entra app registration.",
												},
											},
										},
									},
								},
							},
						},
						FieldExternalConnectionMetadataGcp: {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "GCP-specific metadata.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldExternalConnectionMetadataGcpSAEmails: {
										Type:        schema.TypeMap,
										Optional:    true,
										Description: "Per-feature customer service account emails the customer created in their GCP project. Keyed by feature ID.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
			FieldExternalConnectionCreateTime: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the connection was created.",
			},
			FieldExternalConnectionUpdateTime: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the connection was last updated.",
			},
		},
	}
}

func resourceExternalConnectionCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).externalConnectionsClient
	orgID, diags := getExternalConnectionOrgID(ctx, data, meta)
	if diags.HasError() {
		return diags
	}

	req := buildUpsertRequest(data, orgID)

	resp, err := client.ExternalConnectionsAPIUpsertConnectionWithResponse(ctx, orgID, req)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.Errorf("creating external connection: %v", e)
	}

	if resp.JSON200 == nil {
		return diag.Errorf("creating external connection: empty response body")
	}

	conn := resp.JSON200.Connection
	data.SetId(conn.Id)

	return resourceExternalConnectionRead(ctx, data, meta)
}

func resourceExternalConnectionRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if data.Id() == "" {
		return nil
	}

	client := meta.(*ProviderConfig).externalConnectionsClient
	orgID, diags := getExternalConnectionOrgID(ctx, data, meta)
	if diags.HasError() {
		return diags
	}

	resp, err := client.ExternalConnectionsAPIGetConnectionWithResponse(ctx, orgID, data.Id())
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.Errorf("retrieving external connection: %v", e)
	}

	if resp.JSON200 == nil {
		data.SetId("")
		return nil
	}

	conn := resp.JSON200
	if err := setExternalConnectionState(data, conn); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceExternalConnectionUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).externalConnectionsClient
	orgID, diags := getExternalConnectionOrgID(ctx, data, meta)
	if diags.HasError() {
		return diags
	}

	req := buildUpsertRequest(data, orgID)

	resp, err := client.ExternalConnectionsAPIUpsertConnectionWithResponse(ctx, orgID, req)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.Errorf("updating external connection: %v", e)
	}

	if resp.JSON200 != nil {
		data.SetId(resp.JSON200.Connection.Id)
	}

	return resourceExternalConnectionRead(ctx, data, meta)
}

func resourceExternalConnectionDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if data.Id() == "" {
		return nil
	}

	client := meta.(*ProviderConfig).externalConnectionsClient
	orgID, diags := getExternalConnectionOrgID(ctx, data, meta)
	if diags.HasError() {
		return diags
	}

	resp, err := client.ExternalConnectionsAPIDeleteConnectionWithResponse(ctx, orgID, data.Id())
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.Errorf("deleting external connection: %v", e)
	}

	return nil
}

// getExternalConnectionOrgID resolves the organization ID from the resource schema or the provider config.
func getExternalConnectionOrgID(ctx context.Context, data *schema.ResourceData, meta interface{}) (string, diag.Diagnostics) {
	if v, ok := data.GetOk(FieldExternalConnectionOrganizationID); ok {
		return v.(string), nil
	}
	id, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return "", diag.Errorf("resolving organization id: %v", err)
	}
	if err := data.Set(FieldExternalConnectionOrganizationID, id); err != nil {
		return "", diag.Errorf("setting organization_id: %v", err)
	}
	return id, nil
}

func buildUpsertRequest(data *schema.ResourceData, orgID string) external_connections.UpsertConnectionRequest {
	req := external_connections.UpsertConnectionRequest{
		Cloud:           external_connections.UpsertConnectionRequestCloud(data.Get(FieldExternalConnectionCloud).(string)),
		ConnectionScope: external_connections.UpsertConnectionRequestConnectionScope(data.Get(FieldExternalConnectionScope).(string)),
		ScopeKey:        data.Get(FieldExternalConnectionScopeKey).(string),
		ResourceSuffix:  data.Get(FieldExternalConnectionResourceSuffix).(string),
		EnabledFeatures: buildFeatureSelections(data),
		OrganizationId:  orgID,
	}

	if v, ok := data.GetOk(FieldExternalConnectionMetadata); ok && len(v.([]any)) > 0 {
		req.Metadata = buildConnectionMetadata(v.([]any)[0].(map[string]any))
	}

	return req
}

func buildFeatureSelections(data *schema.ResourceData) []external_connections.FeatureSelection {
	features := data.Get(FieldExternalConnectionEnabledFeatures).([]any)
	out := make([]external_connections.FeatureSelection, 0, len(features))
	for _, f := range features {
		m := f.(map[string]any)
		fs := external_connections.FeatureSelection{
			Feature: external_connections.FeatureSelectionFeature(m[FieldExternalConnectionEFFeature].(string)),
		}
		if rv, ok := m[FieldExternalConnectionEFRegistryVersion].(string); ok && rv != "" {
			fs.RegistryVersion = toPtr(rv)
		}
		if sfs, ok := m[FieldExternalConnectionEFSubFeatures].([]any); ok && len(sfs) > 0 {
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

func buildConnectionMetadata(m map[string]any) *external_connections.ConnectionMetadata {
	md := &external_connections.ConnectionMetadata{}
	if awsRaw, ok := m[FieldExternalConnectionMetadataAws].([]any); ok && len(awsRaw) > 0 {
		awsMap := awsRaw[0].(map[string]any)
		awsMd := &external_connections.AWSConnectionMetadata{}
		if v, ok := awsMap[FieldExternalConnectionMetadataAwsRoleArn].(string); ok && v != "" {
			awsMd.RoleArn = toPtr(v)
		}
		md.Aws = awsMd
	}
	if azureRaw, ok := m[FieldExternalConnectionMetadataAzure].([]any); ok && len(azureRaw) > 0 {
		azureMap := azureRaw[0].(map[string]any)
		azureMd := &external_connections.AzureConnectionMetadata{}
		if v, ok := azureMap[FieldExternalConnectionMetadataAzureSubscription].(string); ok && v != "" {
			azureMd.SubscriptionId = toPtr(v)
		}
		if appsRaw, ok := azureMap[FieldExternalConnectionMetadataAzureApps].([]any); ok && len(appsRaw) > 0 {
			appsMap := make(map[string]external_connections.AzureApp, len(appsRaw))
			for _, a := range appsRaw {
				appMap := a.(map[string]any)
				feature := appMap["feature"].(string)
				appsMap[feature] = external_connections.AzureApp{
					ClientId: appMap["client_id"].(string),
					TenantId: appMap["tenant_id"].(string),
				}
			}
			if len(appsMap) > 0 {
				azureMd.Apps = &appsMap
			}
		}
		md.Azure = azureMd
	}
	if gcpRaw, ok := m[FieldExternalConnectionMetadataGcp].([]any); ok && len(gcpRaw) > 0 {
		gcpMap := gcpRaw[0].(map[string]any)
		gcpMd := &external_connections.GCPConnectionMetadata{}
		if sa, ok := gcpMap[FieldExternalConnectionMetadataGcpSAEmails].(map[string]any); ok && len(sa) > 0 {
			saMap := make(map[string]string, len(sa))
			for k, v := range sa {
				if s, ok := v.(string); ok {
					saMap[k] = s
				}
			}
			if len(saMap) > 0 {
				gcpMd.ServiceAccountEmails = &saMap
			}
		}
		md.Gcp = gcpMd
	}
	return md
}

func setExternalConnectionState(data *schema.ResourceData, conn *external_connections.Connection) error {
	if err := data.Set(FieldExternalConnectionCloud, string(conn.Cloud)); err != nil {
		return fmt.Errorf("setting cloud: %w", err)
	}
	if err := data.Set(FieldExternalConnectionScope, string(conn.Scope)); err != nil {
		return fmt.Errorf("setting connection_scope: %w", err)
	}
	if err := data.Set(FieldExternalConnectionScopeKey, conn.ScopeKey); err != nil {
		return fmt.Errorf("setting scope_key: %w", err)
	}
	if err := data.Set(FieldExternalConnectionResourceSuffix, conn.ResourceSuffix); err != nil {
		return fmt.Errorf("setting resource_suffix: %w", err)
	}
	if err := data.Set(FieldExternalConnectionOrganizationID, conn.OrganizationId); err != nil {
		return fmt.Errorf("setting organization_id: %w", err)
	}
	if err := data.Set(FieldExternalConnectionEnabledFeatures, flattenEnabledFeatures(conn.EnabledFeatures)); err != nil {
		return fmt.Errorf("setting enabled_features: %w", err)
	}
	if err := data.Set(FieldExternalConnectionCreateTime, conn.CreateTime.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("setting create_time: %w", err)
	}
	if err := data.Set(FieldExternalConnectionUpdateTime, conn.UpdateTime.Format(time.RFC3339)); err != nil {
		return fmt.Errorf("setting update_time: %w", err)
	}
	return nil
}

func flattenEnabledFeatures(features []external_connections.EnabledFeature) []map[string]any {
	out := make([]map[string]any, 0, len(features))
	for _, f := range features {
		m := map[string]any{
			FieldExternalConnectionEFFeature:         string(f.Feature),
			FieldExternalConnectionEFRegistryVersion: f.RegistryVersion,
		}
		if f.SubFeatures != nil {
			subs := make([]string, 0, len(*f.SubFeatures))
			for _, sf := range *f.SubFeatures {
				subs = append(subs, string(sf))
			}
			m[FieldExternalConnectionEFSubFeatures] = subs
		}
		out = append(out, m)
	}
	return out
}
