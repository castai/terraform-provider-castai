package sdk

import (
	"encoding/json"
	"fmt"
)

type AzureCredentials struct {
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret,omitempty"`
	FederationID   string `json:"federationId,omitempty"`
	SubscriptionID string `json:"subscriptionId"`
	TenantID       string `json:"tenantId"`
}

func ToCloudCredentialsAzure(clientID, clientSecret, federationID, tenantID, subscriptionID string) (string, error) {
	credentialsJson, err := json.Marshal(AzureCredentials{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		FederationID:   federationID,
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		return "", fmt.Errorf("building Azure credentials json: %v", err)
	}

	return string(credentialsJson), nil
}
