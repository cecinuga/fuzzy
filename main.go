package main

import (
	"fuzzy/internal/config"
	"fuzzy/internal/fuzzer"
	"fuzzy/internal/http"
)

func main() {
	cfg := config.ParseFlags()

	client := http.CreateClient(cfg.InsecureConnection)

	fuzzer.Run(cfg, client)
}
