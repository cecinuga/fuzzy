package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var apiKey = ""

var methodFlag string
var endpointFlag string
var bodyFlag string
var dictionaryFlag string
var insecureFlag bool

func main() {
	flag.StringVar(&endpointFlag, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&methodFlag, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&bodyFlag, "bp", "", "[#] HTTP Body request path")
	flag.StringVar(&dictionaryFlag, "dict", "", "[#] Dictionary file")
	flag.BoolVar(&insecureFlag, "k", false, "[#] Skip TLS certificate verification (insecure)")

	flag.Parse()

	transport := &http.Transport{}
	if insecureFlag {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	
	client := &http.Client{
		Transport: transport,
	}

	fuzz(client)
}

func parseBody(key, value string) *bytes.Reader {
	bodyJson := make(map[string]any)
	bodyJson[key] = value

	bodyBuf, err := json.Marshal(bodyJson)
	if err != nil {
		log.Fatalf("Error Unmarshalling file: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBuf)

	return bodyReader
}

func getDictionaryFile() (name string, file *os.File) {
	name = filepath.Base(dictionaryFlag)

	file, err := os.Open(dictionaryFlag)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}

	return name, file
}

func buildRequest(fuzzKey, fuzzValue string) *http.Request {
	body := parseBody(fuzzKey, fuzzValue)

	req, err := http.NewRequest(methodFlag, endpointFlag, body)
	if err != nil {
		log.Fatal(err)
	}
	req.Method = methodFlag
	req.Header.Add("Content-Type", "application/json")

	return req
}

func fuzz(client *http.Client) {
	dictName, dictFile := getDictionaryFile()
	fuzzKey:= strings.Replace(dictName, ".txt", "", 1)
	defer dictFile.Close()

	valuesScanner := bufio.NewScanner(dictFile)
	for valuesScanner.Scan() {
		fuzzValue := valuesScanner.Text()

		req := buildRequest(fuzzKey, fuzzValue)
		res, err := client.Do(req)

		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		fmt.Printf("[+] Request body {... '%v':%v ... }\n", fuzzKey, fuzzValue)
		fmt.Printf("[+] Response status: %s\n\n", res.Status)
	}
	if err := valuesScanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}
