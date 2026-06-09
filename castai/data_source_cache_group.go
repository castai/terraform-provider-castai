package castai

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const FieldCacheGroupEndpointConnectionString = "connection_string"

func dataSourceCacheGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCacheGroupRead,
		Description: "Retrieve CAST AI DBO Cache Group by ID.",
		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsUUID),
				Description:      "The unique identifier of the cache group.",
			},
			FieldCacheGroupName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Display name for the cache group.",
			},
			FieldCacheGroupProtocolType: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Database protocol type. Values: MySQL or PostgreSQL.",
			},
			FieldCacheGroupDirectMode: {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Direct mode for the cache group.",
			},
			FieldCacheGroupEndpoints: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Connection endpoints for the cache group.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldCacheGroupEndpointHostname: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Database instance hostname.",
						},
						FieldCacheGroupEndpointPort: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Database instance port.",
						},
						FieldCacheGroupEndpointName: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name for the endpoint.",
						},
						FieldCacheGroupEndpointConnectionString: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Connection string for the endpoint. Only available once DBO is deployed and running on the Kubernetes cluster.",
						},
					},
				},
			},
		},
	}
}

func dataSourceCacheGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	id := d.Get("id").(string)

	resp, err := client.DboAPIGetCacheGroupWithResponse(ctx, id, &sdk.DboAPIGetCacheGroupParams{})
	if err != nil {
		return diag.FromErr(fmt.Errorf("retrieving cache group: %w", err))
	}

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("retrieving cache group: %w", err))
	}

	if resp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("cache group data not returned from API"))
	}

	cacheGroup := resp.JSON200
	d.SetId(id)

	name := ""
	if cacheGroup.Name != nil {
		name = *cacheGroup.Name
	}
	if err := d.Set(FieldCacheGroupName, name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
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
		endpoints := flattenEndpointsDataSource(*cacheGroup.Endpoints)
		if err := d.Set(FieldCacheGroupEndpoints, endpoints); err != nil {
			return diag.FromErr(fmt.Errorf("setting endpoints: %w", err))
		}
	}

	return nil
}

func flattenEndpointsDataSource(endpoints []sdk.DboV1Endpoint) []interface{} {
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

		if endpoint.ConnectionString != nil {
			item[FieldCacheGroupEndpointConnectionString] = *endpoint.ConnectionString
		}

		result = append(result, item)
	}

	return result
}
