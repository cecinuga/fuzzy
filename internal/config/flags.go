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
	Parameters string
	FuzzyKey string
	InsecureConnection bool
}

func ParseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.Endpoint, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&config.Method, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&config.Body, "b", "", "[#] HTTP Request Body <'{...}'|/path/body.json>")
	flag.StringVar(&config.Parameters, "p", "", "[#] HTTP Request Parameters <key=value&key1=value1...>")
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

	if len(cfg.Parameters) > 0 {
		utils.CheckParameters(cfg.Parameters)
	}
	if len(cfg.Body) > 0 {
		utils.CheckBody(cfg.Body)
	}

	utils.CheckUrl(cfg.Endpoint)
}