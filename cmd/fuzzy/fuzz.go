package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"fuzzy/internal/config"
	"fuzzy/internal/fuzzer"
	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
	"fuzzy/internal/progress"
	"fuzzy/internal/report"
)

// fuzz loads the inputs, runs the fuzzer and writes the report.
func fuzz(ctx context.Context, cfg *config.Config, stdout, stderr io.Writer) error {
	doc, err := openapi.Load(cfg.SpecPath)
	if err != nil {
		return err
	}

	var dict *generator.Dictionary
	if cfg.DictPath != "" {
		if dict, err = generator.LoadDictionary(cfg.DictPath); err != nil {
			return err
		}
	}

	gen := generator.New(cfg.Bounds, dict, cfg.Seed)
	targets, skipped := fuzzer.Compile(doc, gen, cfg.Remote)
	if len(targets) == 0 {
		return errors.New("no endpoint of the spec can be fuzzed")
	}

	// Opened up front: a bad path must fail before the fuzzing, not after it.
	out, err := os.Create(cfg.Output)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	defer out.Close()

	requests := cfg.Requests
	if requests == 0 {
		requests = max(config.DefaultRequests, dict.Size())
	}

	fmt.Fprintf(stdout, "[+] fuzzing %d endpoints of %s, %d requests each (seed %d)\n", len(targets), cfg.Remote, requests, cfg.Seed)

	display := progress.New(stderr)
	started := time.Now()
	outcomes := fuzzer.Run(ctx, fuzzer.Options{
		Requests:      requests,
		Concurrency:   cfg.Concurrency,
		Timeout:       cfg.Timeout,
		Insecure:      cfg.Insecure,
		Headers:       cfg.Headers,
		Seed:          cfg.Seed,
		Filter:        cfg.Filter,
		Progress:      display.Update,
		ProgressEvery: display.Interval(),
	}, targets)
	display.Finish()

	buffered := bufio.NewWriter(out)
	err = report.Write(buffered, report.Report{
		SpecTitle:   doc.Info.Title,
		SpecVersion: doc.Info.Version,
		Remote:      cfg.Remote.String(),
		Spec:        cfg.SpecPath,
		Dictionary:  cfg.DictPath,
		Filter:      cfg.Filter,
		Bounds:      describeBounds(cfg.Bounds),
		Seed:        cfg.Seed,
		Started:     started,
		Duration:    time.Since(started),
		Interrupted: ctx.Err() != nil,
		Warnings:    gen.Warnings(),
		Outcomes:    outcomes,
		Skipped:     skipped,
	})
	if err == nil {
		err = buffered.Flush()
	}
	if err == nil {
		err = out.Close()
	}
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	sent, failed := 0, 0
	for i := range outcomes {
		sent += outcomes[i].Sent
		failed += outcomes[i].Counts[fuzzer.TransportError]
	}
	fmt.Fprintf(stdout, "[+] %d requests sent, %d without a response. Report: %s\n", sent, failed, cfg.Output)
	return nil
}

func describeBounds(b generator.Bounds) string {
	return fmt.Sprintf("string length %d..%d (%s), int %d..%d, float %g..%g",
		b.StringMin, b.StringMax, b.Charset, b.IntMin, b.IntMax, b.FloatMin, b.FloatMax)
}
