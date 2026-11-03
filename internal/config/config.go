// Package config turns command line arguments into a validated configuration.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/generator"
)

const DefaultRequests = 100

// Config is everything a run needs.
type Config struct {
	Remote      *url.URL
	SpecPath    string
	DictPath    string
	Output      string
	Filter      fuzzer.StatusFilter
	Bounds      generator.Bounds
	Requests    int // 0 means: DefaultRequests, or the dictionary size if larger
	Concurrency int
	Timeout     time.Duration
	Insecure    bool
	Headers     http.Header
	Seed        uint64
}

// headerList collects a repeatable -H flag.
type headerList struct{ header http.Header }

func (h *headerList) String() string { return "" }

func (h *headerList) Set(raw string) error {
	name, value, ok := strings.Cut(raw, ":")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return fmt.Errorf("want 'Name: value', got %q", raw)
	}
	h.header.Add(name, strings.TrimSpace(value))
	return nil
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

// Parse reads args (without the program name). Usage goes to output. A help
// request is reported as flag.ErrHelp.
func Parse(args []string, output io.Writer) (*Config, error) {
	bounds := generator.DefaultBounds()
	headers := headerList{http.Header{}}

	fs := flag.NewFlagSet("fuzzy", flag.ContinueOnError)
	fs.SetOutput(output)

	remote := fs.String("remote", "", "API to fuzz: URL, FQDN or host:port (required)")
	spec := fs.String("spec", "", "OpenAPI 3.0 spec, JSON format (required)")
	status := fs.String("status", "", "status codes in the report, e.g. 200,404,5xx,400-404,err (default all)")
	dict := fs.String("dict", "", "wordlist, one value per line (default: random values only)")
	fs.IntVar(&bounds.StringMin, "string-min", bounds.StringMin, "random string minimum length")
	fs.IntVar(&bounds.StringMax, "string-max", bounds.StringMax, "random string maximum length")
	fs.Int64Var(&bounds.IntMin, "int-min", bounds.IntMin, "random integer minimum")
	fs.Int64Var(&bounds.IntMax, "int-max", bounds.IntMax, "random integer maximum")
	fs.Float64Var(&bounds.FloatMin, "float-min", bounds.FloatMin, "random float minimum")
	fs.Float64Var(&bounds.FloatMax, "float-max", bounds.FloatMax, "random float maximum")
	charset := fs.String("charset", string(bounds.Charset), "random string characters: unicode or alnum")
	requests := fs.Int("n", 0, fmt.Sprintf("requests per endpoint (default %d, or the dictionary size if larger)", DefaultRequests))
	concurrency := fs.Int("c", 20, "concurrent workers")
	timeout := fs.Duration("timeout", 10*time.Second, "timeout of each request")
	reportPath := fs.String("o", "fuzzy-report.md", "markdown report path")
	insecure := fs.Bool("k", false, "skip TLS certificate verification")
	seed := fs.Uint64("seed", 0, "seed to reproduce a run's inputs (default random)")
	fs.Var(&headers, "H", "header for every request, 'Name: value' (repeatable)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if *remote == "" {
		return nil, errors.New("-remote is required")
	}
	if *spec == "" {
		return nil, errors.New("-spec is required")
	}

	target, err := parseRemote(*remote)
	if err != nil {
		return nil, fmt.Errorf("-remote: %w", err)
	}

	filter, err := fuzzer.ParseStatusFilter(*status)
	if err != nil {
		return nil, fmt.Errorf("-status: %w", err)
	}

	bounds.Charset = generator.Charset(*charset)
	if err := bounds.Validate(); err != nil {
		return nil, err
	}

	switch {
	case *requests < 0:
		return nil, fmt.Errorf("-n must not be negative (got %d)", *requests)
	case *concurrency < 1:
		return nil, fmt.Errorf("-c must be at least 1 (got %d)", *concurrency)
	case *timeout <= 0:
		return nil, fmt.Errorf("-timeout must be positive (got %s)", *timeout)
	}

	if *seed == 0 {
		*seed = rand.Uint64()
	}

	return &Config{
		Remote:      target,
		SpecPath:    *spec,
		DictPath:    *dict,
		Output:      *reportPath,
		Filter:      filter,
		Bounds:      bounds,
		Requests:    *requests,
		Concurrency: *concurrency,
		Timeout:     *timeout,
		Insecure:    *insecure,
		Headers:     headers.header,
		Seed:        *seed,
	}, nil
}
