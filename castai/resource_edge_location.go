package castai

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk/omni"
)

const (
	FieldEdgeLocationOrganizationID = "organization_id"
	FieldEdgeLocationClusterID      = "cluster_id"
	FieldEdgeLocationName           = "name"
	FieldEdgeLocationDescription    = "description"
	FieldEdgeLocationRegion         = "region"
	FieldEdgeLocationZones          = "zones"
	FieldEdgeLocationAWS            = "aws"
	FieldEdgeLocationGCP            = "gcp"
	FieldEdgeLocationOCI            = "oci"
)

func resourceEdgeLocation() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEdgeLocationCreate,
		ReadContext:   resourceEdgeLocationRead,
		UpdateContext: resourceEdgeLocationUpdate,
		DeleteContext: resourceEdgeLocationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceEdgeLocationImport,
		},
		Description: "Manage CAST AI Edge Location for edge computing deployments",

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			FieldEdgeLocationOrganizationID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "CAST AI organization ID",
			},
			FieldEdgeLocationClusterID: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "CAST AI cluster ID",
			},
			FieldEdgeLocationName: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "Name of the edge location. Must be unique and relatively short as it's used for creating service accounts.",
			},
			FieldEdgeLocationDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the edge location",
			},
			"credentials_revision": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hash of credentials used to detect credential changes",
			},
			FieldEdgeLocationRegion: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsNotWhiteSpace),
				Description:      "The region where the edge location is deployed",
			},
			FieldEdgeLocationZones: {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The ID of the zone",
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the zone",
						},
					},
				},
				Description: "List of availability zones for the edge location",
			},
			FieldEdgeLocationAWS: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"account_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "AWS account ID",
						},
						"access_key_id": {
							Type:        schema.TypeString,
							Required:    true,
							WriteOnly:   true,
							Sensitive:   true,
							Description: "AWS access key ID",
						},
						"secret_access_key": {
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							WriteOnly:   true,
							Description: "AWS secret access key",
						},
						"vpc_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "VPC ID to be used in the selected region",
						},
						"security_group_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Security group ID to be used in the selected region",
						},
						"subnet_ids": {
							Type:     schema.TypeMap,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Map of zone names to subnet IDs to be used in the selected region",
						},
						"name_tag": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The value of a 'Name' tag applied to VPC resources",
						},
					},
				},
				ExactlyOneOf: []string{FieldEdgeLocationAWS, FieldEdgeLocationGCP, FieldEdgeLocationOCI},
			},
			FieldEdgeLocationGCP: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"project_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "GCP project ID where edges run",
						},
						"client_service_account_json": {
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							WriteOnly:   true,
							Description: "Base64 encoded service account JSON for provisioning edge resources",
						},
						"network_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the network to be used in the selected region",
						},
						"subnet_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the subnetwork to be used in the selected region",
						},
						"network_tags": {
							Type:     schema.TypeSet,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Description: "Tags applied on the provisioned cloud resources and the firewall rule",
						},
					},
				},
				ExactlyOneOf: []string{FieldEdgeLocationAWS, FieldEdgeLocationGCP, FieldEdgeLocationOCI},
			},
			FieldEdgeLocationOCI: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tenancy_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "OCI tenancy ID of the account",
						},
						"compartment_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "OCI compartment ID of edge location",
						},
						"user_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "User ID used to authenticate OCI",
							WriteOnly:   true,
						},
						"fingerprint": {
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							Description: "API key fingerprint",
						},
						"private_key": {
							WriteOnly:   true,
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							Description: "Base64 encoded API private key",
						},
						"vcn_id": {
							Type:        schema.TypeString,
							Required:    true,
							WriteOnly:   true,
							Description: "OCI virtual cloud network ID",
						},
						"subnet_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "OCI subnet ID of edge location",
						},
					},
				},
				ExactlyOneOf: []string{FieldEdgeLocationAWS, FieldEdgeLocationGCP, FieldEdgeLocationOCI},
			},
		},
	}
}

func resourceEdgeLocationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniClient

	organizationID := d.Get(FieldEdgeLocationOrganizationID).(string)
	clusterID := d.Get(FieldEdgeLocationClusterID).(string)

	req := omni.EdgeLocationsAPICreateEdgeLocationJSONRequestBody{
		Name:   d.Get(FieldEdgeLocationName).(string),
		Region: d.Get(FieldEdgeLocationRegion).(string),
	}

	if v, ok := d.GetOk(FieldEdgeLocationDescription); ok {
		req.Description = toPtr(v.(string))
	}

	if v, ok := d.GetOk(FieldEdgeLocationZones); ok {
		req.Zones = toEdgeLocationZones(v.([]interface{}))
	}

	// Map cloud provider specific configurations.
	if v, ok := d.GetOk(FieldEdgeLocationAWS); ok && len(v.([]interface{})) > 0 {
		req.Aws = toAWSConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldEdgeLocationGCP); ok && len(v.([]interface{})) > 0 {
		req.Gcp = toGCPConfig(v.([]interface{})[0].(map[string]interface{}))
	}
	if v, ok := d.GetOk(FieldEdgeLocationOCI); ok && len(v.([]interface{})) > 0 {
		req.Oci = toOCIConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	resp, err := client.EdgeLocationsAPICreateEdgeLocationWithResponse(ctx, organizationID, clusterID, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("creating edge location: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("unexpected status code creating edge location: %d, body: %s", resp.StatusCode(), string(resp.Body)))
	}

	if resp.JSON200 == nil || resp.JSON200.Id == nil {
		return diag.FromErr(fmt.Errorf("edge location ID not returned in response"))
	}

	d.SetId(*resp.JSON200.Id)

	return resourceEdgeLocationRead(ctx, d, meta)
}

func resourceEdgeLocationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniClient

	organizationID := d.Get(FieldEdgeLocationOrganizationID).(string)
	clusterID := d.Get(FieldEdgeLocationClusterID).(string)

	resp, err := client.EdgeLocationsAPIGetEdgeLocationWithResponse(ctx, organizationID, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("reading edge location: %w", err))
	}

	if resp.StatusCode() == http.StatusNotFound {
		if !d.IsNewResource() {
			log.Printf("[WARN] Edge location (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("edge location not found: %s", d.Id()))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("unexpected status code reading edge location: %d, body: %s", resp.StatusCode(), string(resp.Body)))
	}

	if resp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("edge location response is nil"))
	}

	edgeLocation := resp.JSON200

	if err := d.Set(FieldEdgeLocationName, edgeLocation.Name); err != nil {
		return diag.FromErr(fmt.Errorf("setting name: %w", err))
	}
	if err := d.Set(FieldEdgeLocationDescription, edgeLocation.Description); err != nil {
		return diag.FromErr(fmt.Errorf("setting description: %w", err))
	}
	if err := d.Set(FieldEdgeLocationRegion, edgeLocation.Region); err != nil {
		return diag.FromErr(fmt.Errorf("setting region: %w", err))
	}
	if err := d.Set(FieldEdgeLocationZones, flattenEdgeLocationZones(edgeLocation.Zones)); err != nil {
		return diag.FromErr(fmt.Errorf("setting zones: %w", err))
	}

	// Set cloud provider specific configurations
	// Preserve write-only credentials from existing state
	if err := d.Set(FieldEdgeLocationAWS, flattenAWSConfig(edgeLocation.Aws)); err != nil {
		return diag.Errorf("error setting aws config: %v", err)
	}
	if err := d.Set(FieldEdgeLocationGCP, flattenGCPConfig(edgeLocation.Gcp)); err != nil {
		return diag.Errorf("error setting gcp config: %v", err)
	}
	if err := d.Set(FieldEdgeLocationOCI, flattenOCIConfig(edgeLocation.Oci)); err != nil {
		return diag.Errorf("error setting oci config: %v", err)
	}

	// Compute credentials revision hash
	credentialsHash := computeCredentialsHash(d)
	if err := d.Set("credentials_revision", credentialsHash); err != nil {
		return diag.Errorf("error setting credentials_revision: %v", err)
	}

	return nil
}

func resourceEdgeLocationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if !d.HasChanges(
		FieldEdgeLocationDescription,
		FieldEdgeLocationZones,
		FieldEdgeLocationAWS,
		FieldEdgeLocationGCP,
		FieldEdgeLocationOCI,
	) {
		log.Printf("[INFO] Nothing to update in edge location")
		return nil
	}

	client := meta.(*ProviderConfig).omniClient

	organizationID := d.Get(FieldEdgeLocationOrganizationID).(string)
	clusterID := d.Get(FieldEdgeLocationClusterID).(string)

	req := omni.EdgeLocationsAPIUpdateEdgeLocationJSONRequestBody{}

	if v, ok := d.GetOk(FieldEdgeLocationDescription); ok {
		req.Description = toPtr(v.(string))
	}

	if v, ok := d.GetOk(FieldEdgeLocationZones); ok {
		req.Zones = toEdgeLocationZones(v.([]interface{}))
	}

	// Map cloud provider specific configurations for update
	if d.HasChange(FieldEdgeLocationAWS) {
		if v, ok := d.GetOk(FieldEdgeLocationAWS); ok && len(v.([]interface{})) > 0 {
			req.Aws = toAWSConfig(v.([]interface{})[0].(map[string]interface{}))
		}
	}
	if d.HasChange(FieldEdgeLocationGCP) {
		if v, ok := d.GetOk(FieldEdgeLocationGCP); ok && len(v.([]interface{})) > 0 {
			req.Gcp = toGCPConfig(v.([]interface{})[0].(map[string]interface{}))
		}
	}
	if d.HasChange(FieldEdgeLocationOCI) {
		if v, ok := d.GetOk(FieldEdgeLocationOCI); ok && len(v.([]interface{})) > 0 {
			req.Oci = toOCIConfig(v.([]interface{})[0].(map[string]interface{}))
		}
	}

	resp, err := client.EdgeLocationsAPIUpdateEdgeLocationWithResponse(ctx, organizationID, clusterID, d.Id(), nil, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("updating edge location: %w", err))
	}

	if resp.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("unexpected status code updating edge location: %d, body: %s", resp.StatusCode(), string(resp.Body)))
	}

	return resourceEdgeLocationRead(ctx, d, meta)
}

func resourceEdgeLocationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).omniClient

	organizationID := d.Get(FieldEdgeLocationOrganizationID).(string)
	clusterID := d.Get(FieldEdgeLocationClusterID).(string)

	resp, err := client.EdgeLocationsAPIDeleteEdgeLocationWithResponse(ctx, organizationID, clusterID, d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("deleting edge location: %w", err))
	}

	if resp.StatusCode() == http.StatusNotFound {
		log.Printf("[DEBUG] Edge location (%s) not found, skipping delete", d.Id())
		return nil
	}

	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return diag.FromErr(fmt.Errorf("unexpected status code deleting edge location: %d, body: %s", resp.StatusCode(), string(resp.Body)))
	}

	return nil
}

// Helper functions to convert between Terraform schema and SDK types

func toEdgeLocationZones(zones []interface{}) *[]omni.Zone {
	if len(zones) == 0 {
		return nil
	}

	result := make([]omni.Zone, 0, len(zones))
	for _, z := range zones {
		zoneMap := z.(map[string]interface{})
		zone := omni.Zone{}
		if id, ok := zoneMap["id"].(string); ok && id != "" {
			zone.Id = toPtr(id)
		}
		if name, ok := zoneMap["name"].(string); ok && name != "" {
			zone.Name = toPtr(name)
		}
		result = append(result, zone)
	}
	return &result
}

func flattenEdgeLocationZones(zones *[]omni.Zone) []interface{} {
	if zones == nil || len(*zones) == 0 {
		return nil
	}

	result := make([]interface{}, 0, len(*zones))
	for _, zone := range *zones {
		m := make(map[string]interface{})
		if zone.Id != nil {
			m["id"] = *zone.Id
		}
		if zone.Name != nil {
			m["name"] = *zone.Name
		}
		result = append(result, m)
	}
	return result
}

func toAWSConfig(obj map[string]interface{}) *omni.AWSParam {
	if obj == nil {
		return nil
	}

	out := &omni.AWSParam{}
	if v, ok := obj["account_id"].(string); ok && v != "" {
		out.AccountId = toPtr(v)
	}

	// Credentials
	if _, hasKey := obj["access_key_id"]; hasKey {
		creds := &omni.AWSParamCredentials{}
		if v, ok := obj["access_key_id"].(string); ok {
			creds.AccessKeyId = v
		}
		if v, ok := obj["secret_access_key"].(string); ok {
			creds.SecretAccessKey = v
		}
		out.Credentials = creds
	}

	// Networking
	if _, hasNetworking := obj["vpc_id"]; hasNetworking {
		networking := &omni.AWSParamAWSNetworking{}
		if v, ok := obj["vpc_id"].(string); ok {
			networking.VpcId = v
		}
		if v, ok := obj["security_group_id"].(string); ok {
			networking.SecurityGroupId = v
		}
		if v, ok := obj["subnet_ids"].(map[string]interface{}); ok {
			subnetIds := make(map[string]string)
			for k, val := range v {
				subnetIds[k] = val.(string)
			}
			networking.SubnetIds = subnetIds
		}
		if v, ok := obj["name_tag"].(string); ok {
			networking.NameTag = v
		}
		out.Networking = networking
	}

	return out
}

func flattenAWSConfig(config *omni.AWSParam) []map[string]interface{} {
	if config == nil {
		return nil
	}

	m := make(map[string]interface{})

	if config.AccountId != nil {
		m["account_id"] = *config.AccountId
	}
	if config.Networking != nil {
		m["vpc_id"] = config.Networking.VpcId
		m["security_group_id"] = config.Networking.SecurityGroupId
		m["subnet_ids"] = config.Networking.SubnetIds
		m["name_tag"] = config.Networking.NameTag
	}

	return []map[string]interface{}{m}
}

func toGCPConfig(obj map[string]interface{}) *omni.GCPParam {
	if obj == nil {
		return nil
	}

	out := &omni.GCPParam{}
	if v, ok := obj["project_id"].(string); ok {
		out.ProjectId = v
	}

	// Credentials
	if v, ok := obj["client_service_account_json"].(string); ok && v != "" {
		out.Credentials = &omni.GCPParamCredentials{
			ClientServiceAccountJsonBase64: v,
		}
	}

	// Networking
	if _, hasNetworking := obj["network_name"]; hasNetworking {
		networking := &omni.GCPParamGCPNetworking{}
		if v, ok := obj["network_name"].(string); ok {
			networking.NetworkName = v
		}
		if v, ok := obj["subnet_name"].(string); ok {
			networking.SubnetName = v
		}
		if v, ok := obj["network_tags"].(*schema.Set); ok && v.Len() > 0 {
			tags := make([]string, 0, v.Len())
			for _, tag := range v.List() {
				tags = append(tags, tag.(string))
			}
			networking.Tags = tags
		}
		out.Networking = networking
	}

	return out
}

func flattenGCPConfig(config *omni.GCPParam) []map[string]interface{} {
	if config == nil {
		return nil
	}

	m := make(map[string]interface{})

	m["project_id"] = config.ProjectId
	if config.Networking != nil {
		m["network_name"] = config.Networking.NetworkName
		m["subnet_name"] = config.Networking.SubnetName
		m["network_tags"] = config.Networking.Tags
	}

	return []map[string]interface{}{m}
}

func toOCIConfig(obj map[string]interface{}) *omni.OCIParam {
	if obj == nil {
		return nil
	}

	out := &omni.OCIParam{}
	if v, ok := obj["tenancy_id"].(string); ok && v != "" {
		out.TenancyId = toPtr(v)
	}
	if v, ok := obj["compartment_id"].(string); ok && v != "" {
		out.CompartmentId = toPtr(v)
	}

	// Credentials
	if _, hasCredentials := obj["user_id"]; hasCredentials {
		creds := &omni.OCIParamCredentials{}
		if v, ok := obj["user_id"].(string); ok {
			creds.UserId = v
		}
		if v, ok := obj["fingerprint"].(string); ok {
			creds.Fingerprint = v
		}
		if v, ok := obj["private_key"].(string); ok {
			creds.PrivateKeyBase64 = v
		}
		out.Credentials = creds
	}

	// Networking
	if _, hasNetworking := obj["vcn_id"]; hasNetworking {
		networking := &omni.OCIParamNetworking{}
		if v, ok := obj["vcn_id"].(string); ok {
			networking.VcnId = v
		}
		if v, ok := obj["subnet_id"].(string); ok {
			networking.SubnetId = v
		}
		out.Networking = networking
	}

	return out
}

func flattenOCIConfig(config *omni.OCIParam) []map[string]interface{} {
	if config == nil {
		return nil
	}

	m := make(map[string]interface{})

	if config.TenancyId != nil {
		m["tenancy_id"] = *config.TenancyId
	}
	if config.CompartmentId != nil {
		m["compartment_id"] = *config.CompartmentId
	}
	if config.Networking != nil {
		m["vcn_id"] = config.Networking.VcnId
		m["subnet_id"] = config.Networking.SubnetId
	}

	return []map[string]interface{}{m}
}

func resourceEdgeLocationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Import format: organization_id/cluster_id/edge_location_id
	ids := strings.Split(d.Id(), "/")
	if len(ids) != 3 || ids[0] == "" || ids[1] == "" || ids[2] == "" {
		return nil, fmt.Errorf("invalid import ID format, expected: organization_id/cluster_id/edge_location_id, got: %q", d.Id())
	}

	organizationID := ids[0]
	clusterID := ids[1]
	edgeLocationID := ids[2]

	if err := d.Set(FieldEdgeLocationOrganizationID, organizationID); err != nil {
		return nil, fmt.Errorf("setting organization_id: %w", err)
	}
	if err := d.Set(FieldEdgeLocationClusterID, clusterID); err != nil {
		return nil, fmt.Errorf("setting cluster_id: %w", err)
	}

	d.SetId(edgeLocationID)

	// Verify the edge location exists
	if diags := resourceEdgeLocationRead(ctx, d, meta); diags.HasError() {
		return nil, fmt.Errorf("failed to read edge location during import: %v", diags)
	}

	return []*schema.ResourceData{d}, nil
}

func computeCredentialsHash(d *schema.ResourceData) string {
	hasher := sha256.New()

	// Hash AWS credentials
	if aws, ok := d.GetOk(FieldEdgeLocationAWS); ok && len(aws.([]interface{})) > 0 {
		awsMap := aws.([]interface{})[0].(map[string]interface{})
		if accessKey, ok := awsMap["access_key_id"].(string); ok {
			hasher.Write([]byte("aws_access_key:"))
			hasher.Write([]byte(accessKey))
		}
		if secretKey, ok := awsMap["secret_access_key"].(string); ok {
			hasher.Write([]byte("aws_secret_key:"))
			hasher.Write([]byte(secretKey))
		}
	}

	// Hash GCP credentials
	if gcp, ok := d.GetOk(FieldEdgeLocationGCP); ok && len(gcp.([]interface{})) > 0 {
		gcpMap := gcp.([]interface{})[0].(map[string]interface{})
		if serviceAccount, ok := gcpMap["client_service_account_json"].(string); ok {
			hasher.Write([]byte("gcp_service_account:"))
			hasher.Write([]byte(serviceAccount))
		}
	}

	// Hash OCI credentials
	if oci, ok := d.GetOk(FieldEdgeLocationOCI); ok && len(oci.([]interface{})) > 0 {
		ociMap := oci.([]interface{})[0].(map[string]interface{})
		if userId, ok := ociMap["user_id"].(string); ok {
			hasher.Write([]byte("oci_user_id:"))
			hasher.Write([]byte(userId))
		}
		if fingerprint, ok := ociMap["fingerprint"].(string); ok {
			hasher.Write([]byte("oci_fingerprint:"))
			hasher.Write([]byte(fingerprint))
		}
		if privateKey, ok := ociMap["private_key"].(string); ok {
			hasher.Write([]byte("oci_private_key:"))
			hasher.Write([]byte(privateKey))
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil))
}
