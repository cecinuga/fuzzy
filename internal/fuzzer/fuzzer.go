package fuzzer

import (
	"bufio"
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/internal/request"
	"fuzzy/utils"
	"log"
	"net/http"
	"sync"
)

func Run(cfg *config.Config, client *http.Client) {
	body := request.FuzzTarget{}
	queryParams := request.FuzzTarget{}

	// Controlla se il body è stato fornito
	if bodyStr := string(cfg.Body); bodyStr != "" {
		body.BuildData(bodyStr)
		body.BuildPointer(string(cfg.FuzzyKey))
	}

	if queryStr := string(cfg.QueryParameters); queryStr != "" {
		queryParams.BuildData(queryStr)
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
		body request.FuzzTarget ){

	var chGroup sync.WaitGroup
	var bodyMutex sync.Mutex

	responses := make(chan string)

	for scanner.Scan() {
		chGroup.Add(1)
		value := scanner.Text()
		
		go func(value string){
			defer chGroup.Done()
			
			bodyMutex.Lock()

			body.SetTarget(value)
			bodyData := body.Get()
			req := request.BuildRequest(cfg, bodyData)
			
			bodyMutex.Unlock()

			response := request.SendRequest(client, req)

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
