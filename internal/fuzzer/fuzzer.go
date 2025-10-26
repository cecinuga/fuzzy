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

type FuzzTarget struct {
	data map[string]any
	target *map[string]any
	key string
}

func (obj *FuzzTarget) BuildData(source string) {	
	if utils.IsPath(source){ 
		utils.LoadJsonFile(source, &obj.data)
	} else {
		data := []byte(source)
		json.Unmarshal(data, &obj.data)
	}
}
func (obj FuzzTarget) GetPointerToValue(root *map[string]any, value string) (*map[string]any, any) { // GESTIRE GLI ERRORI
	for k, v := range *root{
		if v == value {
			return root, k
		}

		childBody, ok := v.(map[string]any)

		if ok {
			child, key := obj.GetPointerToValue(&childBody, value)
			if len(key.(string)) > 0 {
				return child, key
			}
		}
	}  // QUANDO NON TROVI UNA FUZZY KEY LANCIA UN ERRORE
	return root, ""
}
func (obj *FuzzTarget) BuildPointer(value string){
	child, key := obj.GetPointerToValue(&obj.data, value)
	
	obj.target = child
	obj.key = key.(string)
}
func (obj *FuzzTarget) Assign(value string){
	(*obj.target)[obj.key] = value
}

func Run(cfg *config.Config, client *http.Client) {
	body := FuzzTarget{}

	// Controlla se il body è stato fornito
	if bodyStr := string(cfg.Body); bodyStr != "" {
		body.BuildData(bodyStr)
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
		body FuzzTarget ){

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
			req := BuildRequest(cfg, body.data)
			
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
