package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni_provisioner"
)

const (
	FieldOmniEdgeConfigurationOrganizationID  = "organization_id"
	FieldOmniEdgeConfigurationClusterID       = "cluster_id"
	FieldOmniEdgeConfigurationEdgeLocationID  = "edge_location_id"
	FieldOmniEdgeConfigurationName            = "name"
	FieldOmniEdgeConfigurationVersion         = "version"
	FieldOmniEdgeConfigurationDefault         = "default"
	FieldOmniEdgeConfigurationEdgeCount       = "edge_count"
	FieldOmniEdgeConfigurationAWS             = "aws"
	FieldOmniEdgeConfigurationGCP             = "gcp"
)

func dataSourceOmniEdgeConfiguration() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOmniEdgeConfigurationRead,

		Schema: map[string]*schema.Schema{
			FieldOmniEdgeConfigurationOrganizationID: {
				Type:        schema.TypeString,
				Description: "Organization ID",
				Required:    true,
			},
			FieldOmniEdgeConfigurationClusterID: {
				Type:        schema.TypeString,
				Description: "Omni cluster ID",
				Required:    true,
			},
			FieldOmniEdgeConfigurationEdgeLocationID: {
				Type:        schema.TypeString,
				Description: "Edge location ID (optional, for filtering)",
				Optional:    true,
			},
			FieldOmniEdgeConfigurationName: {
				Type:        schema.TypeString,
				Description: "Configuration name to filter by",
				Optional:    true,
			},
			FieldOmniEdgeConfigurationVersion: {
				Type:        schema.TypeString,
				Description: "Configuration version",
				Computed:    true,
			},
			FieldOmniEdgeConfigurationDefault: {
				Type:        schema.TypeBool,
				Description: "Whether this is the default configuration",
				Computed:    true,
			},
			FieldOmniEdgeConfigurationEdgeCount: {
				Type:        schema.TypeInt,
				Description: "Number of edges using this configuration",
				Computed:    true,
			},
			FieldOmniEdgeConfigurationAWS: {
				Type:        schema.TypeList,
				Description: "AWS configuration details",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ami_id": {
							Type:        schema.TypeString,
							Description: "AMI ID",
							Computed:    true,
						},
						"instance_profile_arn": {
							Type:        schema.TypeString,
							Description: "Instance profile ARN",
							Computed:    true,
						},
						"key_pair_name": {
							Type:        schema.TypeString,
							Description: "EC2 key pair name",
							Computed:    true,
						},
					},
				},
			},
			FieldOmniEdgeConfigurationGCP: {
				Type:        schema.TypeList,
				Description: "GCP configuration details",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image_project": {
							Type:        schema.TypeString,
							Description: "Image project",
							Computed:    true,
						},
						"image_family": {
							Type:        schema.TypeString,
							Description: "Image family",
							Computed:    true,
						},
						"service_account_email": {
							Type:        schema.TypeString,
							Description: "Service account email",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceOmniEdgeConfigurationRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniProvisionerClient

	organizationID := data.Get(FieldOmniEdgeConfigurationOrganizationID).(string)
	clusterID := data.Get(FieldOmniEdgeConfigurationClusterID).(string)

	resp, err := client.ListEdgeConfigurationsWithResponse(ctx, organizationID, clusterID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("listing edge configurations: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("listing edge configurations: unexpected status code %d", resp.StatusCode()))
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return diag.Errorf("edge configurations response is nil")
	}

	// Filter by name or edge location if provided
	filterName := data.Get(FieldOmniEdgeConfigurationName).(string)
	filterEdgeLocationID := data.Get(FieldOmniEdgeConfigurationEdgeLocationID).(string)

	var selectedConfig *omni_provisioner.CastaiOmniProvisionerV1beta1EdgeConfiguration
	for _, config := range *resp.JSON200.Items {
		if filterName != "" && config.Name != nil && *config.Name != filterName {
			continue
		}
		if filterEdgeLocationID != "" && config.EdgeLocationId != nil && *config.EdgeLocationId != filterEdgeLocationID {
			continue
		}

		selectedConfig = &config
		break
	}

	if selectedConfig == nil {
		return diag.Errorf("no edge configuration found matching the criteria")
	}

	if selectedConfig.Id == nil {
		return diag.Errorf("edge configuration ID is nil")
	}

	data.SetId(*selectedConfig.Id)

	if selectedConfig.Name != nil {
		if err := data.Set(FieldOmniEdgeConfigurationName, *selectedConfig.Name); err != nil {
			return diag.FromErr(fmt.Errorf("setting name: %w", err))
		}
	}

	if selectedConfig.Version != nil {
		if err := data.Set(FieldOmniEdgeConfigurationVersion, *selectedConfig.Version); err != nil {
			return diag.FromErr(fmt.Errorf("setting version: %w", err))
		}
	}

	if selectedConfig.Default != nil {
		if err := data.Set(FieldOmniEdgeConfigurationDefault, *selectedConfig.Default); err != nil {
			return diag.FromErr(fmt.Errorf("setting default: %w", err))
		}
	}

	if selectedConfig.EdgeCount != nil {
		if err := data.Set(FieldOmniEdgeConfigurationEdgeCount, int(*selectedConfig.EdgeCount)); err != nil {
			return diag.FromErr(fmt.Errorf("setting edge_count: %w", err))
		}
	}

	if selectedConfig.EdgeLocationId != nil {
		if err := data.Set(FieldOmniEdgeConfigurationEdgeLocationID, *selectedConfig.EdgeLocationId); err != nil {
			return diag.FromErr(fmt.Errorf("setting edge_location_id: %w", err))
		}
	}

	return nil
}
