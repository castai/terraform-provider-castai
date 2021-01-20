package castai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	ClusterFieldName        = "name"
	ClusterFieldStatus      = "status"
	ClusterFieldRegion      = "region"
	ClusterFieldCredentials = "credentials"
	ClusterFieldKubeconfig  = "kubeconfig"

	ClusterFieldInitializeParams = "initialize_params"
	ClusterFieldNodes            = "nodes"
	ClusterFieldNodesCloud       = "cloud"
	ClusterFieldNodesRole        = "role"
	ClusterFieldNodesShape       = "shape"
)

func resourceCastaiCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCastaiClusterCreateOrUpdate,
		ReadContext:   resourceCastaiClusterRead,
		UpdateContext: resourceCastaiClusterCreateOrUpdate,
		DeleteContext: resourceCastaiClusterDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(45 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			ClusterFieldName: {
				Type:             schema.TypeString,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Required:         true,
				ForceNew:         true,
			},
			ClusterFieldRegion: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			ClusterFieldCredentials: {
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
			},
			ClusterFieldStatus: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			ClusterFieldInitializeParams: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						ClusterFieldNodes: {
							Type:     schema.TypeList,
							MinItems: 1,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									ClusterFieldNodesCloud: {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(sdk.SupportedClouds(), false)),
									},
									ClusterFieldNodesRole: {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"master", "worker"}, false)),
									},
									ClusterFieldNodesShape: {
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"x-small", "small", "medium", "large", "x-large", "2x-large"}, false)),
									},
								},
							},
						},
					},
				},
			},
			ClusterFieldKubeconfig: {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceCastaiClusterCreateOrUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	var nodes []sdk.Node
	for _, val := range data.Get(ClusterFieldInitializeParams + ".0." + ClusterFieldNodes).([]interface{}) {
		nodeData := val.(map[string]interface{})
		nodes = append(nodes, sdk.Node{
			Role:  sdk.NodeType(nodeData[ClusterFieldNodesRole].(string)),
			Cloud: sdk.CloudType(nodeData[ClusterFieldNodesCloud].(string)),
			Shape: sdk.NodeShape(nodeData[ClusterFieldNodesShape].(string)),
		})
	}

	cluster := sdk.CreateNewClusterJSONRequestBody{
		Name:                data.Get(ClusterFieldName).(string),
		Region:              data.Get(ClusterFieldRegion).(string),
		CloudCredentialsIDs: convertStringArr(data.Get(ClusterFieldCredentials).(*schema.Set).List()),
		Nodes:               nodes,
	}

	log.Printf("[INFO] Creating new cluster: %#v", cluster)

	response, err := client.CreateNewClusterWithResponse(ctx, cluster)
	if checkErr := sdk.CheckCreateResponse(response, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	data.SetId(response.JSON201.Id)

	log.Printf("[DEBUG] Waiting for cluster to reach `ready` status, id=%q name=%q", data.Id(), data.Get(ClusterFieldName))
	err = resource.RetryContext(ctx, data.Timeout(schema.TimeoutCreate), waitForClusterToReachCreatedFunc(ctx, client, data.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Cluster %q has reached `ready` status", data.Id())
	return resourceCastaiClusterRead(ctx, data, meta)
}

func resourceCastaiClusterRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	response, err := client.GetClusterWithResponse(ctx, sdk.ClusterId(data.Id()))
	if err != nil {
		return diag.FromErr(err)
	} else if response.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Removing cluster %s from state because it no longer exists in CAST.AI", data.Id())
		data.SetId("")
		return nil
	}

	data.Set(ClusterFieldName, response.JSON200.Name)
	data.Set(ClusterFieldRegion, response.JSON200.Region)
	data.Set(ClusterFieldStatus, response.JSON200.Status)
	data.Set(ClusterFieldCredentials, response.JSON200.CloudCredentialsIDs)

	kubeconfig, err := client.GetClusterKubeconfigWithResponse(ctx, sdk.ClusterId(data.Id()))
	if checkErr := sdk.CheckGetResponse(kubeconfig, err); checkErr == nil {
		data.Set(ClusterFieldKubeconfig, string(kubeconfig.Body))
	} else {
		log.Printf("[WARN] kubeconfig is not available for cluster %q: %v", data.Id(), checkErr)
		data.Set(ClusterFieldKubeconfig, nil)
	}

	return nil
}

func resourceCastaiClusterDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	if err := sdk.CheckDeleteResponse(client.DeleteClusterWithResponse(ctx, sdk.ClusterId(data.Id()))); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Waiting for cluster to reach `deleted` status, id=%q name=%q", data.Id(), data.Get(ClusterFieldName))
	err := resource.RetryContext(ctx, data.Timeout(schema.TimeoutDelete), waitForClusterStatusDeletedFunc(ctx, client, data.Id()))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func waitForClusterToReachCreatedFunc(ctx context.Context, client *sdk.ClientWithResponses, id string) resource.RetryFunc {
	return waitForClusterToReachStatusFunc(ctx, client, id, "ready", []string{"creating", "warning"})
}

func waitForClusterStatusDeletedFunc(ctx context.Context, client *sdk.ClientWithResponses, id string) resource.RetryFunc {
	return waitForClusterToReachStatusFunc(ctx, client, id, "deleted", []string{"deleting", "warning"})
}

func waitForClusterToReachStatusFunc(ctx context.Context, client *sdk.ClientWithResponses, id string, targetStatus string, retryableStatuses []string) resource.RetryFunc {
	return func() *resource.RetryError {
		response, err := client.GetClusterWithResponse(ctx, sdk.ClusterId(id))
		if err != nil || response.JSON200 == nil {
			return resource.NonRetryableError(err)
		}

		cluster := response.JSON200

		if cluster.Status == targetStatus {
			return nil
		}

		for _, retryableStatus := range retryableStatuses {
			if cluster.Status == retryableStatus {
				return resource.RetryableError(fmt.Errorf("waiting for cluster to reach %q status, id=%q name=%q, status=%s", targetStatus, cluster.Id, cluster.Name, cluster.Status))
			}
		}

		return resource.NonRetryableError(fmt.Errorf("cluster has reached unexpected status, id=%q name=%q, status=%s", cluster.Id, cluster.Name, cluster.Status))
	}
}
