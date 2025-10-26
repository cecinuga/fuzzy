package request

import (
	"fuzzy/internal/config"
	"fuzzy/utils/query"
	"fuzzy/utils/json"
	"log"
	"net/http"
	"strings"
)

func BuildRequest(cfg *config.Config, body, queryParams map[string]any) *http.Request {
	var encodedEndpoint string

	encodedQuery := query.Encode(queryParams)
	encodedEndpoint = strings.Join([]string{string(cfg.Endpoint), encodedQuery}, "")

	bodyReader := json.Marshal(body)

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