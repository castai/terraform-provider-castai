package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer"
)

const (
	FieldAIOptimizerAPIKeyOrganizationID = "organization_id"
	FieldAIOptimizerAPIKeyName           = "name"
	FieldAIOptimizerAPIKeyToken          = "token"
)

func resourceAIOptimizerAPIKey() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIOptimizerAPIKeyCreate,
		ReadContext:   resourceAIOptimizerAPIKeyRead,
		DeleteContext: resourceAIOptimizerAPIKeyDelete,
		Description:   "CAST AI AI Optimizer API Key resource to manage AI Optimizer API keys",

		Schema: map[string]*schema.Schema{
			FieldAIOptimizerAPIKeyOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI organization ID.",
			},
			FieldAIOptimizerAPIKeyName: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the API key.",
			},
			FieldAIOptimizerAPIKeyToken: {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The generated API key token.",
			},
		},
	}
}

func resourceAIOptimizerAPIKeyCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient
	organizationID := data.Get(FieldAIOptimizerAPIKeyOrganizationID).(string)
	name := data.Get(FieldAIOptimizerAPIKeyName).(string)

	// The request body is just an APIKey with optional token field
	// Since we're creating a new key, we don't provide a token
	req := ai_optimizer.APIKeysAPICreateAPIKeyJSONRequestBody{}

	resp, err := client.APIKeysAPICreateAPIKeyWithResponse(ctx, organizationID, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("creating AI Optimizer API key: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Token == nil {
		return diag.FromErr(fmt.Errorf("unexpected response when creating AI Optimizer API key"))
	}

	// Store the token
	if err := data.Set(FieldAIOptimizerAPIKeyToken, *resp.JSON200.Token); err != nil {
		return diag.FromErr(fmt.Errorf("setting token: %w", err))
	}

	// Use name as ID since API doesn't return a separate ID
	data.SetId(name)

	return resourceAIOptimizerAPIKeyRead(ctx, data, meta)
}

func resourceAIOptimizerAPIKeyRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// The AI Optimizer API doesn't provide a way to list or read API keys after creation
	// So we just verify the resource still exists in state
	return nil
}

func resourceAIOptimizerAPIKeyDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// The AI Optimizer API doesn't provide a delete endpoint for API keys
	// Keys can only be deleted through the CAST AI console
	// We'll just remove it from state
	data.SetId("")
	return nil
}
