package fuzzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/utils"
	"log"
	"net/http"
	"os"
	"strings"
)

func Run(cfg *config.Config, client *http.Client) {
	bodyMap := make(map[string]any)
	data := []byte{}
	var err error

	if !utils.IsJSON(cfg.Body){
		data, err = os.ReadFile(cfg.Body)
		if err != nil {
			log.Fatalf("[!] %v", err)
		}
		
	} else {
		data = []byte(cfg.Body)
	}
	json.Unmarshal(data, &bodyMap)

	dictName, dictFile := GetFile(cfg.Dictionary)
	fuzzKey := strings.Replace(dictName, ".txt", "", 1)

	defer dictFile.Close()

	values := bufio.NewScanner(dictFile)

	for values.Scan() {
		fuzzValue := values.Text()
		bodyMap[fuzzKey] = fuzzValue

		req := BuildRequest(cfg, bodyMap)
		res, err := client.Do(req)

		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()

		fmt.Printf("[+] Response status: %s\n\n", res.Status)
	}
	if err := values.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}

func sendRequest(){
	
}
