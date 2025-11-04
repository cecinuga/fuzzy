package config

import (
	"fuzzy/internal/utils"
	"fuzzy/pkg/flaggy"
)

type Config struct {
	Endpoint string
	Method string
	Body string
	QueryParameters string
	InsecureConnection bool
	Dictionary string
	FuzzyKey string
}

func CreateConfig() Config{
	config := Config{}
	flags := make(flaggy.Flags)
	
	// Definisce i flag usando la nuova API
	method := flags.String("m", "GET", "[#] HTTP Request Method", utils.IsHttpMethod)
	endpoint := flags.String("e", "", "[#] Endpoint u wanna call", utils.ValidateEndpoint)
	body := flags.String("b", "", "[#] HTTP Request Body <'{...}'|/path/body.json>", utils.ValidateBody)
	query := flags.String("q", "", "[#] HTTP Request QueryParameters <key=value&key1=value1...>", utils.IsHttpQueryParameters)
	dict := flags.String("dict", "", "[#] Dictionary file", utils.ValidateDict)
	key := flags.String("key", "FUZZY", "[#] Where fuzzy found that key, replace with dictionary values.", utils.IsAlphabetic)
	
	// TODO: Implementare Bool() in flaggy.go per il flag insecure
	// k := flags.Bool("k", false, "[#] Skip TLS certificate verification (insecure).")
	
	// Popola la config con i puntatori ai valori dei flag
	flags.Parse()

	config.Method = *method
	config.Endpoint = *endpoint
	config.Body = *body
	config.QueryParameters = *query
	config.Dictionary = *dict
	config.FuzzyKey = *key
	config.InsecureConnection = false // TODO: usare il flag Bool quando implementato
	
	return config
}
