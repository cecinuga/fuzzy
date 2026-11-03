package config

import "github.com/spf13/pflag"

// maxShortName is the longest a flag's short name may be.
const maxShortName = 3

// shortNames maps each flag without a single-letter shorthand to its short name.
// A flag whose name is already short enough needs no entry.
var shortNames = map[string]string{
	"remote":      "rem",
	"spec":        "spc",
	"status":      "sta",
	"include":     "icl",
	"exclude":     "exc",
	"list":        "lst",
	"dict":        "dic",
	"string-min":  "smn",
	"string-max":  "smx",
	"int-min":     "imn",
	"int-max":     "imx",
	"float-min":   "fmn",
	"float-max":   "fmx",
	"charset":     "chr",
	"timeout":     "tmo",
	"format":      "fmt",
	"incremental": "inc",
	"seed":        "sd",
}

var longNames = func() map[string]string {
	long := make(map[string]string, len(shortNames))
	for name, short := range shortNames {
		long[short] = name
	}
	return long
}()

// normalizeFlagName makes a short name read as the flag it stands for, wherever
// pflag looks a flag up.
func normalizeFlagName(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	if long, ok := longNames[name]; ok {
		return pflag.NormalizedName(long)
	}
	return pflag.NormalizedName(name)
}

// ShortName returns the short name of f, or "" if it has none.
func ShortName(f *pflag.Flag) string {
	switch short, listed := shortNames[f.Name]; {
	case f.Shorthand != "":
		return f.Shorthand
	case listed:
		return short
	case len(f.Name) <= maxShortName:
		return f.Name
	}
	return ""
}
