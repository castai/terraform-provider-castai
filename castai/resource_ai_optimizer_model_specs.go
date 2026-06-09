package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer"
)

const (
	fieldAIModelSpecsModel           = "model"
	fieldAIModelSpecsRegistryType    = "registry_type"
	fieldAIModelSpecsDescription     = "description"
	fieldAIModelSpecsType            = "type"
	fieldAIModelSpecsRoutable        = "routable"
	fieldAIModelSpecsHuggingFace     = "huggingface"
	fieldAIModelSpecsPrivateRegistry = "private_registry"
	fieldAIModelSpecsHFModelName     = "model_name"
	fieldAIModelSpecsPRBaseModelID   = "base_model_id"
	fieldAIModelSpecsPRRegistryID    = "registry_id"
)

func resourceAIModelSpecs() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIModelSpecsCreate,
		ReadContext:   resourceAIModelSpecsRead,
		DeleteContext: resourceAIModelSpecsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			fieldAIModelSpecsModel: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Model name.",
			},
			fieldAIModelSpecsRegistryType: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Registry type: HUGGING_FACE or PRIVATE.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{
					string(ai_optimizer.ModelSpecsRegistryTypeHUGGINGFACE),
					string(ai_optimizer.ModelSpecsRegistryTypePRIVATE),
				}, false)),
			},
			fieldAIModelSpecsDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Model description.",
			},
			fieldAIModelSpecsType: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Model type (chat, embeddings, completion, etc.).",
			},
			fieldAIModelSpecsRoutable: {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "Whether the model is routable.",
			},
			fieldAIModelSpecsHuggingFace: {
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				MaxItems:      1,
				Description:   "HuggingFace registry configuration. Required when registry_type is HUGGING_FACE.",
				ConflictsWith: []string{fieldAIModelSpecsPrivateRegistry},
				AtLeastOneOf:  []string{fieldAIModelSpecsHuggingFace, fieldAIModelSpecsPrivateRegistry},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						fieldAIModelSpecsHFModelName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "HuggingFace model name (e.g. meta-llama/Llama-3.1-8B-Instruct).",
						},
					},
				},
			},
			fieldAIModelSpecsPrivateRegistry: {
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      true,
				MaxItems:      1,
				Description:   "Private registry configuration. Required when registry_type is PRIVATE.",
				ConflictsWith: []string{fieldAIModelSpecsHuggingFace},
				AtLeastOneOf:  []string{fieldAIModelSpecsHuggingFace, fieldAIModelSpecsPrivateRegistry},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						fieldAIModelSpecsPRBaseModelID: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Base model identifier.",
						},
						fieldAIModelSpecsPRRegistryID: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the castai_ai_optimizer_model_registry resource.",
						},
					},
				},
			},
		},
	}
}

func resourceAIModelSpecsCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	body := ai_optimizer.ModelSpecs{
		Model:        d.Get(fieldAIModelSpecsModel).(string),
		RegistryType: ai_optimizer.ModelSpecsRegistryType(d.Get(fieldAIModelSpecsRegistryType).(string)),
	}

	if v, ok := d.GetOk(fieldAIModelSpecsDescription); ok {
		desc := v.(string)
		body.Description = &desc
	}
	if v, ok := d.GetOk(fieldAIModelSpecsType); ok {
		t := v.(string)
		body.Type = &t
	}
	if v, ok := d.GetOk(fieldAIModelSpecsRoutable); ok {
		r := v.(bool)
		body.Routable = &r
	}
	if v, ok := d.GetOk(fieldAIModelSpecsHuggingFace); ok {
		hfList := v.([]interface{})
		if len(hfList) > 0 {
			hf := hfList[0].(map[string]interface{})
			body.HuggingFace = &ai_optimizer.HuggingFaceModel{
				ModelName: hf[fieldAIModelSpecsHFModelName].(string),
			}
		}
	}
	if v, ok := d.GetOk(fieldAIModelSpecsPrivateRegistry); ok {
		prList := v.([]interface{})
		if len(prList) > 0 {
			pr := prList[0].(map[string]interface{})
			body.PrivateRegistry = &ai_optimizer.PrivateRegistryModel{
				BaseModelId: pr[fieldAIModelSpecsPRBaseModelID].(string),
				RegistryId:  pr[fieldAIModelSpecsPRRegistryID].(string),
			}
		}
	}

	tflog.Debug(ctx, "Creating AI model specs", map[string]any{"model": body.Model})

	resp, err := client.ModelSpecsAPICreateModelSpecsWithResponse(ctx, orgID, body)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("creating model specs: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from create model specs"))
	}

	d.SetId(*resp.JSON200.Id)

	return resourceAIModelSpecsRead(ctx, d, meta)
}

func resourceAIModelSpecsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	resp, err := client.ModelSpecsAPIGetModelSpecsWithResponse(ctx, orgID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "AI model specs not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, nil); err != nil {
		return diag.FromErr(fmt.Errorf("reading model specs: %w", err))
	}

	ms := resp.JSON200

	if err := d.Set(fieldAIModelSpecsModel, ms.Model); err != nil {
		return diag.FromErr(fmt.Errorf("setting model: %w", err))
	}
	if err := d.Set(fieldAIModelSpecsRegistryType, string(ms.RegistryType)); err != nil {
		return diag.FromErr(fmt.Errorf("setting registry_type: %w", err))
	}
	if ms.Description != nil {
		if err := d.Set(fieldAIModelSpecsDescription, *ms.Description); err != nil {
			return diag.FromErr(fmt.Errorf("setting description: %w", err))
		}
	}
	if ms.Type != nil {
		if err := d.Set(fieldAIModelSpecsType, *ms.Type); err != nil {
			return diag.FromErr(fmt.Errorf("setting type: %w", err))
		}
	}
	if ms.Routable != nil {
		if err := d.Set(fieldAIModelSpecsRoutable, *ms.Routable); err != nil {
			return diag.FromErr(fmt.Errorf("setting routable: %w", err))
		}
	}
	if ms.HuggingFace != nil {
		hf := []map[string]interface{}{
			{fieldAIModelSpecsHFModelName: ms.HuggingFace.ModelName},
		}
		if err := d.Set(fieldAIModelSpecsHuggingFace, hf); err != nil {
			return diag.FromErr(fmt.Errorf("setting huggingface: %w", err))
		}
	}
	if ms.PrivateRegistry != nil {
		pr := []map[string]interface{}{
			{
				fieldAIModelSpecsPRBaseModelID: ms.PrivateRegistry.BaseModelId,
				fieldAIModelSpecsPRRegistryID:  ms.PrivateRegistry.RegistryId,
			},
		}
		if err := d.Set(fieldAIModelSpecsPrivateRegistry, pr); err != nil {
			return diag.FromErr(fmt.Errorf("setting private_registry: %w", err))
		}
	}

	return nil
}

func resourceAIModelSpecsDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	tflog.Debug(ctx, "Deleting AI model specs", map[string]any{"id": d.Id()})

	resp, err := client.ModelSpecsAPIDeleteModelSpecsWithResponse(ctx, orgID, d.Id())
	if resp != nil && resp.StatusCode() == http.StatusNotFound {
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("deleting model specs: %w", err))
	}

	return nil
}
