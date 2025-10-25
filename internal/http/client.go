package http

import (
	"net/http"
	"crypto/tls"
)


func CreateClient(insecureConnection bool) *http.Client {
	transport := &http.Transport{}
	if insecureConnection {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	
	client := &http.Client{
		Transport: transport,
	}

	return client
}