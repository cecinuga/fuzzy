package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	if targets, skipped, err = cfg.Selection.Apply(targets, skipped); err != nil {
		return err
	}
	if len(targets) == 0 {
		if cfg.Selection.Active() {
			return errors.New("--include and --exclude leave no endpoint to fuzz")
		}
		return errors.New("no endpoint of the spec can be fuzzed")
	}

	if cfg.List {
		listEndpoints(stdout, stderr, targets, skipped, len(doc.Operations))
		return nil
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

	count := strconv.Itoa(len(targets))
	if cfg.Selection.Active() {
		count = fmt.Sprintf("%d of %d", len(targets), len(doc.Operations))
	}
	fmt.Fprintf(stdout, "[+] fuzzing %s endpoints of %s, %d requests each (seed %d)\n", count, cfg.Remote, requests, cfg.Seed)

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
	err = report.Write(buffered, cfg.Format, report.Report{
		SpecTitle:   doc.Info.Title,
		SpecVersion: doc.Info.Version,
		Remote:      cfg.Remote.String(),
		Spec:        cfg.SpecPath,
		Dictionary:  cfg.DictPath,
		Filter:      cfg.Filter,
		Selection:   cfg.Selection,
		LeftOut:     len(doc.Operations) - len(targets) - len(skipped),
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
	fmt.Fprintf(stdout, "[+] %d requests sent, %d without a response. Report: %s\n", sent, failed, absolutePath(cfg.Output))
	return nil
}

// listEndpoints prints the selected endpoints on stdout, one per line, with any
// tags after a tab: `cut -f1` gives lines that work as selectors. Everything else
// goes to stderr, to keep stdout usable by scripts.
func listEndpoints(stdout, stderr io.Writer, targets []*fuzzer.Target, skipped []fuzzer.Skipped, total int) {
	for _, t := range targets {
		line := progress.TerminalSafe(t.Op.Method + " " + t.Op.Path)
		if len(t.Op.Tags) > 0 {
			line += "\t" + progress.TerminalSafe(strings.Join(t.Op.Tags, ", "))
		}
		fmt.Fprintln(stdout, line)
	}
	fmt.Fprintf(stderr, "[+] %d of %d endpoints would be fuzzed, no request sent\n", len(targets), total)
	for _, s := range skipped {
		fmt.Fprintln(stderr, progress.TerminalSafe(fmt.Sprintf("[!] skipped %s %s: %s", s.Op.Method, s.Op.Path, s.Reason)))
	}
}

// absolutePath returns path made absolute, or as given if the working directory
// can't be determined: the report is already written, so that must not fail the run.
func absolutePath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// describeBounds returns the random limits of each data type, one entry per type.
func describeBounds(b generator.Bounds) []string {
	return []string{
		fmt.Sprintf("string: length %d..%d (%s)", b.StringMin, b.StringMax, b.Charset),
		fmt.Sprintf("int: %d..%d", b.IntMin, b.IntMax),
		fmt.Sprintf("float: %g..%g", b.FloatMin, b.FloatMax),
	}
}
