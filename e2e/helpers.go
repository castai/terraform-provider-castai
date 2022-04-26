package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"

	"github.com/castai/terraform-provider-castai/castai/sdk"
)

func createClient(apiURL, apiToken string) (*sdk.ClientWithResponses, error) {
	httpClientOption := func(client *sdk.Client) error {
		client.Client = &http.Client{
			Transport: logging.NewTransport("CAST.AI", http.DefaultTransport),
			Timeout:   1 * time.Minute,
		}
		return nil
	}

	apiTokenOption := sdk.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-API-Key", apiToken)
		return nil
	})

	apiClient, err := sdk.NewClientWithResponses(apiURL, httpClientOption, apiTokenOption)
	if err != nil {
		return nil, err
	}

	if checkErr := sdk.CheckGetResponse(apiClient.ListAuthTokensWithResponse(context.Background(), &sdk.ListAuthTokensParams{})); checkErr != nil {
		return nil, fmt.Errorf("validating api token (by listing auth tokens): %v", checkErr)
	}

	return apiClient, nil
}

func createVarsFile(vars map[string]interface{}, testName string) (string, error) {
	file, err := ioutil.TempFile("", testName)
	if err != nil {
		return "", err
	}

	defer file.Close()
	body := ""
	for k, v := range vars {
		body += fmt.Sprintf("%s=\"%s\"\n", k, v)
	}
	body += "\n"

	file.WriteString(body)
	fmt.Println("Path", file.Name())
	return file.Name(), nil
}
