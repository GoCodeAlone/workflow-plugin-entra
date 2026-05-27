package internal

import (
	"context"
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/microsoftgraph/msgraph-sdk-go/groups"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

type stepConstructor func(name string, config map[string]any) (sdk.StepInstance, error)

var stepRegistry = map[string]stepConstructor{
	"step.entra_auth_provider_describe": newAuthProviderDescribeStep,
	"step.entra_user_create":            newGraphStep(entraUserCreate),
	"step.entra_user_get":               newGraphStep(entraUserGet),
	"step.entra_user_list":              newGraphStep(entraUserList),
	"step.entra_user_update":            newGraphStep(entraUserUpdate),
	"step.entra_user_delete":            newGraphStep(entraUserDelete),
	"step.entra_group_create":           newGraphStep(entraGroupCreate),
	"step.entra_group_get":              newGraphStep(entraGroupGet),
	"step.entra_group_list":             newGraphStep(entraGroupList),
	"step.entra_group_add_member":       newGraphStep(entraGroupAddMember),
	"step.entra_group_remove_member":    newGraphStep(entraGroupRemoveMember),
	"step.entra_application_create":     newGraphStep(entraApplicationCreate),
	"step.entra_application_get":        newGraphStep(entraApplicationGet),
	"step.entra_application_list":       newGraphStep(entraApplicationList),
	"step.entra_application_update":     newGraphStep(entraApplicationUpdate),
	"step.entra_application_delete":     newGraphStep(entraApplicationDelete),
	"step.entra_directory_role_list":    newGraphStep(entraDirectoryRoleList),
	"step.entra_service_principal_list": newGraphStep(entraServicePrincipalList),
}

func allStepTypes() []string {
	return []string{
		"step.entra_auth_provider_describe",
		"step.entra_user_create",
		"step.entra_user_get",
		"step.entra_user_list",
		"step.entra_user_update",
		"step.entra_user_delete",
		"step.entra_group_create",
		"step.entra_group_get",
		"step.entra_group_list",
		"step.entra_group_add_member",
		"step.entra_group_remove_member",
		"step.entra_application_create",
		"step.entra_application_get",
		"step.entra_application_list",
		"step.entra_application_update",
		"step.entra_application_delete",
		"step.entra_directory_role_list",
		"step.entra_service_principal_list",
	}
}

func createStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	constructor, ok := stepRegistry[typeName]
	if !ok {
		return nil, fmt.Errorf("entra plugin: unknown step type %q", typeName)
	}
	return constructor(name, config)
}

type graphHandler func(context.Context, *msgraphsdk.GraphServiceClient, map[string]any) (map[string]any, error)

type graphStep struct {
	name       string
	moduleName string
	handler    graphHandler
}

func newGraphStep(handler graphHandler) stepConstructor {
	return func(name string, config map[string]any) (sdk.StepInstance, error) {
		moduleName := stringValue(config, "module")
		if moduleName == "" {
			moduleName = "entra"
		}
		return &graphStep{name: name, moduleName: moduleName, handler: handler}, nil
	}
}

func (s *graphStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _, config map[string]any) (*sdk.StepResult, error) {
	client, ok := GetClient(s.moduleName)
	if !ok {
		return &sdk.StepResult{Output: map[string]any{"error": "entra client not found: " + s.moduleName}}, nil
	}
	output, err := s.handler(ctx, client.Graph, mergeMaps(config, current))
	if err != nil {
		return &sdk.StepResult{Output: errResult(err)}, nil
	}
	return &sdk.StepResult{Output: output}, nil
}

func entraUserCreate(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	user, err := buildUser(values, true)
	if err != nil {
		return nil, err
	}
	created, err := client.Users().Post(ctx, user, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"user": userToMap(created)}, nil
}

func entraUserGet(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "user_id", "userId", "id")
	if id == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	user, err := client.Users().ByUserId(id).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"user": userToMap(user)}, nil
}

func entraUserList(ctx context.Context, client *msgraphsdk.GraphServiceClient, _ map[string]any) (map[string]any, error) {
	resp, err := client.Users().Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	users := []map[string]any{}
	if resp != nil {
		for _, user := range resp.GetValue() {
			users = append(users, userToMap(user))
		}
	}
	return map[string]any{"users": users}, nil
}

func entraUserUpdate(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "user_id", "userId", "id")
	if id == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	source := values
	if payload := mapValue(values, "user"); payload != nil {
		source = payload
	}
	user, err := buildUser(source, false)
	if err != nil {
		return nil, err
	}
	updated, err := client.Users().ByUserId(id).Patch(ctx, user, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"updated": true, "user_id": id, "user": userToMap(updated)}, nil
}

func entraUserDelete(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "user_id", "userId", "id")
	if id == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if err := client.Users().ByUserId(id).Delete(ctx, nil); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "user_id": id}, nil
}

func entraGroupCreate(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	group, err := buildGroup(values, true)
	if err != nil {
		return nil, err
	}
	created, err := client.Groups().Post(ctx, group, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"group": groupToMap(created)}, nil
}

func entraGroupGet(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "group_id", "groupId", "id")
	if id == "" {
		return nil, fmt.Errorf("group_id is required")
	}
	group, err := client.Groups().ByGroupId(id).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"group": groupToMap(group)}, nil
}

func entraGroupList(ctx context.Context, client *msgraphsdk.GraphServiceClient, _ map[string]any) (map[string]any, error) {
	resp, err := client.Groups().Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	groupsOut := []map[string]any{}
	if resp != nil {
		for _, group := range resp.GetValue() {
			groupsOut = append(groupsOut, groupToMap(group))
		}
	}
	return map[string]any{"groups": groupsOut}, nil
}

func entraGroupAddMember(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	groupID, memberID, err := groupMemberIDs(values)
	if err != nil {
		return nil, err
	}
	ref := models.NewReferenceCreate()
	odataID := graphDirectoryObjectURL(memberID)
	ref.SetOdataId(&odataID)
	if err := client.Groups().ByGroupId(groupID).Members().Ref().Post(ctx, ref, nil); err != nil {
		return nil, err
	}
	return map[string]any{"added": true, "group_id": groupID, "member_id": memberID}, nil
}

func entraGroupRemoveMember(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	groupID, memberID, err := groupMemberIDs(values)
	if err != nil {
		return nil, err
	}
	odataID := graphDirectoryObjectURL(memberID)
	config := &groups.ItemMembersRefRequestBuilderDeleteRequestConfiguration{
		QueryParameters: &groups.ItemMembersRefRequestBuilderDeleteQueryParameters{Id: &odataID},
	}
	if err := client.Groups().ByGroupId(groupID).Members().Ref().Delete(ctx, config); err != nil {
		return nil, err
	}
	return map[string]any{"removed": true, "group_id": groupID, "member_id": memberID}, nil
}

func entraApplicationCreate(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	app, err := buildApplication(values, true)
	if err != nil {
		return nil, err
	}
	created, err := client.Applications().Post(ctx, app, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"application": applicationToMap(created)}, nil
}

func entraApplicationGet(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "application_id", "applicationId", "id")
	if id == "" {
		return nil, fmt.Errorf("application_id is required")
	}
	app, err := client.Applications().ByApplicationId(id).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"application": applicationToMap(app)}, nil
}

func entraApplicationList(ctx context.Context, client *msgraphsdk.GraphServiceClient, _ map[string]any) (map[string]any, error) {
	resp, err := client.Applications().Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	apps := []map[string]any{}
	if resp != nil {
		for _, app := range resp.GetValue() {
			apps = append(apps, applicationToMap(app))
		}
	}
	return map[string]any{"applications": apps}, nil
}

func entraApplicationUpdate(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "application_id", "applicationId", "id")
	if id == "" {
		return nil, fmt.Errorf("application_id is required")
	}
	source := values
	if payload := mapValue(values, "application"); payload != nil {
		source = payload
	}
	app, err := buildApplication(source, false)
	if err != nil {
		return nil, err
	}
	updated, err := client.Applications().ByApplicationId(id).Patch(ctx, app, nil)
	if err != nil {
		return nil, err
	}
	return map[string]any{"updated": true, "application_id": id, "application": applicationToMap(updated)}, nil
}

func entraApplicationDelete(ctx context.Context, client *msgraphsdk.GraphServiceClient, values map[string]any) (map[string]any, error) {
	id := firstNonEmpty(values, "application_id", "applicationId", "id")
	if id == "" {
		return nil, fmt.Errorf("application_id is required")
	}
	if err := client.Applications().ByApplicationId(id).Delete(ctx, nil); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "application_id": id}, nil
}

func entraDirectoryRoleList(ctx context.Context, client *msgraphsdk.GraphServiceClient, _ map[string]any) (map[string]any, error) {
	resp, err := client.DirectoryRoles().Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	roles := []map[string]any{}
	if resp != nil {
		for _, role := range resp.GetValue() {
			roles = append(roles, directoryRoleToMap(role))
		}
	}
	return map[string]any{"directory_roles": roles}, nil
}

func entraServicePrincipalList(ctx context.Context, client *msgraphsdk.GraphServiceClient, _ map[string]any) (map[string]any, error) {
	resp, err := client.ServicePrincipals().Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	servicePrincipals := []map[string]any{}
	if resp != nil {
		for _, servicePrincipal := range resp.GetValue() {
			servicePrincipals = append(servicePrincipals, servicePrincipalToMap(servicePrincipal))
		}
	}
	return map[string]any{"service_principals": servicePrincipals}, nil
}

func buildUser(values map[string]any, requireCreateFields bool) (models.Userable, error) {
	user := models.NewUser()
	if value := firstNonEmpty(values, "display_name", "displayName"); value != "" {
		user.SetDisplayName(&value)
	}
	if value := firstNonEmpty(values, "user_principal_name", "userPrincipalName"); value != "" {
		user.SetUserPrincipalName(&value)
	}
	if value := firstNonEmpty(values, "mail_nickname", "mailNickname"); value != "" {
		user.SetMailNickname(&value)
	}
	if value := firstNonEmpty(values, "given_name", "givenName"); value != "" {
		user.SetGivenName(&value)
	}
	if value := firstNonEmpty(values, "surname"); value != "" {
		user.SetSurname(&value)
	}
	if value, ok := boolValue(values, "account_enabled", "accountEnabled"); ok {
		user.SetAccountEnabled(&value)
	}
	if password := firstNonEmpty(values, "password", "temporary_password", "temporaryPassword"); password != "" {
		profile := models.NewPasswordProfile()
		profile.SetPassword(&password)
		forceChange := true
		if value, ok := boolValue(values, "force_change_password_next_sign_in", "forceChangePasswordNextSignIn"); ok {
			forceChange = value
		}
		profile.SetForceChangePasswordNextSignIn(&forceChange)
		user.SetPasswordProfile(profile)
	}
	if requireCreateFields {
		if user.GetDisplayName() == nil || *user.GetDisplayName() == "" {
			return nil, fmt.Errorf("display_name is required")
		}
		if user.GetUserPrincipalName() == nil || *user.GetUserPrincipalName() == "" {
			return nil, fmt.Errorf("user_principal_name is required")
		}
		if user.GetMailNickname() == nil || *user.GetMailNickname() == "" {
			return nil, fmt.Errorf("mail_nickname is required")
		}
		if user.GetAccountEnabled() == nil {
			enabled := true
			user.SetAccountEnabled(&enabled)
		}
		if user.GetPasswordProfile() == nil {
			return nil, fmt.Errorf("password or temporary_password is required")
		}
	}
	return user, nil
}

func buildGroup(values map[string]any, requireCreateFields bool) (models.Groupable, error) {
	group := models.NewGroup()
	if value := firstNonEmpty(values, "display_name", "displayName"); value != "" {
		group.SetDisplayName(&value)
	}
	if value := firstNonEmpty(values, "mail_nickname", "mailNickname"); value != "" {
		group.SetMailNickname(&value)
	}
	if value := stringValue(values, "description"); value != "" {
		group.SetDescription(&value)
	}
	if value, ok := boolValue(values, "mail_enabled", "mailEnabled"); ok {
		group.SetMailEnabled(&value)
	}
	if value, ok := boolValue(values, "security_enabled", "securityEnabled"); ok {
		group.SetSecurityEnabled(&value)
	}
	if requireCreateFields {
		if group.GetDisplayName() == nil || *group.GetDisplayName() == "" {
			return nil, fmt.Errorf("display_name is required")
		}
		if group.GetMailNickname() == nil || *group.GetMailNickname() == "" {
			return nil, fmt.Errorf("mail_nickname is required")
		}
		if group.GetMailEnabled() == nil {
			enabled := false
			group.SetMailEnabled(&enabled)
		}
		if group.GetSecurityEnabled() == nil {
			enabled := true
			group.SetSecurityEnabled(&enabled)
		}
	}
	return group, nil
}

func buildApplication(values map[string]any, requireCreateFields bool) (models.Applicationable, error) {
	app := models.NewApplication()
	if value := firstNonEmpty(values, "display_name", "displayName"); value != "" {
		app.SetDisplayName(&value)
	}
	if value := stringValue(values, "description"); value != "" {
		app.SetDescription(&value)
	}
	if value := firstNonEmpty(values, "sign_in_audience", "signInAudience"); value != "" {
		app.SetSignInAudience(&value)
	}
	if items := stringSliceValue(values, "identifier_uris"); len(items) > 0 {
		app.SetIdentifierUris(items)
	} else if items := stringSliceValue(values, "identifierUris"); len(items) > 0 {
		app.SetIdentifierUris(items)
	}
	if items := stringSliceValue(values, "tags"); len(items) > 0 {
		app.SetTags(items)
	}
	if requireCreateFields && (app.GetDisplayName() == nil || *app.GetDisplayName() == "") {
		return nil, fmt.Errorf("display_name is required")
	}
	return app, nil
}

func groupMemberIDs(values map[string]any) (string, string, error) {
	groupID := firstNonEmpty(values, "group_id", "groupId", "id")
	if groupID == "" {
		return "", "", fmt.Errorf("group_id is required")
	}
	memberID := firstNonEmpty(values, "member_id", "memberId", "user_id", "userId")
	if memberID == "" {
		return "", "", fmt.Errorf("member_id is required")
	}
	return groupID, memberID, nil
}

func graphDirectoryObjectURL(id string) string {
	return "https://graph.microsoft.com/v1.0/directoryObjects/" + id
}

func userToMap(user models.Userable) map[string]any {
	if user == nil {
		return nil
	}
	out := map[string]any{
		"id":                  stringPtrValue(user.GetId()),
		"display_name":        stringPtrValue(user.GetDisplayName()),
		"user_principal_name": stringPtrValue(user.GetUserPrincipalName()),
		"mail":                stringPtrValue(user.GetMail()),
		"mail_nickname":       stringPtrValue(user.GetMailNickname()),
		"account_enabled":     boolPtrValue(user.GetAccountEnabled()),
		"user_type":           stringPtrValue(user.GetUserType()),
	}
	return compactMap(out)
}

func groupToMap(group models.Groupable) map[string]any {
	if group == nil {
		return nil
	}
	out := map[string]any{
		"id":               stringPtrValue(group.GetId()),
		"display_name":     stringPtrValue(group.GetDisplayName()),
		"description":      stringPtrValue(group.GetDescription()),
		"mail":             stringPtrValue(group.GetMail()),
		"mail_nickname":    stringPtrValue(group.GetMailNickname()),
		"mail_enabled":     boolPtrValue(group.GetMailEnabled()),
		"security_enabled": boolPtrValue(group.GetSecurityEnabled()),
		"group_types":      group.GetGroupTypes(),
	}
	return compactMap(out)
}

func applicationToMap(app models.Applicationable) map[string]any {
	if app == nil {
		return nil
	}
	out := map[string]any{
		"id":               stringPtrValue(app.GetId()),
		"app_id":           stringPtrValue(app.GetAppId()),
		"display_name":     stringPtrValue(app.GetDisplayName()),
		"description":      stringPtrValue(app.GetDescription()),
		"sign_in_audience": stringPtrValue(app.GetSignInAudience()),
		"identifier_uris":  app.GetIdentifierUris(),
		"tags":             app.GetTags(),
	}
	return compactMap(out)
}

func directoryRoleToMap(role models.DirectoryRoleable) map[string]any {
	if role == nil {
		return nil
	}
	out := map[string]any{
		"id":            stringPtrValue(role.GetId()),
		"display_name":  stringPtrValue(role.GetDisplayName()),
		"description":   stringPtrValue(role.GetDescription()),
		"role_template": stringPtrValue(role.GetRoleTemplateId()),
	}
	return compactMap(out)
}

func servicePrincipalToMap(servicePrincipal models.ServicePrincipalable) map[string]any {
	if servicePrincipal == nil {
		return nil
	}
	out := map[string]any{
		"id":                     stringPtrValue(servicePrincipal.GetId()),
		"app_id":                 stringPtrValue(servicePrincipal.GetAppId()),
		"display_name":           stringPtrValue(servicePrincipal.GetDisplayName()),
		"service_principal_type": stringPtrValue(servicePrincipal.GetServicePrincipalType()),
		"account_enabled":        boolPtrValue(servicePrincipal.GetAccountEnabled()),
		"tags":                   servicePrincipal.GetTags(),
	}
	return compactMap(out)
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values, key); value != "" {
			return value
		}
	}
	return ""
}
