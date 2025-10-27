package request

import (
	"fuzzy/internal/config"
	"fuzzy/internal/utils"
	"log"
	"net/http"
	"strings"
)

func BuildRequest(cfg *config.Config, body, queryParams map[string]any) *http.Request {
	var encodedEndpoint string

	encodedQuery := utils.EncodeQuery(queryParams)
	encodedEndpoint = strings.Join([]string{string(cfg.Endpoint), encodedQuery}, "")

	bodyReader := utils.MarshalJson(body)

	req, err := http.NewRequest(string(cfg.Method), encodedEndpoint, bodyReader)
	
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	return req
}

func SendRequest(client *http.Client, req *http.Request) string {
	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	return res.Status
}