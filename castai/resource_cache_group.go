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
	FieldCacheGroupName             = "name"
	FieldCacheGroupProtocolType     = "protocol_type"
	FieldCacheGroupDirectMode       = "direct_mode"
	FieldCacheGroupEndpoints        = "endpoints"
	FieldCacheGroupEndpointHostname = "hostname"
	FieldCacheGroupEndpointPort     = "port"
	FieldCacheGroupEndpointName     = "name"
)

func resourceCacheGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCacheGroupCreate,
		ReadContext:   resourceCacheGroupRead,
		UpdateContext: resourceCacheGroupUpdate,
		DeleteContext: resourceCacheGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Manage CAST AI DBO Cache Group. Cache groups enable query caching for database workloads.",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldCacheGroupProtocolType: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"MySQL", "PostgreSQL"}, false)),
				Description:      "Database protocol type. Valid values: MySQL or PostgreSQL",
			},
			FieldCacheGroupName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Display name for the cache group.",
			},
			FieldCacheGroupDirectMode: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable direct mode for the cache group.",
			},
			FieldCacheGroupEndpoints: {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MinItems:    1,
				Description: "Connection endpoints for the cache group. At least one endpoint is required when specified.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldCacheGroupEndpointHostname: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Database instance hostname.",
						},
						FieldCacheGroupEndpointPort: {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Database instance port.",
						},
						FieldCacheGroupEndpointName: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name for the endpoint.",
						},
					},
				},
			},
		},
	}
}

func resourceCacheGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.DboAPICreateCacheGroupWithResponse(ctx, buildCacheGroupRequest(d))
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.FromErr(fmt.Errorf("cache group ID not returned from API"))
	}

	d.SetId(*resp.JSON200.Id)
	tflog.Info(ctx, "Cache group created", map[string]any{"id": d.Id()})

	return resourceCacheGroupRead(ctx, d, meta)
}

func resourceCacheGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.DboAPIGetCacheGroupWithResponse(ctx, d.Id(), &sdk.DboAPIGetCacheGroupParams{})
	if err != nil {
		return diag.FromErr(err)
	}

	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "Cache group not found, removing from state", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	if resp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("cache group data not returned from API"))
	}

	cacheGroup := resp.JSON200

	if cacheGroup.Name != nil {
		if err := d.Set(FieldCacheGroupName, *cacheGroup.Name); err != nil {
			return diag.FromErr(fmt.Errorf("setting name: %w", err))
		}
	}

	if err := d.Set(FieldCacheGroupProtocolType, string(cacheGroup.ProtocolType)); err != nil {
		return diag.FromErr(fmt.Errorf("setting protocol_type: %w", err))
	}

	if cacheGroup.DirectMode != nil {
		if err := d.Set(FieldCacheGroupDirectMode, *cacheGroup.DirectMode); err != nil {
			return diag.FromErr(fmt.Errorf("setting direct_mode: %w", err))
		}
	}

	if cacheGroup.Endpoints != nil {
		endpoints := flattenEndpoints(*cacheGroup.Endpoints)
		if err := d.Set(FieldCacheGroupEndpoints, endpoints); err != nil {
			return diag.FromErr(fmt.Errorf("setting endpoints: %w", err))
		}
	}

	return nil
}

func resourceCacheGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if d.HasChanges(FieldCacheGroupName, FieldCacheGroupDirectMode, FieldCacheGroupProtocolType, FieldCacheGroupEndpoints) {
		resp, err := client.DboAPIUpdateCacheGroupWithResponse(ctx, d.Id(), buildCacheGroupRequest(d))
		if err := sdk.CheckOKResponse(resp, err); err != nil {
			return diag.FromErr(err)
		}

		tflog.Info(ctx, "Cache group updated", map[string]any{"id": d.Id()})
	}

	return resourceCacheGroupRead(ctx, d, meta)
}

func resourceCacheGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	resp, err := client.DboAPIDeleteCacheGroupWithResponse(ctx, d.Id())
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Cache group deleted", map[string]any{"id": d.Id()})
	return nil
}

func buildCacheGroupRequest(d *schema.ResourceData) sdk.DboV1CacheGroup {
	name := d.Get(FieldCacheGroupName).(string)
	req := sdk.DboV1CacheGroup{
		ProtocolType: sdk.DboV1CacheGroupProtocolType(d.Get(FieldCacheGroupProtocolType).(string)),
		Name:         &name,
	}

	if v, ok := d.GetOk(FieldCacheGroupDirectMode); ok {
		directMode := v.(bool)
		req.DirectMode = &directMode
	}

	if v, ok := d.GetOk(FieldCacheGroupEndpoints); ok {
		endpoints := expandEndpoints(v.([]interface{}))
		if endpoints != nil {
			req.Endpoints = endpoints
		}
	}

	return req
}

func expandEndpoints(endpoints []interface{}) *[]sdk.DboV1Endpoint {
	if len(endpoints) == 0 {
		return nil
	}

	result := make([]sdk.DboV1Endpoint, 0, len(endpoints))
	for _, item := range endpoints {
		endpoint := item.(map[string]any)

		name := endpoint[FieldCacheGroupEndpointName].(string)
		ep := sdk.DboV1Endpoint{
			Hostname: endpoint[FieldCacheGroupEndpointHostname].(string),
			Port:     int32(endpoint[FieldCacheGroupEndpointPort].(int)),
			Suffix:   &name,
		}

		result = append(result, ep)
	}

	return &result
}

func flattenEndpoints(endpoints []sdk.DboV1Endpoint) []interface{} {
	if len(endpoints) == 0 {
		return nil
	}

	result := make([]interface{}, 0, len(endpoints))
	for _, endpoint := range endpoints {
		item := map[string]any{
			FieldCacheGroupEndpointHostname: endpoint.Hostname,
			FieldCacheGroupEndpointPort:     endpoint.Port,
		}

		if endpoint.Suffix != nil {
			item[FieldCacheGroupEndpointName] = *endpoint.Suffix
		}

		result = append(result, item)
	}

	return result
}
