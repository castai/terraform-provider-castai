package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer"
)

const (
	FieldAIOptimizerHostedModelsOrganizationID = "organization_id"
	FieldAIOptimizerHostedModelsClusterID      = "cluster_id"
	FieldAIOptimizerHostedModelsModels         = "models"
	FieldAIOptimizerHostedModelsModelID        = "id"
	FieldAIOptimizerHostedModelsModelName      = "name"
	FieldAIOptimizerHostedModelsModelStatus    = "status"
)

func dataSourceAIOptimizerHostedModels() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAIOptimizerHostedModelsRead,
		Description: "Retrieve AI Optimizer hosted models for a cluster",

		Schema: map[string]*schema.Schema{
			FieldAIOptimizerHostedModelsOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "CAST AI organization ID.",
			},
			FieldAIOptimizerHostedModelsClusterID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "CAST AI cluster ID.",
			},
			FieldAIOptimizerHostedModelsModels: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of hosted models",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldAIOptimizerHostedModelsModelID: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Model ID",
						},
						FieldAIOptimizerHostedModelsModelName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Model name",
						},
						FieldAIOptimizerHostedModelsModelStatus: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Model status",
						},
					},
				},
			},
		},
	}
}

func dataSourceAIOptimizerHostedModelsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient
	organizationID := data.Get(FieldAIOptimizerHostedModelsOrganizationID).(string)
	clusterID := data.Get(FieldAIOptimizerHostedModelsClusterID).(string)

	resp, err := client.HostedModelsAPIListHostedModelsWithResponse(ctx, organizationID, clusterID, &ai_optimizer.HostedModelsAPIListHostedModelsParams{})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving AI Optimizer hosted models: %w", err))
	}

	if resp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("unexpected response when retrieving hosted models"))
	}

	models := make([]map[string]interface{}, 0)
	if resp.JSON200.Items != nil {
		for _, model := range resp.JSON200.Items {
			modelMap := map[string]interface{}{
				FieldAIOptimizerHostedModelsModelID:     lo.FromPtr(model.Id),
				FieldAIOptimizerHostedModelsModelName:   lo.FromPtr(model.Model),
				FieldAIOptimizerHostedModelsModelStatus: string(lo.FromPtr(model.Status)),
			}
			models = append(models, modelMap)
		}
	}

	if err := data.Set(FieldAIOptimizerHostedModelsModels, models); err != nil {
		return diag.FromErr(fmt.Errorf("setting models: %w", err))
	}

	data.SetId(fmt.Sprintf("%s/%s", organizationID, clusterID))

	return nil
}
