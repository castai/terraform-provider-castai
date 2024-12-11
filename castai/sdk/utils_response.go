package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func CheckGetResponse(response Response, err error) error {
	return checkResponse(response, err, http.StatusOK)
}

func CheckOKResponse(response Response, err error) error {
	return checkResponse(response, err, http.StatusOK)
}

func CheckResponseNoContent(response Response, err error) error {
	return checkResponse(response, err, http.StatusNoContent)
}

func CheckRawResponseNoContent(response *http.Response, err error) error {
	return checkRawResponse(response, err, http.StatusNoContent)
}

func CheckResponseCreated(response Response, err error) error {
	return checkResponse(response, err, http.StatusCreated)
}

func StatusOk(resp Response) error {
	return checkResponse(resp, nil, http.StatusOK)
}

func checkResponse(response Response, err error, expectedStatus int) error {
	if err != nil {
		return err
	}

	if response.StatusCode() != expectedStatus {
		return fmt.Errorf("expected status code %d, received: status=%d body=%s", expectedStatus, response.StatusCode(), string(response.GetBody()))
	}

	return nil
}

func checkRawResponse(response *http.Response, err error, expectedStatus int) error {
	if err != nil {
		return err
	}

	if response.StatusCode != expectedStatus {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}
		return fmt.Errorf("expected status code %d, received: status=%d body=%s", expectedStatus, response.StatusCode, body)
	}
	return nil
}

type ErrorResponse struct {
	Message         string `json:"message"`
	FieldViolations []struct {
		Field       string `json:"field"`
		Description string `json:"description"`
	} `json:"fieldViolations"`
}

func IsCredentialsError(response Response) bool {
	buf := response.GetBody()

	var errResponse ErrorResponse
	err := json.Unmarshal(buf, &errResponse)
	if err != nil {
		return false
	}

	return errResponse.Message == "Forbidden" && len(errResponse.FieldViolations) > 0 && errResponse.FieldViolations[0].Field == "credentials"
}
