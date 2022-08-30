package sdk

import (
	"fmt"
	"net/http"
)

func CheckGetResponse(response Response, err error) error {
	return checkResponse(response, err, http.StatusOK)
}

func CheckCreateResponse(response Response, err error) error {
	return checkResponse(response, err, http.StatusCreated)
}

func CheckOKResponse(response Response, err error) error {
	return checkResponse(response, err, http.StatusOK)
}

func CheckResponseNoContent(response Response, err error) error {
	return checkResponse(response, err, http.StatusNoContent)
}

func CheckDeleteResponse(response Response, err error) error {
	return CheckResponseNoContent(response, err)
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
