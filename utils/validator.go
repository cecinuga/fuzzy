package utils

import (
	"encoding/json"
	"log"
	"regexp"
)

const URL_RE = `^(https?:\/\/)?([\d\w\.-]+)\.([a-z\.]+)([\/\w \.-]*)*\/?$`
const LOCALHOST_URL_RE = `^(https?:\/\/)?(localhost(:[0-9])?)([\/\w \.-]*)*\/?$`
const HOST_URL_RE = `^(https?:\/\/)?(([0-9\.]+)(:[0-9])?)([\/\w \.-]*)*\/?$`
const HTTP_METHOD_RE = `^[POST|GET|PUT|DELETE|PATCH|OPTIONS|TRACE|CONNECT|HEAD]`
const PATH_RE = `([\/\w \.-]*)*\/?$`
const JSON_RE = ``

func match(pattern, source string) bool{
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Fatal(err)
	}
	if !re.MatchString(source){
		return false
	}

	return true
}

func check(pattern, source string){
	if !match(pattern, source){
		log.Fatalf("[!] %v not valid", source)
	}
}

func IsPath(path string) bool{
	return match(PATH_RE, path)
}

func IsJSON(s string) bool {
	var js any
	return json.Unmarshal([]byte(s), &js) == nil
}

func IsLocalhostUrl(url string) bool {
	return match(LOCALHOST_URL_RE, url)
}

func IsHostUrl(url string) bool {
	return match(HOST_URL_RE, url)
}

func IsUrl(url string) bool {
	return match(URL_RE, url)
}

func CheckUrl(url string){
	if !IsUrl(url) && !IsHostUrl(url) && !IsLocalhostUrl(url){
		log.Fatal("[!] url not valid")
	}
}

func CheckMethod(method string){
	check(HTTP_METHOD_RE, method)
}

func CheckBody(body string){
	if !IsPath(body) && !IsJSON(body){
		log.Fatal("[!] body not valid")
	}
}