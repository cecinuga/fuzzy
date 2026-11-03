package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"fuzzy/internal/config"
)

// flagUsages lists a command's flags. The root command shows them the way it
// documents them, short name first: --rem, -remote. Subcommands keep Cobra's
// format, because canonicalArgs doesn't apply to them, so those spellings wouldn't
// work there.
func flagUsages(cmd *cobra.Command) string {
	flags := cmd.LocalFlags()
	if cmd.HasParent() {
		return flags.FlagUsages()
	}

	type row struct{ names, usage string }
	var rows []row
	width := 0
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		names := "-" + f.Name
		if short := config.ShortName(f); short != "" {
			names = "--" + short + ", " + names
		}
		if value, _ := pflag.UnquoteUsage(f); value != "" {
			names += " " + value
		}
		width = max(width, len(names))
		rows = append(rows, row{names, describeFlag(f)})
	})

	var b strings.Builder
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-*s   %s\n", width, r.names, r.usage)
	}
	return b.String()
}

// describeFlag returns f's usage text as pflag renders it, with its default. pflag
// separates the names from the text with three spaces.
func describeFlag(f *pflag.Flag) string {
	single := pflag.NewFlagSet("", pflag.ContinueOnError)
	single.AddFlag(f)
	_, usage, _ := strings.Cut(strings.TrimSpace(single.FlagUsages()), "   ")
	return usage
}
