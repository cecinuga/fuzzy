package config

import (
	"flag"
	"fuzzy/utils"
)

type Config struct {
	Method string
	Endpoint string
	Body string
	Dictionary string
	QueryParameters string
	FuzzyKey string
	InsecureConnection bool
}

func ParseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.Endpoint, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&config.Method, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&config.Body, "b", "", "[#] HTTP Request Body <'{...}'|/path/body.json>")
	flag.StringVar(&config.QueryParameters, "query", "", "[#] HTTP Request QueryParameters <key=value&key1=value1...>")
	flag.StringVar(&config.Dictionary, "dict", "", "[#] Dictionary file")
	flag.StringVar(&config.FuzzyKey, "key", "", "[#] Where fuzzy found that key, replace with dictionary values.")
	flag.BoolVar(&config.InsecureConnection, "k", false, "[#] Skip TLS certificate verification (insecure)")

	flag.Parse()

	CheckConfig(config)

	return config
}

func CheckConfig(cfg *Config){
	utils.CheckMethod(cfg.Method)
	utils.CheckFuzzKey(cfg.FuzzyKey)

	if len(cfg.QueryParameters) > 0 {
		utils.CheckQueryParameters(cfg.QueryParameters)
	}
	if len(cfg.Body) > 0 {
		utils.CheckBody(cfg.Body)
	}

	utils.CheckUrl(cfg.Endpoint)
}