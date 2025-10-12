package utils

import (
	"log"
	"regexp"
)

const URL_RE = `^(https?:\/\/)?([\d\w\.-]+)\.([a-z\.]+)([\/\w \.-]*)*\/?$`
const HTTP_METHOD_RE = `^[POST|GET|PUT|DELETE|PATCH|OPTIONS|TRACE|CONNECT|HEAD]`

func check(pattern, source string){
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Fatal(err)
	}
	if !re.MatchString(source){
		log.Fatalf("[!] %v not valid", source)
	}
}

func CheckUrl(url string){
	check(URL_RE, url)
}

func CheckMethod(method string){
	check(HTTP_METHOD_RE, method)
}