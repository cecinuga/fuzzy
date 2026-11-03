// Package config defines the command line flags and turns their values into a
// validated configuration.
package config

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/generator"
	"fuzzy/internal/report"
)

const (
	DefaultRequests = 100

	// ReportCheckpoints is how many times --incremental saves the report before
	// the end of the run.
	ReportCheckpoints = 10

	defaultConcurrency = 20
	defaultTimeout     = 10 * time.Second
	defaultOutputName  = "fuzzy-report" // the extension comes from the format
)

// Config is everything a run needs.
type Config struct {
	Remote      *url.URL // empty when listing without --remote
	SpecPath    string
	DictPath    string
	Output      string
	Format      report.Format
	LogPath     string // empty means no log file
	Filter      fuzzer.StatusFilter
	Selection   fuzzer.Selection
	List        bool // print the selected endpoints instead of fuzzing them
	Bounds      generator.Bounds
	Requests    int // 0 means: DefaultRequests, or the dictionary size if larger
	Concurrency int
	Timeout     time.Duration
	Insecure    bool
	Headers     http.Header
	Seed        uint64
	Incremental bool // also save the report during the run, not only at its end
}

// Flags holds the flag values as typed, before validation.
type Flags struct {
	remote      string
	spec        string
	status      string
	include     []string
	exclude     []string
	list        bool
	dict        string
	bounds      generator.Bounds
	requests    int
	concurrency int
	timeout     time.Duration
	output      string
	format      string
	log         string
	headers     []string
	insecure    bool
	seed        uint64
	incremental bool
}

// Register defines the flags on fs, with their defaults. A word in backquotes
// names the flag's value in the help.
func (f *Flags) Register(fs *pflag.FlagSet) {
	d := generator.DefaultBounds()

	fs.SetNormalizeFunc(normalizeFlagName)
	fs.StringVar(&f.remote, "remote", "", "API to fuzz: `url`, FQDN or host:port (required)")
	fs.StringVar(&f.spec, "spec", "", "OpenAPI 3.0 spec, JSON `file` (required)")
	fs.StringVar(&f.status, "status", "", "status `codes` in the report, e.g. 200,404,5xx,400-404,err (default all)")
	// Arrays, not slices: a slice would split selectors such as 'GET,POST /a' on
	// their commas.
	fs.StringArrayVar(&f.include, "include", nil, "only fuzz endpoints matching `selector`: GET, GET,POST, /users/*, /users/**, tag:billing or 'GET /user/{id}' (repeatable)")
	fs.StringArrayVar(&f.exclude, "exclude", nil, "don't fuzz endpoints matching `selector`, even if included (repeatable)")
	fs.BoolVar(&f.list, "list", false, "print the selected endpoints and exit without sending requests (--remote not needed)")
	fs.StringVar(&f.dict, "dict", "", "values `file`, one value per line (default random values only)")
	fs.IntVar(&f.bounds.StringMin, "string-min", d.StringMin, "random string minimum length")
	fs.IntVar(&f.bounds.StringMax, "string-max", d.StringMax, "random string maximum length")
	fs.Int64Var(&f.bounds.IntMin, "int-min", d.IntMin, "random integer minimum")
	fs.Int64Var(&f.bounds.IntMax, "int-max", d.IntMax, "random integer maximum")
	fs.Float64Var(&f.bounds.FloatMin, "float-min", d.FloatMin, "random float minimum")
	fs.Float64Var(&f.bounds.FloatMax, "float-max", d.FloatMax, "random float maximum")
	fs.StringVar((*string)(&f.bounds.Charset), "charset", string(d.Charset), "random string characters: unicode or alnum")
	fs.IntVarP(&f.requests, "requests", "n", 0, fmt.Sprintf("requests per endpoint (default %d, or the dictionary size if larger)", DefaultRequests))
	fs.IntVarP(&f.concurrency, "concurrency", "c", defaultConcurrency, "concurrent workers")
	fs.DurationVar(&f.timeout, "timeout", defaultTimeout, "timeout of each request")
	fs.StringVarP(&f.output, "output", "o", "", "report `path`; its extension picks the format (default "+defaultOutputName+" with the format's extension)")
	fs.StringVar(&f.format, "format", "", "report format: markdown, json, txt, csv, xml or html (default from the -o extension, else markdown)")
	fs.BoolVar(&f.incremental, "incremental", false, fmt.Sprintf("also save the report %d times during the run (default only at the end)", ReportCheckpoints))
	fs.StringVar(&f.log, "log", "", "append a line for every request to this `file` (default no log)")
	// An array, not a slice: a slice would split header values on their commas.
	fs.StringArrayVarP(&f.headers, "header", "H", nil, "`header` for every request, 'Name: value' (repeatable)")
	fs.BoolVarP(&f.insecure, "insecure", "k", false, "skip TLS certificate verification")
	fs.Uint64Var(&f.seed, "seed", 0, "seed to reproduce a run's inputs (default random)")
}

// Listing reports whether --list was given.
func (f *Flags) Listing() bool { return f.list }

// Config validates the flag values. Requiring a flag to be set is up to the
// command, so here an empty value is only an invalid one.
func (f *Flags) Config() (*Config, error) {
	if f.spec == "" {
		return nil, errors.New("--spec must not be empty")
	}

	// Listing sends nothing, so it has no use for a target unless one is given.
	target := &url.URL{}
	if !f.list || f.remote != "" {
		var err error
		if target, err = parseRemote(f.remote); err != nil {
			return nil, fmt.Errorf("--remote: %w", err)
		}
	}

	filter, err := fuzzer.ParseStatusFilter(f.status)
	if err != nil {
		return nil, fmt.Errorf("--status: %w", err)
	}

	selection, err := fuzzer.ParseSelection(f.include, f.exclude)
	if err != nil {
		return nil, err
	}

	format, err := f.reportFormat()
	if err != nil {
		return nil, fmt.Errorf("--format: %w", err)
	}

	output := f.output
	if output == "" {
		output = defaultOutputName + format.Extension()
	}
	if f.log != "" && filepath.Clean(f.log) == filepath.Clean(output) {
		return nil, fmt.Errorf("--log: %q is also the report's path", f.log)
	}

	if err := f.bounds.Validate(); err != nil {
		return nil, err
	}

	headers, err := parseHeaders(f.headers)
	if err != nil {
		return nil, fmt.Errorf("--header: %w", err)
	}

	switch {
	case f.requests < 0:
		return nil, fmt.Errorf("--requests must not be negative (got %d)", f.requests)
	case f.concurrency < 1:
		return nil, fmt.Errorf("--concurrency must be at least 1 (got %d)", f.concurrency)
	case f.timeout <= 0:
		return nil, fmt.Errorf("--timeout must be positive (got %s)", f.timeout)
	}

	seed := f.seed
	if seed == 0 {
		seed = rand.Uint64()
	}

	return &Config{
		Remote:      target,
		SpecPath:    f.spec,
		DictPath:    f.dict,
		Output:      output,
		Format:      format,
		LogPath:     f.log,
		Filter:      filter,
		Selection:   selection,
		List:        f.list,
		Bounds:      f.bounds,
		Requests:    f.requests,
		Concurrency: f.concurrency,
		Timeout:     f.timeout,
		Insecure:    f.insecure,
		Headers:     headers,
		Seed:        seed,
		Incremental: f.incremental,
	}, nil
}

// reportFormat lets --format decide, then the extension of --output. A path whose
// extension names no format gets the default one, as any path used to.
func (f *Flags) reportFormat() (report.Format, error) {
	if f.format != "" {
		return report.ParseFormat(f.format)
	}
	if format, ok := report.FormatOfPath(f.output); ok {
		return format, nil
	}
	return report.FormatMarkdown, nil
}

// parseHeaders reads 'Name: value' pairs. The value may itself contain colons.
func parseHeaders(raw []string) (http.Header, error) {
	headers := http.Header{}
	for _, entry := range raw {
		name, value, ok := strings.Cut(entry, ":")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			return nil, fmt.Errorf("want 'Name: value', got %q", entry)
		}
		headers.Add(name, strings.TrimSpace(value))
	}
	return headers, nil
}

// parseRemote accepts a full http(s) URL, an FQDN or a host:port. Without a
// scheme, a port other than 443 means http and anything else https: redirects
// are not followed, so plain http to a public host would only report 301s.
func parseRemote(raw string) (*url.URL, error) {
	hasScheme := strings.Contains(raw, "://")
	withScheme := raw
	if !hasScheme {
		withScheme = "http://" + raw
	}

	target, err := url.Parse(withScheme)
	if err != nil {
		return nil, err
	}
	if (target.Scheme != "http" && target.Scheme != "https") || target.Hostname() == "" {
		return nil, fmt.Errorf("want a URL, FQDN or host:port, got %q", raw)
	}
	port := target.Port()
	if n, err := strconv.Atoi(port); port != "" && (err != nil || n < 1 || n > 65535) {
		return nil, fmt.Errorf("invalid port in %q", raw)
	}
	if !hasScheme && (port == "" || port == "443") {
		target.Scheme = "https"
	}
	return target, nil
}
