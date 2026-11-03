// Package fuzzer sends generated requests to a remote API with a bounded worker
// pool and collects what comes back.
package fuzzer

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	snippetSize = 1 << 10 // bytes of a response body inspected for the message
	drainLimit  = 1 << 20 // bytes drained to keep a connection reusable
	maxMsgRunes = 200
)

// Options configures a run.
type Options struct {
	Requests    int // requests per operation
	Concurrency int
	Timeout     time.Duration
	Insecure    bool
	Headers     http.Header // sent with every request, overriding generated ones
	Seed        uint64
	Filter      StatusFilter

	// Progress, if set, receives a snapshot every ProgressEvery. It runs on the
	// goroutine that collects results, so it must return quickly.
	Progress      func(Progress)
	ProgressEvery time.Duration
}

// Progress is a run's state at one moment. Like an Outcome's Counts, it covers
// every call, whatever the status filter keeps for the report.
type Progress struct {
	Done            int // calls completed so far
	Total           int // calls the run will make if it isn't interrupted
	ServerErrors    int // calls answered with a 5xx
	TransportErrors int // calls that got no response
	Elapsed         time.Duration

	// Endpoints has one entry per target, in target order. The slice is reused
	// between snapshots, so it is only valid during the Progress call.
	Endpoints []EndpointProgress

	// Final marks the last snapshot, sent once when the run ends even if it was
	// interrupted, whatever ProgressEvery says.
	Final bool
}

// EndpointState says how far a run has got with one endpoint.
type EndpointState int

const (
	EndpointPending EndpointState = iota // no request sent yet
	EndpointTesting                      // sending requests
	EndpointDone                         // every request sent
)

// EndpointProgress is one endpoint in a Progress snapshot.
type EndpointProgress struct {
	Name  string // as written in the spec, so not safe to print on a terminal as is
	State EndpointState
}

// Result is the outcome of one call: (endpoint, status code, message, input).
type Result struct {
	Endpoint string
	Status   int // TransportError when no response was received
	Message  string
	Input    string
	iter     int
}

// Outcome aggregates the calls made to one target. Counts always covers every
// call; Results only holds those that passed the status filter.
type Outcome struct {
	Target  *Target
	Sent    int
	Counts  map[int]int
	Results []Result
}

type job struct{ target, iter int }

type record struct {
	target int
	result Result
}

// Run fuzzes all targets and returns one Outcome per target, in target order.
// Cancelling ctx stops the run early; what was collected so far is still returned.
func Run(ctx context.Context, opts Options, targets []*Target) []Outcome {
	client := newClient(opts)
	jobs := make(chan job)
	records := make(chan record, opts.Concurrency)
	started := make([]atomic.Bool, len(targets))

	// Rounds go over all targets rather than target by target, so an
	// interrupted run still has covered every endpoint evenly.
	go func() {
		defer close(jobs)
		for iter := range opts.Requests {
			for i := range targets {
				select {
				case jobs <- job{i, iter}:
					if iter == 0 {
						started[i].Store(true)
					}
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	var workers sync.WaitGroup
	for range opts.Concurrency {
		workers.Go(func() {
			w := newWorker(client, opts)
			for j := range jobs {
				result, ok := w.call(ctx, targets[j.target], j)
				if ok {
					records <- record{j.target, result}
				}
			}
		})
	}
	go func() {
		workers.Wait()
		close(records)
	}()

	return collect(records, opts, targets, started)
}

// collect is the only owner of the outcomes, so they need no locking.
func collect(records <-chan record, opts Options, targets []*Target, dispatched []atomic.Bool) []Outcome {
	outcomes := make([]Outcome, len(targets))
	for i, t := range targets {
		outcomes[i] = Outcome{Target: t, Counts: map[int]int{}}
	}

	// A nil channel is never ready, which turns the tick case off.
	var tick <-chan time.Time
	if opts.Progress != nil {
		ticker := time.NewTicker(opts.ProgressEvery)
		defer ticker.Stop()
		tick = ticker.C
	}

	began := time.Now()
	progress := Progress{Total: len(targets) * opts.Requests}
	if opts.Progress != nil {
		progress.Endpoints = make([]EndpointProgress, len(targets))
		for i, t := range targets {
			progress.Endpoints[i].Name = t.Op.Method + " " + t.Op.Path
		}
	}
	report := func() {
		progress.Elapsed = time.Since(began)
		for i := range outcomes {
			progress.Endpoints[i].State = endpointState(outcomes[i].Sent, opts.Requests, dispatched[i].Load())
		}
		opts.Progress(progress)
	}

	for {
		select {
		case rec, open := <-records:
			if !open {
				for i := range outcomes {
					// Workers finish out of order; sorting makes the report reproducible.
					slices.SortFunc(outcomes[i].Results, func(a, b Result) int { return a.iter - b.iter })
				}
				if opts.Progress != nil {
					progress.Final = true
					report()
				}
				return outcomes
			}
			progress.Done++
			switch status := rec.result.Status; {
			case status == TransportError:
				progress.TransportErrors++
			case status >= 500:
				progress.ServerErrors++
			}
			out := &outcomes[rec.target]
			out.Sent++
			out.Counts[rec.result.Status]++
			if opts.Filter.Match(rec.result.Status) {
				out.Results = append(out.Results, rec.result)
			}
		case <-tick:
			report()
		}
	}
}

// endpointState goes by the calls completed, so an endpoint interrupted before
// its last response never counts as done. dispatched is checked too because a
// call in flight has no record yet.
func endpointState(sent, requests int, dispatched bool) EndpointState {
	switch {
	case sent >= requests:
		return EndpointDone
	case sent > 0 || dispatched:
		return EndpointTesting
	}
	return EndpointPending
}

func newClient(opts Options) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = opts.Concurrency
	transport.MaxIdleConnsPerHost = opts.Concurrency
	transport.DisableCompression = true
	if opts.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
		// A redirect is itself a result, and following it could leave the target.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// worker holds the per-goroutine state that makes calls allocation-light.
type worker struct {
	client  *http.Client
	opts    Options
	pcg     *rand.PCG
	rng     *rand.Rand
	snippet []byte
}

func newWorker(client *http.Client, opts Options) *worker {
	pcg := rand.NewPCG(0, 0)
	return &worker{client: client, opts: opts, pcg: pcg, rng: rand.New(pcg), snippet: make([]byte, snippetSize)}
}

// call performs one job. The random stream derives only from the seed and the
// job, never from scheduling, so a seed reproduces the same inputs at any
// concurrency. ok is false when the call was abandoned because of cancellation.
func (w *worker) call(ctx context.Context, t *Target, j job) (Result, bool) {
	w.pcg.Seed(w.opts.Seed, mix(uint64(j.target)<<32|uint64(j.iter)))

	req, input, err := t.build(ctx, w.rng, j.iter)
	result := Result{Endpoint: t.Op.Method + " " + t.Op.Path, Input: input, iter: j.iter}
	if err != nil {
		result.Message = err.Error()
		return result, true
	}
	result.Endpoint = t.Op.Method + " " + req.URL.EscapedPath()
	w.applyHeaders(req)

	res, err := w.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return result, false
		}
		result.Message = transportMessage(err)
		return result, true
	}
	defer res.Body.Close()

	n, _ := io.ReadFull(res.Body, w.snippet)
	_, _ = io.CopyN(io.Discard, res.Body, drainLimit)

	result.Status = res.StatusCode
	result.Message = message(w.snippet[:n], res.StatusCode)
	return result, true
}

func (w *worker) applyHeaders(req *http.Request) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "fuzzy")
	}
	for name, values := range w.opts.Headers {
		if strings.EqualFold(name, "Host") {
			req.Host = values[len(values)-1]
			continue
		}
		req.Header[name] = values
	}
}

// mix is the SplitMix64 finalizer: consecutive job numbers must not produce
// visibly related random streams.
func mix(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	x = (x ^ (x >> 30)) * 0xBF58476D1CE4E5B9
	x = (x ^ (x >> 27)) * 0x94D049BB133111EB
	return x ^ (x >> 31)
}

// transportMessage drops url.Error's method and URL: the result already has them.
func transportMessage(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err.Error()
	}
	return err.Error()
}

// message condenses a response into one line: the start of the body, or the
// standard status text when the body is empty.
func message(body []byte, status int) string {
	text := strings.Join(strings.Fields(strings.ToValidUTF8(string(body), "")), " ")
	if text == "" {
		return http.StatusText(status)
	}
	if runes := []rune(text); len(runes) > maxMsgRunes {
		return string(runes[:maxMsgRunes]) + "…"
	}
	return text
}
