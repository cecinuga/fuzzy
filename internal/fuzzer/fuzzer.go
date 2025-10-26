package fuzzer

import (
	"bufio"
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/internal/request"
	"log"
	"net/http"
	"os"
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
		queryParams.BuildPointer(string(cfg.FuzzyKey))
	}

	dictFile, err := os.Open(string(cfg.Dictionary))
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}

	defer dictFile.Close()

	dictScanner := bufio.NewScanner(dictFile)

	spawner(cfg, client, dictScanner, body, queryParams) 
}

func spawner(
		cfg *config.Config, 
		client *http.Client, 
		scanner *bufio.Scanner, 
		body request.FuzzTarget, queryParams request.FuzzTarget ){

	var chGroup sync.WaitGroup
	var reqMutex sync.Mutex

	responses := make(chan string)

	for scanner.Scan() {
		chGroup.Add(1)
		value := scanner.Text()
		
		go func(value string){
			defer chGroup.Done()
			
			reqMutex.Lock()

			body.SetTarget(value)
			bodyData := body.GetMap()

			queryParams.SetTarget(value)
			queryData := queryParams.GetMap()

			req := request.BuildRequest(cfg, bodyData, queryData)
			
			reqMutex.Unlock()

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
