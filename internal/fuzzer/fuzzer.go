package fuzzer

import (
	"log"
	"fmt"
	"sync"
	"bufio"
	"net/http"
	"fuzzy/utils"
	"encoding/json"
	"fuzzy/internal/config"
)

func Run(cfg *config.Config, client *http.Client) {
	body := PointerMap{}
	body.BuildDataFromJson(cfg.Body)
	body.BuildPointer(cfg.FuzzyKey)

	_, dictFile := utils.GetFile(cfg.Dictionary)
	defer dictFile.Close()

	dictScanner := bufio.NewScanner(dictFile)

	spawner(cfg, client, dictScanner, body) 
}

func spawner(
		cfg *config.Config, 
		client *http.Client, 
		scanner *bufio.Scanner, 
		body PointerMap ){

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


type PointerMap struct {
	data map[string]any

	child *map[string]any
	key string
}

func (obj *PointerMap) BuildDataFromJson(object string) {
	if utils.IsPath(object){ 
		utils.LoadJsonFile(object, &obj.data)
	} else {
		data := []byte(object)
		json.Unmarshal(data, &obj.data)
	}
}


func (obj PointerMap) GetPointerToValue(root *map[string]any, value string) (*map[string]any, any) { // GESTIRE GLI ERRORI
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

func (obj *PointerMap) BuildPointer(value string){
	child, key := obj.GetPointerToValue(&obj.data, value)
	
	obj.child = child
	obj.key = key.(string)
}

func (obj *PointerMap) Assign(value string){
	(*obj.child)[obj.key] = value
}