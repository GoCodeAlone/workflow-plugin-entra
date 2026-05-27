package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

const graphDefaultScope = "https://graph.microsoft.com/.default"

type entraModule struct {
	name   string
	config map[string]any
}

func newEntraModule(name string, config map[string]any) (*entraModule, error) {
	return &entraModule{name: name, config: config}, nil
}

func (m *entraModule) Init() error {
	tenantID := firstNonEmpty(m.config, "tenant_id", "tenantId", "tenant")
	if tenantID == "" {
		return fmt.Errorf("entra.provider %q: tenant_id is required", m.name)
	}
	tenantID = normalizeTenantID(tenantID)
	clientID := firstNonEmpty(m.config, "client_id", "clientId")
	if clientID == "" {
		return fmt.Errorf("entra.provider %q: client_id is required", m.name)
	}
	clientSecret := firstNonEmpty(m.config, "client_secret", "clientSecret")
	if clientSecret == "" {
		return fmt.Errorf("entra.provider %q: client_secret is required", m.name)
	}

	credential, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return fmt.Errorf("entra.provider %q: create client secret credential: %w", m.name, err)
	}
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(credential, []string{graphDefaultScope})
	if err != nil {
		return fmt.Errorf("entra.provider %q: create Microsoft Graph client: %w", m.name, err)
	}
	RegisterClient(m.name, &EntraClient{Graph: client, TenantID: tenantID})
	return nil
}

func (m *entraModule) Start(context.Context) error { return nil }

func (m *entraModule) Stop(context.Context) error {
	UnregisterClient(m.name)
	return nil
}

func normalizeTenantID(tenantID string) string {
	return strings.Trim(strings.TrimSpace(tenantID), "/")
}
