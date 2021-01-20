package sdk

import (
	"encoding/json"
	"fmt"
)

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
