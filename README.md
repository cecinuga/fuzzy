# 🦄 Fuzzy - HTTP API Fuzzer 🦄

> **Fuzzy** — A simple shiny HTTP API fuzzing tool written in **Go** (Golang).
> Give it an OpenAPI 3.0 spec and it sends semantically valid requests, with random or dictionary values, to every endpoint.

![fuzzy with little bit of anxiety because the job is not done yet](assets/fuzzy.png)

---

Fuzzy is shy, a little anxious when he is among many people, but he works hard because he loves his job.
He is always eager to learn new things and improve his skills, even if it means stepping out of his comfort zone.

## Table of Contents 🦆

* [Description](#description)
* [Key Features](#key-features)
* [Installation](#installation)
* [Usage](#usage)
* [Examples](#examples)
* [File Structure](#file-structure)
* [Command Line Options](#command-line-options)
* [Contributing](#contributing)

---

## Description 🦑

Fuzzy is a newbie-friendly, concurrent command-line HTTP API fuzzer. You give it the address of an API and its OpenAPI 3.0 spec, and it works out on its own which endpoints exist and what each one accepts.

For example, from the spec it learns:

```
GET /user/{id}
  id: integer
```

Then it calls every endpoint with requests that respect the spec, using values generated at random or taken from your dictionary, and writes what happened to a Markdown or JSON report.

---

## Key Features 🐞

* **OpenAPI 3.0 Driven**: Reads a JSON spec and fuzzes all of its endpoints, no request templates to write
* **Semantically Valid Requests**: Respects types, formats, ranges, lengths, patterns, enums, required fields, `$ref`, `allOf`, `oneOf`, `anyOf` and discriminators
* **Your Choice of Endpoints**: Include or exclude them by method, path pattern or tag, and preview the result with `--list`
* **Everywhere a Value Can Go**: Path, query, header and cookie parameters, plus JSON, form, multipart, text and binary bodies
* **Flexible Input**: Random values within bounds you choose, or a text file with values to test (one value per line)
* **Dictionary With a Safety Net**: A dictionary value is only used where the spec allows it; anything it doesn't cover gets a random valid value
* **Markdown or JSON Report**: Each endpoint with its spec and its results, filtered by the status codes you care about
* **Fast and Concurrent**: A bounded worker pool on a shared connection pool, with memory that stays low on big specs
* **Reproducible**: The same seed produces the same inputs, at any concurrency
* **Live Status Bar**: A single line that updates in place with the requests sent so far, the speed, the time left and the failures
* **Endpoint Colors**: Every endpoint is listed live on the terminal: red when not tested yet, yellow while sending requests, white once all of them are sent
* **Interrupt Friendly**: Ctrl+C stops the run and still writes the report, marked as partial
* **Lightweight**: A single binary, whose only direct dependency is [Cobra](https://github.com/spf13/cobra) and its flag library

---

## Requirements 🐿️

* Go 1.25.1+
* OS: Linux/macOS/Windows (Linux recommended for heavy testing)
* (Optional) Docker for target isolation

---

## Installation 🐝

1. **Prerequisites**: Go 1.25.1 or higher

2. **Clone and build**:
```bash
git clone https://github.com/cecinuga/fuzzy.git
cd fuzzy
go build -o fuzzy ./cmd/fuzzy
```

3. **Run the tests** (optional):
```bash
go test -race ./...
```

---

## Usage 🦥

Fuzzy needs two things: where the API lives and its OpenAPI 3.0 spec. Everything else is optional.

### Basic Syntax 🦩

```bash
./fuzzy --remote <URL|FQDN|HOST:PORT> --spec <OPENAPI_JSON> [OPTIONS]
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

A result is what one call to an endpoint produced: its endpoint, status code, message and input.

`--status` decides which results end up in the report. It takes a comma separated list of exact codes, classes, ranges and `err`, which stands for requests that got no response at all (timeouts, refused connections):

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --status 200,404,5xx,400-403,err
```

Without `--status`, every result is reported. Filtered results still count in each endpoint's totals, so you always see how many requests got each status code.

### Choosing the Report Format 🦜

The report is Markdown by default, which is made to be read. `--format json` writes the same report as JSON, for scripts and CI:

```bash
./fuzzy --remote http://localhost:8080 --spec api.json --status 5xx,err --format json
jq -r '.endpoints[].results.items[] | select(.status >= 500) | "\(.status) \(.endpoint)"' fuzzy-report.json
```

The default file name follows the format: `fuzzy-report.md` or `fuzzy-report.json`. `-o` writes to any path you choose, and its extension doesn't change the format, which only `--format` decides.

Both formats have the same structure and the same information:

| Markdown | JSON |
|---|---|
| Warning about a partial run | `interrupted` |
| Overview table | `overview` |
| Summary | `summary` |
| A section for each endpoint, with its parameters and results | `endpoints`, each with its `parameters` and `results` |
| Skipped endpoints | `skipped` |
| Warnings | `warnings` |

```json
{
  "interrupted": false,
  "overview": {
    "target": "http://localhost:8080",
    "spec": { "path": "api.json", "title": "Demo", "version": "2" },
    "dictionary": "",
    "randomBounds": ["string: length 1..10 (unicode)", "int: 0..1000", "float: 0..1000"],
    "statusFilter": "5xx,err",
    "selection": { "include": [], "exclude": [], "endpointsLeftOut": 0 },
    "seed": "42",
    "started": "2026-01-02T03:04:05Z",
    "duration": "1.5s",
    "requestsSent": 100,
    "resultsReported": 18
  },
  "summary": [
    { "endpoint": "GET /user/{id}", "sent": 100, "reported": 18, "statusCodes": { "404": 82, "500": 17, "ERR": 1 } }
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
        "items": [
          { "endpoint": "GET /v1/user/427", "status": 500, "message": "{\"error\":\"db timeout\"}", "input": "id=427" },
          { "endpoint": "GET /v1/user/98", "status": null, "message": "context deadline exceeded", "input": "id=98" }
        ]
      }
    }
  ],
  "skipped": [],
  "warnings": []
}
```

A few values are written differently, to suit the format:

- **`status`**: A number, or `null` when the request got no response. Markdown shows that case as `ERR`, and `statusCodes` counts it under the `ERR` key.
- **`seed`**: A string, because seeds use 64 bits and many JSON tools would round such a large number, and a rounded seed doesn't reproduce the run.
- **`selection`**: Always there in JSON. Markdown only adds its rows to the overview when a run used `--include` or `--exclude`.
- **Empty values**: Every key is always there. An empty list is `[]`, and text that doesn't apply is `""`, like `dictionary` when values are only random.
- **Text from the server**: Markdown neutralizes markup and control characters so a response can't break the document. JSON keeps what the server really sent, so treat `message` as untrusted if you display it.

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


---

## Examples

### Example 1: Hunting Server Errors 🐻

**Command** (`testdata/api.json` is a sample spec you can try):
```bash
./fuzzy --remote http://localhost:8080 --spec testdata/api.json --status 5xx,err --seed 42
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

Then a section for each endpoint with its spec and results:
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
| GET /v1/user/427 | 500 | {"error":"db timeout"} | id=427 |
| GET /v1/user/98 | 500 | {"error":"db timeout"} | id=98, fields=\["age","name","email"\], X-Request-Id=c8ac4929-b19e-4fe3-ba2f-b60236b4d4e0 |
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

### Example 3: Trying It on a Local API 🐸

`examples/crud` is a small in-memory CRUD invoice API, safe to fuzz, written in Go. It starts from the sample invoices in `db.json`, which it never changes, and has simple business rules that answer `409`. At startup it writes its spec to `artifacts/openapi.json`.

**Start the API** (from the repository root):
```bash
go run ./examples/crud
```

**Command** (from the repository root, in another terminal):
```bash
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json
```

### Where the Requests Go 🐳

`--remote` can be a full URL, an FQDN or a `host:port`. Without a scheme, Fuzzy uses `https` when there is no port or the port is 443, and `http` for any other port. Write the scheme yourself to choose differently, for example `http://api.example.com`.

| `--remote` | Same as |
|---|---|
| `api.example.com` | `https://api.example.com` |
| `api.example.com:443` | `https://api.example.com:443` |
| `localhost:8080` | `http://localhost:8080` |
| `127.0.0.1:8080/v1` | `http://127.0.0.1:8080/v1` |

If `--remote` has no path, the path of the spec's `servers` entry is added to it. If it has one, it's used as given. Either way the host always comes from `--remote`, so Fuzzy never calls the server named in the spec.

| `--remote` | `servers` in the spec | Request |
|---|---|---|
| `http://localhost:8080` | `https://prod.example.com/api/v3` | `http://localhost:8080/api/v3/user/1` |
| `http://localhost:8080/custom` | `https://prod.example.com/api/v3` | `http://localhost:8080/custom/user/1` |

---

## File Structure 🦢

```
fuzzy/
├── README.md            # This documentation
├── CLAUDE.md            # Project guidance for Claude Code
├── go.mod, go.sum       # Go module definition and checksums
├── assets/              # Project assets
│   ├── fuzzy.png        # Project logo/mascot image
│   └── fuzzyPersonality.txt # Fuzzy's character description
├── cmd/fuzzy/           # The Cobra command line and the run pipeline
├── examples/crud/       # Local CRUD invoice API to try Fuzzy on
├── internal/
│   ├── config/          # Flags and their validation
│   ├── openapi/         # Loads the spec, resolves $ref and allOf
│   ├── generator/       # Turns schemas into valid values, dictionary included
│   ├── fuzzer/          # Encodes requests, runs the worker pool, filters statuses
│   ├── progress/        # Shows a run's progress: status bar or plain lines
│   └── report/          # Writes the Markdown or JSON report
└── testdata/
    ├── api.json         # Sample OpenAPI spec
    └── name.txt         # Sample dictionary
```

---

## Command Line Options 🐊

| Flag | Description | Example |
|------|-------------|---------|
| `--remote` | API to fuzz: URL, FQDN or `host:port` (required) | `--remote localhost:8080` |
| `--spec` | OpenAPI 3.0 spec in JSON format (required) | `--spec api.json` |
| `--status` | Status codes in the report (default all) | `--status 200,404,5xx,400-403,err` |
| `--include` | Only fuzz the endpoints matching this selector, repeatable | `--include 'GET /users/*'` |
| `--exclude` | Don't fuzz the endpoints matching this selector, even if included; repeatable | `--exclude DELETE` |
| `--list` | Print the selected endpoints and exit, without sending requests (`--remote` not needed) | `--list` |
| `--dict` | Values file, one value per line (default random values only) | `--dict values.txt` |
| `--string-min` | Random string minimum length (default 1) | `--string-min 3` |
| `--string-max` | Random string maximum length (default 10) | `--string-max 50` |
| `--int-min` | Random integer minimum (default 0) | `--int-min -100` |
| `--int-max` | Random integer maximum (default 1000) | `--int-max 100` |
| `--float-min` | Random float minimum (default 0) | `--float-min 0.5` |
| `--float-max` | Random float maximum (default 1000) | `--float-max 1.5` |
| `--charset` | Random string characters, `unicode` or `alnum` (default `unicode`) | `--charset alnum` |
| `-n`, `--requests` | Requests per endpoint (default 100, or the dictionary size if larger) | `-n 500` |
| `-c`, `--concurrency` | Concurrent workers (default 20) | `-c 50` |
| `--timeout` | Timeout of each request (default 10s) | `--timeout 2s` |
| `-o`, `--output` | Report path (default `fuzzy-report.md`, or `fuzzy-report.json` with `--format json`) | `-o report.md` |
| `--format` | Report format, `markdown` or `json` (default `markdown`) | `--format json` |
| `-H`, `--header` | Header for every request, repeatable; overrides generated ones | `-H "Authorization: Bearer abc"` |
| `-k`, `--insecure` | Skip TLS certificate verification (useful for self-signed certs) | `-k` |
| `--seed` | Seed to reproduce a run's inputs (default random, also for `0`; printed in the report) | `--seed 42` |

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
- **Response Bodies**: The message of a result is the start of the response body (200 characters at most), or the standard status text if the body is empty.
- **Dictionary Coverage**: Values take turns, so with enough requests each one gets used. On an optional parameter, which is only sent about half the time, some can be missed: raise `-n` if you need them all.
- **Enums**: Values come only from the list, ahead of the dictionary and random values. With `nullable: true`, `null` is sent only if the list includes it.
- **Schemas Without a Type**: Fuzzy infers it from the other keywords: `properties` or `required` mean `object`, `items` or `minItems` mean `array`, `format: int64` means `integer`, `minimum` means `number`. With no hint at all, it uses `string`.
- **Null Values**: `null` is only sent in JSON and multipart bodies, where a multipart `null` is a part with `Content-Type: application/json`. Parameters, form bodies, text bodies and binary bodies can't represent it, so they always get a real value.
- **Spec Format**: Only OpenAPI 3.0.x in JSON. Swagger 2.0, OpenAPI 3.1 and YAML are refused with an error, and so are `$ref`s that point outside the document.
- **Progress and Exit Codes**: In a terminal, a status bar on stderr updates in place during the run and disappears when it ends. Above it, the endpoints change color: red when not tested yet, yellow while sending requests, white once all of them are sent. The list stays on screen after the run; if it doesn't fit, the endpoints still in progress come first and the last line counts the rest. With `NO_COLOR` set or on Windows there is only the bar. When stderr is redirected, or with `TERM=dumb`, Fuzzy prints a plain `[+] 1200/6000 requests` line every 5 seconds instead. The `5xx` and `err` counters cover every request, whatever `--status` keeps for the report. Fuzzy exits with `0` when the run completes, whatever the API answered, `1` if it can't run (for example, an unreadable spec) and `2` for invalid arguments.

### Known Limitations 🐙

- **XML Bodies**: An operation whose required body is only available as XML is skipped and listed in the report. If the body is optional, the operation is called without it.
- **Overlapping `oneOf`**: When branches overlap in the spec itself (`{}` can match two of them), a value can match more than one. Branches that differ by their `required` fields are told apart.
- **Unsupported Keywords and Patterns**: `not` and discriminators inherited through `allOf` are ignored. Patterns must use Go regular expression syntax, so one with a lookahead is ignored and listed under the report's warnings.
- **Formats and Lengths**: Values generated for a `format` such as `email` or `uuid` follow the format, not `maxLength` or `--string-max`.

---

## Use Cases 🦀

This fuzzer is particularly useful for:

- **Authentication Testing**: Test login endpoints with different password combinations
- **Parameter Validation**: Check that APIs accept every value their spec says is valid
- **Security Testing**: Send injection, XSS and path traversal payloads from a dictionary, wherever the spec allows them
- **Error Handling**: Find valid requests that still make the API fail (`--status 5xx,err`)
- **Rate Limiting**: Test API rate limiting mechanisms
- **Input Sanitization**: Check if the API properly sanitizes user input

---

## Best Practices 🦋

1. **Start Small**: Begin with a low `-n`, a small set of test values and a few endpoints (check them with `--list`) and expand gradually
2. **Monitor Responses**: Pay attention to different HTTP status codes (200, 401, 403, 500)
3. **Rate Limiting**: Lower `-c` if needed to avoid overwhelming the target
4. **Legal Considerations**: Only test systems you own or have explicit permission to test
5. **Report Analysis**: Use `--status 5xx,err` to focus the report on what went wrong

---
## Contributing 🐜

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.

1. Fork the repo
2. Create a branch `feature/xxx`
3. Add tests for new features
4. Open a PR with description and use case

Guidelines:

* Maintain CLI backward compatibility when possible
* Add unit tests for new schema keywords and formats
* Document new flags in the Command Line Options table of this README

## License 🐮

This project is open source. Please check the license file for details.

---

## Disclaimer 🐰

This tool is for educational and authorized testing purposes only. Always ensure you have proper permission before testing any systems. The authors are not responsible for any misuse of this tool.

---

## Best practices and security 🐶

* Run fuzzing in isolated environments (containers, VMs) to avoid harming real systems.
* Set resource limits (CPU, memory) for the target.
* Do not fuzz sensitive production data.
* Record seeds and metadata to reproduce results.
* Use real-world corpus inputs (telemetry/customer data) to improve effectiveness — anonymize sensitive data first.

---
