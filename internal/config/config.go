package config

import (
	"fuzzy/internal/utils"
	"fuzzy/pkg/flaggy"
)

type Config struct {
	Endpoint 			string
	Method 				string
	Body 				string
	QueryParameters 	string
	Dictionary 			string
	LogFile 			string
	FuzzyKey 			string
	Verbosity			string
	InsecureConnection  bool
}

func CreateConfig() Config{
	config := Config{}
	flags := make(flaggy.Flags)
	
	// Definisce i flag usando la nuova API
	method := flags.String("m", "GET", "[#] HTTP req method.", utils.IsHttpMethod)
	endpoint := flags.String("e", "", "[#] Endpoint u wanna call.", utils.ValidateEndpoint)
	body := flags.String("b", "", "[#] HTTP req body <'{...}'|/path/body.json>", utils.ValidateBody)
	query := flags.String("q", "", "[#] HTTP req query params <key=value&key1=value1...>", utils.IsHttpQueryParameters)
	dict := flags.String("dict", "", "[#] Dictionary file.", utils.ValidateDict)
	key := flags.String("key", "FUZZY", "[#] Replace key with dict values.", utils.IsAlphabetic)
	out := flags.String("o", "", "[#] Redirect stdout, if doesn't exist, will created.", utils.IsPath)
	verbosity := flags.String("v", "0", "[#] Log verbosity, <1|2|3>", utils.IsVerbosity)
	
	// TODO: Implementare Bool() in flaggy.go per il flag insecure
	// k := flags.Bool("k", false, "[#] Skip TLS certificate verification (insecure).")
	
	// Popola la config con i puntatori ai valori dei flag
	flags.Parse()

	config.Method = 		 	*method
	config.Endpoint = 		 	*endpoint
	config.Body = 			 	*body
	config.QueryParameters = 	*query
	config.Dictionary = 	 	*dict
	config.LogFile = 		 	*out
	config.FuzzyKey = 		 	*key
	config.Verbosity = 		 	*verbosity
	config.InsecureConnection = false 
	
	return config
}
