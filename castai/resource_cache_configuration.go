package castai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldCacheConfigurationCacheGroupID = "cache_group_id"
	FieldCacheConfigurationDatabaseName = "database_name"
	FieldCacheConfigurationMode         = "mode"
)

func resourceCacheConfiguration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCacheConfigurationCreate,
		ReadContext:   resourceCacheConfigurationRead,
		UpdateContext: resourceCacheConfigurationUpdate,
		DeleteContext: resourceCacheConfigurationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Manage CAST AI DBO Cache Configuration. Cache configurations define caching behavior for specific databases within a cache group.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldCacheConfigurationCacheGroupID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the cache group this configuration belongs to.",
			},
			FieldCacheConfigurationDatabaseName: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Logical database name to cache.",
			},
			FieldCacheConfigurationMode: {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"Auto", "DontCache", "Manual"}, false)),
				Description:      "Caching mode for this database. Valid values: Auto, DontCache, Manual.",
			},
		},
	}
}

func resourceCacheConfigurationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	cacheGroupID := d.Get(FieldCacheConfigurationCacheGroupID).(string)
	databaseName := d.Get(FieldCacheConfigurationDatabaseName).(string)

	// Handle auto-discovered databases by updating existing configuration if present
	listResp, err := client.DboAPIListCacheConfigurationsWithResponse(ctx, cacheGroupID, &sdk.DboAPIListCacheConfigurationsParams{
		DatabaseName: &databaseName,
	})
	if err := sdk.CheckOKResponse(listResp, err); err == nil {
		if listResp.JSON200 != nil && listResp.JSON200.Items != nil && len(*listResp.JSON200.Items) > 0 {
			existingConfig := (*listResp.JSON200.Items)[0]
			if existingConfig.Id != nil {
				tflog.Info(ctx, "Cache configuration already exists, updating instead of creating", map[string]any{
					"id":             *existingConfig.Id,
					"cache_group_id": cacheGroupID,
					"database_name":  databaseName,
				})

				d.SetId(*existingConfig.Id)
				return resourceCacheConfigurationUpdate(ctx, d, meta)
			}
		}
	}

	mode := sdk.DboV1TTLMode(d.Get(FieldCacheConfigurationMode).(string))
	req := sdk.DboAPICreateCacheConfigurationJSONRequestBody{
		DatabaseName: databaseName,
		Mode:         &mode,
	}

	resp, err := client.DboAPICreateCacheConfigurationWithResponse(ctx, cacheGroupID, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.FromErr(fmt.Errorf("cache configuration ID not returned from API"))
	}

	d.SetId(*resp.JSON200.Id)
	tflog.Info(ctx, "Cache configuration created", map[string]any{
		"id":             *resp.JSON200.Id,
		"cache_group_id": cacheGroupID,
		"database_name":  req.DatabaseName,
	})

	return resourceCacheConfigurationRead(ctx, d, meta)
}

func resourceCacheConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	cacheGroupID := d.Get(FieldCacheConfigurationCacheGroupID).(string)
	configID := d.Id()

	resp, err := client.DboAPIListCacheConfigurationsWithResponse(ctx, cacheGroupID, &sdk.DboAPIListCacheConfigurationsParams{})
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "Cache group not found, removing configuration from state", map[string]any{
			"id":             configID,
			"cache_group_id": cacheGroupID,
		})
		d.SetId("")
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return diag.FromErr(fmt.Errorf("cache configuration data not returned from API"))
	}

	var config *sdk.DboV1CacheConfiguration
	for _, cfg := range *resp.JSON200.Items {
		if cfg.Id != nil && *cfg.Id == configID {
			config = &cfg
			break
		}
	}

	if config == nil {
		if !d.IsNewResource() {
			tflog.Warn(ctx, "Cache configuration not found, removing from state", map[string]any{
				"id":             configID,
				"cache_group_id": cacheGroupID,
			})
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("cache configuration with ID %s not found in cache group %s", configID, cacheGroupID))
	}

	if err := d.Set(FieldCacheConfigurationCacheGroupID, cacheGroupID); err != nil {
		return diag.FromErr(fmt.Errorf("setting cache_group_id: %w", err))
	}

	if err := d.Set(FieldCacheConfigurationDatabaseName, config.DatabaseName); err != nil {
		return diag.FromErr(fmt.Errorf("setting database_name: %w", err))
	}

	if config.Mode != nil {
		if err := d.Set(FieldCacheConfigurationMode, string(*config.Mode)); err != nil {
			return diag.FromErr(fmt.Errorf("setting mode: %w", err))
		}
	}

	return nil
}

func resourceCacheConfigurationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	cacheGroupID := d.Get(FieldCacheConfigurationCacheGroupID).(string)
	configID := d.Id()

	mode := sdk.DboV1TTLMode(d.Get(FieldCacheConfigurationMode).(string))
	req := sdk.DboAPIUpdateCacheConfigurationJSONRequestBody{
		DatabaseName: d.Get(FieldCacheConfigurationDatabaseName).(string),
		Mode:         &mode,
	}

	resp, err := client.DboAPIUpdateCacheConfigurationWithResponse(ctx, cacheGroupID, configID, req)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Cache configuration updated", map[string]any{
		"id":             configID,
		"cache_group_id": cacheGroupID,
	})

	return resourceCacheConfigurationRead(ctx, d, meta)
}

func resourceCacheConfigurationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	cacheGroupID := d.Get(FieldCacheConfigurationCacheGroupID).(string)
	configID := d.Id()

	resp, err := client.DboAPIDeleteCacheConfigurationWithResponse(ctx, cacheGroupID, configID)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Cache configuration deleted", map[string]any{
		"id":             configID,
		"cache_group_id": cacheGroupID,
	})
	return nil
}
