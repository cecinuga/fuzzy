package utils

import (
	"bytes"
	"encoding/json"
	"log"
)

func MarshalJson (data map[string]any) *bytes.Reader {
	bodyBuf, err := json.Marshal(data)

	if err != nil {
		log.Fatalf("Error Unmarshalling file: %v", err)
	}
	
	return bytes.NewReader(bodyBuf)
}