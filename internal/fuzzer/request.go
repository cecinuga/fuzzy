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

func CreateBody(key, value string) *bytes.Reader {
	body := make(map[string]any) 
	body[key] = value

	bodyBuf, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("Error Unmarshalling file: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBuf)

	return bodyReader
}

func UpdateBody(body map[string]any, key, value string) *bytes.Reader {
	body[key] = value

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

func BuildRequest(cfg *config.Config, fuzzKey, fuzzValue string) *http.Request {
	body := CreateBody(fuzzKey, fuzzValue)

	req, err := http.NewRequest(cfg.Method, cfg.Endpoint, body)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")

	return req
}