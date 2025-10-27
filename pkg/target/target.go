package target

import (
	"encoding/json"
	"fuzzy/internal/utils"
	"fuzzy/internal/utils/query"
	"log"
	"os"
)

type FuzzTarget struct {
	data map[string]any
	target *map[string]any
	key string
}

func (obj *FuzzTarget) Get(key string) any {
	return obj.data[key]
}

func (obj *FuzzTarget) Set(key string, val any) {
	if (obj.data != nil){
		obj.data[key] = val
	}
}

func (obj *FuzzTarget) GetMap() map[string]any {
	return obj.data
}

func (obj *FuzzTarget) SetTarget(value string){
	if obj.target != nil{
		(*obj.target)[obj.key] = value
	}
}

func (obj *FuzzTarget) BuildData(source string) {	
	if utils.IsHttpQueryParameters(source){
		obj.data = query.ParseQuery(source)
	} else {
		var data []byte
		var err error

		if utils.IsPath(source){ 
			data, err = os.ReadFile(source)
			if err != nil {
				log.Fatalf("[!] %v", err)
			}			
		} else {
			data = []byte(source)
		}

		json.Unmarshal(data, &obj.data)
	}
}
func (obj FuzzTarget) GetPointerToValue(root *map[string]any, value string) (*map[string]any, any) { // GESTIRE GLI ERRORI
	for k, v := range *root {
		if v == value {
			return root, k
		}

		childBody, ok := v.(map[string]any)

		if ok {
			child, key := obj.GetPointerToValue(&childBody, value)
			if keyStr := key.(string); keyStr != "" {
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
