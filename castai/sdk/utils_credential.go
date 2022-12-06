package sdk

import (
	"encoding/json"
	"fmt"
)

type AzureCredentials struct {
	ClientID       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`
	TenantID       string `json:"tenantId"`
	SubscriptionID string `json:"subscriptionId"`
}

func ToCloudCredentialsAzure(clientID, clientSecret, tenantID, subscriptionID string) (string, error) {
	credentialsJson, err := json.Marshal(AzureCredentials{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		return "", fmt.Errorf("building Azure credentials json: %v", err)
	}

	return string(credentialsJson), nil
}
