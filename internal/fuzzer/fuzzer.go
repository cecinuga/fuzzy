package fuzzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"fuzzy/internal/config"
	"fuzzy/utils"
	"log"
	"net/http"
	"strings"
	"sync"
)

func Run(cfg *config.Config, client *http.Client) {
	body := make(map[string]any) //DEVI CREARE UNA FUNZIONE CHE INCAPSULA FINO A RIGA .28, DEVE RITORNARE UN RIFERIMENTO

	if !utils.IsJson(cfg.Body){ //ALLA CHIAVE IL CUI VALORE È UGUALE A FUZZ KEY, COSI CHE A RIGA .40 VIENE ASSEGNATA A QUEL RIFERIMENTO
		utils.LoadJsonFile(cfg.Body, &body)
	} else {
		data := []byte(cfg.Body)
		json.Unmarshal(data, &body)
	}

	dictName, dictFile := utils.GetFile(cfg.Dictionary)
	key := strings.Replace(dictName, ".txt", "", 1)

	defer dictFile.Close()

	values := bufio.NewScanner(dictFile)
	responses := make(chan string)

	var wg sync.WaitGroup

	for values.Scan() {
		wg.Add(1)
		go func(value string){
			defer wg.Done()

			body[key] = value
			req := BuildRequest(cfg, body)

			responses <- SendRequest(client, req)	
		}(values.Text())
	}

	go func(){
		wg.Wait()
		close(responses)
	}()
	
	for status := range responses {
		fmt.Printf("[+] Response status: %v\n", status)
	}
	
	if err := values.Err(); err != nil {
		log.Fatalf("Error scanning value file: %v", err)
	}
}