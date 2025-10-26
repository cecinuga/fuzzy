package config

import (
	"flag"
)

func ParseFlags() *Config {
	// Definire i flag PRIMA di flag.Parse()
	methodFlag := flag.String("m", "GET", "[#] HTTP Request Method")
	endpointFlag := flag.String("e", "", "[#] Endpoint u wanna call")
	bodyFlag := flag.String("b", "", "[#] HTTP Request Body <'{...}'|/path/body.json>")
	queryFlag := flag.String("query", "", "[#] HTTP Request QueryParameters <key=value&key1=value1...>")
	insecureFlag := flag.Bool("k", false, "[#] Skip TLS certificate verification (insecure).")
	dictFlag := flag.String("dict", "", "[#] Dictionary file")
	keyFlag := flag.String("key", "", "[#] Where fuzzy found that key, replace with dictionary values.")

	// Ora possiamo chiamare Parse
	flag.Parse()

	// Creare i tipi con puntatori per i metodi Check
	method := HttpMethod(*methodFlag)
	endpoint := URL(*endpointFlag)
	bodyJson := HttpBodyJson(*bodyFlag)
	queryParams := HttpQueryParameters(*queryFlag)
	dictionary := FilePath(*dictFlag)
	fuzzyKey := Key(*keyFlag)

	config := Config{
		Method:  method,
		Endpoint: endpoint,
		Body: bodyJson,
		QueryParameters: queryParams,
		InsecureConnection: *insecureFlag,
		Dictionary: dictionary,
		FuzzyKey: fuzzyKey,
	}

	config.checkConfig()

	return &config
}

func (cfg Config) checkConfig(){
	cfg.Method.Check()
	cfg.FuzzyKey.Check()
	cfg.Endpoint.Check()
	cfg.Dictionary.Check()

	// Controlla Body se fornito
	if bodyStr := string(cfg.Body); bodyStr != "" {
		cfg.Body.Check()
	}

	// Controlla QueryParameters se forniti
	if queryStr := string(cfg.QueryParameters); queryStr != "" {
		cfg.QueryParameters.Check()
	}
}