package main

import (
	"bufio"
	"bytes"
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

func main() {
	flag.StringVar(&methodFlag, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&endpointFlag, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&bodyFlag, "bp", "", "[#] HTTP Body request path")
	flag.StringVar(&fieldsFlag, "fp", "", "[#] File with values to fuzz")

	flag.Parse()

	fuzz()
}

func fuzz() {
	client := &http.Client{}
	fieldValuesFilename := filepath.Base(fieldsFlag)
	keyToValorize := strings.Replace(fieldValuesFilename, ".txt", "", 1)

	switch methodFlag {
	case http.MethodPost:
		fuzzPost(client, fieldValuesFilename, keyToValorize)
	case http.MethodGet:
		fuzzGet(client, fieldValuesFilename, keyToValorize)
	}
}

func fuzzPost(client *http.Client, fieldValuesFilename, keyToValorize string) {
	bodyBuf, err := os.ReadFile(bodyFlag)
	if err != nil {
		log.Fatalf("Error reading bidy file: %v", err)
	}
	var bodyJson map[string]any
	json.Unmarshal(bodyBuf, &bodyJson)

	valuesFile, err := os.Open(fieldsFlag)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer valuesFile.Close()

	valuesScanner := bufio.NewScanner(valuesFile)
	for valuesScanner.Scan() {
		value := valuesScanner.Text()

		bodyJson[keyToValorize] = value
		bodyBuf, err := json.Marshal(bodyJson)
		if err != nil {
			log.Fatalf("Error Unmarshalling file: %v", err)
		}
		bodyReader := bytes.NewReader(bodyBuf)

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

func fuzzGet(client *http.Client, fieldValuesFilename, keyToValorize string) {
	valuesFile, err := os.Open(fieldsFlag)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer valuesFile.Close()

	valuesScanner := bufio.NewScanner(valuesFile)
	for valuesScanner.Scan() {
		value := valuesScanner.Text()

		bodyReader := bytes.NewReader(nil)
		endpointWithParam := fmt.Sprintf("%v%v", endpointFlag, value)
		req, err := http.NewRequest(http.MethodGet, endpointWithParam, bodyReader)
		if err != nil {
			log.Fatal(err)
		}
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

		fmt.Printf("[+] Endpoint: %v\n", endpointWithParam)
		fmt.Printf("[+] Response status: %s\n\n", res.Status)
		//fmt.Printf("%v\n\n", string(resBody))
	}
	if err := valuesScanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}
