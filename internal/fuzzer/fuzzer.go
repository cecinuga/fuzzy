package fuzzer

import (
	"bufio"
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/utils"
	"log"
	"net/http"
	"sync"
)

func Run(cfg *config.Config, client *http.Client) {
	body := config.FuzzTarget{}

	// Controlla se il body è stato fornito
	if bodyStr := string(cfg.Body.Source.(config.HttpBodyJson)); bodyStr != "" {
		body.BuildDataFromJson(bodyStr)
		body.BuildPointer(string(cfg.FuzzyKey))
	}

	_, dictFile := utils.GetFile(string(cfg.Dictionary))
	defer dictFile.Close()

	dictScanner := bufio.NewScanner(dictFile)

	spawner(cfg, client, dictScanner, body) 
}

func spawner(
		cfg *config.Config, 
		client *http.Client, 
		scanner *bufio.Scanner, 
		body config.FuzzTarget ){

	var chGroup sync.WaitGroup
	var bodyMutex sync.Mutex

	responses := make(chan string)

	for scanner.Scan() {
		chGroup.Add(1)
		value := scanner.Text()
		
		go func(value string){
			defer chGroup.Done()
			
			bodyMutex.Lock()

			body.Assign(value)
			req := BuildRequest(cfg, body.Data)
			
			bodyMutex.Unlock()

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
