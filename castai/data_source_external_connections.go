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
	FieldDSExternalConnectionsOrganizationID = "organization_id"
	FieldDSExternalConnectionsFilterCloud    = "filter_cloud"
	FieldDSExternalConnectionsItems          = "items"

	// connection item fields
	FieldDSExtConnID            = "id"
	FieldDSExtConnCloud         = "cloud"
	FieldDSExtConnScope         = "scope"
	FieldDSExtConnScopeKey      = "scope_key"
	FieldDSExtConnResourceSuffix = "resource_suffix"
	FieldDSExtConnEnabledFeatures = "enabled_features"
	FieldDSExtConnCreateTime     = "create_time"
	FieldDSExtConnUpdateTime     = "update_time"
)

func dataSourceExternalConnections() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceExternalConnectionsRead,
		Description: "Lists all external cloud-account connections for an organization. Supports filtering by cloud provider.",
		Schema: map[string]*schema.Schema{
			FieldDSExternalConnectionsOrganizationID: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "The organization ID to list connections for. If not provided, will attempt to infer it using the CAST AI API client.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
			},
			FieldDSExternalConnectionsFilterCloud: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Filter connections by cloud provider. Valid values: AWS, AZURE, GCP, ORACLE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS", "AZURE", "GCP", "ORACLE"}, false)),
			},
			FieldDSExternalConnectionsItems: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of external connections.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldDSExtConnID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the connection.",
						},
						FieldDSExtConnCloud: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The cloud provider.",
						},
						FieldDSExtConnScope: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The connection scope (account or organization).",
						},
						FieldDSExtConnScopeKey: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Unique identifier for the scope.",
						},
						FieldDSExtConnResourceSuffix: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Suffix for generated cloud resources.",
						},
						FieldDSExtConnEnabledFeatures: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Features enabled for this connection.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"feature": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The feature type.",
									},
									"registry_version": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Registry version of the feature at the time the connection was created.",
									},
								},
							},
						},
						FieldDSExtConnCreateTime: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp when the connection was created.",
						},
						FieldDSExtConnUpdateTime: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Timestamp when the connection was last updated.",
						},
					},
				},
			},
		},
	}
}

func dataSourceExternalConnectionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).externalConnectionsClient

	orgID, diags := getDataSourceOrgID(ctx, d, meta, FieldDSExternalConnectionsOrganizationID)
	if diags.HasError() {
		return diags
	}

	params := &external_connections.ExternalConnectionsAPIListConnectionsParams{}
	if cloud, ok := d.GetOk(FieldDSExternalConnectionsFilterCloud); ok {
		c := external_connections.ExternalConnectionsAPIListConnectionsParamsFilterCloud(cloud.(string))
		params.FilterCloud = &c
	}

	resp, err := client.ExternalConnectionsAPIListConnectionsWithResponse(ctx, orgID, params)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.FromErr(fmt.Errorf("listing external connections: %w", e))
	}

	if resp.JSON200 == nil {
		return diag.Errorf("listing external connections: empty response body")
	}

	items := make([]map[string]any, 0, len(resp.JSON200.Items))
	for _, conn := range resp.JSON200.Items {
		item := map[string]any{
			FieldDSExtConnID:             conn.Id,
			FieldDSExtConnCloud:          string(conn.Cloud),
			FieldDSExtConnScope:          string(conn.Scope),
			FieldDSExtConnScopeKey:       conn.ScopeKey,
			FieldDSExtConnResourceSuffix: conn.ResourceSuffix,
			FieldDSExtConnCreateTime:     conn.CreateTime.Format(time.RFC3339),
			FieldDSExtConnUpdateTime:     conn.UpdateTime.Format(time.RFC3339),
		}

		features := make([]map[string]any, 0, len(conn.EnabledFeatures))
		for _, ef := range conn.EnabledFeatures {
			features = append(features, map[string]any{
				"feature":          string(ef.Feature),
				"registry_version": ef.RegistryVersion,
			})
		}
		item[FieldDSExtConnEnabledFeatures] = features
		items = append(items, item)
	}

	d.SetId(orgID)
	if err := d.Set(FieldDSExternalConnectionsItems, items); err != nil {
		return diag.FromErr(fmt.Errorf("setting items: %w", err))
	}

	return nil
}

// getDataSourceOrgID resolves the organization ID from the data source schema or the provider config.
func getDataSourceOrgID(ctx context.Context, d *schema.ResourceData, meta interface{}, fieldName string) (string, diag.Diagnostics) {
	if v, ok := d.GetOk(fieldName); ok {
		return v.(string), nil
	}
	id, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return "", diag.Errorf("resolving organization id: %v", err)
	}
	if err := d.Set(fieldName, id); err != nil {
		return "", diag.Errorf("setting organization_id: %v", err)
	}
	return id, nil
}
