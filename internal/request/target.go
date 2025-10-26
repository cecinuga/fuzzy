package request

import (
	"encoding/json"
	"fuzzy/utils"
)

type FuzzTarget struct {
	data map[string]any
	target *map[string]any
	key string
}

func (obj *FuzzTarget) Get() map[string]any{
	return obj.data
}

func (obj *FuzzTarget) SetTarget(value string){
	(*obj.target)[obj.key] = value
}

func (obj *FuzzTarget) BuildData(source string) {	
	if utils.IsHttpQueryParameters(source){
		//params, ok := url.ParseQuery(source)
		// SCRIVERE PARSING QUERY
	} else if utils.IsPath(source){ 
		utils.LoadJsonFile(source, &obj.data)
	} else {
		data := []byte(source)
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