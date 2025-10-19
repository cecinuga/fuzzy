package utils

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

func GetFile(path string) (name string, file *os.File) {
	name = filepath.Base(path)

	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error reading values file: %v", err)
	}

	return name, file
}

func LoadJsonFile(path string, file *map[string]any) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("[!] %v", err)
	}
		
	json.Unmarshal(data, &file)
}