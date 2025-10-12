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

func GetFile(path string) (name string, file *os.File) {
	name = filepath.Base(path)

	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer file.Close()

	return name, file
}