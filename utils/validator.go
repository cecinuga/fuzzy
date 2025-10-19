package utils

import (
	"encoding/json"
	"log"
	"regexp"
	"flag"
)

const URL_RE = `^(https?:\/\/)?([\d\w\.-]+)\.([a-z\.]+)([\/\w \.-]*)*\/?$`
const LOCALHOST_URL_RE = `^(https?:\/\/)?(localhost(:[0-9])?)([\/\w \.-]*)*\/?$`
const HOST_URL_RE = `^(https?:\/\/)?(([0-9\.]+)(:[0-9])?)([\/\w \.-]*)*\/?$`
const HTTP_METHOD_RE = `^(POST|GET|PUT|DELETE|PATCH|OPTIONS|TRACE|CONNECT|HEAD)$`
const PATH_RE = `^([\/\w \.-]*)+\/?$`

type matcher func (string) bool

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

func CheckUrl(url string){
	Check("url", url, IsUrl, IsHostUrl, IsLocalhostUrl)
}

func CheckMethod(method string){
	Check("http method", method, IsHttpMethod)
}

func CheckBody(body string){
	Check("body", body, IsPath, IsJson)
}

func Check(name, source string, matchers ...matcher){
	if !atLeastOne(source, matchers...) {
		flag.Usage()
		log.Fatalf("[!] %v not valid: ( %v ). check help manual", name, source)
	}
}

func match(source, pattern string) bool{
	res, err := regexp.MatchString(pattern, source)
	if err != nil {
		log.Fatal(err)
	}
	return res
}

func atLeastOne(source string, matchers ...matcher) bool {
	for _, matcher := range(matchers){
		if matcher(source){
			return  true
		}
	}
	return false
}