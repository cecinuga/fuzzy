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
	InsecureConnection bool
}

func ParseFlags() *Config{
	config := &Config{}

	flag.StringVar(&config.Endpoint, "e", "", "[#] Endpoint u wanna call")
	flag.StringVar(&config.Method, "m", "GET", "[#] HTTP Request Method")
	flag.StringVar(&config.Body, "b", "", "[#] HTTP Request Body <'{...}'|/path/body.json>")
	flag.StringVar(&config.Dictionary, "dict", "", "[#] Dictionary file")
	flag.BoolVar(&config.InsecureConnection, "k", false, "[#] Skip TLS certificate verification (insecure)")

	flag.Parse()

	CheckConfig(config)

	return config
}

func CheckConfig(cfg *Config){
	utils.CheckUrl(cfg.Endpoint)
	utils.CheckMethod(cfg.Method)
	utils.CheckBody(cfg.Body)
}