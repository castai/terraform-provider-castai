package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/external_connections"
)

const (
	FieldDSExtConnFeaturesClouds      = "clouds"
	FieldDSExtConnFeaturesItems       = "items"
	FieldDSExtConnFeatureID           = "id"
	FieldDSExtConnFeatureName         = "name"
	FieldDSExtConnFeatureType         = "type"
	FieldDSExtConnFeatureCategory     = "category"
	FieldDSExtConnFeatureDescription  = "description"
	FieldDSExtConnFeatureOwner        = "owner"
	FieldDSExtConnFeatureCurrentVersion = "current_version"
	FieldDSExtConnFeatureBasePermCount = "base_permission_count"
	FieldDSExtConnFeatureSubFeatures   = "sub_features"
)

func dataSourceExternalConnectionFeatures() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceExternalConnectionFeaturesRead,
		Description: "Lists all available features and their sub-features from the CAST AI features catalog. Supports optional filtering by cloud provider.",
		Schema: map[string]*schema.Schema{
			FieldDSExtConnFeaturesClouds: {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Optional cloud-provider filter. When non-empty, the response is filtered both at the feature and sub-feature level: sub-features that don't support any of the listed clouds are dropped, and a feature with no remaining sub-features is dropped entirely. Valid values: AWS, AZURE, GCP, ORACLE.",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"AWS", "AZURE", "GCP", "ORACLE"}, false)),
				},
			},
			FieldDSExtConnFeaturesItems: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available features.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldDSExtConnFeatureID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Stable string identifier (e.g. cloud_connect).",
						},
						FieldDSExtConnFeatureName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Human-readable name of the feature.",
						},
						FieldDSExtConnFeatureType: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Enum identifier mirroring the id.",
						},
						FieldDSExtConnFeatureCategory: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Category at which the feature applies (CLUSTER or ORGANIZATION).",
						},
						FieldDSExtConnFeatureDescription: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Description of what the feature does.",
						},
						FieldDSExtConnFeatureOwner: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The team/owner responsible for the feature.",
						},
						FieldDSExtConnFeatureCurrentVersion: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Current registry version for this feature.",
						},
						FieldDSExtConnFeatureBasePermCount: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Number of permission units defined in the feature's base_permissions block.",
						},
						FieldDSExtConnFeatureSubFeatures: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of sub-features belonging to this feature.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Stable string identifier of the sub-feature.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Human-readable name of the sub-feature.",
									},
									"type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Enum identifier mirroring the id.",
									},
									"description": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "Description of what the sub-feature does.",
									},
									"required": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Whether this sub-feature is required.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceExternalConnectionFeaturesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).externalConnectionsClient

	params := &external_connections.ExternalConnectionsAPIListFeaturesParams{}
	if clouds, ok := d.GetOk(FieldDSExtConnFeaturesClouds); ok {
		cloudList := clouds.([]any)
		if len(cloudList) > 0 {
			filterClouds := make([]external_connections.ExternalConnectionsAPIListFeaturesParamsClouds, 0, len(cloudList))
			for _, c := range cloudList {
				if s, ok := c.(string); ok && s != "" {
					filterClouds = append(filterClouds, external_connections.ExternalConnectionsAPIListFeaturesParamsClouds(s))
				}
			}
			if len(filterClouds) > 0 {
				params.Clouds = &filterClouds
			}
		}
	}

	resp, err := client.ExternalConnectionsAPIListFeaturesWithResponse(ctx, params)
	if e := sdk.CheckOKResponse(resp, err); e != nil {
		return diag.FromErr(fmt.Errorf("listing external connection features: %w", e))
	}

	if resp.JSON200 == nil {
		return diag.Errorf("listing external connection features: empty response body")
	}

	items := make([]map[string]any, 0, len(resp.JSON200.Items))
	for _, f := range resp.JSON200.Items {
		item := map[string]any{
			FieldDSExtConnFeatureID:            f.Id,
			FieldDSExtConnFeatureName:         f.Name,
			FieldDSExtConnFeatureType:         string(f.Type),
			FieldDSExtConnFeatureCategory:     string(f.Category),
			FieldDSExtConnFeatureDescription:  f.Description,
			FieldDSExtConnFeatureOwner:        f.Owner,
			FieldDSExtConnFeatureBasePermCount: int(f.BasePermissionCount),
		}
		if f.CurrentVersion != nil {
			item[FieldDSExtConnFeatureCurrentVersion] = *f.CurrentVersion
		} else {
			item[FieldDSExtConnFeatureCurrentVersion] = ""
		}

		subFeatures := make([]map[string]any, 0, len(f.SubFeatures))
		for _, sf := range f.SubFeatures {
			subFeatures = append(subFeatures, map[string]any{
				"id":          sf.Id,
				"name":        sf.Name,
				"type":        string(sf.Type),
				"description": sf.Description,
				"required":    sf.Required,
			})
		}
		item[FieldDSExtConnFeatureSubFeatures] = subFeatures
		items = append(items, item)
	}

	d.SetId("external_connection_features")
	if err := d.Set(FieldDSExtConnFeaturesItems, items); err != nil {
		return diag.FromErr(fmt.Errorf("setting items: %w", err))
	}

	return nil
}
