package fuzzer

import (
	"bytes"
	"encoding/json"
	"fuzzy/internal/config"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func parseBody(body map[string]any) *bytes.Reader { 
	bodyBuf, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("Error Unmarshalling file: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBuf)

	return bodyReader
}

func GetDictionary(dictPath string) (name string, file *os.File) {
	name = filepath.Base(dictPath)

	file, err := os.Open(dictPath)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer file.Close()

	return name, file
}

func BuildRequest(cfg *config.Config, body map[string]any) *http.Request {
	bodyBuf := parseBody(body)

	req, err := http.NewRequest(cfg.Method, cfg.Endpoint, bodyBuf)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	return req
}