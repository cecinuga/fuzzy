// Command fuzzy sends schema-valid requests to every endpoint of an OpenAPI 3.0
// spec and writes what came back to a Markdown report.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fuzzy/internal/config"
	"fuzzy/internal/fuzzer"
	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
	"fuzzy/internal/report"
)

const (
	exitOK = iota
	exitFailure
	exitUsage
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// The first signal stops the run and still writes the report; a second one
	// gets the default behavior and kills the process.
	context.AfterFunc(ctx, stop)

	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cfg, err := config.Parse(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	if err != nil {
		fmt.Fprintf(stderr, "[!] %v\n", err)
		return exitUsage
	}
	if err := fuzz(ctx, cfg, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "[!] %v\n", err)
		return exitFailure
	}
	return exitOK
}

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

	started := time.Now()
	outcomes := fuzzer.Run(ctx, fuzzer.Options{
		Requests:    requests,
		Concurrency: cfg.Concurrency,
		Timeout:     cfg.Timeout,
		Insecure:    cfg.Insecure,
		Headers:     cfg.Headers,
		Seed:        cfg.Seed,
		Filter:      cfg.Filter,
		Progress:    stderr,
	}, targets)

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
