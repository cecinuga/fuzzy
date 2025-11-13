package main

import (
	"fuzzy/internal/config"
	"fuzzy/pkg/fuzzer"
)

func main() {
	config := config.CreateConfig()
	f := fuzzer.New(&config)

	f.Run()
}
