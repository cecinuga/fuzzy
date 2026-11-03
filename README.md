# 🦄 Fuzzy - HTTP API Fuzzer 🦄

> Give Fuzzy an OpenAPI 3.0 or Swagger 2.0 spec and it sends **schema-valid** requests to every endpoint, so they get past your validators and reach the code behind them. No templates, no config, one binary.

![fuzzy with little bit of anxiety because the job is not done yet](assets/fuzzy.png)

Fuzzy is shy, a little anxious when he is among many people, but he works hard because he loves his job.

## Why Fuzzy 🦑

Random input usually dies at the front door with a `400` or `422`. Fuzzy's requests pass validation, so what it finds comes from your business logic. A real result from the bundled example API:

```
POST /invoices → 409 {"detail":"due_date 2000-10-24 is before issue_date 2031-06-02"}
{"body":{"currency":"USD","customer":"õひल","due_date":"2000-10-24","issue_date":"2031-06-02","lines":[{"description":"乷ग𠁞Â걎","quantity":317,"unit_price":354}],"tax_rate":20.028}}
```

- **Valid requests**: types, formats, ranges, lengths, patterns, enums, required fields, `$ref`, `allOf`, `oneOf`, `anyOf`, discriminators; in path, query, header and cookie parameters and in JSON, form, multipart, text, binary and XML bodies (SOAP envelopes included).
- **Dictionary values only where the spec allows them**, random valid values everywhere else.
- **Reproducible**: the same seed gives the same inputs at any concurrency.
- **Reports** in Markdown, HTML, text, JSON, XML or CSV, with per-endpoint response-time percentiles.
- **Stays on target**: the host always comes from `--remote`; redirects are reported, never followed.
- **Ctrl+C still writes the report**, marked as partial.
- **Light**: a bounded worker pool, Cobra as the only dependency.

| Measured (local `examples/crud`, `-c 20 --status 5xx,err`) | |
|---|---|
| Speed | ~5,700 req/s (120,000 requests in 20.9 s) |
| Peak memory | 19 MB, at both 6,000 and 120,000 requests |
| Requests rejected as invalid (`422`) | 0 of 120,000 |
| Same seed at `-c 1` and `-c 50` | 600 of 600 inputs identical |

**Is it for you?** You need an OpenAPI 3.0.x or Swagger 2.0 spec **in JSON** and a disposable environment. Fuzzy doesn't send malformed requests on purpose, it really calls `DELETE` and `PUT`, and authentication is up to you (`-H`).

---

## Quick Start 🐇

Requires Go 1.25.1+. The repository ships a safe in-memory invoice API with no planted bugs.

```bash
git clone https://github.com/cecinuga/fuzzy.git && cd fuzzy
go run ./examples/crud                      # terminal 1, listens on :8000
go build -o fuzzy ./cmd/fuzzy               # terminal 2
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json --seed 42
```

```
[+] fuzzing 6 endpoints of http://localhost:8000, 100 requests each (seed 42)
[+] 600 requests sent, 0 without a response. Report: /home/you/fuzzy/fuzzy-report.md
```

**In CI**, Fuzzy exits `0` whatever the API answered, so gate on the JSON report:

```bash
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json --status 5xx,err --format json
jq -e '.overview.resultsReported == 0' fuzzy-report.json
```

---

## Usage 🦥

```bash
./fuzzy --remote <URL|FQDN|HOST:PORT> --spec <SPEC_JSON> [OPTIONS]
```

### Choosing the Endpoints 🦔

`--include` keeps matching endpoints, `--exclude` drops them; both repeat, and `--exclude` always wins.

| Selector | Matches |
|---|---|
| `GET` or `GET,POST` | Those methods |
| `/users/*` | Exactly one segment: `/users/{id}` |
| `/users/**` | Zero or more segments: `/users`, `/users/{id}/orders` |
| `tag:billing` | Operations with that tag |
| `GET /user/{id}`, `POST tag:billing` | Method and path or tag, both must match |

Paths are written as in the spec, without the `servers` base path. `--list` prints what would run and exits (no `--remote` needed). **A selector that matches nothing stops the run with exit `1`**, so a typo in `--exclude` can't hit what it was meant to protect. Selection never changes an endpoint's inputs, so `--include 'GET /user/{id}' --seed 42` replays it from a bigger run.

### Choosing the Results 🐠

A result is `(endpoint, status code, message, input)` plus its duration. `--status` filters the report only, and filtered results still count in each endpoint's totals:

```bash
--status 200,404,5xx,400-403,err      # err = no response (timeout, refused connection)
```

Useful filters: `5xx,err` for crashes, `400,422` for spec drift, `429` for rate limiting.

### Report Formats 🦜

The extension of `-o` picks the format (`.md`, `.html`, `.txt`, `.json`, `.xml`, `.csv`); `--format` beats it; anything else gets Markdown. The default file is `fuzzy-report.<ext>`.

- Every format but CSV holds the same information: overview, summary, one section per endpoint (spec + results), skipped endpoints, warnings. CSV exports only the results, with `seed` and `interrupted` on every row.
- `input` is always a JSON object as a string, keyed by `path`, `query`, `header`, `cookie`, `body` (use `jq '.input | fromjson'`).
- In JSON, a missing response is `"status": null` (`ERR` elsewhere) and `seed` is a string, since 64-bit numbers get rounded by many tools.
- Server text is neutralized in Markdown, HTML, text and CSV (formula injection); JSON and XML keep it raw, so treat `message` as untrusted.
- `--incremental` also saves the report atomically at 10 checkpoints, so a killed run keeps its last one.

### Response Times 🐎

Each endpoint reports min, mean, p50, p95, p99 and max in ms, over every response regardless of `--status`, excluding requests without one. Percentiles come from a fixed histogram (~6% accuracy) so memory doesn't grow with `-n`.

### Keeping a Log 🦫

`--log run.log` appends one logfmt line per request, whatever `--status` keeps, written immediately so a killed run keeps it. Headers (and so credentials) are never logged. `--status 5xx,err --log run.log` keeps every request on disk instead of in memory.

### Choosing the Values 🦊

Random defaults: strings of 1–10 characters (`--charset unicode` or `alnum`), integers 0–1000, floats 0–1000.0. **The spec always wins**: values come from the intersection of the flags and the schema, or from the schema alone when they don't overlap.

With `--dict values.txt` each line serves every type it can parse as (`42` → integer, float, string) and is used only where it fits the schema; the rest gets random valid values. Dictionaries above 25 MiB stay on disk and are read by position (a 400 MB file peaked at ~60 MB of memory), keeping their duplicate lines. The default `-n` grows to the dictionary size, so pair big dictionaries with `--status 5xx,err`.

### Where the Requests Go 🐳

Without a scheme, `--remote` uses `https` for no port or `443`, `http` otherwise. **The host always comes from `--remote`**: if it has no path, the path of the spec's `servers` entry is appended; if it has one, it is used as given.

| `--remote` | `servers` in the spec | Request |
|---|---|---|
| `http://localhost:8080` | `https://prod.example.com/api/v3` | `http://localhost:8080/api/v3/user/1` |
| `http://localhost:8080/custom` | `https://prod.example.com/api/v3` | `http://localhost:8080/custom/user/1` |

### Reproducing a Run 🦎

Every run has a seed (`--seed`, or a random one printed on stdout, in the report, the log and every CSV row). To reproduce the inputs, keep the seed, the spec and the value flags.

| Inputs stay the same when you change | Inputs can change when you change |
|---|---|
| `-c` | `--seed` |
| `--remote`, `-H`, `-k`, `--timeout` | The spec (one new endpoint can change every input) |
| `--status`, `--format`, `-o`, `--log`, `--incremental` | `--dict` or its contents |
| `--include`, `--exclude` (for endpoints that still run) | `--string-*`, `--int-*`, `--float-*`, `--charset` |
| `-n` (a smaller run sends the first inputs of a bigger one) | The version of Fuzzy |

It holds because each request's random stream derives only from the seed, the endpoint's position among all compiled endpoints and the request number; enum and dictionary lists start at a seed-chosen offset; and the spec is resolved in a fixed order. A seed doesn't reproduce **answers** (they depend on server state and arrival order) or multipart boundaries. For stateful failures, reset the target and use `-c 1`.

---

## Command Line Options 🐊

Long names use one dash, short names two (`-remote` / `--rem`); older spellings such as `--remote` and `-n` keep working.

| Flag | Description | Example |
|------|-------------|---------|
| `--rem`, `-remote` | API to fuzz: URL, FQDN or `host:port` (required) | `--remote localhost:8080` |
| `--spc`, `-spec` | OpenAPI 3.0 or Swagger 2.0 spec in JSON, detected automatically (required) | `--spec api.json` |
| `--sta`, `-status` | Status codes in the report (default all) | `--status 200,404,5xx,400-403,err` |
| `--icl`, `-include` | Only fuzz endpoints matching this selector, repeatable | `--include 'GET /users/*'` |
| `--exc`, `-exclude` | Don't fuzz endpoints matching this selector, repeatable | `--exclude DELETE` |
| `--lst`, `-list` | Print the selected endpoints and exit | `--list` |
| `--dic`, `-dict` | Values file, one value per line | `--dict values.txt` |
| `--smn`, `-string-min` | Random string minimum length (default 1) | `--string-min 3` |
| `--smx`, `-string-max` | Random string maximum length (default 10) | `--string-max 50` |
| `--imn`, `-int-min` | Random integer minimum (default 0) | `--int-min -100` |
| `--imx`, `-int-max` | Random integer maximum (default 1000) | `--int-max 100` |
| `--fmn`, `-float-min` | Random float minimum (default 0) | `--float-min 0.5` |
| `--fmx`, `-float-max` | Random float maximum (default 1000) | `--float-max 1.5` |
| `--chr`, `-charset` | `unicode` or `alnum` (default `unicode`) | `--charset alnum` |
| `--n`, `-requests` | Requests per endpoint (default 100, or the dictionary size if larger) | `-n 500` |
| `--c`, `-concurrency` | Concurrent workers (default 20) | `-c 50` |
| `--tmo`, `-timeout` | Timeout of each request (default 10s) | `--timeout 2s` |
| `--o`, `-output` | Report path; its extension picks the format | `-o report.html` |
| `--fmt`, `-format` | `markdown`, `json`, `txt`, `csv`, `xml` or `html` | `--format json` |
| `--inc`, `-incremental` | Also save the report 10 times during the run | `--incremental` |
| `--log`, `-log` | Append a line for every request to this file | `--log run.log` |
| `--H`, `-header` | Header for every request, repeatable; overrides generated ones | `-H "Authorization: Bearer abc"` |
| `--k`, `-insecure` | Skip TLS certificate verification | `-k` |
| `--sd`, `-seed` | Seed to reproduce a run's inputs (default random, also for `0`) | `--seed 42` |

Also: `--help`, `--version`, and `completion bash|zsh|fish|powershell`.

**Exit codes**: `0` the run completed (whatever the API answered), `1` it couldn't run, `2` invalid arguments.

### Behavior Notes 🦖

- **Specs**: OpenAPI 3.0.x and Swagger 2.0, JSON only, detected from the document's version field. OpenAPI 3.1, YAML and external `$ref`s are refused.
- **Swagger 2.0** gets the same requests as the equivalent OpenAPI 3.0 spec, plus discriminators inherited through `allOf`. `basePath` acts as the server path; `host` and `schemes` are ignored; arrays follow `collectionFormat`.
- **Messages** are the first 200 characters of the body; XML and HTML are reduced to their text (a SOAP fault reads `faultcode: soap:Server; faultstring: db down`).
- **XML bodies** follow the schema's `xml` keyword and the spec's property order, and are chosen only when no other media type is offered. SOAP envelopes are added for `application/soap+xml`, or `text/xml` with a `SOAPAction` header parameter.
- **Enums** take precedence over dictionary and random values. **Untyped schemas** get a type inferred from their keywords, else `string`. **`null`** is only sent in JSON and multipart bodies.
- **Progress** is a colored status bar on stderr in a terminal, plain `[+]` lines every 5 s otherwise.

### Known Limitations 🐙

- Operations whose required body uses an unsupported media type (e.g. `application/protobuf`) are skipped and listed.
- Rules stated only in an XSD or WSDL are unknown; `xsi:nil` is never sent.
- Overlapping `oneOf` branches can produce a value matching more than one.
- `not` is ignored, and so are OpenAPI 3.0 discriminators inherited through `allOf`. Patterns must be Go regex (lookaheads are ignored with a warning).
- Swagger 2.0: nested arrays and multipart array fields are written as JSON; `x-nullable` is ignored.
- Values for a `format` such as `email` or `uuid` ignore `maxLength` and `--string-max`.

---

## Safety 🦋

Fuzzy calls every operation with valid values, `DELETE` and `PUT` included, so it **really changes or deletes data**. Only test systems you own or are authorized to test, in a disposable environment, never production. Start small (`--list`, low `-n` and `-c`), and keep the seed of every run.

## Contributing 🐜

Use a `feature/xxx` branch, add tests (`go test -race ./...`), keep the CLI backward compatible, document new flags in the options table, and open a PR with a description and a use case.

## Roadmap 🦉

🔴 high impact · 🟠 medium-high · 🟡 medium · 🟢 low · 🔵 funny little things

- 🔴 Stateful testing: chain requests and reuse values from responses, such as created IDs
- 🔴 Support for certificates
- 🟡 More authentication methods (OAuth, API keys)
- 🟠 Generalize and modularize code for easier extension
- 🟠 Request rate limiting
- 🔵 Calculate and optimize code complexity

## License 🐮

The repository has no license file yet.
