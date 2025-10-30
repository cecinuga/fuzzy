package main

import (
	"fuzzy/internal/config"
	"fuzzy/pkg/fuzzer"
)

func main() {
	cfg := config.CreateConfig()

	f := fuzzer.New(cfg)

	f.Run(cfg)
}
