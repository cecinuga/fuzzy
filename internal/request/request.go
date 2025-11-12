package request

import (
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/internal/utils"
	"net/http"
	"strings"
)

func BuildRequest(cfg *config.Config, body map[string]any, queryParams string) (*http.Request, error) {
	var encodedEndpoint string

	encodedEndpoint = strings.Join([]string{cfg.Endpoint, queryParams}, "")

	bodyReader := utils.MarshalJson(body)

	req, err := http.NewRequest(cfg.Method, encodedEndpoint, bodyReader)
	
	if err != nil {
		return nil, fmt.Errorf("Error creating http client.")
	}
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func SendRequest(client *http.Client, req *http.Request) (string, error) {
	res, err := client.Do(req)

	if err != nil {
		return "", fmt.Errorf("Request not send! %v", req.URL)
	}
	defer res.Body.Close()

	return res.Status, nil
}