package internal

import (
	"context"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-entra/internal/contracts"
	pb "github.com/GoCodeAlone/workflow/plugin/external/proto"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestModuleInitRegistersGraphClient(t *testing.T) {
	module, err := newEntraModule("entra-test", map[string]any{
		"tenantId":     "example.onmicrosoft.com",
		"clientId":     "00000000-0000-0000-0000-000000000001",
		"clientSecret": "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err != nil {
		t.Fatal(err)
	}
	client, ok := GetClient("entra-test")
	if !ok || client == nil {
		t.Fatal("expected registered client")
	}
	if client.Graph == nil {
		t.Fatal("expected graph client")
	}
	if client.TenantID != "example.onmicrosoft.com" {
		t.Fatalf("tenant id = %q", client.TenantID)
	}
	if err := module.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetClient("entra-test"); ok {
		t.Fatal("expected client to be unregistered")
	}
}

func TestModuleInitRequiresCredentials(t *testing.T) {
	module, err := newEntraModule("entra-test", map[string]any{"tenantId": "example.onmicrosoft.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := module.Init(); err == nil {
		t.Fatal("expected missing credentials error")
	}
}

func TestContractRegistryIncludesStrictProtoDescriptors(t *testing.T) {
	provider, ok := NewEntraPlugin().(interface {
		ContractRegistry() *pb.ContractRegistry
	})
	if !ok {
		t.Fatal("plugin does not expose ContractRegistry")
	}
	registry := provider.ContractRegistry()
	if registry == nil || registry.GetFileDescriptorSet() == nil {
		t.Fatal("missing contract registry file descriptors")
	}
	contractsByType := map[string]*pb.ContractDescriptor{}
	for _, contract := range registry.GetContracts() {
		switch contract.GetKind() {
		case pb.ContractKind_CONTRACT_KIND_MODULE:
			contractsByType["module:"+contract.GetModuleType()] = contract
		case pb.ContractKind_CONTRACT_KIND_STEP:
			contractsByType["step:"+contract.GetStepType()] = contract
		}
	}
	module := contractsByType["module:entra.provider"]
	if module == nil || module.GetConfigMessage() != "workflow.plugins.entra.v1.ProviderConfig" {
		t.Fatalf("unexpected module contract: %#v", module)
	}
	for _, stepType := range allStepTypes() {
		contract := contractsByType["step:"+stepType]
		if contract == nil {
			t.Fatalf("missing step contract %s", stepType)
		}
		if contract.GetMode() != pb.ContractMode_CONTRACT_MODE_STRICT_PROTO {
			t.Fatalf("%s mode = %v", stepType, contract.GetMode())
		}
	}
}

func TestDescriptorAdvertisesBackedCapabilities(t *testing.T) {
	step, err := newAuthProviderDescribeStep("describe", map[string]any{"tenantId": "example.onmicrosoft.com"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	providers := result.Output["providers"].([]map[string]any)
	if len(providers) != 1 {
		t.Fatalf("providers = %#v", providers)
	}
	provider := providers[0]
	categories := stringSet(provider["categories"].([]string))
	for _, category := range []string{"identity_management", "oauth2_oidc", "enterprise_sso", "directory_roles"} {
		if !categories[category] {
			t.Fatalf("missing category %q", category)
		}
	}
	if categories["rbac"] {
		t.Fatal("descriptor must not advertise RBAC role management without role mutation steps")
	}
	if categories["mfa"] {
		t.Fatal("descriptor must not advertise MFA management without Entra MFA steps")
	}
	if categories["scim"] {
		t.Fatal("descriptor must not advertise SCIM without provisioning steps")
	}
	capabilities := provider["capabilities"].([]map[string]any)
	if len(capabilities) != 5 {
		t.Fatalf("capability count = %d, want 5", len(capabilities))
	}
	for _, capability := range capabilities {
		if capability["supported"] != true {
			t.Fatalf("%s supported = %#v", capability["key"], capability["supported"])
		}
	}
}

func TestTypedDescriptor(t *testing.T) {
	result, err := typedAuthProviderDescribe(context.Background(), sdk.TypedStepRequest[*contracts.AuthProviderDescribeConfig, *contracts.AuthProviderDescribeInput]{
		Config: &contracts.AuthProviderDescribeConfig{ProviderId: "entra-admin"},
		Input:  &contracts.AuthProviderDescribeInput{TenantId: "example.onmicrosoft.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output == nil || len(result.Output.GetProviders()) != 1 {
		t.Fatalf("providers = %#v", result.Output)
	}
	if result.Output.GetProviders()[0].GetId() != "entra-admin" {
		t.Fatalf("provider id = %q", result.Output.GetProviders()[0].GetId())
	}
}

func TestMissingClientReturnsStepError(t *testing.T) {
	step, err := createStep("step.entra_user_get", "get", map[string]any{"module": "missing"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := step.Execute(context.Background(), nil, nil, nil, nil, map[string]any{"user_id": "entra|123"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output["error"] == nil {
		t.Fatalf("expected error output, got %#v", result.Output)
	}
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}
