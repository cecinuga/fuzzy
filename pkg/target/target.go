package target

import (
	"encoding/json"
	"fmt"
	"fuzzy/internal/utils"
	"log"
	"os"
)

type FuzzTarget struct {
	data map[string]any
	target *map[string]any
	key string
}

func (obj *FuzzTarget) Get(key string) (any, error) {
	if obj.data[key] == nil {
		return nil, utils.KeyNotFoundError{ Key:key, Msg: "FuzzTarget.Get" }
	} 
	return obj.data[key], nil
}

func (obj *FuzzTarget) Set(key string, val any) error {
	if (obj.data == nil){
		msgError := fmt.Sprintf(". %v", val)
		return utils.KeyNotFoundError{ Key:key, Msg: msgError }	
	}

	obj.data[key] = val
	return nil
}

func (obj *FuzzTarget) GetMap() map[string]any {
	return obj.data
}

func (obj *FuzzTarget) SetTarget(value string) error {
	if obj.target == nil {
		return utils.ObjectNotInitialized{ Msg: "FuzzTarget.SetTarget" }
	}

	(*obj.target)[obj.key] = value
	return nil
}

func (obj *FuzzTarget) BuildData(source string) (err error) {	
	if utils.IsHttpQueryParameters(source){
		obj.data, err = utils.ParseQuery(source)
		if err != nil {
			return err
		} 
		
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

		e := json.Unmarshal(data, &obj.data)
		if e != nil {
			return e
		}
	}
	return nil
}
func (obj FuzzTarget) GetPointerToValue(root *map[string]any, value string) (*map[string]any, any, error) { 
	for k, v := range *root {
		if v == value {
			return root, k, nil
		}

		childBody, ok := v.(map[string]any)
		if !ok {
			continue
		}

		child, key, err := obj.GetPointerToValue(&childBody, value)
		if err != nil {
			return child, "", err
		}

		if keyStr := key.(string); keyStr != "" {
			return child, key, nil
		}

	} 
	return root, "", utils.KeyNotFoundError{Key:value, Msg: "FuzzTarget.GetPointerToValue"}
}

func (obj *FuzzTarget) BuildPointer(value string) error {
	child, key, err := obj.GetPointerToValue(&obj.data, value)
	
	if err != nil {
		return err
	}

	obj.target = child
	obj.key = key.(string)
	
	return nil
}
