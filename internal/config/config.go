package config

import (
	"encoding/json"
	"flag"
	"fuzzy/utils"
)

type Checkable interface {
	Check()
}

type HttpMethod string
func (m HttpMethod) Check() {
	utils.Check("http method", string(m), utils.IsHttpMethod)
}

type HttpQueryParameters string
func (p HttpQueryParameters) Check(){
	utils.Check("http query parameters", string(p), utils.IsHttpQueryParameters)
}
 
type HttpBodyJson string
func (b HttpBodyJson) Check(){
	utils.Check("body", string(b), utils.IsPath, utils.IsJson)
}

type URL string
func (u URL) Check(){
	utils.Check("url", string(u), utils.IsUrl, utils.IsHostUrl, utils.IsLocalhostUrl)
}

type FilePath string
func (p FilePath) Check(){
	utils.Check("file path", string(p), utils.IsPath)
}

type Key string
func (k Key) Check(){
	utils.Check("fuzz key", string(k), utils.IsAlphabetic)
}

type FuzzTarget struct {
	Source Checkable

	Data map[string]any
	Target *map[string]any
	Key string
}
func (obj *FuzzTarget) BuildDataFromJson(object string) {
	if utils.IsPath(object){ 
		utils.LoadJsonFile(object, &obj.Data)
	} else {
		data := []byte(object)
		json.Unmarshal(data, &obj.Data)
	}
}
func (obj FuzzTarget) GetPointerToValue(root *map[string]any, value string) (*map[string]any, any) { // GESTIRE GLI ERRORI
	for k, v := range *root{
		if v == value {
			return root, k
		}

		childBody, ok := v.(map[string]any)

		if ok {
			child, key := obj.GetPointerToValue(&childBody, value)
			if len(key.(string)) > 0 {
				return child, key
			}
		}
	}  // QUANDO NON TROVI UNA FUZZY KEY LANCIA UN ERRORE
	return root, ""
}
func (obj *FuzzTarget) BuildPointer(value string){
	child, key := obj.GetPointerToValue(&obj.Data, value)
	
	obj.Target = child
	obj.Key = key.(string)
}
func (obj *FuzzTarget) Assign(value string){
	(*obj.Target)[obj.Key] = value
}

type Config struct {
	Method HttpMethod
	Endpoint URL
	Body FuzzTarget
	QueryParameters FuzzTarget
	InsecureConnection bool
	Dictionary FilePath
	FuzzyKey Key
}

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
		Body: FuzzTarget{
			Source: bodyJson,
		},
		QueryParameters: FuzzTarget{
			Source: queryParams,
		},
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
	if bodyStr := string(cfg.Body.Source.(HttpBodyJson)); bodyStr != "" {
		cfg.Body.Source.Check()
	}

	// Controlla QueryParameters se forniti
	if queryStr := string(cfg.QueryParameters.Source.(HttpQueryParameters)); queryStr != "" {
		cfg.QueryParameters.Source.Check()
	}
}