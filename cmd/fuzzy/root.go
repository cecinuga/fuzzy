package main

import (
	"errors"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"fuzzy/internal/config"
	"fuzzy/internal/generator"
	"fuzzy/internal/report"
)

const longDescription = `Fuzzy sends schema-valid requests, with random or dictionary values, to every
endpoint of an OpenAPI 3.0 spec and writes what came back to a report: Markdown,
JSON, plain text, CSV, XML or HTML.

Fuzzy calls every operation with valid values, DELETE and PUT included, so it
really changes or deletes data. Only run it against disposable or local targets
that you have permission to test, never production.`

const examples = `  # Keep only server errors and requests that got no response
  fuzzy -remote http://localhost:8080 -spec api.json -status 5xx,err -seed 42

  # Authenticated API, with a dictionary
  fuzzy -remote https://api.example.com/v3 -spec api.json -dict values.txt --H "Authorization: Bearer $TOKEN"

  # A JSON report, for scripts and CI
  fuzzy -remote http://localhost:8080 -spec api.json -format json

  # The report's extension picks its format, and the log keeps every request
  fuzzy -remote http://localhost:8080 -spec api.json -status 5xx,err --o results.csv -log run.log

  # Everything except deletions and one slow path, previewed first
  fuzzy --spec api.json --exclude DELETE --exclude 'POST /reports/*' --list
  fuzzy --remote http://localhost:8080 --spec api.json --exclude DELETE --exclude 'POST /reports/*'

  # Shell completion, see 'fuzzy completion --help'
  fuzzy completion bash`

// runError marks a failure of the run itself. Any other error returned by the
// command means the arguments were invalid.
type runError struct{ error }

func (e runError) Unwrap() error { return e.error }

// newRootCommand builds the command line. Fuzzing is what the root command does,
// so that commands written for earlier versions keep working.
func newRootCommand() *cobra.Command {
	var flags config.Flags

	cmd := &cobra.Command{
		Use:     "fuzzy -remote <url> -spec <file> [flags]",
		Short:   "Fuzz an HTTP API from its OpenAPI 3.0 spec",
		Long:    longDescription,
		Example: examples,
		Version: version(),
		Args:    cobra.NoArgs,
		// run reports errors itself, in the [!] format and with the right exit code.
		SilenceErrors: true,
		SilenceUsage:  true,
		// Cobra checks required flags between PreRunE and RunE. Listing sends nothing,
		// so it doesn't need a target.
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if flags.Listing() {
				return cmd.Flags().SetAnnotation("remote", cobra.BashCompOneRequiredFlag, []string{"false"})
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := flags.Config()
			if err != nil {
				return err
			}
			if err := fuzz(cmd.Context(), cfg, cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
				return runError{err}
			}
			return nil
		},
	}

	cmd.Flags().SortFlags = false
	flags.Register(cmd.Flags())

	cobra.AddTemplateFunc("flagUsages", flagUsages)
	cmd.SetUsageTemplate(strings.ReplaceAll(cmd.UsageTemplate(), ".LocalFlags.FlagUsages", "flagUsages ."))

	// These only fail on the name of a flag that doesn't exist: a programming error.
	cobra.CheckErr(errors.Join(
		cmd.MarkFlagRequired("remote"),
		cmd.MarkFlagRequired("spec"),
		cmd.MarkFlagFilename("spec", "json"),
		cmd.RegisterFlagCompletionFunc("remote", cobra.NoFileCompletions),
		cmd.RegisterFlagCompletionFunc("status", cobra.NoFileCompletions),
		cmd.RegisterFlagCompletionFunc("include", cobra.NoFileCompletions),
		cmd.RegisterFlagCompletionFunc("exclude", cobra.NoFileCompletions),
		cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
			formatNames(),
			cobra.ShellCompDirectiveNoFileComp,
		)),
		cmd.RegisterFlagCompletionFunc("charset", cobra.FixedCompletions(
			[]string{string(generator.CharsetUnicode), string(generator.CharsetAlnum)},
			cobra.ShellCompDirectiveNoFileComp,
		)),
	))

	return cmd
}

func formatNames() []string {
	formats := report.Formats()
	names := make([]string, len(formats))
	for i, format := range formats {
		names[i] = string(format)
	}
	return names
}

// version is the module version stamped by the Go toolchain at build time.
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "unknown"
}
