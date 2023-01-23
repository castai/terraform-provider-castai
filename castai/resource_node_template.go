package castai

import (
	"context"
	"fmt"
	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"
	"log"
	"time"
)

const (
	FieldNodeTemplates                 = "template"
	FieldNodeTemplateName              = "name"
	FieldNodeTemplateConfigurationId   = "configuration_id"
	FieldNodeTemplateShouldTaint       = "should_taint"
	FieldNodeTemplateRebalancingConfig = "rebalancing_config"
	FieldNodeTemplateCustomLabel       = "custom_label"
	FieldNodeTemplateConstraints       = "constraints"
)

func resourceNodeTemplate() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceNodeTemplateRead,
		UpdateContext: resourceNodeTemplateUpdate,
		Description:   "CAST AI node template resource to manage autoscaler node templates",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterId: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI cluster id",
			},
			FieldNodeTemplateName: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Name of the node template",
			},
			FieldNodeTemplateConfigurationId: {
				Type:             schema.TypeString,
				Optional:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "CAST AI node configuration id to be used for node template",
			},
			FieldNodeTemplateShouldTaint: {
				Type:        schema.TypeBool,
				Optional:    false,
				Description: "Should taint nodes created from this template",
			},
		},
	}
}

func resourceNodeTemplateRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] List Node Templates get call start")
	defer log.Printf("[INFO] List Node Templates get call end")

	clusterID := getClusterId(data)
	if clusterID == "" {
		log.Print("[INFO] ClusterId is missing. Will skip operation.")
		return nil
	}

	nodeTemplate, err := getNodeTemplateByName(ctx, data, meta, clusterID)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := data.Set(FieldNodeTemplateName, nodeTemplate.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}
	return nil
}
func resourceNodeTemplateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChanges(
		FieldNodeTemplateName,
		FieldNodeTemplateConfigurationId,
		FieldNodeTemplateShouldTaint,
	) {
		log.Printf("[INFO] Nothing to update in node template")
		return nil
	}
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	name := d.Get(FieldNodeTemplateName).(string)

	req := sdk.NodeTemplatesAPIUpdateNodeTemplateJSONRequestBody{}
	if v, ok := d.GetOk(FieldNodeTemplateConfigurationId); ok {
		req.ConfigurationId = toPtr(v.(string))
	}

	if v, ok := d.GetOk(FieldNodeTemplateShouldTaint); ok {
		req.ShouldTaint = toPtr(v.(bool))
	}

	resp, err := client.NodeTemplatesAPIUpdateNodeTemplateWithResponse(ctx, clusterID, name, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return resourceNodeTemplateRead(ctx, d, meta)
}

func getNodeTemplateByName(ctx context.Context, data *schema.ResourceData, meta interface{}, clusterID sdk.ClusterId) (*sdk.NodetemplatesV1NodeTemplate, error) {
	client := meta.(*ProviderConfig).api
	nodeTemplateName := data.Get("name").(string)

	log.Printf("[INFO] Getting current node templates")
	resp, err := client.NodeTemplatesAPIListNodeTemplatesWithResponse(ctx, clusterID)
	notFound := fmt.Errorf("cluster %s node templates not found at CAST AI", clusterID)
	if err != nil {
		return nil, err
	}

	templates := resp.JSON200

	if templates == nil {
		return nil, notFound
	}

	if err != nil {
		log.Printf("[WARN] Getting current node template: %v", err)
		return nil, fmt.Errorf("failed to get current node template from API: %v", err)
	}

	t, ok := lo.Find[sdk.NodetemplatesV1NodeTemplateListItem](lo.FromPtr(templates.Items), func(t sdk.NodetemplatesV1NodeTemplateListItem) bool {
		return t.Template.Name == lo.ToPtr(nodeTemplateName)
	})

	if !ok {
		return nil, fmt.Errorf("failed to find node template with name: %v", nodeTemplateName)
	}

	if err != nil {
		log.Printf("[WARN] Failed merging node template changes: %v", err)
		return nil, fmt.Errorf("failed to merge node template changes: %v", err)
	}

	return t.Template, nil
}
