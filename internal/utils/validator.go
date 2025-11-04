package utils

import (
	"encoding/json"
	"flag"
	"log"
	"regexp"
)

const URL_RE = `^(https?:\/\/)?([\d\w\.-]+)\.([a-z\.]+)([\/\w \.-]*)*\/?$`
const LOCALHOST_URL_RE = `^(https?:\/\/)?(localhost(:[0-9])?)([\/\w \.-]*)*\/?$`
const HOST_URL_RE = `^(https?:\/\/)?(([0-9\.]+)(:[0-9])?)([\/\w \.-]*)*\/?$`
const HTTP_METHOD_RE = `^(POST|GET|PUT|DELETE|PATCH|OPTIONS|TRACE|CONNECT|HEAD)$`
const HTTP_QUERY_PARAMETERS_RE = `^([^=&?]+=[^&#]*)(?:&[^=&?]+=[^&#]*)*$`
const PATH_RE = `^([\/\w \.-]*)+\/?$`
const ALPHABETIC_RE = `^[\w]+$`

type Matcher func (string) bool

func IsAlphabetic(word string) bool {
	return match(word, ALPHABETIC_RE)
}

func IsPath(path string) bool{
	return match(path, PATH_RE)
}

func IsJson(s string) bool {
	var js any
	return json.Unmarshal([]byte(s), &js) == nil
}

func IsLocalhostUrl(url string) bool {
	return match(url, LOCALHOST_URL_RE)
}

func IsHostUrl(url string) bool {
	return match(url, HOST_URL_RE)
}

func IsUrl(url string) bool {
	return match(url, URL_RE)
}

func IsHttpMethod(method string) bool {
	return match(method, HTTP_METHOD_RE)
}

func IsHttpQueryParameters(parameters string) bool {
	return match(parameters, HTTP_QUERY_PARAMETERS_RE)
}

func ValidateEndpoint(url string) bool {
	return atLeastOne(url, IsUrl, IsHostUrl, IsLocalhostUrl)
}

func ValidateBody(body string) bool {
	return atLeastOne(body, IsJson, IsPath)
}

func ValidateDict(dict string) bool {
	return IsPath(dict)
}

func match(source, pattern string) bool{
	res, err := regexp.MatchString(pattern, source)

	if err != nil {
		log.Fatal(err)
	}
	return res
}

func Check(name, source string, matchers ...Matcher){
	if !atLeastOne(source, matchers...) {
		flag.Usage()
		log.Fatalf("[!] %v not valid: ( %v ). check help manual", name, source)
	}
}

func atLeastOne(source string, matchers ...Matcher) bool {
	for _, matcher := range(matchers){
		if matcher(source){
			return  true
		}
	}
	return false
}