package main

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// canonicalArgs rewrites flags to the spelling Cobra expects: two dashes for a
// long name, one for a single letter. Earlier versions used Go's flag package,
// which accepted either, so -remote and --n still have to work. Without this,
// Cobra would read -remote as the bundle of short flags -r -e -m -o -t -e.
//
// The result is never nil, because Cobra falls back to os.Args on nil arguments.
func canonicalArgs(cmd *cobra.Command, args []string) []string {
	out := append([]string{}, args...)

	// Fuzzing takes only flags, so anything else is a subcommand. These include
	// shell completion, whose partial words must reach Cobra untouched.
	if len(out) == 0 || !strings.HasPrefix(out[0], "-") {
		return out
	}

	// Cobra adds these two only when it executes, but -help and -version count too.
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()
	flags := cmd.Flags()

	for i := 0; i < len(out); i++ {
		arg := out[i]
		if arg == "--" {
			break
		}
		body := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
		if body == arg {
			continue // not a flag
		}

		name, _, hasValue := strings.Cut(body, "=")
		var flag *pflag.Flag
		if len(name) == 1 {
			flag = flags.ShorthandLookup(name)
			out[i] = "-" + body
		} else {
			flag = flags.Lookup(name)
			out[i] = "--" + body
		}
		if flag == nil {
			out[i] = arg // not ours: left for Cobra to read or to reject
			continue
		}

		// The next argument is this flag's value, even if it starts with a dash.
		if !hasValue && flag.NoOptDefVal == "" {
			i++
		}
	}
	return out
}
