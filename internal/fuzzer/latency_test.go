package fuzzer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatencyWithoutSamples(t *testing.T) {
	var l Latency
	if l.Mean() != 0 || l.Percentile(99) != 0 || l.Count != 0 {
		t.Errorf("empty latency = %+v", l)
	}
}

func TestLatencyFigures(t *testing.T) {
	var l Latency
	for ms := 1; ms <= 100; ms++ {
		l.add(time.Duration(ms) * time.Millisecond)
	}
	if l.Count != 100 || l.Min != time.Millisecond || l.Max != 100*time.Millisecond {
		t.Errorf("count %d, min %s, max %s", l.Count, l.Min, l.Max)
	}
	if want := 50500 * time.Microsecond; l.Mean() != want {
		t.Errorf("mean = %s, want %s", l.Mean(), want)
	}
	for p, exact := range map[float64]time.Duration{50: 50 * time.Millisecond, 95: 95 * time.Millisecond, 99: 99 * time.Millisecond} {
		got := l.Percentile(p)
		// A bucket spans at most 1/8 of its start, and the end of the bucket is returned.
		if got < exact || got > exact+exact/8 {
			t.Errorf("p%v = %s, want about %s", p, got, exact)
		}
	}
	if got := l.Percentile(100); got != l.Max {
		t.Errorf("p100 = %s, want the maximum", got)
	}
}

func TestLatencySingleSampleIsExact(t *testing.T) {
	var l Latency
	l.add(1234 * time.Microsecond)
	for _, p := range []float64{0, 50, 99, 100} {
		if got := l.Percentile(p); got != 1234*time.Microsecond {
			t.Errorf("p%v = %s", p, got)
		}
	}
}

func TestLatencyBucketsAreContiguousAndInRange(t *testing.T) {
	last := 0
	for _, d := range []time.Duration{-time.Second, 0, time.Microsecond, 15 * time.Microsecond, 16 * time.Microsecond, time.Millisecond, time.Second, time.Hour, 1 << 62} {
		bucket := latencyBucket(d)
		if bucket < last || bucket >= latencyBuckets {
			t.Errorf("bucket(%s) = %d, previous %d", d, bucket, last)
		}
		last = bucket
	}
	for us := int64(0); us < 1<<20; us += 1 + us/64 {
		d := time.Duration(us) * time.Microsecond
		if end := latencyBucketEnd(latencyBucket(d)); end < d {
			t.Fatalf("bucket of %s ends at %s, before it", d, end)
		}
	}
}

func TestRunMeasuresResponsesAndSkipsTransportErrors(t *testing.T) {
	const delay = 20 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { time.Sleep(delay) }))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{}}}`))

	opts := options()
	opts.Requests = 4
	opts.Concurrency = 2
	outcome := Run(context.Background(), opts, targets)[0]
	if outcome.Latency.Count != 4 || outcome.Latency.Min < delay {
		t.Errorf("latency = %+v, want 4 samples of at least %s", outcome.Latency, delay)
	}
	for _, r := range outcome.Results {
		if r.Duration < delay {
			t.Errorf("result took %s, want at least %s", r.Duration, delay)
		}
	}

	srv.Close()
	outcome = Run(context.Background(), opts, targets)[0]
	if outcome.Latency.Count != 0 {
		t.Errorf("requests without a response were measured: %+v", outcome.Latency)
	}
	if len(outcome.Results) == 0 || outcome.Results[0].Status != TransportError {
		t.Errorf("results = %+v", outcome.Results)
	}
}

func TestOnResultSeesWhatTheFilterDrops(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{}}}`))

	filter, err := ParseStatusFilter("5xx")
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	opts := options()
	opts.Requests = 7
	opts.Filter = filter
	opts.OnResult = func(r Result) {
		seen++
		if r.Status != 404 {
			t.Errorf("status = %d", r.Status)
		}
	}
	outcome := Run(context.Background(), opts, targets)[0]
	if seen != 7 || len(outcome.Results) != 0 || outcome.Latency.Count != 7 {
		t.Errorf("seen %d, kept %d, measured %d", seen, len(outcome.Results), outcome.Latency.Count)
	}
}
