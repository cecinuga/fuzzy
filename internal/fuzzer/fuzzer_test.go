package fuzzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
)

func compileSpec(t *testing.T, remote, spec string) ([]*Target, []Skipped) {
	t.Helper()
	doc, err := openapi.Parse([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(remote)
	if err != nil {
		t.Fatal(err)
	}
	targets, skipped := Compile(doc, generator.New(generator.DefaultBounds(), nil, 1), u)
	return targets, skipped
}

func paths(body string) string {
	return `{"openapi":"3.0.0","paths":` + body + `}`
}

func options() Options {
	return Options{Requests: 20, Concurrency: 4, Timeout: 5 * time.Second, Seed: 42}
}

// capture records every request a test server receives.
type capture struct {
	mu       sync.Mutex
	requests []capturedRequest
}

type capturedRequest struct {
	method, escapedPath, rawQuery string
	header                        http.Header
	body                          []byte
}

func (c *capture) server(t *testing.T, status int) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		c.mu.Lock()
		c.requests = append(c.requests, capturedRequest{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, r.Header.Clone(), body})
		c.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSendsSemanticallyValidPathValues(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, paths(`{"/user/{id}":{"get":{"parameters":[
		{"name":"id","in":"path","required":true,"schema":{"type":"integer","minimum":1,"maximum":5}}
	]}}}`))

	outcomes := Run(context.Background(), options(), targets)

	if outcomes[0].Sent != 20 || len(got.requests) != 20 {
		t.Fatalf("sent %d, server saw %d, want 20", outcomes[0].Sent, len(got.requests))
	}
	for _, r := range got.requests {
		id, err := strconv.Atoi(strings.TrimPrefix(r.escapedPath, "/user/"))
		if err != nil || id < 1 || id > 5 {
			t.Errorf("bad path %q", r.escapedPath)
		}
	}
	first := outcomes[0].Results[0]
	if !strings.HasPrefix(first.Endpoint, "GET /user/") || !strings.HasPrefix(first.Input, "id=") {
		t.Errorf("result = %+v", first)
	}
}

func TestUnsafePathValuesStayInOneSegment(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a/{v}/b":{"get":{"parameters":[
		{"name":"v","in":"path","required":true,"schema":{"type":"string","enum":["x/../../y z"]}}
	]}}}`))
	Run(context.Background(), options(), targets)

	if got.requests[0].escapedPath != "/a/x%2F..%2F..%2Fy%20z/b" {
		t.Fatalf("path = %q", got.requests[0].escapedPath)
	}
}

func TestRemoteAndServerBasePath(t *testing.T) {
	spec := `{"openapi":"3.0.0","servers":[{"url":"https://prod.example.com/api/v3"}],"paths":{"/ping":{"get":{}}}}`
	cases := []struct{ suffix, want string }{
		{"", "/api/v3/ping"},
		{"/", "/api/v3/ping"},
		{"/custom", "/custom/ping"},
		{"/custom/", "/custom/ping"},
	}
	for _, tc := range cases {
		var got capture
		srv := got.server(t, 200)
		targets, _ := compileSpec(t, srv.URL+tc.suffix, spec)
		Run(context.Background(), Options{Requests: 1, Concurrency: 1, Timeout: time.Second}, targets)
		if got.requests[0].escapedPath != tc.want {
			t.Errorf("remote suffix %q: requested %q, want %q", tc.suffix, got.requests[0].escapedPath, tc.want)
		}
	}
}

func TestRequiredAndOptionalParameters(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, paths(`{"/s":{"get":{"parameters":[
		{"name":"req","in":"query","required":true,"schema":{"type":"integer"}},
		{"name":"opt","in":"query","schema":{"type":"integer"}}
	]}}}`))
	opts := options()
	opts.Requests = 200
	Run(context.Background(), opts, targets)

	optional := 0
	for _, r := range got.requests {
		q, _ := url.ParseQuery(r.rawQuery)
		if !q.Has("req") {
			t.Fatalf("required parameter missing in %q", r.rawQuery)
		}
		if q.Has("opt") {
			optional++
		}
	}
	if optional == 0 || optional == 200 {
		t.Errorf("optional parameter sent %d/200 times, want a mix", optional)
	}
}

func TestParameterStyles(t *testing.T) {
	bools := func(v bool) *bool { return &v }
	array := []any{"a b", int64(2)}
	object := map[string]any{"x": int64(1), "y": "z w"}

	query := []struct {
		name string
		p    openapi.Parameter
		v    any
		want string
	}{
		{"scalar", openapi.Parameter{}, "a&b", "n=a%26b"},
		{"form explode", openapi.Parameter{}, array, "n=a+b&n=2"},
		{"form plain", openapi.Parameter{Explode: bools(false)}, array, "n=a+b,2"},
		{"space", openapi.Parameter{Style: "spaceDelimited", Explode: bools(false)}, array, "n=a+b%202"},
		{"pipe", openapi.Parameter{Style: "pipeDelimited", Explode: bools(false)}, array, "n=a+b%7C2"},
		{"object explode", openapi.Parameter{}, object, "x=1&y=z+w"},
		{"object plain", openapi.Parameter{Explode: bools(false)}, object, "n=x,1,y,z+w"},
		{"deepObject", openapi.Parameter{Style: "deepObject"}, object, "n%5Bx%5D=1&n%5By%5D=z+w"},
		{"deep array", openapi.Parameter{Style: "deepObject"}, array, "n%5B0%5D=a+b&n%5B1%5D=2"},
		{"deep nested", openapi.Parameter{Style: "deepObject"}, map[string]any{"a": map[string]any{"b": []any{int64(1)}}}, "n%5Ba%5D%5Bb%5D%5B0%5D=1"},
		{"deep scalar", openapi.Parameter{Style: "deepObject"}, int64(5), "n=5"},
		{"content", openapi.Parameter{ContentType: "application/json"}, object, "n=%7B%22x%22%3A1%2C%22y%22%3A%22z+w%22%7D"},
	}
	for _, tc := range query {
		t.Run("query "+tc.name, func(t *testing.T) {
			tc.p.Name, tc.p.In = "n", "query"
			if got := strings.Join(encodeQuery(&tc.p, tc.v), "&"); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}

	path := []struct {
		name string
		p    openapi.Parameter
		v    any
		want string
	}{
		{"simple scalar", openapi.Parameter{}, "a/b", "a%2Fb"},
		{"simple array", openapi.Parameter{}, array, "a%20b,2"},
		{"simple object", openapi.Parameter{}, object, "x,1,y,z%20w"},
		{"simple explode", openapi.Parameter{Explode: bools(true)}, object, "x=1,y=z%20w"},
		{"label", openapi.Parameter{Style: "label"}, array, ".a%20b.2"},
		{"label object", openapi.Parameter{Style: "label", Explode: bools(true)}, object, ".x=1.y=z%20w"},
		{"matrix scalar", openapi.Parameter{Style: "matrix"}, "v", ";n=v"},
		{"matrix array", openapi.Parameter{Style: "matrix"}, array, ";n=a%20b,2"},
		{"matrix explode", openapi.Parameter{Style: "matrix", Explode: bools(true)}, array, ";n=a%20b;n=2"},
		{"matrix object", openapi.Parameter{Style: "matrix", Explode: bools(true)}, object, ";x=1;y=z%20w"},
	}
	for _, tc := range path {
		t.Run("path "+tc.name, func(t *testing.T) {
			tc.p.Name, tc.p.In = "n", "path"
			if got := encodePath(&tc.p, tc.v); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}

	h := openapi.Parameter{Name: "n", In: "header"}
	if got := encodeHeader(&h, array); got != "a b,2" {
		t.Errorf("header = %q", got)
	}
	c := openapi.Parameter{Name: "n", In: "cookie"}
	if got := encodeCookie(&c, "a;b"); got != "n=a%3Bb" {
		t.Errorf("cookie = %q", got)
	}
}

func TestHeaderAndCookieParameters(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, paths(`{"/s":{"get":{"parameters":[
		{"name":"X-Trace","in":"header","required":true,"schema":{"type":"string","format":"uuid"}},
		{"name":"sid","in":"cookie","required":true,"schema":{"type":"integer"}}
	]}}}`))
	opts := options()
	opts.Headers = http.Header{"Authorization": {"Bearer t"}, "X-Trace": {"override"}}
	Run(context.Background(), opts, targets)

	r := got.requests[0]
	if r.header.Get("Authorization") != "Bearer t" {
		t.Errorf("custom header missing: %v", r.header)
	}
	if r.header.Get("X-Trace") != "override" {
		t.Errorf("custom headers must override generated ones: %v", r.header.Get("X-Trace"))
	}
	if !strings.HasPrefix(r.header.Get("Cookie"), "sid=") {
		t.Errorf("cookie = %q", r.header.Get("Cookie"))
	}
}

func TestBodyEncodings(t *testing.T) {
	cases := []struct {
		name, content, wantType string
		check                   func(body []byte) bool
	}{
		{"json", "application/json", "application/json", func(b []byte) bool { return strings.HasPrefix(string(b), `{"a":`) }},
		{"vendor json", "application/vnd.api+json", "application/vnd.api+json", func(b []byte) bool { return strings.HasPrefix(string(b), `{"a":`) }},
		{"wildcard", "*/*", "application/json", func(b []byte) bool { return strings.HasPrefix(string(b), `{"a":`) }},
		{"form", "application/x-www-form-urlencoded", "application/x-www-form-urlencoded", func(b []byte) bool {
			v, err := url.ParseQuery(string(b))
			return err == nil && v.Has("a")
		}},
		{"multipart", "multipart/form-data", "multipart/form-data", func(b []byte) bool { return strings.Contains(string(b), `name="a"`) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got capture
			srv := got.server(t, 200)
			targets, _ := compileSpec(t, srv.URL, paths(`{"/b":{"post":{"requestBody":{"required":true,"content":{"`+tc.content+`":{"schema":
				{"type":"object","required":["a"],"properties":{"a":{"type":"integer"}}}}}}}}}`))
			Run(context.Background(), options(), targets)

			r := got.requests[0]
			if contentType := r.header.Get("Content-Type"); !strings.HasPrefix(contentType, tc.wantType) {
				t.Errorf("Content-Type = %q, want %q", contentType, tc.wantType)
			}
			if !tc.check(r.body) {
				t.Errorf("body = %q", r.body)
			}
		})
	}
}

func TestJSONIsPreferredAmongMediaTypes(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, paths(`{"/b":{"post":{"requestBody":{"required":true,"content":{
		"application/xml":{"schema":{"type":"string"}},
		"application/x-www-form-urlencoded":{"schema":{"type":"string"}},
		"application/json":{"schema":{"type":"string"}}
	}}}}}`))
	Run(context.Background(), options(), targets)
	if contentType := got.requests[0].header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q", contentType)
	}
}

func TestUnsupportedRequiredBodyIsSkippedOptionalIsOmitted(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, skipped := compileSpec(t, srv.URL, paths(`{
		"/required":{"post":{"requestBody":{"required":true,"content":{"application/xml":{"schema":{"type":"string"}}}}}},
		"/optional":{"post":{"requestBody":{"content":{"application/xml":{"schema":{"type":"string"}}}}}}
	}`))
	if len(skipped) != 1 || skipped[0].Op.Path != "/required" || !strings.Contains(skipped[0].Reason, "application/xml") {
		t.Fatalf("skipped = %+v", skipped)
	}
	Run(context.Background(), options(), targets)
	for _, r := range got.requests {
		if r.escapedPath != "/optional" || len(r.body) != 0 {
			t.Errorf("unexpected request %q with body %q", r.escapedPath, r.body)
		}
	}
}

func TestRedirectsAreReportedNotFollowed(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/r":{"get":{}}}`))

	outcomes := Run(context.Background(), options(), targets)

	if outcomes[0].Counts[302] != 20 || hits.Load() != 20 {
		t.Fatalf("counts = %v, hits = %d", outcomes[0].Counts, hits.Load())
	}
}

func TestStatusFilterKeepsCountsButDropsResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/x/"))
		if id%2 == 0 {
			w.WriteHeader(500)
			fmt.Fprint(w, "boom\n  at line 3")
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/x/{id}":{"get":{"parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"integer"}}]}}}`))

	opts := options()
	opts.Requests = 100
	opts.Filter, _ = ParseStatusFilter("5xx")
	out := Run(context.Background(), opts, targets)[0]

	if out.Counts[500]+out.Counts[404] != 100 || out.Counts[500] == 0 || out.Counts[404] == 0 {
		t.Fatalf("counts = %v", out.Counts)
	}
	if len(out.Results) != out.Counts[500] {
		t.Fatalf("kept %d results, want %d", len(out.Results), out.Counts[500])
	}
	for _, res := range out.Results {
		if res.Status != 500 || res.Message != "boom at line 3" {
			t.Fatalf("result = %+v", res)
		}
	}
}

func TestTransportFailuresBecomeErrResults(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	remote := srv.URL
	srv.Close() // nothing listens any more

	targets, _ := compileSpec(t, remote, paths(`{"/x":{"get":{}}}`))
	opts := options()
	opts.Requests = 3
	opts.Filter, _ = ParseStatusFilter("err")
	out := Run(context.Background(), opts, targets)[0]

	if out.Counts[TransportError] != 3 || len(out.Results) != 3 {
		t.Fatalf("outcome = %+v", out)
	}
	if out.Results[0].Message == "" {
		t.Error("the failure reason should be the message")
	}
	if strings.Contains(out.Results[0].Message, remote) {
		t.Errorf("message repeats the URL: %q", out.Results[0].Message)
	}
}

func TestTimeoutIsReported(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)

	targets, _ := compileSpec(t, srv.URL, paths(`{"/slow":{"get":{}}}`))
	opts := Options{Requests: 1, Concurrency: 1, Timeout: 50 * time.Millisecond}
	if out := Run(context.Background(), opts, targets)[0]; out.Counts[TransportError] != 1 {
		t.Fatalf("counts = %v", out.Counts)
	}
}

func TestConcurrencyIsBounded(t *testing.T) {
	var current, peak atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		now := current.Add(1)
		defer current.Add(-1)
		for {
			seen := peak.Load()
			if now <= seen || peak.CompareAndSwap(seen, now) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
	}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{}},"/b":{"get":{}}}`))

	opts := Options{Requests: 40, Concurrency: 5, Timeout: 5 * time.Second}
	outcomes := Run(context.Background(), opts, targets)

	if got := peak.Load(); got > 5 || got < 2 {
		t.Errorf("peak concurrency = %d, want between 2 and 5", got)
	}
	if outcomes[0].Sent != 40 || outcomes[1].Sent != 40 {
		t.Errorf("sent %d and %d, want 40 each", outcomes[0].Sent, outcomes[1].Sent)
	}
}

func TestCancellationStopsTheRunAndKeepsResults(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 10 {
			cancel()
		}
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Millisecond):
		}
	}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{}}}`))

	opts := Options{Requests: 100000, Concurrency: 4, Timeout: 5 * time.Second}
	done := make(chan []Outcome, 1)
	go func() { done <- Run(ctx, opts, targets) }()

	select {
	case outcomes := <-done:
		if outcomes[0].Sent == 0 || outcomes[0].Sent == 100000 {
			t.Errorf("sent = %d, want a partial run", outcomes[0].Sent)
		}
		if outcomes[0].Counts[TransportError] != 0 {
			t.Errorf("cancelled calls must not look like failures of the target: %v", outcomes[0].Counts)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
}

func TestSameSeedSameInputsAtAnyConcurrency(t *testing.T) {
	spec := paths(`{"/a/{id}":{"post":{
		"parameters":[
			{"name":"id","in":"path","required":true,"schema":{"type":"integer"}},
			{"name":"q","in":"query","schema":{"type":"string"}}
		],
		"requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","properties":{
			"s":{"type":"string"},"f":{"type":"number"},"t":{"type":"array","items":{"type":"integer"}}
		}}}}}
	}}}`)
	run := func(concurrency int) []string {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		defer srv.Close()
		targets, _ := compileSpec(t, srv.URL, spec)
		opts := options()
		opts.Requests = 200
		opts.Concurrency = concurrency
		out := Run(context.Background(), opts, targets)[0]

		inputs := make([]string, len(out.Results))
		for i, res := range out.Results {
			inputs[i] = res.Endpoint + " " + res.Input
		}
		return inputs
	}
	serial, parallel := run(1), run(16)
	if !slices.Equal(serial, parallel) {
		t.Fatalf("inputs differ between concurrency levels:\n%v\n%v", serial[:3], parallel[:3])
	}
	if len(slices.Compact(slices.Sorted(slices.Values(serial)))) < 150 {
		t.Error("inputs look like they barely vary")
	}
}

func TestParseStatusFilter(t *testing.T) {
	f, err := ParseStatusFilter("200, 4XX,500-502,err")
	if err != nil {
		t.Fatal(err)
	}
	match := map[int]bool{200: true, 201: false, 400: true, 499: true, 500: true, 502: true, 503: false, TransportError: true}
	for status, want := range match {
		if f.Match(status) != want {
			t.Errorf("Match(%d) = %v, want %v", status, !want, want)
		}
	}

	all, _ := ParseStatusFilter("")
	if !all.Match(TransportError) || !all.Match(204) || all.String() != "all" {
		t.Error("an empty filter keeps everything")
	}
	var zero StatusFilter
	if !zero.Match(500) {
		t.Error("the zero filter keeps everything")
	}
	only5xx, _ := ParseStatusFilter("5xx")
	if only5xx.Match(TransportError) {
		t.Error("errors are only kept when asked for")
	}

	for _, bad := range []string{"abc", "99", "600", "6xx", "0xx", "500-400", "2x", "200,", "-5"} {
		if _, err := ParseStatusFilter(bad); err == nil {
			t.Errorf("ParseStatusFilter(%q) should fail", bad)
		}
	}
}

func TestMessage(t *testing.T) {
	if got := message(nil, 404); got != "Not Found" {
		t.Errorf("empty body: %q", got)
	}
	if got := message([]byte("  a\n\tb  "), 200); got != "a b" {
		t.Errorf("whitespace: %q", got)
	}
	long := strings.Repeat("é", 500)
	if got := message([]byte(long), 200); len([]rune(got)) != maxMsgRunes+1 {
		t.Errorf("long body: %d runes", len([]rune(got)))
	}
	if got := message([]byte{0xff, 0xfe, 'o', 'k'}, 200); got != "ok" {
		t.Errorf("invalid UTF-8: %q", got)
	}
}

func TestLargeResponsesAreBoundedInMemoryAndMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		chunk := []byte(strings.Repeat("x", 1<<16))
		for range 64 { // 4 MiB, beyond the drain limit
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/big":{"get":{}}}`))
	out := Run(context.Background(), options(), targets)[0]
	if out.Counts[200] != 20 {
		t.Fatalf("counts = %v", out.Counts)
	}
	if n := len(out.Results[0].Message); n > 2*maxMsgRunes {
		t.Errorf("message is %d bytes", n)
	}
}

func TestOutcomesFollowTargetOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{}},"/b":{"get":{},"delete":{}},"/c":{"put":{}}}`))
	outcomes := Run(context.Background(), options(), targets)
	got := map[string]bool{}
	for i, o := range outcomes {
		if o.Target != targets[i] {
			t.Errorf("outcome %d does not match its target", i)
		}
		got[o.Target.Op.Method+o.Target.Op.Path] = true
	}
	if want := []string{"DELETE/b", "GET/a", "GET/b", "PUT/c"}; !slices.Equal(slices.Sorted(maps.Keys(got)), want) {
		t.Errorf("targets = %v", got)
	}
}

func TestFormBodyFollowsDeclaredEncoding(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, paths(`{"/f":{"post":{"requestBody":{"required":true,"content":{"application/x-www-form-urlencoded":{
		"encoding":{"deep":{"style":"deepObject","explode":true},"csv":{"style":"form","explode":false}},
		"schema":{"type":"object","required":["deep","csv","flat","many"],"properties":{
			"deep":{"type":"object","required":["k"],"properties":{"k":{"type":"string","enum":["v"]}}},
			"csv":{"type":"array","minItems":2,"maxItems":2,"items":{"type":"string","enum":["x"]}},
			"flat":{"type":"object","required":["p"],"properties":{"p":{"type":"string","enum":["q"]}}},
			"many":{"type":"array","minItems":2,"maxItems":2,"items":{"type":"string","enum":["y"]}}
		}}}}}}}}`))
	Run(context.Background(), options(), targets)

	want := "csv=x,x&deep%5Bk%5D=v&p=q&many=y&many=y"
	if body := string(got.requests[0].body); body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func nullableBodySpec(contentType string) string {
	return paths(`{"/b":{"post":{"requestBody":{"required":true,"content":{"` + contentType + `":{"schema":
		{"type":"object","required":["n","s"],"properties":{"n":{"type":"integer","nullable":true},"s":{"type":"string","nullable":true}}}}}}}}}`)
}

func TestNullInJSONBody(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, nullableBodySpec("application/json"))
	opts := options()
	opts.Requests = 300
	Run(context.Background(), opts, targets)

	nulls := 0
	for _, r := range got.requests {
		var body map[string]any
		if err := json.Unmarshal(r.body, &body); err != nil {
			t.Fatalf("body %q: %v", r.body, err)
		}
		for _, key := range []string{"n", "s"} {
			value, present := body[key]
			if !present {
				t.Fatalf("required %q missing from %q", key, r.body)
			}
			if value == nil {
				nulls++
			}
		}
	}
	if nulls == 0 {
		t.Error("nullable fields should sometimes be a JSON null")
	}
}

func TestNullInMultipartBodyIsAJSONPart(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, nullableBodySpec("multipart/form-data"))
	opts := options()
	opts.Requests = 300
	Run(context.Background(), opts, targets)

	nulls, values := 0, 0
	for _, r := range got.requests {
		_, params, err := mime.ParseMediaType(r.header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}
		reader := multipart.NewReader(bytes.NewReader(r.body), params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("body %q: %v", r.body, err)
			}
			content, _ := io.ReadAll(part)
			if part.Header.Get("Content-Type") == "application/json" {
				nulls++
				if string(content) != "null" || part.FormName() == "" {
					t.Errorf("part %q has content %q", part.FormName(), content)
				}
				continue
			}
			values++
			if len(content) == 0 {
				t.Errorf("part %q is empty: null must not become an empty string", part.FormName())
			}
		}
	}
	if nulls == 0 || values == 0 {
		t.Errorf("%d null parts and %d value parts, want both", nulls, values)
	}
}

func TestNullIsNeverSentWhereItCannotBeRepresented(t *testing.T) {
	t.Run("parameters", func(t *testing.T) {
		var got capture
		srv := got.server(t, 200)
		targets, _ := compileSpec(t, srv.URL, paths(`{"/p/{a}":{"get":{"parameters":[
			{"name":"a","in":"path","required":true,"schema":{"type":"integer","nullable":true}},
			{"name":"q","in":"query","required":true,"schema":{"type":"integer","nullable":true}},
			{"name":"X-H","in":"header","required":true,"schema":{"type":"integer","nullable":true}},
			{"name":"c","in":"cookie","required":true,"schema":{"type":"integer","nullable":true}}
		]}}}`))
		opts := options()
		opts.Requests = 300
		Run(context.Background(), opts, targets)

		for _, r := range got.requests {
			q, _ := url.ParseQuery(r.rawQuery)
			if _, err := strconv.Atoi(strings.TrimPrefix(r.escapedPath, "/p/")); err != nil {
				t.Fatalf("path %q", r.escapedPath)
			}
			if _, err := strconv.Atoi(q.Get("q")); err != nil {
				t.Fatalf("query %q", r.rawQuery)
			}
			if _, err := strconv.Atoi(r.header.Get("X-H")); err != nil {
				t.Fatalf("header %q", r.header.Get("X-H"))
			}
			if _, err := strconv.Atoi(strings.TrimPrefix(r.header.Get("Cookie"), "c=")); err != nil {
				t.Fatalf("cookie %q", r.header.Get("Cookie"))
			}
		}
	})

	t.Run("form", func(t *testing.T) {
		var got capture
		srv := got.server(t, 200)
		targets, _ := compileSpec(t, srv.URL, nullableBodySpec("application/x-www-form-urlencoded"))
		opts := options()
		opts.Requests = 300
		Run(context.Background(), opts, targets)

		for _, r := range got.requests {
			form, err := url.ParseQuery(string(r.body))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := strconv.Atoi(form.Get("n")); err != nil || form.Get("s") == "" {
				t.Fatalf("form body %q carries an empty value", r.body)
			}
		}
	})

	for _, contentType := range []string{"text/plain", "application/octet-stream"} {
		t.Run(contentType, func(t *testing.T) {
			var got capture
			srv := got.server(t, 200)
			targets, _ := compileSpec(t, srv.URL, paths(`{"/t":{"post":{"requestBody":{"required":true,"content":{"`+contentType+`":{"schema":{"type":"string","nullable":true}}}}}}}`))
			opts := options()
			opts.Requests = 300
			Run(context.Background(), opts, targets)

			for _, r := range got.requests {
				if len(r.body) == 0 {
					t.Fatal("a null body became an empty one")
				}
			}
		})
	}
}

func TestProgressReportsRunningTotals(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{"responses":{"200":{"description":"ok"}}}}}`))

	// Run with a filter that keeps nothing: progress must count all the same.
	filter, err := ParseStatusFilter("200")
	if err != nil {
		t.Fatal(err)
	}
	var snapshots []Progress
	opts := options()
	opts.Requests = 100
	opts.Concurrency = 1
	opts.Filter = filter
	opts.ProgressEvery = time.Millisecond
	opts.Progress = func(p Progress) { snapshots = append(snapshots, p) }
	Run(context.Background(), opts, targets)

	if len(snapshots) == 0 {
		t.Fatal("no progress reported")
	}
	last := Progress{}
	for _, p := range snapshots {
		if p.Total != 100 {
			t.Fatalf("total = %d, want 100", p.Total)
		}
		if p.Done < last.Done || p.Elapsed < last.Elapsed {
			t.Fatalf("progress went backwards: %+v after %+v", p, last)
		}
		if p.ServerErrors != p.Done || p.TransportErrors != 0 {
			t.Fatalf("every call is a 500, got %+v", p)
		}
		last = p
	}
	if last.Done == 0 {
		t.Errorf("no call counted in %d snapshots", len(snapshots))
	}
}

func TestProgressCountsTransportErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	remote := srv.URL
	srv.Close() // nothing listens here any more
	targets, _ := compileSpec(t, remote, paths(`{"/a":{"get":{"responses":{"200":{"description":"ok"}}}}}`))

	var last Progress
	opts := options()
	opts.Requests = 2000
	opts.Concurrency = 1
	opts.ProgressEvery = time.Millisecond
	opts.Progress = func(p Progress) { last = p }
	Run(context.Background(), opts, targets)

	if last.Done == 0 || last.TransportErrors != last.Done || last.ServerErrors != 0 {
		t.Errorf("last snapshot = %+v", last)
	}
}

func TestProgressTracksEachEndpoint(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{"responses":{"200":{"description":"ok"}}}},"/b":{"get":{"responses":{"200":{"description":"ok"}}}}}`))

	states := func(p Progress) string {
		return fmt.Sprint(p.Endpoints[0].State, p.Endpoints[1].State)
	}
	var first, last string
	var once sync.Once
	opts := options()
	opts.Concurrency = 1 // only /a can be dispatched while the server holds the first call
	opts.Requests = 2
	opts.ProgressEvery = time.Millisecond
	opts.Progress = func(p Progress) {
		if p.Endpoints[0].Name != "GET /a" || p.Endpoints[1].Name != "GET /b" {
			t.Errorf("endpoints = %+v", p.Endpoints)
		}
		once.Do(func() {
			first = states(p)
			close(release)
		})
		if p.Final {
			last = states(p)
		}
	}
	Run(context.Background(), opts, targets)

	if want := fmt.Sprint(EndpointTesting, EndpointPending); first != want {
		t.Errorf("first snapshot = %s, want %s", first, want)
	}
	if want := fmt.Sprint(EndpointDone, EndpointDone); last != want {
		t.Errorf("final snapshot = %s, want %s", last, want)
	}
}

func TestInterruptedEndpointIsNotDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		if calls.Add(1) == 10 {
			cancel()
		}
	}))
	defer srv.Close()
	targets, _ := compileSpec(t, srv.URL, paths(`{"/a":{"get":{"responses":{"200":{"description":"ok"}}}}}`))

	var final Progress
	opts := options()
	opts.Requests = 100000
	opts.ProgressEvery = time.Hour
	opts.Progress = func(p Progress) { final = p }
	Run(ctx, opts, targets)

	if !final.Final || final.Endpoints[0].State != EndpointTesting {
		t.Errorf("final snapshot = %+v, want the endpoint still testing", final)
	}
}
