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
var fieldsFlag string
var insecureFlag bool

func main() {
	flag.StringVar(&endpointFlag, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&methodFlag, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&bodyFlag, "bp", "", "[#] HTTP Body request path")
	flag.StringVar(&fieldsFlag, "fp", "", "[#] File with values to fuzz")
	flag.BoolVar(&insecureFlag, "k", false, "[#] Skip TLS certificate verification (insecure)")

	flag.Parse()

	fuzz()
}

func parseBody(key string, value string) *bytes.Reader {
	bodyJson := make(map[string]any)
	bodyJson[key] = value

	bodyBuf, err := json.Marshal(bodyJson)
	if err != nil {
		log.Fatalf("Error Unmarshalling file: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBuf)

	return bodyReader
}

func fuzz() {
	transport := &http.Transport{}
	if insecureFlag {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	
	client := &http.Client{
		Transport: transport,
	}

	fieldValuesFilename := filepath.Base(fieldsFlag)
	keyToValorize := strings.Replace(fieldValuesFilename, ".txt", "", 1)

	valuesFile, err := os.Open(fieldsFlag)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer valuesFile.Close()

	valuesScanner := bufio.NewScanner(valuesFile)
	for valuesScanner.Scan() {
		value := valuesScanner.Text()

		bodyReader := parseBody(keyToValorize, value)

		req, err := http.NewRequest(http.MethodPost, endpointFlag, bodyReader)
		if err != nil {
			log.Fatal(err)
		}
		req.Method = http.MethodPost
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-API-KEY", apiKey)

		res, err := client.Do(req)

		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		/*resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Reading response body failed: %v", err)
		}*/

		fmt.Printf("[+] Request body {... '%v':%v ... }\n", keyToValorize, value)
		fmt.Printf("[+] Response status: %s\n\n", res.Status)
		//fmt.Printf("%v\n\n\n", string(resBody))
	}
	if err := valuesScanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}
