package internal

import (
	"sync"

	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

type EntraClient struct {
	Graph    *msgraphsdk.GraphServiceClient
	TenantID string
}

var (
	clientMu       sync.RWMutex
	clientRegistry = map[string]*EntraClient{}
)

func RegisterClient(name string, client *EntraClient) {
	clientMu.Lock()
	defer clientMu.Unlock()
	clientRegistry[name] = client
}

func GetClient(name string) (*EntraClient, bool) {
	clientMu.RLock()
	defer clientMu.RUnlock()
	client, ok := clientRegistry[name]
	return client, ok
}

func UnregisterClient(name string) {
	clientMu.Lock()
	defer clientMu.Unlock()
	delete(clientRegistry, name)
}
