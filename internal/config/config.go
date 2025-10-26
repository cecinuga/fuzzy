package config

import "fuzzy/internal/utils"

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

type Config struct {
	Endpoint URL
	Method HttpMethod
	Body HttpBodyJson
	QueryParameters HttpQueryParameters
	InsecureConnection bool
	Dictionary FilePath
	FuzzyKey Key
}