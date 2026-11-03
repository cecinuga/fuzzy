// Command fuzzy sends schema-valid requests to every endpoint of an OpenAPI 3.0 or
// Swagger 2.0 spec and writes what came back to a report: Markdown, JSON, plain text,
// CSV, XML or HTML.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
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

// run executes the command line and returns the process exit code.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cmd := newRootCommand()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(canonicalArgs(cmd, args))

	err := cmd.ExecuteContext(ctx)
	if err == nil {
		return exitOK
	}

	fmt.Fprintf(stderr, "[!] %v\n", err)
	if failed := (runError{}); errors.As(err, &failed) {
		return exitFailure
	}
	fmt.Fprintf(stderr, "Run '%s --help' for usage.\n", cmd.Name())
	return exitUsage
}
