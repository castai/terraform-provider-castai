package castai

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/ai_optimizer"
)

const (
	fieldAIModelRegistryBucket       = "bucket"
	fieldAIModelRegistryRegion       = "region"
	fieldAIModelRegistryPrefix       = "prefix"
	fieldAIModelRegistryCredentials  = "credentials"
	fieldAIModelRegistryUserName     = "user_name"
	fieldAIModelRegistryStatus       = "status"
	fieldAIModelRegistryStatusReason = "status_reason"
)

func resourceAIModelRegistry() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIModelRegistryCreate,
		ReadContext:   resourceAIModelRegistryRead,
		DeleteContext: resourceAIModelRegistryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			fieldAIModelRegistryBucket: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "S3 bucket name.",
			},
			fieldAIModelRegistryRegion: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "AWS region of the S3 bucket.",
			},
			fieldAIModelRegistryPrefix: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Path prefix within the bucket.",
			},
			fieldAIModelRegistryCredentials: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Sensitive:   true,
				Description: "JSON-encoded credentials for accessing the S3 bucket.",
			},
			fieldAIModelRegistryUserName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IAM user name created by onboarding.",
			},
			fieldAIModelRegistryStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Registry status.",
			},
			fieldAIModelRegistryStatusReason: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Reason for the current status.",
			},
		},
	}
}

func resourceAIModelRegistryCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	s3Config := ai_optimizer.ProviderS3Config{
		Bucket: d.Get(fieldAIModelRegistryBucket).(string),
		Region: d.Get(fieldAIModelRegistryRegion).(string),
	}
	if v, ok := d.GetOk(fieldAIModelRegistryPrefix); ok {
		prefix := v.(string)
		s3Config.Prefix = &prefix
	}

	credentials := d.Get(fieldAIModelRegistryCredentials).(string)
	body := ai_optimizer.ModelRegistry{
		Credentials: &credentials,
		Provider: ai_optimizer.Provider{
			Type: "S3",
			S3:   &s3Config,
		},
	}

	tflog.Debug(ctx, "Creating AI model registry", map[string]any{"bucket": s3Config.Bucket})

	resp, err := client.ModelRegistriesAPICreateModelRegistryWithResponse(ctx, orgID, body)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("creating model registry: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from create model registry"))
	}

	d.SetId(*resp.JSON200.Id)

	return resourceAIModelRegistryRead(ctx, d, meta)
}

func resourceAIModelRegistryRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	resp, err := client.ModelRegistriesAPIGetModelRegistryWithResponse(ctx, orgID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "AI model registry not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, nil); err != nil {
		return diag.FromErr(fmt.Errorf("reading model registry: %w", err))
	}

	reg := resp.JSON200
	if reg == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response reading model registry %q", d.Id()))
	}

	if reg.Provider.S3 != nil {
		if err := d.Set(fieldAIModelRegistryBucket, reg.Provider.S3.Bucket); err != nil {
			return diag.FromErr(fmt.Errorf("setting bucket: %w", err))
		}
		if err := d.Set(fieldAIModelRegistryRegion, reg.Provider.S3.Region); err != nil {
			return diag.FromErr(fmt.Errorf("setting region: %w", err))
		}
		if reg.Provider.S3.Prefix != nil {
			if err := d.Set(fieldAIModelRegistryPrefix, *reg.Provider.S3.Prefix); err != nil {
				return diag.FromErr(fmt.Errorf("setting prefix: %w", err))
			}
		}
		if reg.Provider.S3.UserName != nil {
			if err := d.Set(fieldAIModelRegistryUserName, *reg.Provider.S3.UserName); err != nil {
				return diag.FromErr(fmt.Errorf("setting user_name: %w", err))
			}
		}
	}
	if reg.Status != nil {
		if err := d.Set(fieldAIModelRegistryStatus, string(*reg.Status)); err != nil {
			return diag.FromErr(fmt.Errorf("setting status: %w", err))
		}
	}
	if reg.StatusReason != nil {
		if err := d.Set(fieldAIModelRegistryStatusReason, *reg.StatusReason); err != nil {
			return diag.FromErr(fmt.Errorf("setting status_reason: %w", err))
		}
	}

	return nil
}

func resourceAIModelRegistryDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).aiOptimizerClient

	orgID, err := getDefaultOrganizationId(ctx, meta)
	if err != nil {
		return diag.FromErr(fmt.Errorf("fetching organization ID: %w", err))
	}

	tflog.Debug(ctx, "Deleting AI model registry", map[string]any{"id": d.Id()})

	resp, err := client.ModelRegistriesAPIDeleteModelRegistryWithResponse(ctx, orgID, d.Id())
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("deleting model registry: %w", err))
	}

	return nil
}
