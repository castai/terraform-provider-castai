package castai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"

	"github.com/castai/terraform-provider-castai/castai/sdk"
	"github.com/castai/terraform-provider-castai/castai/sdk/organization_management"
)

const (
	// Field names for the main resource schema
	FieldEnterpriseRoleBindingEnterpriseID   = "enterprise_id"
	FieldEnterpriseRoleBindingOrganizationID = "organization_id"
	FieldEnterpriseRoleBindingID             = "id"
	FieldEnterpriseRoleBindingName           = "name"
	FieldEnterpriseRoleBindingDescription    = "description"
	FieldEnterpriseRoleBindingRoleID         = "role_id"
	FieldEnterpriseRoleBindingSubjects       = "subjects"
	FieldEnterpriseRoleBindingScopes         = "scopes"

	// Field names for subject types
	FieldEnterpriseRoleBindingSubjectUser           = "user"
	FieldEnterpriseRoleBindingSubjectServiceAccount = "service_account"
	FieldEnterpriseRoleBindingSubjectGroup          = "group"
	FieldEnterpriseRoleBindingSubjectID             = "id"

	// Field names for scope types
	FieldEnterpriseRoleBindingScopeOrganization = "organization"
	FieldEnterpriseRoleBindingScopeCluster      = "cluster"
	FieldEnterpriseRoleBindingScopeID           = "id"

	SubjectKindUser           = "user"
	SubjectKindServiceAccount = "service_account"
	SubjectKindGroup          = "group"

	ScopeKindOrganization = "organization"
	ScopeKindCluster      = "cluster"
)

type EnterpriseRoleBinding struct {
	ID             string
	Name           string
	Description    *string
	OrganizationID string
	RoleID         string
	Subjects       []EnterpriseRoleBindingSubject
	Scopes         []EnterpriseRoleBindingScope
	CreateTime     *time.Time
}

type EnterpriseRoleBindingSubject struct {
	Kind string
	ID   string
}

type EnterpriseRoleBindingScope struct {
	Kind string
	ID   string
}

func resourceEnterpriseRoleBinding() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEnterpriseRoleBindingCreate,
		ReadContext:   resourceEnterpriseRoleBindingRead,
		UpdateContext: resourceEnterpriseRoleBindingUpdate,
		DeleteContext: resourceEnterpriseRoleBindingDelete,
		Description:   "CAST AI Enterprise Role Binding resource.",
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			FieldEnterpriseRoleBindingEnterpriseID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Enterprise organization ID.",
			},
			FieldEnterpriseRoleBindingOrganizationID: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Organization ID (either enterprise or it's child) where the role binding is created.",
			},
			FieldEnterpriseRoleBindingID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Role binding ID assigned by the API.",
			},
			FieldEnterpriseRoleBindingName: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the role binding.",
			},
			FieldEnterpriseRoleBindingDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the role binding.",
			},
			FieldEnterpriseRoleBindingRoleID: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Role UUID to bind.",
			},
			FieldEnterpriseRoleBindingSubjects: {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Subjects (users, service accounts, groups) for this role binding.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldEnterpriseRoleBindingSubjectUser: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "User subjects.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnterpriseRoleBindingSubjectID: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "User ID.",
									},
								},
							},
						},
						FieldEnterpriseRoleBindingSubjectServiceAccount: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Service account subjects.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnterpriseRoleBindingSubjectID: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Service account ID.",
									},
								},
							},
						},
						FieldEnterpriseRoleBindingSubjectGroup: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Group subjects.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnterpriseRoleBindingSubjectID: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Group ID.",
									},
								},
							},
						},
					},
				},
			},
			FieldEnterpriseRoleBindingScopes: {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Scopes (organization or cluster) for this role binding.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						FieldEnterpriseRoleBindingScopeOrganization: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Organization scopes.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnterpriseRoleBindingScopeID: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Organization ID.",
									},
								},
							},
						},
						FieldEnterpriseRoleBindingScopeCluster: {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Cluster scopes.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									FieldEnterpriseRoleBindingScopeID: {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Cluster ID.",
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

func resourceEnterpriseRoleBindingCreate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Get(FieldEnterpriseRoleBindingEnterpriseID).(string)

	tflog.Debug(ctx, "Creating enterprise role binding", map[string]any{
		"enterprise_id": enterpriseID,
	})

	createRequest, err := buildBatchCreateRoleBindingRequest(data, enterpriseID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building create request: %w", err))
	}

	resp, err := client.EnterpriseAPIBatchCreateEnterpriseRoleBindingsWithResponse(ctx, enterpriseID, *createRequest)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch create enterprise role bindings failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.RoleBindings == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from batch create"))
	}

	roleBindings := resp.JSON200.RoleBindings
	if len(*roleBindings) != 1 {
		return diag.FromErr(fmt.Errorf("unexpected number of role bindings created: expected 1, got %d", len(*roleBindings)))
	}

	roleBinding, err := convertRoleBindingFromSDK((*roleBindings)[0])
	if err != nil {
		return diag.FromErr(fmt.Errorf("converting created role binding data: %w", err))
	}

	if err = setRoleBindingData(data, roleBinding); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set created role binding data: %w", err))
	}

	data.SetId(roleBinding.ID)

	tflog.Debug(ctx, "Created enterprise role binding", map[string]any{
		"role_binding_id": roleBinding.ID,
	})

	return nil
}

func resourceEnterpriseRoleBindingRead(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient

	roleBindingID := data.Id()

	tflog.Debug(ctx, "Reading enterprise role binding", map[string]any{
		"role_binding_id": roleBindingID,
	})

	enterpriseIDValue, ok := data.GetOk(FieldEnterpriseRoleBindingEnterpriseID)
	if !ok {
		return diag.FromErr(fmt.Errorf("enterprise ID is not set"))
	}
	enterpriseID := enterpriseIDValue.(string)

	organizationIDValue, ok := data.GetOk(FieldEnterpriseRoleBindingOrganizationID)
	if !ok {
		return diag.FromErr(fmt.Errorf("organization ID is not set"))
	}
	organizationID := organizationIDValue.(string)

	resp, err := client.EnterpriseAPIListRoleBindingsWithResponse(ctx, enterpriseID, &organization_management.EnterpriseAPIListRoleBindingsParams{
		OrganizationId: lo.ToPtr([]string{organizationID}),
	})
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("list enterprise role bindings failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from list enterprise role bindings"))
	}

	var rb *organization_management.RoleBinding
	for _, item := range *resp.JSON200.Items {
		if item.Id != nil && *item.Id == roleBindingID {
			rb = &item
			break
		}
	}

	if rb == nil {
		tflog.Warn(ctx, "Role binding not found, removing from state", map[string]any{
			"role_binding_id": roleBindingID,
			"enterprise_id":   enterpriseID,
		})
		data.SetId("")
		return nil
	}

	roleBinding, err := convertRoleBindingFromSDK(*rb)
	if err != nil {
		return diag.FromErr(fmt.Errorf("converting role binding data: %w", err))
	}

	if err = setRoleBindingData(data, roleBinding); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set read role binding data: %w", err))
	}

	tflog.Debug(ctx, "Finished reading enterprise role binding", map[string]any{
		"role_binding_id": roleBindingID,
	})

	return nil
}

func resourceEnterpriseRoleBindingUpdate(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	roleBindingID := data.Id()
	enterpriseID := data.Get(FieldEnterpriseRoleBindingEnterpriseID).(string)

	tflog.Debug(ctx, "Updating enterprise role binding", map[string]any{
		"role_binding_id": roleBindingID,
	})

	if !data.HasChanges(
		FieldEnterpriseRoleBindingName,
		FieldEnterpriseRoleBindingDescription,
		FieldEnterpriseRoleBindingRoleID,
		FieldEnterpriseRoleBindingSubjects,
		FieldEnterpriseRoleBindingScopes,
	) {
		tflog.Debug(ctx, "No changes detected, skipping update", map[string]any{
			"role_binding_id": roleBindingID,
		})
		return nil
	}

	updateRequest, err := buildBatchUpdateRoleBindingRequest(data, enterpriseID, roleBindingID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("building update request: %w", err))
	}

	resp, err := client.EnterpriseAPIBatchUpdateEnterpriseRoleBindingsWithResponse(ctx, enterpriseID, *updateRequest)
	if err := sdk.CheckOKResponse(resp, err); err != nil {
		return diag.FromErr(fmt.Errorf("batch update enterprise role bindings failed: %w", err))
	}

	if resp.JSON200 == nil || resp.JSON200.RoleBindings == nil {
		return diag.FromErr(fmt.Errorf("unexpected empty response from batch update"))
	}

	if len(*resp.JSON200.RoleBindings) != 1 {
		return diag.FromErr(fmt.Errorf("unexpected number of role bindings updated: expected 1, got %d", len(*resp.JSON200.RoleBindings)))
	}

	roleBinding, err := convertRoleBindingFromSDK((*resp.JSON200.RoleBindings)[0])
	if err != nil {
		return diag.FromErr(fmt.Errorf("converting updated role binding data: %w", err))
	}

	if err = setRoleBindingData(data, roleBinding); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set updated role binding data: %w", err))
	}

	tflog.Debug(ctx, "Enterprise role binding updated successfully", map[string]any{
		"role_binding_id": roleBindingID,
	})

	return nil
}

func resourceEnterpriseRoleBindingDelete(ctx context.Context, data *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(*ProviderConfig).organizationManagementClient
	enterpriseID := data.Get(FieldEnterpriseRoleBindingEnterpriseID).(string)
	organizationID := data.Get(FieldEnterpriseRoleBindingOrganizationID).(string)
	roleBindingID := data.Id()

	tflog.Debug(ctx, "Deleting enterprise role binding", map[string]any{
		"role_binding_id": roleBindingID,
		"enterprise_id":   enterpriseID,
	})

	deleteRequest := organization_management.BatchDeleteEnterpriseRoleBindingsRequest{
		EnterpriseId: enterpriseID,
		Requests: []organization_management.BatchDeleteEnterpriseRoleBindingsRequestDeleteRoleBindingRequest{
			{
				Id:             roleBindingID,
				OrganizationId: organizationID,
			},
		},
	}

	resp, err := client.EnterpriseAPIBatchDeleteEnterpriseRoleBindingsWithResponse(ctx, enterpriseID, deleteRequest)
	if err = sdk.CheckOKResponse(resp, err); err != nil {
		return diag.Errorf("batch delete enterprise role bindings failed: %s", err.Error())
	}

	// Clear the resource ID
	data.SetId("")

	tflog.Debug(ctx, "Enterprise role binding deleted successfully", map[string]any{
		"role_binding_id": roleBindingID,
	})

	return nil
}

func buildBatchCreateRoleBindingRequest(data *schema.ResourceData, enterpriseID string) (*organization_management.BatchCreateEnterpriseRoleBindingsRequest, error) {
	roleBinding, err := readRoleBindingData(data)
	if err != nil {
		return nil, fmt.Errorf("reading role binding data: %w", err)
	}

	definition := buildRoleBindingDefinition(roleBinding)

	request := organization_management.BatchCreateEnterpriseRoleBindingsRequest{
		EnterpriseId: enterpriseID,
		Requests: []organization_management.BatchCreateEnterpriseRoleBindingsRequestCreateRoleBindingRequest{
			{
				OrganizationId: roleBinding.OrganizationID,
				RoleBinding: organization_management.BatchCreateEnterpriseRoleBindingsRequestRoleBinding{
					Name:        roleBinding.Name,
					Description: roleBinding.Description,
					Definition:  definition,
				},
			},
		},
	}

	return &request, nil
}

func buildBatchUpdateRoleBindingRequest(data *schema.ResourceData, enterpriseID, roleBindingID string) (*organization_management.BatchUpdateEnterpriseRoleBindingsRequest, error) {
	roleBinding, err := readRoleBindingData(data)
	if err != nil {
		return nil, fmt.Errorf("reading role binding data: %w", err)
	}

	definition := buildRoleBindingDefinition(roleBinding)

	request := organization_management.BatchUpdateEnterpriseRoleBindingsRequest{
		EnterpriseId: enterpriseID,
		Requests: []organization_management.BatchUpdateEnterpriseRoleBindingsRequestUpdateEnterpriseRoleBindingRequest{
			{
				Id:             roleBindingID,
				Name:           roleBinding.Name,
				OrganizationId: roleBinding.OrganizationID,
				Description:    roleBinding.Description,
				Definition:     definition,
			},
		},
	}

	return &request, nil
}

func buildRoleBindingDefinition(roleBinding EnterpriseRoleBinding) organization_management.RoleBindingDefinition {
	subjects := convertEnterpriseRoleBindingSubjectsToSDK(roleBinding.Subjects)
	scopes := convertEnterpriseRoleBindingScopesToSDK(roleBinding.Scopes)

	definition := organization_management.RoleBindingDefinition{
		RoleId: lo.ToPtr(roleBinding.RoleID),
	}

	if len(subjects) > 0 {
		definition.Subjects = &subjects
	}

	if len(scopes) > 0 {
		definition.Scopes = &scopes
	}

	return definition
}

func convertEnterpriseRoleBindingSubjectsToSDK(subjects []EnterpriseRoleBindingSubject) []organization_management.Subject {
	return lo.Map(subjects, func(s EnterpriseRoleBindingSubject, _ int) organization_management.Subject {
		subject := organization_management.Subject{}

		switch strings.ToLower(s.Kind) {
		case SubjectKindUser:
			subject.User = &organization_management.UserSubject{Id: s.ID}
		case SubjectKindServiceAccount:
			subject.ServiceAccount = &organization_management.ServiceAccountSubject{Id: s.ID}
		case SubjectKindGroup:
			subject.Group = &organization_management.GroupSubject{Id: s.ID}
		}

		return subject
	})
}

func convertEnterpriseRoleBindingScopesToSDK(scopes []EnterpriseRoleBindingScope) []organization_management.Scope {
	return lo.Map(scopes, func(s EnterpriseRoleBindingScope, _ int) organization_management.Scope {
		scope := organization_management.Scope{}

		if s.Kind == ScopeKindOrganization {
			scope.Organization = &organization_management.OrganizationScope{Id: s.ID}
		}

		if s.Kind == ScopeKindCluster {
			scope.Cluster = &organization_management.ClusterScope{Id: s.ID}
		}

		return scope
	})
}

func convertRoleBindingFromSDK(rb organization_management.RoleBinding) (EnterpriseRoleBinding, error) {
	if rb.Id == nil {
		return EnterpriseRoleBinding{}, fmt.Errorf("role binding ID is nil")
	}

	if rb.Name == nil {
		return EnterpriseRoleBinding{}, fmt.Errorf("role binding name is nil")
	}

	if rb.OrganizationId == nil {
		return EnterpriseRoleBinding{}, fmt.Errorf("role binding organization ID is nil")
	}

	if rb.Definition == nil {
		return EnterpriseRoleBinding{}, fmt.Errorf("role binding definition is nil")
	}

	if rb.Definition.RoleId == nil {
		return EnterpriseRoleBinding{}, fmt.Errorf("role binding role ID is nil")
	}

	var subjects []EnterpriseRoleBindingSubject
	if rb.Definition != nil && rb.Definition.Subjects != nil {
		converted, err := convertSubjectsFromSDK(*rb.Definition.Subjects)
		if err != nil {
			return EnterpriseRoleBinding{}, fmt.Errorf("converting subjects: %w", err)
		}
		subjects = converted
	}

	var scopes []EnterpriseRoleBindingScope
	if rb.Definition != nil && rb.Definition.Scopes != nil {
		scopes = convertScopesFromSDK(*rb.Definition.Scopes)
	}

	return EnterpriseRoleBinding{
		ID:             *rb.Id,
		Name:           *rb.Name,
		Description:    rb.Description,
		OrganizationID: *rb.OrganizationId,
		RoleID:         *rb.Definition.RoleId,
		Subjects:       subjects,
		Scopes:         scopes,
		CreateTime:     rb.CreateTime,
	}, nil
}

func convertSubjectsFromSDK(sdkSubjects []organization_management.Subject) ([]EnterpriseRoleBindingSubject, error) {
	subjects := make([]EnterpriseRoleBindingSubject, 0, len(sdkSubjects))

	for _, sdkSubject := range sdkSubjects {
		subject := EnterpriseRoleBindingSubject{}

		switch {
		case sdkSubject.User != nil:
			subject.Kind = SubjectKindUser
			subject.ID = sdkSubject.User.Id
		case sdkSubject.ServiceAccount != nil:
			subject.Kind = SubjectKindServiceAccount
			subject.ID = sdkSubject.ServiceAccount.Id
		case sdkSubject.Group != nil:
			subject.Kind = SubjectKindGroup
			subject.ID = sdkSubject.Group.Id
		default:
			return nil, fmt.Errorf("unknown subject type in response")
		}

		subjects = append(subjects, subject)
	}

	return subjects, nil
}

func convertScopesFromSDK(sdkScopes []organization_management.Scope) []EnterpriseRoleBindingScope {
	return lo.Map(sdkScopes, func(s organization_management.Scope, _ int) EnterpriseRoleBindingScope {
		scope := EnterpriseRoleBindingScope{}

		if s.Organization != nil {
			scope.Kind = ScopeKindOrganization
			scope.ID = s.Organization.Id
		}

		if s.Cluster != nil {
			scope.Kind = ScopeKindCluster
			scope.ID = s.Cluster.Id
		}

		return scope
	})
}

func setRoleBindingData(data *schema.ResourceData, rb EnterpriseRoleBinding) error {
	if err := data.Set(FieldEnterpriseRoleBindingID, rb.ID); err != nil {
		return fmt.Errorf("failed to set role binding ID: %w", err)
	}

	if err := data.Set(FieldEnterpriseRoleBindingName, rb.Name); err != nil {
		return fmt.Errorf("failed to set name: %w", err)
	}

	if err := data.Set(FieldEnterpriseRoleBindingDescription, lo.FromPtr(rb.Description)); err != nil {
		return fmt.Errorf("failed to set description: %w", err)
	}

	if err := data.Set(FieldEnterpriseRoleBindingOrganizationID, rb.OrganizationID); err != nil {
		return fmt.Errorf("failed to set organization ID: %w", err)
	}

	if err := data.Set(FieldEnterpriseRoleBindingRoleID, rb.RoleID); err != nil {
		return fmt.Errorf("failed to set role ID: %w", err)
	}

	subjectsData, err := convertSubjectsToTFData(rb.Subjects)
	if err != nil {
		return fmt.Errorf("failed to convert subjects: %w", err)
	}

	if err := data.Set(FieldEnterpriseRoleBindingSubjects, subjectsData); err != nil {
		return fmt.Errorf("failed to set subjects: %w", err)
	}

	scopesData, err := convertScopesToTFData(rb.Scopes)
	if err != nil {
		return fmt.Errorf("failed to convert scopes: %w", err)
	}

	if err := data.Set(FieldEnterpriseRoleBindingScopes, scopesData); err != nil {
		return fmt.Errorf("failed to set scopes: %w", err)
	}

	return nil
}

func convertSubjectsToTFData(subjects []EnterpriseRoleBindingSubject) ([]map[string]any, error) {
	// Group subjects by kind
	users := []map[string]any{}
	serviceAccounts := []map[string]any{}
	groups := []map[string]any{}

	for _, subject := range subjects {
		if subject.ID == "" {
			return nil, fmt.Errorf("subject kind %s missing id", subject.Kind)
		}

		switch strings.ToLower(subject.Kind) {
		case SubjectKindUser:
			users = append(users, map[string]any{
				FieldEnterpriseRoleBindingSubjectID: subject.ID,
			})
		case SubjectKindServiceAccount:
			serviceAccounts = append(serviceAccounts, map[string]any{
				FieldEnterpriseRoleBindingSubjectID: subject.ID,
			})
		case SubjectKindGroup:
			groups = append(groups, map[string]any{
				FieldEnterpriseRoleBindingSubjectID: subject.ID,
			})
		default:
			return nil, fmt.Errorf("unsupported subject kind: %s", subject.Kind)
		}
	}

	// Create single subjects block with all grouped subjects
	subjectsBlock := map[string]any{}
	if len(users) > 0 {
		subjectsBlock[FieldEnterpriseRoleBindingSubjectUser] = users
	}
	if len(serviceAccounts) > 0 {
		subjectsBlock[FieldEnterpriseRoleBindingSubjectServiceAccount] = serviceAccounts
	}
	if len(groups) > 0 {
		subjectsBlock[FieldEnterpriseRoleBindingSubjectGroup] = groups
	}

	return []map[string]any{subjectsBlock}, nil
}

func convertScopesToTFData(scopes []EnterpriseRoleBindingScope) ([]map[string]any, error) {
	// Group scopes by kind
	organizations := []map[string]any{}
	clusters := []map[string]any{}

	for _, scope := range scopes {
		switch strings.ToLower(scope.Kind) {
		case ScopeKindOrganization:
			organizations = append(organizations, map[string]any{
				FieldEnterpriseRoleBindingScopeID: scope.ID,
			})
		case ScopeKindCluster:
			clusters = append(clusters, map[string]any{
				FieldEnterpriseRoleBindingScopeID: scope.ID,
			})
		default:
			return nil, fmt.Errorf("unsupported scope kind: %s", scope.Kind)
		}
	}

	// Create single scopes block with all grouped scopes
	scopesBlock := map[string]any{}
	if len(organizations) > 0 {
		scopesBlock[FieldEnterpriseRoleBindingScopeOrganization] = organizations
	}
	if len(clusters) > 0 {
		scopesBlock[FieldEnterpriseRoleBindingScopeCluster] = clusters
	}

	return []map[string]any{scopesBlock}, nil
}

func readRoleBindingData(data *schema.ResourceData) (EnterpriseRoleBinding, error) {
	rbDescription := ""
	if description, ok := data.GetOk(FieldEnterpriseRoleBindingDescription); ok {
		rbDescription = description.(string)
	}

	subjects, err := parseSubjectsFromTFData(data)
	if err != nil {
		return EnterpriseRoleBinding{}, err
	}

	scopes, err := parseScopesFromTFData(data)
	if err != nil {
		return EnterpriseRoleBinding{}, err
	}

	roleBinding := EnterpriseRoleBinding{
		ID:             data.Get(FieldEnterpriseRoleBindingID).(string),
		Name:           data.Get(FieldEnterpriseRoleBindingName).(string),
		Description:    &rbDescription,
		OrganizationID: data.Get(FieldEnterpriseRoleBindingOrganizationID).(string),
		RoleID:         data.Get(FieldEnterpriseRoleBindingRoleID).(string),
		Subjects:       subjects,
		Scopes:         scopes,
	}

	return roleBinding, nil
}

func parseSubjectsFromTFData(data *schema.ResourceData) ([]EnterpriseRoleBindingSubject, error) {
	subjects := []EnterpriseRoleBindingSubject{}
	if subjectsData, ok := data.GetOk(FieldEnterpriseRoleBindingSubjects); ok {
		if subjectBlocks, ok := subjectsData.([]any); ok && len(subjectBlocks) > 0 {
			if subjectBlock, ok := subjectBlocks[0].(map[string]any); ok {
				subjects = append(subjects, parseSubjectItems(subjectBlock, FieldEnterpriseRoleBindingSubjectUser, SubjectKindUser)...)
				subjects = append(subjects, parseSubjectItems(subjectBlock, FieldEnterpriseRoleBindingSubjectServiceAccount, SubjectKindServiceAccount)...)
				subjects = append(subjects, parseSubjectItems(subjectBlock, FieldEnterpriseRoleBindingSubjectGroup, SubjectKindGroup)...)
			}
		}
	}

	if len(subjects) == 0 {
		return nil, fmt.Errorf("at least one subject (user, service account, or group) must be defined")
	}

	return subjects, nil
}

func parseSubjectItems(block map[string]any, itemKey, kind string) []EnterpriseRoleBindingSubject {
	var items []EnterpriseRoleBindingSubject
	if itemList, ok := block[itemKey].([]any); ok {
		for _, item := range itemList {
			if itemMap, ok := item.(map[string]any); ok {
				if id, ok := itemMap[FieldEnterpriseRoleBindingSubjectID].(string); ok && id != "" {
					items = append(items, EnterpriseRoleBindingSubject{
						Kind: kind,
						ID:   id,
					})
				}
			}
		}
	}
	return items
}

func parseScopesFromTFData(data *schema.ResourceData) ([]EnterpriseRoleBindingScope, error) {
	scopes := []EnterpriseRoleBindingScope{}
	if scopesData, ok := data.GetOk(FieldEnterpriseRoleBindingScopes); ok {
		if scopeBlocks, ok := scopesData.([]any); ok && len(scopeBlocks) > 0 {
			if scopeBlock, ok := scopeBlocks[0].(map[string]any); ok {
				scopes = append(scopes, parseScopeItems(scopeBlock, FieldEnterpriseRoleBindingScopeOrganization, ScopeKindOrganization)...)
				scopes = append(scopes, parseScopeItems(scopeBlock, FieldEnterpriseRoleBindingScopeCluster, ScopeKindCluster)...)
			}
		}
	}

	if len(scopes) == 0 {
		return nil, fmt.Errorf("at least one scope (organization or cluster) must be defined")
	}

	return scopes, nil
}

func parseScopeItems(block map[string]any, itemKey, kind string) []EnterpriseRoleBindingScope {
	var items []EnterpriseRoleBindingScope
	if itemList, ok := block[itemKey].([]any); ok {
		for _, item := range itemList {
			if itemMap, ok := item.(map[string]any); ok {
				if id, ok := itemMap[FieldEnterpriseRoleBindingScopeID].(string); ok && id != "" {
					items = append(items, EnterpriseRoleBindingScope{
						Kind: kind,
						ID:   id,
					})
				}
			}
		}
	}
	return items
}
