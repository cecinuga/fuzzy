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

func Init() *Config {
	config := Config{}
	options := flaggy.Options{}
	
	m, mok := options.String("m", "[#] HTTP Request Method", "GET", utils.IsHttpMethod)
	if mok {
		config.Method = m.Value()
	}
	
	e, eok := options.String("e", "[#] Endpoint u wanna call", "", utils.IsUrl) 
	if eok {
		config.Endpoint = e.Value()
	}

	b, bok := options.String("b", "[#] HTTP Request Body <'{...}'|/path/body.json>", "", utils.IsJson)
	if bok {
		config.Body = b.Value()
	}

	q, qok := options.String("q", "[#] HTTP Request QueryParameters <key=value&key1=value1...>", "", utils.IsHttpQueryParameters)
	if qok {
		config.QueryParameters = q.Value()
	}

	dict, dictOk := options.String("dict", "[#] Dictionary file", "", utils.IsPath)
	if dictOk {
		config.Dictionary = dict.Value()
	}

	key, keyok := options.String("key", "[#] Where fuzzy found that key, replace with dictionary values.", "FUZZY", utils.IsAlphabetic)
	if keyok {
		config.FuzzyKey = key.Value()
	}
	
	k, kok := options.Bool("k", "[#] Skip TLS certificate verification (insecure).", false, func(b bool) bool { return true })
	if kok {
		config.InsecureConnection = k.Value()
	}
	
	return &config
}

func CreateConfig() *Config{
	config := Init()
	options := flaggy.Options{}
	
	options.ParseFlags()

	return config
}