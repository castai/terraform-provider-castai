package castai

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	castval "github.com/castai/terraform-provider-castai/castai/validation"
)

const (
	FieldNodeConfigurationName         = "name"
	FieldNodeConfigurationDiskCpuRatio = "disk_cpu_ratio"
	FieldNodeConfigurationSubnets      = "subnets"
	FieldNodeConfigurationSSHPublicKey = "ssh_public_key"
	FieldNodeConfigurationImage        = "image"
	FieldNodeConfigurationTags         = "tags"
	FieldNodeConfigurationAKS          = "aks"
	FieldNodeConfigurationEKS          = "eks"
	FieldNodeConfigurationKOPS         = "kops"
)

func resourceNodeConfiguration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNodeConfigurationCreate,
		ReadContext:   resourceNodeConfigurationRead,
		UpdateContext: resourceNodeConfigurationUpdate,
		DeleteContext: resourceNodeConfigurationDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldClusterID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI cluster id",
			},
			FieldNodeConfigurationName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Name of the node configuration",
			},
			FieldNodeConfigurationDiskCpuRatio: {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          25,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1)),
				Description:      "Disk to CPU ratio. Sets the number of GiBs to be added for every CPU on the node",
			},
			FieldNodeConfigurationSubnets: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Subnet ids to be used for provisioned nodes",
			},
			FieldNodeConfigurationSSHPublicKey: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "SSH public key to be used for provisioned nodes",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsBase64),
			},
			FieldNodeConfigurationImage: {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "Image to be used while provisioning the node. If nothing is provided will be resolved to latest available image based on Kubernetes version if possible",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
			},
			FieldNodeConfigurationTags: {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Tags to be added on cloud instances for provisioned nodes",
			},
			FieldNodeConfigurationEKS: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"security_groups": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Cluster's security groups configuration for CAST provisioned nodes",
						},
						"dns_cluster_ip": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "IP address to use for DNS queries within the cluster",
							ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPv4Address),
						},
						"instance_profile_arn": {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "Cluster's instance profile ARN used for CAST provisioned nodes",
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
						},
						"key_pair_id": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "AWS key pair ID to be used for provisioned nodes. Has priority over sshPublicKey",
							ValidateDiagFunc: castval.ValidKeyPairFormat(),
						},
					},
				},
			},
			FieldNodeConfigurationAKS: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_pods_per_node": {
							Type:             schema.TypeInt,
							Default:          30,
							Optional:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(10, 250)),
							Description:      "Maximum number of pods that can be run on a node, which affects how many IP addresses you will need for each node. Defaults to 30",
						},
					},
				},
			},
			FieldNodeConfigurationKOPS: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key_pair_id": {
							Type:             schema.TypeString,
							Optional:         true,
							Description:      "AWS key pair ID to be used for provisioned nodes. Has priority over sshPublicKey",
							ValidateDiagFunc: castval.ValidKeyPairFormat(),
						},
					},
				},
			},
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
			return nil
		},
	}
}

func resourceNodeConfigurationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	req := sdk.NodeConfigurationAPICreateConfigurationJSONRequestBody{
		Name:         d.Get(FieldNodeConfigurationName).(string),
		DiskCpuRatio: toPtr(int32(d.Get(FieldNodeConfigurationDiskCpuRatio).(int))),
	}

	if v, ok := d.GetOk(FieldNodeConfigurationSubnets); ok {
		req.Subnets = toPtr(toStringList(v.([]interface{})))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationImage); ok {
		req.Image = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationSSHPublicKey); ok {
		req.SshPublicKey = toPtr(v.(string))
	}
	if v := d.Get(FieldNodeConfigurationTags).(map[string]interface{}); len(v) > 0 {
		req.Tags = &sdk.NodeconfigV1NewNodeConfiguration_Tags{
			AdditionalProperties: toStringMap(v),
		}
	}

	// Map provider specific configurations.
	if v, ok := d.GetOk(FieldNodeConfigurationEKS); ok && len(v.([]interface{})) > 0 {
		req.Eks = toEKSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationKOPS); ok && len(v.([]interface{})) > 0 {
		req.Kops = toKOPSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationAKS); ok && len(v.([]interface{})) > 0 {
		req.Aks = toAKSSConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	resp, err := client.NodeConfigurationAPICreateConfigurationWithResponse(ctx, clusterID, req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	d.SetId(*resp.JSON200.Id)

	return resourceNodeConfigurationRead(ctx, d, meta)
}

func resourceNodeConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	clusterID := d.Get(FieldClusterID).(string)
	resp, err := client.NodeConfigurationAPIGetConfigurationWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	if !d.IsNewResource() && resp.StatusCode() == http.StatusNotFound {
		log.Printf("[WARN] Node configuration (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(err)
	}

	nodeConfig := resp.JSON200

	d.Set(FieldNodeConfigurationName, nodeConfig.Name)
	d.Set(FieldNodeConfigurationDiskCpuRatio, nodeConfig.DiskCpuRatio)
	d.Set(FieldNodeConfigurationSubnets, nodeConfig.Subnets)
	d.Set(FieldNodeConfigurationSSHPublicKey, nodeConfig.SshPublicKey)
	d.Set(FieldNodeConfigurationImage, nodeConfig.Image)
	d.Set(FieldNodeConfigurationTags, nodeConfig.Tags.AdditionalProperties)

	if err := d.Set(FieldNodeConfigurationEKS, flattenEKSConfig(nodeConfig.Eks)); err != nil {
		return diag.Errorf("error setting eks config: %v", err)
	}
	if err := d.Set(FieldNodeConfigurationKOPS, flattenKOPSConfig(nodeConfig.Kops)); err != nil {
		return diag.Errorf("error setting kops config: %v", err)
	}
	if err := d.Set(FieldNodeConfigurationAKS, flattenAKSConfig(nodeConfig.Aks)); err != nil {
		return diag.Errorf("error setting aks config: %v", err)
	}

	return nil
}

func resourceNodeConfigurationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChanges(
		FieldNodeConfigurationDiskCpuRatio,
		FieldNodeConfigurationSubnets,
		FieldNodeConfigurationSSHPublicKey,
		FieldNodeConfigurationImage,
		FieldNodeConfigurationTags,
		FieldNodeConfigurationAKS,
		FieldNodeConfigurationEKS,
		FieldNodeConfigurationKOPS,
	) {
		log.Printf("[INFO] Nothing to update in node configuration")
		return nil
	}

	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)
	req := sdk.NodeConfigurationAPIUpdateConfigurationJSONRequestBody{
		DiskCpuRatio: toPtr(int32(d.Get(FieldNodeConfigurationDiskCpuRatio).(int))),
	}

	if v, ok := d.GetOk(FieldNodeConfigurationSubnets); ok {
		req.Subnets = toPtr(toStringList(v.([]interface{})))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationImage); ok {
		req.Image = toPtr(v.(string))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationSSHPublicKey); ok {
		req.SshPublicKey = toPtr(v.(string))
	}
	if v := d.Get(FieldNodeConfigurationTags).(map[string]interface{}); len(v) > 0 {
		req.Tags = &sdk.NodeconfigV1NodeConfigurationUpdate_Tags{
			AdditionalProperties: toStringMap(v),
		}
	}

	// Map provider specific configurations.
	if v, ok := d.GetOk(FieldNodeConfigurationEKS); ok && len(v.([]interface{})) > 0 {
		req.Eks = toEKSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationKOPS); ok && len(v.([]interface{})) > 0 {
		req.Kops = toKOPSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldNodeConfigurationAKS); ok && len(v.([]interface{})) > 0 {
		req.Aks = toAKSSConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	resp, err := client.NodeConfigurationAPIUpdateConfigurationWithResponse(ctx, clusterID, d.Id(), req)
	if checkErr := sdk.CheckOKResponse(resp, err); checkErr != nil {
		return diag.FromErr(checkErr)
	}

	return resourceNodeConfigurationRead(ctx, d, meta)
}

func resourceNodeConfigurationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api
	clusterID := d.Get(FieldClusterID).(string)

	resp, err := client.NodeConfigurationAPIGetConfigurationWithResponse(ctx, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[DEBUG] Node configuration (%s) not found, skipping delete", d.Id())
		return nil
	}

	if err := sdk.StatusOk(resp); err != nil {
		return diag.FromErr(err)
	}

	if *resp.JSON200.Default {
		log.Printf("[WARN] Default node configuration (%s) can't be deleted, removing from state", d.Id())
		return nil
	}

	del, err := client.NodeConfigurationAPIDeleteConfigurationWithResponse(ctx, clusterID, d.Id())
	if err := sdk.CheckOKResponse(del, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func toEKSConfig(obj map[string]interface{}) *sdk.NodeconfigV1EKSConfig {
	if obj == nil {
		return nil
	}

	out := &sdk.NodeconfigV1EKSConfig{}
	if v, ok := obj["dns_cluster_ip"].(string); ok && v != "" {
		out.DnsClusterIp = toPtr(v)
	}
	if v, ok := obj["instance_profile_arn"].(string); ok {
		out.InstanceProfileArn = v
	}
	if v, ok := obj["key_pair_id"].(string); ok && v != "" {
		out.KeyPairId = toPtr(v)
	}
	if v, ok := obj["security_groups"].([]interface{}); ok && len(v) > 0 {
		out.SecurityGroups = toPtr(toStringList(v))
	}

	return out
}

func flattenEKSConfig(config *sdk.NodeconfigV1EKSConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}

	m := map[string]interface{}{
		"instance_profile_arn": config.InstanceProfileArn,
	}
	if v := config.KeyPairId; v != nil {
		m["key_paid_id"] = toStringValue(v)
	}
	if v := config.DnsClusterIp; v != nil {
		m["dns_cluster_ip"] = toStringValue(v)
	}
	if v := config.SecurityGroups; v != nil {
		m["security_groups"] = *config.SecurityGroups
	}

	return []map[string]interface{}{m}
}

func toKOPSConfig(obj map[string]interface{}) *sdk.NodeconfigV1KOPSConfig {
	if obj == nil {
		return nil
	}

	out := &sdk.NodeconfigV1KOPSConfig{}
	if v, ok := obj["key_pair_id"].(string); ok && v != "" {
		out.KeyPairId = toPtr(v)
	}

	return out
}

func flattenKOPSConfig(config *sdk.NodeconfigV1KOPSConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}
	m := map[string]interface{}{}
	if v := config.KeyPairId; v != nil {
		m["key_paid_id"] = toStringValue(v)
	}

	return []map[string]interface{}{m}
}

func toAKSSConfig(obj map[string]interface{}) *sdk.NodeconfigV1AKSConfig {
	if obj == nil {
		return nil
	}

	out := &sdk.NodeconfigV1AKSConfig{}
	if v, ok := obj["max_pods_per_node"].(int); ok {
		out.MaxPodsPerNode = toPtr(int32(v))
	}

	return out
}

func flattenAKSConfig(config *sdk.NodeconfigV1AKSConfig) []map[string]interface{} {
	if config == nil {
		return nil
	}
	m := map[string]interface{}{}
	if v := config.MaxPodsPerNode; v != nil {
		m["max_pods_per_node"] = *config.MaxPodsPerNode
	}

	return []map[string]interface{}{m}
}
