# 🦄 Fuzzy - HTTP API Fuzzer 🦄

> **Your API's validation layer stops most fuzz traffic. Fuzzy's requests get through it.**
>
> Give Fuzzy an OpenAPI 3.0 or Swagger 2.0 spec and it sends schema-valid requests to every endpoint, so they reach the code behind your validators. No templates, no config, one binary.

![fuzzy with little bit of anxiety because the job is not done yet](assets/fuzzy.png)

Fuzzy is shy, a little anxious when he is among many people, but he works hard because he loves his job.

## Table of Contents 🦆

* [Why Fuzzy](#why-fuzzy-)
* [Quick Start](#quick-start-)
* [Installation](#installation-)
* [Usage](#usage-)
* [Examples](#examples)
* [Command Line Options](#command-line-options-)
* [Known Limitations](#known-limitations-)
* [Use Cases](#use-cases-)
* [Safety and Best Practices](#safety-and-best-practices-)
* [Contributing](#contributing-)
* [Roadmap](#roadmap-)

---

## Why Fuzzy 🦑

Random input usually dies at the front door: a wrong type or a missing field earns a `400` or a `422`, and the code behind the validator never runs. Fuzzy reads your spec and sends requests that pass validation, so what you find comes from the code behind it.

This is a real result from the bundled example API:

```
POST /invoices → 409 {"detail":"due_date 2000-10-24 is before issue_date 2031-06-02"}
{"body":{"currency":"USD","customer":"õひल","due_date":"2000-10-24","issue_date":"2031-06-02","lines":[{"description":"乷ग𠁞Â걎","quantity":317,"unit_price":354}],"tax_rate":20.028}}
```

The body matches the schema in every field, so it got past validation and reached a business rule that the spec can't express. That's the code you want a fuzzer exercising.

### What You Get 🐞

* **Zero setup**: Two flags, `--remote` and `--spec`. Fuzzy works out the endpoints and what each one accepts.
* **Valid requests, however complex the spec**: Respects types, formats, ranges, lengths, patterns, enums, required fields, `$ref`, `allOf`, `oneOf`, `anyOf` and discriminators.
* **Your choice of endpoints**: Include or exclude them by method, path pattern or tag, and preview the result with `--list`.
* **Every place a value can go**: Path, query, header and cookie parameters, plus JSON, form, multipart, text, binary and XML bodies, SOAP envelopes included.
* **Your payloads, where they fit**: Dictionary values are only used where the spec allows them. Anything they don't cover gets a random valid value.
* **Failures you can reproduce**: Every report records its seed, and the same seed produces the same inputs at any concurrency. See [Reproducing a Run](#reproducing-a-run-).
* **Reports for people and scripts**: Markdown, HTML or plain text to read, JSON or XML for CI, CSV for a spreadsheet, chosen by the file extension. Filter by status code without losing each endpoint's totals.
* **Response times**: Each endpoint's minimum, mean, percentiles and maximum, and the time of every reported result.
* **Messages you can read**: XML and HTML answers are reduced to their text, so a SOAP fault reads `faultcode: soap:Server; faultstring: db down`.
* **A log of everything**: `--log` keeps a line for every request, whatever `--status` leaves out of the report.
* **Built for pipelines**: Stable exit codes, and a plain-line progress mode when stderr isn't a terminal.
* **It stays on target**: The host always comes from `--remote`. Fuzzy never calls the server named in the spec, and it reports redirects instead of following them.
* **Ctrl+C loses nothing**: An interrupted run still writes its report, marked as partial. Add `--incremental` and even a killed run leaves its last checkpoint.
* **Easy on your machine**: A single binary, a bounded worker pool on a shared connection pool, and only [Cobra](https://github.com/spf13/cobra) and its flag library as direct dependencies.

### Measured 🐆

| What | Result |
|---|---|
| Speed | 120,000 requests in 20.9 s, about 5,700 req/s |
| Peak memory | 17 MB, at both 6,000 and 120,000 requests |
| Requests rejected as invalid (`422`) | 0 of 120,000 |
| Same seed at `-c 1` and `-c 50` | 600 of 600 inputs identical |
| Binary size | 13 MB |

These come from one Linux machine, fuzzing the in-memory `examples/crud` API on localhost with `-c 20 --status 5xx,err`. A real API will set its own pace. Results that pass `--status` stay in memory for the whole run, so a run that keeps every result uses more.

### Is It for You? 🦝

Fuzzy fits if you have an OpenAPI 3.0.x or Swagger 2.0 spec in JSON and a disposable environment to test against. Keep in mind:

- **It stays inside the spec**: Fuzzy doesn't send malformed requests on purpose, so it won't test how your API handles invalid input.
- **It really calls everything**: `DELETE` and `PUT` included, with valid values, so it changes real data. [Never point it at production](#safety-and-best-practices-).
- **Authentication is up to you**: Pass your credentials with `-H`.

---

## Quick Start 🐇

The repository includes a small in-memory invoice API that is safe to fuzz. It has no planted bugs, so a healthy run reports no `5xx` and no `err`.

**Terminal 1**, start the API:
```bash
git clone https://github.com/cecinuga/fuzzy.git
cd fuzzy
go run ./examples/crud
```

**Terminal 2**, from the same directory, fuzz it:
```bash
go build -o fuzzy ./cmd/fuzzy
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json --seed 42
```

```
[+] fuzzing 6 endpoints of http://localhost:8000, 100 requests each (seed 42)
[+] 600 requests sent, 0 without a response. Report: /home/you/fuzzy/fuzzy-report.md
```

`fuzzy-report.md` opens with a summary, followed by each endpoint's spec and results. This one is from a real run that took 38 ms:

| Endpoint | Sent | Status codes |
|---|---|---|
| GET /invoices | 100 | 200 ×100 |
| POST /invoices | 100 | 201 ×48, 409 ×52 |
| GET /invoices/{invoice_id} | 100 | 200 ×2, 404 ×98 |
| PUT /invoices/{invoice_id} | 100 | 200 ×1, 404 ×99 |
| DELETE /invoices/{invoice_id} | 100 | 204 ×3, 404 ×97 |
| PUT /invoices/{invoice_id}/status | 100 | 200 ×1, 404 ×96, 409 ×3 |

Your counts may differ: the seed fixes the inputs, but the answers depend on the server's state and the order the requests arrive in.

**Add it to CI.** Fuzzy exits with `0` whatever the API answered, so let the JSON report decide:
```bash
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json --status 5xx,err --format json
jq -e '.overview.resultsReported == 0' fuzzy-report.json   # fails on any 5xx or request without a response
```

---

## Installation 🐝

Requires Go 1.25.1+, on Linux, macOS or Windows (Linux recommended for heavy testing).

```bash
git clone https://github.com/cecinuga/fuzzy.git
cd fuzzy
go build -o fuzzy ./cmd/fuzzy
```

---

## Usage 🦥

Fuzzy needs two things: where the API lives and its OpenAPI 3.0 or Swagger 2.0 spec, which it tells apart by itself. Everything else is optional.

### Basic Syntax 🦩

```bash
./fuzzy --remote <URL|FQDN|HOST:PORT> --spec <SPEC_JSON> [OPTIONS]
```

### Choosing the Endpoints 🦔

Fuzzy calls every endpoint of the spec unless you say otherwise. `--include` keeps only the endpoints that match, and `--exclude` leaves out the ones that match. Both can be repeated:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --include '/users/**'
./fuzzy --remote http://localhost:8080 --spec api.json --exclude DELETE --exclude 'POST /reports/*'
```

| Selector | Matches |
|---|---|
| `GET` or `GET,POST` | Those methods |
| `/users/*` | One path segment in place of the `*`: `/users/{id}`, but not `/users` or `/users/{id}/orders` |
| `/users/**` | Any number of segments, none included: `/users`, `/users/{id}`, `/users/{id}/orders` |
| `/user/{id}` | That exact path |
| `tag:billing` | The operations with that tag in the spec |
| `GET /user/{id}`, `POST tag:billing` | Methods followed by a path or a tag, both must match |

An endpoint runs if it matches any `--include`, or if there are none, and no `--exclude`, so `--exclude` always wins. Paths are written as in the spec, without the base path of its `servers`, and a wildcard has to be a whole segment. Quote selectors with `*`, `{` or spaces, or your shell may change them.

`--list` shows what a run would call and exits, without sending a request or writing a report. It doesn't need `--remote`:

```bash
./fuzzy --spec api.json --exclude DELETE --list
```

Each line has an endpoint, followed by a tab and its tags if it has any, so `cut -f1` gives lines you can pass back as selectors. Endpoints that can't be fuzzed and the totals go to stderr.

A selector that matches no endpoint of the spec stops the run with exit code `1` before any request is sent. Otherwise, a typo in an `--exclude` would quietly call the endpoints it was meant to protect.

An endpoint gets the same inputs whichever others are left out, so you can replay one from a bigger run with its seed and the same spec and value flags:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --include 'GET /user/{id}' --seed 42
```

The report says which selectors were used and how many endpoints they left out. Endpoints left out don't appear under skipped endpoints, which are the ones that couldn't be fuzzed. Warnings still cover the whole spec, because every schema is compiled to keep the seed's inputs stable.

### Choosing the Results 🐠

A result is what one call to an endpoint produced: its endpoint, status code, message and input, and how long it took.

`--status` decides which results end up in the report. It takes a comma separated list of exact codes, classes, ranges and `err`, which stands for requests that got no response at all (timeouts, refused connections):

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --status 200,404,5xx,400-403,err
```

Without `--status`, every result is reported. Filtered results still count in each endpoint's totals, so you always see how many requests got each status code.

### Choosing the Report Format 🦜

The extension of `-o` picks the format, and without `-o` the report is `fuzzy-report.md`:

| Extension | Format | Made for |
|---|---|---|
| `.md`, `.markdown` | `markdown` | Reading, and rendering on a code host |
| `.html`, `.htm` | `html` | Reading in a browser: one page with nothing to download |
| `.txt` | `txt` | Reading in a terminal |
| `.json` | `json` | Scripts and CI |
| `.xml` | `xml` | Scripts and CI, in tools that expect XML |
| `.csv` | `csv` | Spreadsheets: the results as a table |

```bash
./fuzzy --remote http://localhost:8080 --spec api.json -o report.html
```

`--format` beats the extension, for any path: `--format json -o results.out` writes JSON. With `--format` and no `-o`, the file name follows the format, like `fuzzy-report.json`. An extension that isn't listed, or none at all, gets Markdown. A script that used `-o report.json` or `-o report.txt` without `--format` used to get Markdown in that file, so add `--format markdown` to keep it.

For scripts and CI, JSON is the usual choice:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --status 5xx,err --format json
jq -r '.endpoints[].results.items[] | select(.status >= 500) | "\(.status) \(.endpoint)"' fuzzy-report.json
```

The report is written once, when the run ends. With `--incremental`, it's also saved at 10 evenly spaced checkpoints during the run, so you can open it while Fuzzy works and a killed run keeps its last checkpoint. Each save replaces the file in one step, so you never read a half-written report. Until the final save the report is marked as partial (`interrupted` is `true` in JSON), and CI should only trust a report once Fuzzy has exited. The file stays empty until the first checkpoint, and a run of fewer than 11 requests has fewer checkpoints. A pipe, a device such as `/dev/stdout`, or a file in a read-only directory can't be replaced, so it only gets the final report.

Every format but CSV has the same structure and the same information. XML uses the element names of the JSON keys, inside a `<report>` root:

| Markdown, HTML, text | JSON, XML |
|---|---|
| Warning about a partial run | `interrupted` |
| Overview | `overview` |
| Summary | `summary` |
| A section for each endpoint, with its parameters and results | `endpoints`, each with its `parameters` and `results` |
| Skipped endpoints | `skipped` |
| Warnings | `warnings` |

CSV exports only the results, with one row for each reported result, since the overview and the totals don't fit in a table of results. So an endpoint with nothing reported has no row, and `--status` matters more here. `seed` and `interrupted` repeat on every row, so a row can be reproduced, and a partial run recognized, after it's imported somewhere else:

```csv
operation,endpoint,status,message,input,durationMs,seed,interrupted
GET /user/{id},GET /v1/user/427,500,"{""error"":""db timeout""}","{""path"":{""id"":427}}",12.345,42,false
```

<details>
<summary>Full JSON example</summary>

```json
{
  "interrupted": false,
  "overview": {
    "target": "http://localhost:8080",
    "spec": { "path": "api.json", "title": "Demo", "version": "2" },
    "dictionary": "",
    "randomBounds": {
      "string": { "min": 1, "max": 10 },
      "int": { "min": 0, "max": 1000 },
      "float": { "min": 0, "max": 1000 }
    },
    "charset": "unicode",
    "statusFilter": "5xx,err",
    "selection": { "include": [], "exclude": [], "endpointsLeftOut": 0 },
    "seed": "42",
    "started": "2026-01-02T03:04:05Z",
    "duration": "1.5s",
    "requestsSent": 100,
    "resultsReported": 18
  },
  "summary": [
    {
      "endpoint": "GET /user/{id}",
      "sent": 100,
      "reported": 18,
      "statusCodes": { "404": 82, "500": 17, "ERR": 1 },
      "responseTimeMs": { "count": 99, "min": 1.2, "mean": 8.1, "p50": 6.5, "p95": 21.9, "p99": 39.9, "max": 41.7 }
    }
  ],
  "endpoints": [
    {
      "method": "GET",
      "path": "/user/{id}",
      "fullPath": "/v1/user/{id}",
      "summary": "Fetch a user",
      "operationId": "getUser",
      "parameters": [
        { "name": "id", "in": "path", "type": "integer (int32)", "required": true, "constraints": "min 1; max 500" }
      ],
      "results": {
        "sent": 100,
        "reported": 18,
        "filter": "5xx,err",
        "statusCodes": { "404": 82, "500": 17, "ERR": 1 },
        "responseTimeMs": { "count": 99, "min": 1.2, "mean": 8.1, "p50": 6.5, "p95": 21.9, "p99": 39.9, "max": 41.7 },
        "items": [
          { "endpoint": "GET /v1/user/427", "status": 500, "message": "{\"error\":\"db timeout\"}", "input": "{\"path\":{\"id\":427}}", "durationMs": 12.345 },
          { "endpoint": "GET /v1/user/98", "status": null, "message": "context deadline exceeded", "input": "{\"path\":{\"id\":98}}", "durationMs": 10000.2 }
        ]
      }
    }
  ],
  "skipped": [],
  "warnings": []
}
```

</details>

A few values are written differently, to suit the format:

- **`status`**: A number, or `null` in JSON when the request got no response. Every other format shows that case as `ERR`, and `statusCodes` counts it under the `ERR` key.
- **`seed`**: A string in JSON, because seeds use 64 bits and many JSON tools would round such a large number, and a rounded seed doesn't reproduce the run.
- **`selection`**: Always there in JSON and XML. Markdown, text and HTML only show it when a run used `--include` or `--exclude`. CSV has no overview, so it doesn't show it.
- **Empty values**: In JSON and XML, every key is always there. An empty list is `[]` or an empty element, and text that doesn't apply is empty, like `dictionary` when values are only random.
- **Text from the server**: Markdown, HTML and text neutralize markup and control characters so a response can't break the document or a terminal, and the HTML page also forbids scripts. JSON and XML keep what the server really sent, apart from characters XML can't hold, so treat `message` as untrusted if you display it. CSV keeps it too, but puts a `'` before a cell starting with `=`, `+`, `-`, `@`, a tab or a carriage return, because a spreadsheet would run it as a formula.

In every format, `input` is a JSON object written as a string. It holds the generated values by where they went: `path`, `query`, `header` and `cookie` map names to values, and `body` holds the body. Keys appear only when the request used them, so a request with no values has `{}`. To read it with `jq`, use `.input | fromjson`.

### Response Times 🐎

Each endpoint reports the minimum, mean, `p50`, `p95`, `p99` and maximum of its response times, and each reported result its own, all in milliseconds:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --status 5xx,err --format json
jq -r '.summary[] | select(.responseTimeMs.p95 > 500) | .endpoint' fuzzy-report.json   # the slow endpoints
```

A time goes from sending the request to reading the response, up to 1 MiB of its body. Generating the values isn't included. The statistics cover every response, whatever `--status` keeps, but not requests without a response: those would measure `--timeout`, not the API. A result without a response still shows how long it took to fail. Percentiles are accurate to about 6%, because Fuzzy keeps a small histogram per endpoint instead of every time, so memory doesn't grow with `-n`. The minimum, mean and maximum are exact. With a high `-c` on one machine the times include Fuzzy's own load, so compare them between runs rather than trusting them to the millisecond.

### Keeping a Log 🦫

`--log` writes what happened during the run to a file, in order, with one line for each request, whatever `--status` leaves out of the report:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --status 5xx,err --log run.log
```

```
time=2026-01-02T03:04:05.712+01:00 level=INFO msg="run started" target=http://localhost:8080 spec=api.json seed=42 endpoints=6 requests=100 concurrency=20 timeout=10s
time=2026-01-02T03:04:05.734+01:00 level=INFO msg=request endpoint="GET /v1/user/12" status=404 duration=1.2ms message="Not Found" input="{\"path\":{\"id\":12}}"
time=2026-01-02T03:04:05.735+01:00 level=ERROR msg=request endpoint="GET /v1/user/427" status=500 duration=12.3ms message="{\"error\":\"db timeout\"}" input="{\"path\":{\"id\":427}}"
time=2026-01-02T03:04:06.018+01:00 level=INFO msg="run finished" sent=600 failed=0 interrupted=false duration=305ms report=/home/you/fuzzy/fuzzy-report.md
```

It's plain text in the `key=value` format that `grep` and log tools read, whatever the file's extension. `level=ERROR` marks server errors and requests without a response (`status=ERR`), and `level=WARN` marks skipped endpoints and warnings about the spec. A new run appends to the file, so the log keeps the history. Each line is written straight away, so a killed run keeps its log up to the end. Values are quoted, so a response can't forge a line. Headers aren't logged, so neither are the credentials you pass with `-H`, but inputs are. A log failure, such as a full disk, is reported as a warning at the end and doesn't fail the run. A log takes about 370 bytes per request, and it keeps memory low: `--status 5xx,err --log run.log` holds every request on disk instead of in memory.

### Choosing the Values 🦊

By default values are random. Random strings have 1 to 10 characters, integers go from 0 to 1000, floats from 0 to 1000.0. Strings mix letters of several Unicode scripts with ASCII letters and digits, and `--charset alnum` keeps only the ASCII ones. You can change all of it:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json \
  --string-min 3 --string-max 50 --int-min -100 --int-max 100 --float-min 0 --float-max 1 --charset alnum
```

The spec always wins: an `id` with `minimum: 5000` gets values from 5000 up, even if `--int-max` is lower, because the request has to stay valid.

With a dictionary, Fuzzy reads one value per line and works out the types each line can serve: `42` works as an integer, a float and a string, `true` as a boolean and a string, `admin` only as a string. A value is used only where it fits the spec. Parameters with no fitting value get random valid inputs instead:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --dict values.txt
```

A dictionary above 25 MiB stays on disk: Fuzzy indexes it, and the workers read only the pieces they need. In our tests, a run with a 400 MB file peaked at about 60 MB of memory, where loading it took several GB. A seed gives the same inputs either way, with one difference: a file this big keeps its duplicate lines, because finding them takes memory for every line. They are sent more than once and count towards the default `-n`, so run `sort -u` on the file first if that matters. Don't change the file during a run: Fuzzy can't always notice, and when a value can't be read it sends a random valid one and adds a warning to the report.

The default `-n` follows the size of the dictionary, for each endpoint, and every result that passes `--status` stays in memory until the report is written. So a run that could keep a million results or more starts with a warning and an estimate of the memory. To avoid that, use `--status 5xx,err` or a lower `-n`.

### Where the Requests Go 🐳

`--remote` can be a full URL, an FQDN or a `host:port`. Without a scheme, Fuzzy uses `https` when there is no port or the port is 443, and `http` for any other port. Write the scheme yourself to choose differently, for example `http://api.example.com`.

| `--remote` | Same as |
|---|---|
| `api.example.com` | `https://api.example.com` |
| `api.example.com:443` | `https://api.example.com:443` |
| `localhost:8080` | `http://localhost:8080` |
| `127.0.0.1:8080/v1` | `http://127.0.0.1:8080/v1` |

If `--remote` has no path, the path of the spec's `servers` entry is added to it. If it has one, it's used as given. Either way the host always comes from `--remote`, so Fuzzy never calls the server named in the spec. A spec can't get around that: a path that doesn't start with `/` is refused with an error, and a relative `servers` URL, such as `.example.com`, only ever adds to the path.

| `--remote` | `servers` in the spec | Request |
|---|---|---|
| `http://localhost:8080` | `https://prod.example.com/api/v3` | `http://localhost:8080/api/v3/user/1` |
| `http://localhost:8080/custom` | `https://prod.example.com/api/v3` | `http://localhost:8080/custom/user/1` |

### Reproducing a Run 🦎

Run Fuzzy again with the same seed and it generates the same inputs: the same optional parameters left out or sent, and the same values in them. Every run has a seed. Pass your own with `--seed`, or Fuzzy picks a random one (also for `--seed 0`) and prints it on stdout and in the report, the log and every CSV row.

To reproduce a run, keep the seed, the spec file and the flags that shape values:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --seed 42 --dict values.txt --string-max 20
```

| Inputs stay the same when you change | Inputs can change when you change |
|---|---|
| `-c`, so any concurrency gives the same inputs | `--seed` |
| `--remote`, `-H`, `-k` and `--timeout` | The spec, where even one new endpoint can change every input |
| `--status`, `--format`, `-o`, `--log` and `--incremental` | `--dict`, or the contents of the dictionary |
| `--include` and `--exclude`, for the endpoints that still run | `--string-min`, `--string-max`, `--int-min`, `--int-max`, `--float-min`, `--float-max` and `--charset` |
| `-n`, where a smaller run sends the first inputs of a bigger one | The version of Fuzzy |

**How it works.** Three things make this hold:

1. **Each request has its own random stream.** It's derived from the seed, the endpoint's position among all the endpoints of the spec and the request's number, and never from which worker sends it or when. That's why concurrency doesn't matter, and why leaving endpoints out doesn't change the others.
2. **The seed picks where each list starts.** Enum and dictionary values aren't drawn at random: the seed chooses a starting point in each list, and the request's number moves through it, so every value gets its turn. All schemas are compiled before endpoints are selected, so these starting points don't depend on the selection either.
3. **The spec is always read in the same order.** Paths, schemas and `allOf` merges are processed in a fixed order, so a spec resolves to the same schemas on every run.

**What a seed doesn't reproduce.** It fixes what Fuzzy sends, not what comes back:

- **Answers**: Status codes, messages and response times depend on the server. So does the outcome of a request that needs earlier data: `GET /invoices/{id}` only finds an invoice that an earlier `POST` created, and with `-c` above 1 the requests can arrive in a different order.
- **Multipart boundaries**: They're random on every request, so the raw bytes of a multipart body differ even though its fields don't.

To chase a failure that depends on state, reset the target between runs and use `-c 1`, so the requests arrive in the same order each time.

---

## Examples

### Example 1: Hunting Server Errors 🐻

This example shows what a finding looks like, on an illustrative API with a bug in `GET /user/{id}`.

**Command**:
```bash
./fuzzy --remote http://localhost:8080 --spec api.json --status 5xx,err --seed 42
```

**Output** (the middle line is the status bar, which updates in place while the requests run):
```
[+] fuzzing 6 endpoints of http://localhost:8080, 100 requests each (seed 42)
⠹ [██████████░░░░░░░░░░]  52% 312/600 req · 287 req/s · ETA 1s · 5xx 17 · err 0
[+] 600 requests sent, 0 without a response. Report: /home/you/fuzzy/fuzzy-report.md
```

**Report** (`fuzzy-report.md`), first an overview and a summary:
```markdown
## Summary

| Endpoint | Sent | Reported | Status codes |
|---|---|---|---|
| POST /login | 100 | 0 | 200 ×100 |
| GET /user/{id} | 100 | 17 | 404 ×83, 500 ×17 |
```

Then a section for each endpoint with its spec and results, including the exact input behind each failure:
```markdown
## GET /user/{id}

Fetch a user

- **HTTP verb:** GET
- **Path:** /v1/user/{id}
- **Operation ID:** getUser

### Parameters

| Name | In | Type | Required | Constraints |
|---|---|---|---|---|
| id | path | integer (int32) | yes | min 1; max 500 |
| fields | query | array of string | no | min items 1; unique items |
| X-Request-Id | header | string (uuid) | no |  |

### Results

100 sent, 17 reported (filter: 5xx,err). Status codes: 404 ×83, 500 ×17.

| Endpoint | Status | Message | Input |
|---|---|---|---|
| GET /v1/user/427 | 500 | {"error":"db timeout"} | {"path":{"id":427}} |
| GET /v1/user/98 | 500 | {"error":"db timeout"} | {"header":{"X-Request-Id":"c8ac4929-b19e-4fe3-ba2f-b60236b4d4e0"},"path":{"id":98},"query":{"fields":\["age","name","email"\]}} |
```

Requests that got no response appear with status `ERR` and the failure as their message.

### Example 2: Testing an Authenticated API With a Dictionary 🐮

**Values file** (`values.txt`):
```
1
999999
admin
root
../../../etc/passwd
<script>alert(1)</script>
```

**Command**:
```bash
./fuzzy --remote https://api.example.com/v3 --spec api.json --dict values.txt -H "Authorization: Bearer $TOKEN"
```

Here `1` and `999999` are also tried as integers, and the rest only as strings. A string parameter with `maxLength: 5` would only get `1`, `admin` and `root`, and one with `format: email` would get none of them, so it falls back to random valid emails.

---

## Command Line Options 🐊

Every flag has a short name with two dashes and a long name with one, as `--help` shows: `--rem` and `-remote`. The spellings of earlier versions, such as `--remote` and `-n`, keep working, and the examples in this README use them.

| Flag | Description | Example |
|------|-------------|---------|
| `--rem`, `-remote` | API to fuzz: URL, FQDN or `host:port` (required) | `--remote localhost:8080` |
| `--spc`, `-spec` | OpenAPI 3.0 or Swagger 2.0 spec in JSON format, detected automatically (required) | `--spec api.json` |
| `--sta`, `-status` | Status codes in the report (default all) | `--status 200,404,5xx,400-403,err` |
| `--icl`, `-include` | Only fuzz the endpoints matching this selector, repeatable | `--include 'GET /users/*'` |
| `--exc`, `-exclude` | Don't fuzz the endpoints matching this selector, even if included; repeatable | `--exclude DELETE` |
| `--lst`, `-list` | Print the selected endpoints and exit, without sending requests (`--remote` not needed) | `--list` |
| `--dic`, `-dict` | Values file, one value per line (default random values only) | `--dict values.txt` |
| `--smn`, `-string-min` | Random string minimum length (default 1) | `--string-min 3` |
| `--smx`, `-string-max` | Random string maximum length (default 10) | `--string-max 50` |
| `--imn`, `-int-min` | Random integer minimum (default 0) | `--int-min -100` |
| `--imx`, `-int-max` | Random integer maximum (default 1000) | `--int-max 100` |
| `--fmn`, `-float-min` | Random float minimum (default 0) | `--float-min 0.5` |
| `--fmx`, `-float-max` | Random float maximum (default 1000) | `--float-max 1.5` |
| `--chr`, `-charset` | Random string characters, `unicode` or `alnum` (default `unicode`) | `--charset alnum` |
| `--n`, `-requests` | Requests per endpoint (default 100, or the dictionary size if larger) | `-n 500` |
| `--c`, `-concurrency` | Concurrent workers (default 20) | `-c 50` |
| `--tmo`, `-timeout` | Timeout of each request (default 10s) | `--timeout 2s` |
| `--o`, `-output` | Report path; its extension picks the format (default `fuzzy-report` with the format's extension) | `-o report.html` |
| `--fmt`, `-format` | Report format: `markdown`, `json`, `txt`, `csv`, `xml` or `html` (default from the `-o` extension, else `markdown`) | `--format json` |
| `--inc`, `-incremental` | Also save the report 10 times during the run (default only at the end) | `--incremental` |
| `--log`, `-log` | Append a line for every request to this file (default no log) | `--log run.log` |
| `--H`, `-header` | Header for every request, repeatable; overrides generated ones | `-H "Authorization: Bearer abc"` |
| `--k`, `-insecure` | Skip TLS certificate verification (useful for self-signed certs) | `-k` |
| `--sd`, `-seed` | Seed to reproduce a run's inputs (default random, also for `0`; printed in the report) | `--seed 42` |

### Help, Version and Shell Completion 🦭

```bash
./fuzzy --help
./fuzzy --version
source <(./fuzzy completion bash)   # also zsh, fish and powershell, see ./fuzzy completion --help
```

Completion knows the flag names, the `--charset` and `--format` values and that `--spec` wants a `.json` file.

Older spellings still work, so existing scripts don't need to change: `-remote` is the same as `--remote`, and `--n` the same as `-n`. New short bundles such as `-kn 5` are read the POSIX way, so a mistyped `-nope` means `-n ope` and fails as an invalid number.

### Important Notes 🦖

- **Fuzzy Means It**: Every operation in the spec gets called unless you [leave it out](#choosing-the-endpoints-), `DELETE` and `PUT` included, with values that are valid on purpose. Against a real system this really changes or deletes data, so point it at a disposable environment.
- **Authentication**: Security schemes in the spec are not handled automatically. Pass your credentials with `-H`.
- **Redirects**: They are reported as results and not followed, so a `302` shows up as a `302`, and a redirect can never take Fuzzy away from the target.
- **Response Bodies**: The message of a result is the start of the response body (200 characters at most), or the standard status text if the body is empty. For XML and HTML, as said by `Content-Type` or by the body when that's missing, it's the text instead of the markup: `name: text` pairs for XML, attributes included, and the words of the page for HTML, skipping scripts and styles. Fuzzy looks at the first 16 KiB of these, and a body it can't get any text from is shown as it came. Entities in a document type are never expanded.
- **XML Bodies**: These follow the `xml` keyword of the schema: `name`, `namespace`, `prefix`, `attribute` and `wrapped`. Without a `name`, the root is named after the schema's component, or `root` for an inline schema, and an array repeats its items unless it's `wrapped`. Properties keep the spec's order, because XML services often enforce one. A name that isn't valid in XML, for example from `additionalProperties`, has its invalid characters replaced by `_`. An operation offering another media type uses that one, so XML is only the choice when nothing else is available. The report's `input` shows the values as JSON, not the XML that was sent.
- **SOAP**: A body gets an envelope when its media type is `application/soap+xml` (SOAP 1.2), or `text/xml` on an operation with a `SOAPAction` header parameter (SOAP 1.1). A schema whose root is already named `Envelope` is sent as it is. If your spec doesn't declare `SOAPAction`, model the envelope in the schema and pass the header with `-H`.
- **Dictionary Coverage**: Values take turns, so with enough requests each one gets used. On an optional parameter, which is only sent about half the time, some can be missed: raise `-n` if you need them all.
- **Enums**: Values come only from the list, ahead of the dictionary and random values. With `nullable: true`, `null` is sent only if the list includes it.
- **Schemas Without a Type**: Fuzzy infers it from the other keywords: `properties` or `required` mean `object`, `items` or `minItems` mean `array`, `format: int64` means `integer`, `minimum` means `number`. With no hint at all, it uses `string`.
- **Null Values**: `null` is only sent in JSON and multipart bodies, where a multipart `null` is a part with `Content-Type: application/json`. Parameters, form bodies, text bodies, binary bodies and XML bodies can't represent it, so they always get a real value.
- **Spec Format**: OpenAPI 3.0.x and Swagger 2.0 in JSON. There is no flag for it: Fuzzy reads the document's `openapi` or `swagger` field, and writes what it found to the `--log` file. OpenAPI 3.1, other versions and YAML are refused with an error, and so are `$ref`s that point outside the document.
- **Swagger 2.0**: A spec gets the same requests whichever version it's written in, apart from inherited discriminators, which only Swagger 2.0 specs get. `basePath` works like a server's path, and `host` and `schemes` are ignored. A `body` parameter is sent in the media types of `consumes`, or as JSON if there are none. `formData` parameters become a form body, or a multipart one if there is a `file`. Arrays follow `collectionFormat`, which defaults to one comma-separated value. A reference to a schema with a `discriminator` gets one of the definitions extending it, with the property set to that definition's name. Only the most derived definitions are used, because a base is often abstract. If the property has an `enum` that doesn't list those names, its values are used instead.
- **Progress and Exit Codes**: In a terminal, a status bar on stderr updates in place during the run and disappears when it ends. It drops details such as the speed when the terminal is narrow. Above it, the endpoints change color: red when not tested yet, yellow while sending requests, white once all of them are sent. The list stays on screen after the run; if it doesn't fit, the endpoints still in progress come first and the last line counts the rest. With `NO_COLOR` set or on Windows there is only the bar. When stderr is redirected, or with `TERM=dumb`, Fuzzy prints a plain `[+] 1200/6000 requests` line every 5 seconds instead. The `5xx` and `err` counters cover every request, whatever `--status` keeps for the report. Fuzzy exits with `0` when the run completes, whatever the API answered, `1` if it can't run (for example, an unreadable spec) and `2` for invalid arguments.

### Known Limitations 🐙

- **Other Media Types**: An operation whose required body is only available in a media type Fuzzy can't build, such as `application/protobuf`, is skipped and listed in the report. If the body is optional, the operation is called without it.
- **XML Schemas and WSDL**: Fuzzy only reads the OpenAPI spec, so a rule that only an XSD or a WSDL states isn't known to it, and an `xsi:nil` value is never sent.
- **Overlapping `oneOf`**: When branches overlap in the spec itself (`{}` can match two of them), a value can match more than one. Branches that differ by their `required` fields are told apart.
- **Unsupported Keywords and Patterns**: `not` and, in OpenAPI 3.0, discriminators inherited through `allOf` are ignored. Patterns must use Go regular expression syntax, so one with a lookahead is ignored and listed under the report's warnings.
- **Swagger 2.0 Arrays and Nulls**: An array nested in another array ignores the inner `collectionFormat` and is written as JSON, and so is an array field in a multipart body. `x-nullable` is ignored, so these specs never get a `null` from it.
- **Formats and Lengths**: Values generated for a `format` such as `email` or `uuid` follow the format, not `maxLength` or `--string-max`.

---

## Use Cases 🦀

- **Crash Hunting**: Find valid requests that still make the API fail (`--status 5xx,err`).
- **Spec Drift**: A request built from the spec shouldn't be rejected as invalid, so a `400` or `422` points at a spec and an implementation that disagree (`--status 400,422`).
- **Security Testing**: Send injection, XSS and path traversal payloads from a dictionary, wherever the spec allows them, and check how the API handles them.
- **CI Smoke Tests**: Fuzz a fresh deployment on every pipeline run and gate on the JSON report.
- **Rate Limiting**: Raise `-c` and see which endpoints answer `429` (`--status 429`).

---

## Safety and Best Practices 🦋

Fuzzy really changes or deletes data, so only test systems you own or have explicit permission to test, and never production. This tool is for educational and authorized testing purposes only, and the authors are not responsible for any misuse of it.

1. **Isolate the Target**: Run it against a disposable environment, such as a container or a VM, with limits on its CPU and memory. Don't fuzz sensitive production data.
2. **Start Small**: Begin with a low `-n`, a small set of test values and a few endpoints (check them with `--list`), then expand gradually. Lower `-c` to avoid overwhelming the target.
3. **Focus the Report**: Use `--status 5xx,err` to see what went wrong.
4. **Keep the Seed**: It's in every report, and it [reproduces the inputs of the run](#reproducing-a-run-).
5. **Use Real Inputs**: Real-world values in your dictionary improve the results. Anonymize sensitive data first.

---

## File Structure 🦢

```
fuzzy/
├── README.md            # This documentation
├── CLAUDE.md            # Project guidance for Claude Code
├── go.mod, go.sum       # Go module definition and checksums
├── assets/              # Project logo and Fuzzy's character description
├── cmd/fuzzy/           # The Cobra command line and the run pipeline
├── examples/crud/       # Local CRUD invoice API to try Fuzzy on
└── internal/
    ├── config/          # Flags and their validation
    ├── openapi/         # Loads the spec, converts Swagger 2.0, resolves $ref and allOf
    ├── generator/       # Turns schemas into valid values, dictionary included
    ├── fuzzer/          # Encodes requests, runs the worker pool, filters statuses
    ├── progress/        # Shows a run's progress: status bar or plain lines
    ├── runlog/          # Writes the log file of a run
    └── report/          # Writes the report in each format
```

---

## Contributing 🐜

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.

1. Fork the repo
2. Create a branch `feature/xxx`
3. Add tests for new features, and run `go test -race ./...`
4. Open a PR with description and use case

Guidelines:

* Maintain CLI backward compatibility when possible
* Add unit tests for new schema keywords and formats
* Document new flags in the Command Line Options table of this README

---

## Roadmap 🦉

- 🔴 high impact features or improvements or bug fixes should be prioritized.
- 🟠 medium-high general improvements to code quality, usability, performance and testing.
- 🟡 medium features, improvement or refactoring.
- 🟢 low urgency features, minor tweaks, documentation updates.
- 🔵 funny little things.

### To-Do Functionality List 🦞

- 🔴 Add stateful testing: chain requests and reuse values from responses, such as created IDs, so calls like `GET /users/{id}` reach the code behind the "not found" check
- 🔴 Add support for certificates
- 🟡 Add support for more authentication methods (e.g., OAuth, API keys)

### To-Do Improvement List 🐍

- 🟠 Generalize and modularize code for easier extension
- 🟠 Implement request rate limiting
- 🔵 Calculate and optimize code complexity

---

## License 🐮

The repository has no license file yet.
