package main

import (
	"fuzzy/internal/config"
	"fuzzy/internal/http"
	"fuzzy/pkg/fuzzer"
)

func main() {
	cfg := config.ParseFlags()

	client := http.CreateClient(cfg.InsecureConnection)

	fuzzer.Run(cfg, client)
}
