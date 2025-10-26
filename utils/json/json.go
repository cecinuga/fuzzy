package json

import (
	"bytes"
	"encoding/json"
	"log"
)

func Marshal (data map[string]any) *bytes.Reader {
	bodyBuf, err := json.Marshal(data)

	if err != nil {
		log.Fatalf("Error Unmarshalling file: %v", err)
	}
	
	return bytes.NewReader(bodyBuf)
}