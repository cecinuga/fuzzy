package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"software.sslmate.com/src/go-pkcs12"
)

var apiKey = ""
var certPath = "./cert.pfx"
var certPwd = "leonet1397"

var methodFlag string
var endpointFlag string
var bodyFlag string
var fieldsFlag string

func Init() {
	flag.StringVar(&methodFlag, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&endpointFlag, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&bodyFlag, "bp", "", "[#] HTTP Body request path")
	flag.StringVar(&fieldsFlag, "fp", "", "[#] File with values to fuzz")

	flag.Parse()
}

func main() {
	fmt.Fprintf(os.Stdout, helpManRoutine())

	flag.StringVar(&methodFlag, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&endpointFlag, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&bodyFlag, "bp", "", "[#] HTTP Body request path")
	flag.StringVar(&fieldsFlag, "fp", "", "[#] File with values to fuzz")

	flag.Parse()

	certData, err := os.ReadFile(certPath)
	if err != nil {
		log.Fatalf("Error reading PFX file: %v", err)
	}
	privateKey, certificate, caCerts, err := pkcs12.DecodeChain(certData, certPwd)
	if err != nil {
		log.Fatalf("Error decoding PFX data: %v", err)
	}

	certChain := [][]byte{certificate.Raw}
	for _, caCert := range caCerts {
		certChain = append(certChain, caCert.Raw)
	}

	cert := tls.Certificate{
		Certificate: certChain,
		PrivateKey:  privateKey,
	}

	rootCAs := x509.NewCertPool()
	for _, caCert := range caCerts {
		rootCAs.AddCert(caCert)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            rootCAs, // Optional: if you need to trust those CA certs
		InsecureSkipVerify: true,    // Don't use unless testing!
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	switch methodFlag {
	case http.MethodPost:
		fuzzPost(client, endpointFlag, bodyFlag, fieldsFlag)
	case http.MethodGet:
		fuzzGet(client, endpointFlag, fieldsFlag)
	}
}

func fuzzPost(client *http.Client, endpoint string, bodyPath string, fieldValuesPath string) {
	fieldValuesFilename := filepath.Base(fieldValuesPath)
	keyToValorize := strings.Replace(fieldValuesFilename, ".txt", "", 1)

	bodyBuf, err := os.ReadFile(bodyPath)
	if err != nil {
		log.Fatalf("Error reading bidy file: %v", err)
	}
	var bodyJson map[string]any
	json.Unmarshal(bodyBuf, &bodyJson)

	valuesFile, err := os.Open(fieldValuesPath)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer valuesFile.Close()

	found := false
	foundedValue := ""

	valuesScanner := bufio.NewScanner(valuesFile)
	for valuesScanner.Scan() {
		value := valuesScanner.Text()
		bodyJson[keyToValorize] = value
		bodyBuf, err := json.Marshal(bodyJson)
		if err != nil {
			log.Fatalf("Error Unmarshalling file: %v", err)
		}
		bodyReader := bytes.NewReader(bodyBuf)

		req, err := http.NewRequest(http.MethodPost, endpoint, bodyReader)
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

		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Reading response body failed: %v", err)
		}

		if res.StatusCode != http.StatusNotFound {
			found = true
			foundedValue = value
		}

		fmt.Printf("[+] Request body {... '%v':%v ... }\n", keyToValorize, value)
		fmt.Printf("[+] Response status: %s\n", res.Status)
		fmt.Printf("%v\n\n\n", string(resBody))
	}
	if err := valuesScanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}

	fmt.Printf("[#] Something Found: %v\n", found)
	if found {
		fmt.Printf("[+]\t%v: %v", keyToValorize, foundedValue)
	}
}

func fuzzGet(client *http.Client, endpoint string, fieldValuesPath string) {
	valuesFile, err := os.Open(fieldValuesPath)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer valuesFile.Close()

	found := false
	foundedValue := ""

	valuesScanner := bufio.NewScanner(valuesFile)
	for valuesScanner.Scan() {
		value := valuesScanner.Text()

		bodyReader := bytes.NewReader(nil)
		endpointWithParam := fmt.Sprintf("%v%v", endpoint, value)
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

		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalf("Reading response body failed: %v", err)
		}

		if res.StatusCode != http.StatusNotFound {
			found = true
			foundedValue = value
		}

		fmt.Printf("[+] Endpoint: %v\n", endpointWithParam)
		fmt.Printf("[+] Response status: %s\n", res.Status)
		fmt.Printf("%v\n\n", string(resBody))
	}
	if err := valuesScanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}

	fmt.Printf("[#] Something Found: %v\n", found)
	if found {
		fmt.Printf("[+]\t%v%v", endpoint, foundedValue)
	}
}

func helpManRoutine() string {
	helpMan := "[#] -m\tHTTP Request Method\n[#] -e\tEndpoint u wanna call\n[#] -bp\tHTTP Body request path\n[#] -fp\tFile with values to fuzz\n\nUsage: fuzzy.go -m GET -e <endpoint> file.txt"

	if len(os.Args) == 1 {
		return helpMan
	}
	for _, flag := range os.Args {
		if flag == "--help" {
			return helpMan
		}
	}
	return ""
}
