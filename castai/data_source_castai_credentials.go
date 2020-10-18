package castai

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func dataSourceCastaiCredentials() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceCastaiCredentialsRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:             schema.TypeString,
				Optional:         true,
				ExactlyOneOf:     []string{"id", "name"},
				ValidateDiagFunc: toDiagFunc(validation.IsUUID),
			},
			"name": {
				Type:             schema.TypeString,
				Optional:         true,
				ExactlyOneOf:     []string{"id", "name"},
				ValidateDiagFunc: toDiagFunc(validation.StringIsNotWhiteSpace),
			},
			"cloud": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCastaiCredentialsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).api

	var credentials *sdk.CloudCredentials

	if id, ok := data.GetOk("id"); ok {
		response, err := client.GetCloudCredentialsWithResponse(ctx, sdk.CredentialsId(id.(string)))
		if checkErr := sdk.CheckGetResponse(response, err); checkErr != nil {
			return diag.Errorf("fetching cluster by id=%s: %v", id, checkErr)
		}

		credentials = response.JSON200
	} else if name, ok := data.GetOk("name"); ok {
		response, err := client.ListCloudCredentialsWithResponse(ctx)
		if checkErr := sdk.CheckGetResponse(response, err); checkErr != nil {
			return diag.Errorf("reading list of credentials: %v", checkErr)
		}

		var foundCredentials *sdk.CloudCredentials
		for _, item := range response.JSON200.Items {
			if item.Name == name {
				foundCredentials = &item
				break
			}
		}

		if foundCredentials == nil {
			return diag.Errorf("credentials was not found by name=%s", name)
		}

		credentials = foundCredentials
	}

	log.Printf("[INFO] found cloud credentials: %v", credentials)

	data.SetId(credentials.Id)
	data.Set("id", credentials.Id)
	data.Set("name", credentials.Name)
	data.Set("cloud", credentials.Cloud)
	return nil
}
