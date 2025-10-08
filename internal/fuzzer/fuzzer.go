package fuzzer

import (
	"bufio"
	"fmt"
	"fuzzy/internal/config"
	"log"
	"net/http"
	"strings"
)

func Run(cfg *config.Config, client *http.Client) {
	dictName, dictFile := GetDictionary(cfg.Dictionary)
	fuzzKey := strings.Replace(dictName, ".txt", "", 1)

	valuesScanner := bufio.NewScanner(dictFile)
	bodyMap := make(map[string]any)

	for valuesScanner.Scan() {
		fuzzValue := valuesScanner.Text()
		bodyMap[fuzzKey] = fuzzValue

		req := BuildRequest(cfg, bodyMap)
		res, err := client.Do(req)

		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		fmt.Printf("[+] Response status: %s\n\n", res.Status)
	}
	if err := valuesScanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}
