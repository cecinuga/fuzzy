package fuzzer

import (
	"log"
	"bytes"
	"net/http"
	"encoding/json"
	"fuzzy/internal/config"
)

func BuildRequest(cfg *config.Config, body map[string]any) *http.Request {
	bodyBuf, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("Error Unmarshalling file: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBuf)

	req, err := http.NewRequest(cfg.Method, cfg.Endpoint, bodyReader)
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