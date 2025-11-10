package request

import (
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/internal/utils"
	"net/http"
	"strings"
)

func BuildRequest(cfg *config.Config, body, queryParams map[string]any) (*http.Request, error) {
	var encodedEndpoint string

	encodedQuery := utils.EncodeQuery(queryParams)
	encodedEndpoint = strings.Join([]string{string(cfg.Endpoint), encodedQuery}, "")

	bodyReader := utils.MarshalJson(body)

	req, err := http.NewRequest(string(cfg.Method), encodedEndpoint, bodyReader)
	
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