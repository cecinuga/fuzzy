package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"fuzzy/internal/config"
	"fuzzy/internal/fuzzer"
	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
	"fuzzy/internal/progress"
	"fuzzy/internal/report"
	"fuzzy/internal/runlog"
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

	// Opened up front: a bad path must fail before the fuzzing, not after it. The
	// log goes first because appending to it can't harm an existing file, while
	// opening the report empties one.
	var log *runlog.Logger
	if cfg.LogPath != "" {
		if log, err = runlog.Open(cfg.LogPath); err != nil {
			return fmt.Errorf("open log: %w", err)
		}
		defer log.Close()
	}

	out, err := openReportFile(cfg.Output, cfg.Incremental)
	if err != nil {
		return fmt.Errorf("create report: %w", err)
	}
	defer out.close()

	requests := cfg.Requests
	if requests == 0 {
		requests = max(config.DefaultRequests, dict.Size())
	}

	fmt.Fprintf(stdout, "[+] fuzzing %d endpoints of %s, %d requests each (seed %d)\n", len(targets), cfg.Remote, requests, cfg.Seed)

	log.Info("run started",
		"target", cfg.Remote.Redacted(), "spec", cfg.SpecPath, "seed", cfg.Seed, "endpoints", len(targets),
		"requests", requests, "concurrency", cfg.Concurrency, "timeout", cfg.Timeout)
	for _, s := range skipped {
		log.Warn("endpoint skipped", "endpoint", s.Op.Method+" "+s.Op.Path, "reason", s.Reason)
	}
	for _, warning := range gen.Warnings() {
		log.Warn("spec warning", "warning", warning)
	}

	display := progress.New(stderr)
	started := time.Now()
	writeReport := func(outcomes []fuzzer.Outcome, interrupted bool) error {
		return out.write(func(w io.Writer) error {
			return report.Write(w, cfg.Format, report.Report{
				SpecTitle:   doc.Info.Title,
				SpecVersion: doc.Info.Version,
				Remote:      cfg.Remote.String(),
				Spec:        cfg.SpecPath,
				Dictionary:  cfg.DictPath,
				Filter:      cfg.Filter,
				Bounds:      cfg.Bounds,
				Seed:        cfg.Seed,
				Started:     started,
				Duration:    time.Since(started),
				Interrupted: interrupted,
				Warnings:    gen.Warnings(),
				Outcomes:    outcomes,
				Skipped:     skipped,
			})
		})
	}

	opts := fuzzer.Options{
		Requests:      requests,
		Concurrency:   cfg.Concurrency,
		Timeout:       cfg.Timeout,
		Insecure:      cfg.Insecure,
		Headers:       cfg.Headers,
		Seed:          cfg.Seed,
		Filter:        cfg.Filter,
		Progress:      display.Update,
		ProgressEvery: display.Interval(),
	}
	if log != nil {
		opts.OnResult = log.Result
	}
	// A checkpoint that fails costs only that checkpoint, not the run. It is
	// reported after the run so that it doesn't break up the status bar.
	var checkpointErr error
	if cfg.Incremental && !out.replaceable() {
		fmt.Fprintf(stderr, "[!] --incremental: %s can't be replaced, so the report is only written at the end\n", cfg.Output)
	}
	if out.replaceable() {
		opts.Checkpoints = config.ReportCheckpoints
		opts.Checkpoint = func(outcomes []fuzzer.Outcome) {
			if err := writeReport(outcomes, true); err != nil && checkpointErr == nil {
				checkpointErr = err
			}
		}
	}
	outcomes := fuzzer.Run(ctx, opts, targets)
	display.Finish()
	if checkpointErr != nil {
		fmt.Fprintf(stderr, "[!] report checkpoint failed: %v\n", checkpointErr)
	}

	err = writeReport(outcomes, ctx.Err() != nil)
	if err == nil {
		err = out.close()
	}
	if err != nil {
		log.Error("run failed", "error", err)
		return fmt.Errorf("write report: %w", err)
	}

	sent, failed := 0, 0
	for i := range outcomes {
		sent += outcomes[i].Sent
		failed += outcomes[i].Counts[fuzzer.TransportError]
	}
	log.Info("run finished",
		"sent", sent, "failed", failed, "interrupted", ctx.Err() != nil, "duration", time.Since(started), "report", absolutePath(cfg.Output))
	// The run's outcome doesn't depend on its log, so a failing log is only a warning.
	if err := log.Close(); err != nil {
		fmt.Fprintf(stderr, "[!] log: %v\n", err)
	} else if log != nil {
		fmt.Fprintf(stdout, "[+] Log: %s\n", absolutePath(cfg.LogPath))
	}
	fmt.Fprintf(stdout, "[+] %d requests sent, %d without a response. Report: %s\n", sent, failed, absolutePath(cfg.Output))
	return nil
}

// absolutePath returns path made absolute, or as given if the working directory
// can't be determined: the report is already written, so that must not fail the run.
func absolutePath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}
