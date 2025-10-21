package fuzzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/utils"
	"log"
	"net/http"
	"sync"
)

func Run(cfg *config.Config, client *http.Client) {
	body := make(map[string]any) 

	if !utils.IsJson(cfg.Body){ 
		utils.LoadJsonFile(cfg.Body, &body)
	} else {
		data := []byte(cfg.Body)
		json.Unmarshal(data, &body)
	}

	_, dictFile := utils.GetFile(cfg.Dictionary)
	defer dictFile.Close()

	objToFuzz, fuzzKey := getFuzzValuePointer(&body, cfg.FuzzyKey)

	responses := make(chan string)

	var chGroup sync.WaitGroup

	values := bufio.NewScanner(dictFile)
	for values.Scan() {
		chGroup.Add(1)
		value := values.Text()
		
		go func(value string){
			defer chGroup.Done()
			
			(*objToFuzz)[fuzzKey] = value

			req := BuildRequest(cfg, body)
			response := SendRequest(client, req)

			responses <- response
		}(value)
	}

	go func(){
		chGroup.Wait()
		close(responses)
	}()
	
	for status := range responses {
		fmt.Printf("[+] Response status: %v\n", status)
	}
	
	if err := values.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}

func getFuzzValuePointer(body *map[string]any, fuzzValue string) (*map[string]any, string) {
	for k, v := range *body{
		if v == fuzzValue {
			return body, k
		}

		childBody, ok := v.(map[string]any)

		if ok {
			return getFuzzValuePointer(&childBody, fuzzValue)
		}
	}
	return body, ""
}