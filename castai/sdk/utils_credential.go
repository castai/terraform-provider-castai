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

func ToCloudCredentialsAWS(accessKeyId string, secretAccessKey string) (string, error) {
	credentialsJson, err := json.Marshal(struct {
		AccessKeyId     string `json:"accessKeyId"`
		SecretAccessKey string `json:"secretAccessKey"`
	}{
		AccessKeyId:     accessKeyId,
		SecretAccessKey: secretAccessKey,
	})
	if err != nil {
		return "", fmt.Errorf("building aws credentials json: %v", err)
	}

	return string(credentialsJson), nil
}

func ToCloudCredentialsDO(token string) (string, error) {
	credentialsJson, err := json.Marshal(struct {
		Token string `json:"token"`
	}{
		Token: token,
	})
	if err != nil {
		return "", fmt.Errorf("building do credentials json: %v", err)
	}

	return string(credentialsJson), nil
}

func ToCloudCredentialsAzure(clientID, clientSecret, tenantID, subscriptionID string) (string, error) {
	credentialsJson, err := json.Marshal(AzureCredentials{
		ClientID: clientID,
		ClientSecret: clientSecret,
		TenantID: tenantID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		return "", fmt.Errorf("building Azure credentials json: %v", err)
	}

	return string(credentialsJson), nil
}
