package sdk

import (
	"fmt"
	"io"
	"net/http"
)

func CheckGetResponse(response *http.Response, err error) error {
	return checkResponse(response, err, http.StatusOK)
}

func CheckOKResponse(response *http.Response, err error) error {
	return checkResponse(response, err, http.StatusOK)
}

func CheckResponseNoContent(response *http.Response, err error) error {
	return checkResponse(response, err, http.StatusNoContent)
}

func StatusOk(resp *http.Response) error {
	return checkResponse(resp, nil, http.StatusOK)
}

func checkResponse(response *http.Response, err error, expectedStatus int) error {
	if err != nil {
		return err
	}

	if response.StatusCode != expectedStatus {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("expected status code %d, received: status=%d body=<failed to read body>", expectedStatus, response.StatusCode)
		}
		response.Body.Close()
		return fmt.Errorf("expected status code %d, received: status=%d body=%s", expectedStatus, response.StatusCode, string(body))
	}

	return nil
}
