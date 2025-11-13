package fuzzer

import (
	"bufio"
	"fuzzy/internal/client"
	"fuzzy/internal/config"
	"fuzzy/internal/request"
	"fuzzy/internal/utils"
	"fuzzy/pkg/target"
	"log"
	"net/http"
	"os"
	"sync"
)

type Fuzzer struct {
    config *config.Config
    client *http.Client
}

func New(cfg *config.Config) *Fuzzer {
    return &Fuzzer{
        config: cfg,
        client: client.CreateClient(cfg.InsecureConnection),
    }
}


func (f *Fuzzer) Run() {
	body := target.FuzzTarget{}
	queryParams := target.FuzzTarget{}

	// Controlla se il body è stato fornito
	if f.config.Body != "" {
		body.BuildData(f.config.Body)
		body.BuildPointer(f.config.FuzzyKey)
	}
	if f.config.QueryParameters != "" {
		queryParams.BuildData(f.config.QueryParameters)
		queryParams.BuildPointer(f.config.FuzzyKey)
	}

	dictFile, err := os.Open(f.config.Dictionary)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}
	defer dictFile.Close()

	if f.config.LogFile != "" {
		logFile, err := os.Open(f.config.LogFile)
		
		if os.IsNotExist(err) {
			logFile, err = os.Create(f.config.LogFile)
		} else if err != nil {
			log.Fatalf("Error reading values file: %v", err)
		}
		defer logFile.Close()
		os.Stdout = logFile
	}
	
	dictScanner := bufio.NewScanner(dictFile)

	f.spawner(dictScanner, body, queryParams) 
}

func (f *Fuzzer) spawner(
		scanner *bufio.Scanner, 
		body target.FuzzTarget, 
		queryParams target.FuzzTarget ){

	var chGroup sync.WaitGroup
	var reqMutex sync.Mutex

	responses := make(chan utils.ResponseMsg)

	for scanner.Scan() {
		chGroup.Add(1)
		value := scanner.Text()
		
		go func(fuzzValue string){
			defer chGroup.Done()
			
			reqMutex.Lock()
			body.SetTarget(fuzzValue)
			bodyData := body.GetMap()

			queryParams.SetTarget(fuzzValue)
			queryData := queryParams.GetMap()
			encodedQuery := utils.EncodeQuery(queryData)

			req, err := request.BuildRequest(f.config, bodyData, encodedQuery)
			reqMutex.Unlock()
			
			message := utils.ResponseMsg{} 

			var response string
			if err != nil {
				message.Status = err.Error()
				message.Error = true
			} else {
				response, err = request.SendRequest(f.client, req)
				if err != nil {
					message.Status = err.Error()
					message.Error = true
				} else {
					message.Status = response
				}
			}
			
			message.FuzzValue = value
			message.QueryParams = encodedQuery

			responses <- message
		}(value)
	}

	go func(){
		chGroup.Wait()
		close(responses)
	}()
	
	for res := range responses {
		utils.Log(res)
	}
	
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}
