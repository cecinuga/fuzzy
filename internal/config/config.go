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

func Init() flaggy.Specs {
	specs := flaggy.Specs{}
	
	specs.String("m", "[#] HTTP Request Method", "GET", utils.IsHttpMethod)
	specs.String("e", "[#] Endpoint u wanna call", "", utils.IsUrl) 
	specs.String("b", "[#] HTTP Request Body <'{...}'|/path/body.json>", "", utils.IsJson)
	specs.String("q", "[#] HTTP Request QueryParameters <key=value&key1=value1...>", "", utils.IsHttpQueryParameters)
	specs.String("dict", "[#] Dictionary file", "", utils.IsPath)
	specs.String("key", "[#] Where fuzzy found that key, replace with dictionary values.", "FUZZY", utils.IsAlphabetic)
	specs.Bool("k", "[#] Skip TLS certificate verification (insecure).", "false", func(b bool) bool { return true })
	
	return specs
}

func CreateConfig() *Config{
	specs := Init()
	options := flaggy.Options{}
	
	options.ParseFlags(specs)

	return nil
}