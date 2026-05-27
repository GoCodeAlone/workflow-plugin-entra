package internal

import (
	"context"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

type authProviderDescribeStep struct {
	name   string
	config map[string]any
}

func newAuthProviderDescribeStep(name string, config map[string]any) (sdk.StepInstance, error) {
	return &authProviderDescribeStep{name: name, config: config}, nil
}

func (s *authProviderDescribeStep) Execute(_ context.Context, _ map[string]any, _ map[string]map[string]any, current, _, _ map[string]any) (*sdk.StepResult, error) {
	values := mergeMaps(s.config, current)
	providerID := firstNonEmpty(values, "provider_id", "providerId")
	if providerID == "" {
		providerID = "entra"
	}
	tenantID := firstNonEmpty(values, "tenant_id", "tenantId", "tenant")
	return &sdk.StepResult{Output: map[string]any{
		"providers": []map[string]any{entraProviderDescriptor(providerID, tenantID)},
	}}, nil
}

func entraProviderDescriptor(providerID, tenantID string) map[string]any {
	return map[string]any{
		"id":             providerID,
		"label":          "Microsoft Entra ID",
		"description":    "Microsoft Entra identity, group, application registration, service principal, and directory role administration through Microsoft Graph.",
		"categories":     []string{"identity_management", "oauth2_oidc", "enterprise_sso", "directory_roles"},
		"implementation": "workflow-plugin-entra",
		"version":        Version,
		"docs_url":       "https://github.com/GoCodeAlone/workflow-plugin-entra",
		"support_level":  "management",
		"capabilities": []map[string]any{
			entraCapability("entra_users", "Users", "identity_management", "Create, read, update, and delete Microsoft Entra users.", []string{"User.ReadWrite.All"}, entraFields(tenantID)),
			entraCapability("entra_groups", "Groups", "identity_management", "Create and read groups, and manage direct group membership.", []string{"Group.ReadWrite.All"}, entraFields(tenantID)),
			entraCapability("entra_applications", "Application registrations", "oauth2_oidc", "Create, read, update, and delete app registrations used by OAuth 2.0 and OpenID Connect clients.", []string{"Application.ReadWrite.All"}, entraFields(tenantID)),
			entraCapability("entra_service_principals", "Service principals", "enterprise_sso", "List service principals backing enterprise applications and SSO integrations.", []string{"Application.Read.All"}, entraFields(tenantID)),
			entraCapability("entra_directory_roles", "Directory roles", "directory_roles", "List active directory roles for tenant administration review.", []string{"Directory.Read.All"}, entraFields(tenantID)),
		},
	}
}

func entraCapability(key, label, category, description string, graphScopes []string, fields []map[string]any) map[string]any {
	return map[string]any{
		"key":                key,
		"label":              label,
		"category":           category,
		"description":        description,
		"supported":          true,
		"app_scopes":         graphScopes,
		"admin_read_scopes":  []string{"admin.auth.providers.read"},
		"admin_write_scopes": []string{"admin.auth.providers.write"},
		"config_fields":      fields,
	}
}

func entraFields(tenantID string) []map[string]any {
	return []map[string]any{
		entraField("entra_tenant_id", "Tenant ID or domain", "text", "Microsoft Entra tenant ID or verified tenant domain.", "Use the tenant GUID or domain that owns the app registration.", false, true, optionIfSet(normalizeTenantID(tenantID))),
		entraField("entra_client_id", "Application client ID", "text", "Client ID for the Entra application registration used by Workflow.", "Grant this app registration least-privilege Microsoft Graph application permissions for the enabled capabilities.", false, true, nil),
		entraField("entra_client_secret", "Client secret", "secret", "Client secret for the Entra application registration used by Workflow.", "Write-only secret. Store through the application's secret provider and rotate regularly.", true, true, nil),
		entraField("entra_graph_permissions", "Graph application permissions", "multiselect", "Microsoft Graph application permissions required by enabled capabilities.", "Select the permissions granted by admin consent to the Workflow app registration.", false, false, entraGraphPermissionOptions()),
	}
}

func entraField(key, label, inputType, description, helpText string, secret, required bool, options []map[string]any) map[string]any {
	return map[string]any{
		"key":         key,
		"label":       label,
		"input_type":  inputType,
		"description": description,
		"help_text":   helpText,
		"secret":      secret,
		"required":    required,
		"options":     options,
	}
}

func optionIfSet(value string) []map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return []map[string]any{{"value": value, "label": value}}
}

func entraGraphPermissionOptions() []map[string]any {
	permissions := []map[string]string{
		{"value": "User.ReadWrite.All", "label": "User.ReadWrite.All", "description": "Create, read, update, and delete users."},
		{"value": "Group.ReadWrite.All", "label": "Group.ReadWrite.All", "description": "Create and read groups and manage direct memberships."},
		{"value": "Application.ReadWrite.All", "label": "Application.ReadWrite.All", "description": "Create, read, update, and delete application registrations."},
		{"value": "Application.Read.All", "label": "Application.Read.All", "description": "Read service principals and application metadata."},
		{"value": "Directory.Read.All", "label": "Directory.Read.All", "description": "Read directory roles and tenant metadata."},
	}
	options := make([]map[string]any, 0, len(permissions))
	for _, permission := range permissions {
		options = append(options, map[string]any{
			"value":       permission["value"],
			"label":       permission["label"],
			"description": permission["description"],
		})
	}
	return options
}
