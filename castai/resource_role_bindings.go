package castai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

const (
	FieldRoleBindingsOrganizationID          = "organization_id"
	FieldRoleBindingsName                    = "name"
	FieldRoleBindingsDescription             = "description"
	FieldRoleBindingsRoleID                  = "role_id"
	FieldRoleBindingsScope                   = "scope"
	FieldRoleBindingsScopeKind               = "kind"
	FieldRoleBindingsScopeResourceID         = "resource_id"
	FieldRoleBindingsSubjects                = "subjects"
	FieldRoleBindingsSubject                 = "subject"
	FieldRoleBindingsSubjectKind             = "kind"
	FieldRoleBindingsSubjectUserID           = "user_id"
	FieldRoleBindingsSubjectServiceAccountID = "service_account_id"
	FieldRoleBindingsSubjectGroupID          = "group_id"

	RoleBindingScopeKindOrganization = "organization"
	RoleBindingScopeKindCluster      = "cluster"

	RoleBindingSubjectKindUser           = "user"
	RoleBindingSubjectKindServiceAccount = "service_account"
	RoleBindingSubjectKindGroup          = "group"
)

var (
	supportedScopeKinds   = []string{RoleBindingScopeKindOrganization, RoleBindingScopeKindCluster}
	supportedSubjectKinds = []string{RoleBindingSubjectKindUser, RoleBindingSubjectKindServiceAccount, RoleBindingSubjectKindGroup}
)

func resourceRoleBindings() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceRoleBindingsRead,
		CreateContext: resourceRoleBindingsCreate,
		UpdateContext: resourceRoleBindingsUpdate,
		DeleteContext: resourceRoleBindingsDelete,
		Description:   "CAST AI organization group resource to manage organization groups",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(2 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldRoleBindingsOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "CAST AI organization ID.",
			},
			FieldRoleBindingsName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of role binding.",
			},
			FieldRoleBindingsDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Description of the role binding.",
			},
			FieldRoleBindingsRoleID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of role from role binding.",
			},
			FieldRoleBindingsScope: {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Scope of the role binding.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldRoleBindingsScopeKind: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      fmt.Sprintf("Scope of the role binding Supported values include: %s.", strings.Join(supportedScopeKinds, ", ")),
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedScopeKinds, true)),
							DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
								return strings.EqualFold(oldValue, newValue)
							},
						},
						FieldRoleBindingsScopeResourceID: {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ID of the scope resource.",
						},
					},
				},
			},
			FieldRoleBindingsSubjects: {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldRoleBindingsSubject: {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldRoleBindingsSubjectKind: {
										Type:             schema.TypeString,
										Required:         true,
										Description:      fmt.Sprintf("Kind of the subject. Supported values include: %s.", strings.Join(supportedSubjectKinds, ", ")),
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(supportedSubjectKinds, true)),
										DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
											return strings.EqualFold(oldValue, newValue)
										},
									},
									FieldRoleBindingsSubjectUserID: {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: fmt.Sprintf("Optional, required only if `%s` is `%s`.", FieldRoleBindingsSubjectKind, RoleBindingSubjectKindUser),
									},
									FieldRoleBindingsSubjectServiceAccountID: {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: fmt.Sprintf("Optional, required only if `%s` is `%s`.", FieldRoleBindingsSubjectKind, RoleBindingSubjectKindServiceAccount),
									},
									FieldRoleBindingsSubjectGroupID: {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: fmt.Sprintf("Optional, required only if `%s` is `%s`.", FieldRoleBindingsSubjectKind, RoleBindingSubjectKindGroup),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceRoleBindingsRead(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	roleBindingID := data.Id()
	if roleBindingID == "" {
		return diag.Errorf("role binding ID is not set")
	}

	organizationID := data.Get(FieldRoleBindingsOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.FromErr(fmt.Errorf("getting default organization: %w", err))
		}
	}

	client := meta.(*ProviderConfig).api

	roleBinding, err := getRoleBinding(client, ctx, organizationID, roleBindingID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("getting role binding for read: %w", err))
	}

	if err := assignRoleBindingData(roleBinding, data); err != nil {
		return diag.FromErr(fmt.Errorf("assigning role binding data for read: %w", err))
	}

	return nil
}

func resourceRoleBindingsCreate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	organizationID := data.Get(FieldRoleBindingsOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.FromErr(fmt.Errorf("getting default organization: %w", err))
		}
	}

	client := meta.(*ProviderConfig).api

	subjects, err := convertSubjectsToSDK(data)
	if err != nil {
		return diag.FromErr(err)
	}
	scope := convertScopeToSDK(data)

	resp, err := client.RbacServiceAPICreateRoleBindingsWithResponse(ctx, organizationID, sdk.RbacServiceAPICreateRoleBindingsJSONRequestBody{
		{
			Definition: sdk.CastaiRbacV1beta1RoleBindingDefinition{
				RoleId:   data.Get(FieldRoleBindingsRoleID).(string),
				Scope:    scope,
				Subjects: &subjects,
			},
			Description: lo.ToPtr(data.Get(FieldRoleBindingsDescription).(string)),
			Name:        data.Get(FieldRoleBindingsName).(string),
		},
	})

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("create role binding: %w", err))
	}

	if len(*resp.JSON200) == 0 {
		return diag.FromErr(errors.New("unknown error with creating role binding"))
	}

	data.SetId(*(*resp.JSON200)[0].Id)

	return resourceRoleBindingsRead(ctx, data, meta)
}

func resourceRoleBindingsUpdate(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	roleBindingID := data.Id()
	if roleBindingID == "" {
		return diag.Errorf("role binding ID is not set")
	}

	organizationID := data.Get(FieldRoleBindingsOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.FromErr(fmt.Errorf("getting default organization: %w", err))
		}
	}

	client := meta.(*ProviderConfig).api

	subjects, err := convertSubjectsToSDK(data)
	if err != nil {
		return diag.FromErr(err)
	}
	scope := convertScopeToSDK(data)

	resp, err := client.RbacServiceAPIUpdateRoleBindingWithResponse(ctx, organizationID, roleBindingID, sdk.RbacServiceAPIUpdateRoleBindingJSONRequestBody{
		Definition: sdk.CastaiRbacV1beta1RoleBindingDefinition{
			RoleId:   data.Get(FieldRoleBindingsRoleID).(string),
			Scope:    scope,
			Subjects: &subjects,
		},
		Description: lo.ToPtr(data.Get(FieldRoleBindingsDescription).(string)),
		Name:        data.Get(FieldRoleBindingsName).(string),
	})

	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("update role binding: %w", err))
	}

	return resourceRoleBindingsRead(ctx, data, meta)
}

func resourceRoleBindingsDelete(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	roleBindingID := data.Id()
	if roleBindingID == "" {
		return diag.Errorf("role binding ID is not set")
	}

	organizationID := data.Get(FieldRoleBindingsOrganizationID).(string)
	if organizationID == "" {
		var err error
		organizationID, err = getDefaultOrganizationId(ctx, meta)
		if err != nil {
			return diag.FromErr(fmt.Errorf("getting default organization: %w", err))
		}
	}

	client := meta.(*ProviderConfig).api

	resp, err := client.RbacServiceAPIDeleteRoleBindingWithResponse(ctx, organizationID, roleBindingID)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("destroy role binding: %w", err))
	}

	return nil
}

func getRoleBinding(client *sdk.ClientWithResponses, ctx context.Context, organizationID, roleBindingID string) (*sdk.CastaiRbacV1beta1RoleBinding, error) {
	resp, err := client.RbacServiceAPIGetRoleBindingWithResponse(ctx, organizationID, roleBindingID)
	if err != nil {
		return nil, fmt.Errorf("fetching role binding: %w", err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		return nil, fmt.Errorf("role binding %s not found", roleBindingID)
	}
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return nil, fmt.Errorf("retrieving role binding: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("role binding %s not found", roleBindingID)
	}
	return resp.JSON200, nil
}

func assignRoleBindingData(roleBinding *sdk.CastaiRbacV1beta1RoleBinding, data *schema.ResourceData) error {
	if err := data.Set(FieldRoleBindingsOrganizationID, roleBinding.OrganizationId); err != nil {
		return fmt.Errorf("setting organization_id: %w", err)
	}
	if err := data.Set(FieldRoleBindingsDescription, roleBinding.Description); err != nil {
		return fmt.Errorf("setting description: %w", err)
	}
	if err := data.Set(FieldRoleBindingsName, roleBinding.Name); err != nil {
		return fmt.Errorf("setting role binding name: %w", err)
	}
	if err := data.Set(FieldRoleBindingsRoleID, roleBinding.Definition.RoleId); err != nil {
		return fmt.Errorf("setting role binding role id: %w", err)
	}

	if roleBinding.Definition.Scope.Organization != nil {
		err := data.Set(FieldRoleBindingsScope, []any{
			map[string]any{
				FieldRoleBindingsScopeKind:       RoleBindingScopeKindOrganization,
				FieldRoleBindingsScopeResourceID: roleBinding.Definition.Scope.Organization.Id,
			},
		})
		if err != nil {
			return fmt.Errorf("parsing scope organization: %w", err)
		}
	} else if roleBinding.Definition.Scope.Cluster != nil {
		err := data.Set(FieldRoleBindingsScope, []any{
			map[string]any{
				FieldRoleBindingsScopeKind:       RoleBindingScopeKindCluster,
				FieldRoleBindingsScopeResourceID: roleBinding.Definition.Scope.Cluster.Id,
			},
		})
		if err != nil {
			return fmt.Errorf("parsing scope cluster: %w", err)
		}
	}

	if roleBinding.Definition.Subjects != nil {
		var subjects []map[string]string
		for _, subject := range *roleBinding.Definition.Subjects {

			if subject.User != nil {
				subjects = append(subjects, map[string]string{
					FieldRoleBindingsSubjectKind:   RoleBindingSubjectKindUser,
					FieldRoleBindingsSubjectUserID: subject.User.Id,
				})
			} else if subject.Group != nil {
				subjects = append(subjects, map[string]string{
					FieldRoleBindingsSubjectKind:    RoleBindingSubjectKindGroup,
					FieldRoleBindingsSubjectGroupID: subject.Group.Id,
				})
			} else if subject.ServiceAccount != nil {
				subjects = append(subjects, map[string]string{
					FieldRoleBindingsSubjectKind:             RoleBindingSubjectKindServiceAccount,
					FieldRoleBindingsSubjectServiceAccountID: subject.ServiceAccount.Id,
				})
			}
		}
		err := data.Set(FieldRoleBindingsSubjects, []any{
			map[string]any{
				FieldRoleBindingsSubject: subjects,
			},
		})
		if err != nil {
			return fmt.Errorf("parsing roleBinding subjects: %w", err)
		}
	}

	return nil
}

func convertScopeToSDK(data *schema.ResourceData) sdk.CastaiRbacV1beta1Scope {
	scopes := data.Get(FieldRoleBindingsScope).([]any)
	if len(scopes) == 0 {
		return sdk.CastaiRbacV1beta1Scope{}
	}

	scope := scopes[0].(map[string]any)

	switch scope[FieldRoleBindingsScopeKind].(string) {
	case RoleBindingScopeKindOrganization:
		return sdk.CastaiRbacV1beta1Scope{
			Organization: &sdk.CastaiRbacV1beta1OrganizationScope{
				Id: scope[FieldRoleBindingsScopeResourceID].(string),
			},
		}
	case RoleBindingScopeKindCluster:
		return sdk.CastaiRbacV1beta1Scope{
			Cluster: &sdk.CastaiRbacV1beta1ClusterScope{
				Id: scope[FieldRoleBindingsScopeResourceID].(string),
			},
		}
	default:
		return sdk.CastaiRbacV1beta1Scope{}
	}
}

func convertSubjectsToSDK(data *schema.ResourceData) ([]sdk.CastaiRbacV1beta1Subject, error) {
	var subjects []sdk.CastaiRbacV1beta1Subject

	for _, dataSubjectsDef := range data.Get(FieldRoleBindingsSubjects).([]any) {
		for i, dataSubject := range dataSubjectsDef.(map[string]any)[FieldRoleBindingsSubject].([]any) {

			switch dataSubject.(map[string]any)[FieldRoleBindingsSubjectKind].(string) {
			case RoleBindingSubjectKindUser:
				if dataSubject.(map[string]any)[FieldRoleBindingsSubjectUserID].(string) == "" {
					return nil, fmt.Errorf("missing `%s` value for subject no. %d", FieldRoleBindingsSubjectUserID, i)
				}

				subjects = append(subjects, sdk.CastaiRbacV1beta1Subject{
					User: &sdk.CastaiRbacV1beta1UserSubject{
						Id: dataSubject.(map[string]any)[FieldRoleBindingsSubjectUserID].(string),
					},
				})
			case RoleBindingSubjectKindServiceAccount:
				if dataSubject.(map[string]any)[FieldRoleBindingsSubjectServiceAccountID].(string) == "" {
					return nil, fmt.Errorf("missing `%s` value for subject no. %d", FieldRoleBindingsSubjectServiceAccountID, i)
				}

				subjects = append(subjects, sdk.CastaiRbacV1beta1Subject{
					ServiceAccount: &sdk.CastaiRbacV1beta1ServiceAccountSubject{
						Id: dataSubject.(map[string]any)[FieldRoleBindingsSubjectServiceAccountID].(string),
					},
				})
			case RoleBindingSubjectKindGroup:
				if dataSubject.(map[string]any)[FieldRoleBindingsSubjectGroupID].(string) == "" {
					return nil, fmt.Errorf("missing `%s` value for subject no. %d", FieldRoleBindingsSubjectGroupID, i)
				}

				subjects = append(subjects, sdk.CastaiRbacV1beta1Subject{
					Group: &sdk.CastaiRbacV1beta1GroupSubject{
						Id: dataSubject.(map[string]any)[FieldRoleBindingsSubjectGroupID].(string),
					},
				})
			}
		}
	}

	return subjects, nil
}
