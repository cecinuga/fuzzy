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

	dictScanner := bufio.NewScanner(dictFile)

	toFuzzObj, toFuzzKey := getFuzzValuePointer(&body, cfg.FuzzyKey)
	toFuzz := (*toFuzzObj)[toFuzzKey].(string)

	spawner(cfg, client, &body, dictScanner, &toFuzz)

}

func getFuzzValuePointer(body *map[string]any, fuzzValue string) (*map[string]any, string) { // GESTIRE GLI ERRORI
	for k, v := range *body{
		if v == fuzzValue {
			return body, k
		}

		childBody, ok := v.(map[string]any)

		if ok {
			childToFuzz, childToFuzzKey := getFuzzValuePointer(&childBody, fuzzValue)
			if len(childToFuzzKey) > 0 && (*childToFuzz)[childToFuzzKey].(string) == fuzzValue {
				return childToFuzz, childToFuzzKey
			}
		}
	}
	return body, "" // QUANDO NON TROVI UNA FUZZY KEY LANCIA UN ERRORE
}

func spawner(cfg *config.Config, client *http.Client, body *map[string]any, scanner *bufio.Scanner, toFuzz *string){
	var chGroup sync.WaitGroup
	responses := make(chan string)

	for scanner.Scan() {
		chGroup.Add(1)
		value := scanner.Text()
		
		go func(value string){
			defer chGroup.Done()
			
			*toFuzz = value

			req := BuildRequest(cfg, *body)
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
	
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}