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

Then it calls every endpoint with requests that respect the spec, using values generated at random or taken from your dictionary, and writes what happened to a Markdown report.

---

## Key Features 🐞

* **OpenAPI 3.0 Driven**: Reads a JSON spec and fuzzes all of its endpoints, no request templates to write
* **Semantically Valid Requests**: Respects types, formats, ranges, lengths, patterns, enums, required fields, `$ref`, `allOf`, `oneOf`, `anyOf` and discriminators
* **Everywhere a Value Can Go**: Path, query, header and cookie parameters, plus JSON, form, multipart, text and binary bodies
* **Flexible Input**: Random values within bounds you choose, or a text file with values to test (one value per line)
* **Dictionary With a Safety Net**: A dictionary value is only used where the spec allows it; anything it doesn't cover gets a random valid value
* **Markdown Report**: Each endpoint with its spec and its results, filtered by the status codes you care about
* **Fast and Concurrent**: A bounded worker pool on a shared connection pool, with memory that stays low on big specs
* **Reproducible**: The same seed produces the same inputs, at any concurrency
* **Interrupt Friendly**: Ctrl+C stops the run and still writes the report, marked as partial
* **Lightweight**: Single binary with no external dependencies

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
./fuzzy -remote <URL|FQDN|HOST:PORT> -spec <OPENAPI_JSON> [OPTIONS]
```

### Choosing the Results 🐠

A result is what one call to an endpoint produced: its endpoint, status code, message and input.

`-status` decides which results end up in the report. It takes a comma separated list of exact codes, classes, ranges and `err`, which stands for requests that got no response at all (timeouts, refused connections):

```bash
./fuzzy -remote http://localhost:8080 -spec api.json -status 200,404,5xx,400-403,err
```

Without `-status`, every result is reported. Filtered results still count in each endpoint's totals, so you always see how many requests got each status code.

### Choosing the Values 🦊

By default values are random. Random strings have 1 to 10 characters, integers go from 0 to 1000, floats from 0 to 1000.0. Strings mix letters of several Unicode scripts with ASCII letters and digits, and `-charset alnum` keeps only the ASCII ones. You can change all of it:

```bash
./fuzzy -remote http://localhost:8080 -spec api.json \
  -string-min 3 -string-max 50 -int-min -100 -int-max 100 -float-min 0 -float-max 1 -charset alnum
```

The spec always wins: an `id` with `minimum: 5000` gets values from 5000 up, even if `-int-max` is lower, because the request has to stay valid.

With a dictionary, Fuzzy reads one value per line and works out the types each line can serve: `42` works as an integer, a float and a string, `true` as a boolean and a string, `admin` only as a string. A value is used only where it fits the spec. Parameters with no fitting value get random valid inputs instead:

```bash
./fuzzy -remote http://localhost:8080 -spec api.json -dict values.txt
```


---

## Examples

### Example 1: Hunting Server Errors 🐻

**Command** (`testdata/api.json` is a sample spec you can try):
```bash
./fuzzy -remote http://localhost:8080 -spec testdata/api.json -status 5xx,err -seed 42
```

**Output**:
```
[+] fuzzing 6 endpoints of http://localhost:8080, 100 requests each (seed 42)
[+] 600 requests sent, 0 without a response. Report: fuzzy-report.md
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
./fuzzy -remote https://api.example.com/v3 -spec api.json -dict values.txt -H "Authorization: Bearer $TOKEN"
```

Here `1` and `999999` are also tried as integers, and the rest only as strings. A string parameter with `maxLength: 5` would only get `1`, `admin` and `root`, and one with `format: email` would get none of them, so it falls back to random valid emails.

### Example 3: Trying It on a Local API 🐸

`examples/crud` is a small in-memory CRUD API, safe to fuzz, written in Go. At startup it writes its spec to `artifacts/openapi.json`.

**Start the API** (from the repository root):
```bash
go run ./examples/crud
```

**Command** (from the repository root, in another terminal):
```bash
./fuzzy -remote http://localhost:8000 -spec examples/crud/artifacts/openapi.json
```

### Where the Requests Go 🐳

`-remote` can be a full URL, an FQDN or a `host:port`. Without a scheme, Fuzzy uses `https` when there is no port or the port is 443, and `http` for any other port. Write the scheme yourself to choose differently, for example `http://api.example.com`.

| `-remote` | Same as |
|---|---|
| `api.example.com` | `https://api.example.com` |
| `api.example.com:443` | `https://api.example.com:443` |
| `localhost:8080` | `http://localhost:8080` |
| `127.0.0.1:8080/v1` | `http://127.0.0.1:8080/v1` |

If `-remote` has no path, the path of the spec's `servers` entry is added to it. If it has one, it's used as given. Either way the host always comes from `-remote`, so Fuzzy never calls the server named in the spec.

| `-remote` | `servers` in the spec | Request |
|---|---|---|
| `http://localhost:8080` | `https://prod.example.com/api/v3` | `http://localhost:8080/api/v3/user/1` |
| `http://localhost:8080/custom` | `https://prod.example.com/api/v3` | `http://localhost:8080/custom/user/1` |

---

## File Structure 🦢

```
fuzzy/
├── README.md            # This documentation
├── CLAUDE.md            # Project guidance for Claude Code
├── go.mod               # Go module definition
├── assets/              # Project assets
│   ├── fuzzy.png        # Project logo/mascot image
│   └── fuzzyPersonality.txt # Fuzzy's character description
├── cmd/fuzzy/           # The command line entry point
├── examples/crud/       # Local CRUD API to try Fuzzy on
├── internal/
│   ├── config/          # Flags and their validation
│   ├── openapi/         # Loads the spec, resolves $ref and allOf
│   ├── generator/       # Turns schemas into valid values, dictionary included
│   ├── fuzzer/          # Encodes requests, runs the worker pool, filters statuses
│   └── report/          # Writes the Markdown report
└── testdata/
    ├── api.json         # Sample OpenAPI spec
    └── name.txt         # Sample dictionary
```

---

## Command Line Options 🐊

| Flag | Description | Example |
|------|-------------|---------|
| `-remote` | API to fuzz: URL, FQDN or `host:port` (required) | `-remote localhost:8080` |
| `-spec` | OpenAPI 3.0 spec in JSON format (required) | `-spec api.json` |
| `-status` | Status codes in the report (default all) | `-status 200,404,5xx,400-403,err` |
| `-dict` | Values file, one value per line (default random values only) | `-dict values.txt` |
| `-string-min` | Random string minimum length (default 1) | `-string-min 3` |
| `-string-max` | Random string maximum length (default 10) | `-string-max 50` |
| `-int-min` | Random integer minimum (default 0) | `-int-min -100` |
| `-int-max` | Random integer maximum (default 1000) | `-int-max 100` |
| `-float-min` | Random float minimum (default 0) | `-float-min 0.5` |
| `-float-max` | Random float maximum (default 1000) | `-float-max 1.5` |
| `-charset` | Random string characters, `unicode` or `alnum` (default `unicode`) | `-charset alnum` |
| `-n` | Requests per endpoint (default 100, or the dictionary size if larger) | `-n 500` |
| `-c` | Concurrent workers (default 20) | `-c 50` |
| `-timeout` | Timeout of each request (default 10s) | `-timeout 2s` |
| `-o` | Markdown report path (default `fuzzy-report.md`) | `-o report.md` |
| `-H` | Header for every request, repeatable; overrides generated ones | `-H "Authorization: Bearer abc"` |
| `-k` | Skip TLS certificate verification (useful for self-signed certs) | `-k` |
| `-seed` | Seed to reproduce a run's inputs (default random, also for `0`; printed in the report) | `-seed 42` |

### Important Notes 🦖

- **Fuzzy Means It**: Every operation in the spec gets called, `DELETE` and `PUT` included, with values that are valid on purpose. Against a real system this really changes or deletes data, so point it at a disposable environment.
- **Authentication**: Security schemes in the spec are not handled automatically. Pass your credentials with `-H`.
- **Redirects**: They are reported as results and not followed, so a `302` shows up as a `302`, and a redirect can never take Fuzzy away from the target.
- **Response Bodies**: The message of a result is the start of the response body (200 characters at most), or the standard status text if the body is empty.
- **Dictionary Coverage**: Values take turns, so with enough requests each one gets used. On an optional parameter, which is only sent about half the time, some can be missed: raise `-n` if you need them all.
- **Enums**: Values come only from the list, ahead of the dictionary and random values. With `nullable: true`, `null` is sent only if the list includes it.
- **Schemas Without a Type**: Fuzzy infers it from the other keywords: `properties` or `required` mean `object`, `items` or `minItems` mean `array`, `format: int64` means `integer`, `minimum` means `number`. With no hint at all, it uses `string`.
- **Null Values**: `null` is only sent in JSON and multipart bodies, where a multipart `null` is a part with `Content-Type: application/json`. Parameters, form bodies, text bodies and binary bodies can't represent it, so they always get a real value.
- **Spec Format**: Only OpenAPI 3.0.x in JSON. Swagger 2.0, OpenAPI 3.1 and YAML are refused with an error, and so are `$ref`s that point outside the document.
- **Progress and Exit Codes**: Long runs print their progress to stderr every 5 seconds. Fuzzy exits with `0` when the run completes, whatever the API answered, `1` if it can't run (for example, an unreadable spec) and `2` for invalid arguments.

### Known Limitations 🐙

- **XML Bodies**: An operation whose required body is only available as XML is skipped and listed in the report. If the body is optional, the operation is called without it.
- **Overlapping `oneOf`**: When branches overlap in the spec itself (`{}` can match two of them), a value can match more than one. Branches that differ by their `required` fields are told apart.
- **Unsupported Keywords and Patterns**: `not` and discriminators inherited through `allOf` are ignored. Patterns must use Go regular expression syntax, so one with a lookahead is ignored and listed under the report's warnings.
- **Formats and Lengths**: Values generated for a `format` such as `email` or `uuid` follow the format, not `maxLength` or `-string-max`.

---

## Use Cases 🦀

This fuzzer is particularly useful for:

- **Authentication Testing**: Test login endpoints with different password combinations
- **Parameter Validation**: Check that APIs accept every value their spec says is valid
- **Security Testing**: Send injection, XSS and path traversal payloads from a dictionary, wherever the spec allows them
- **Error Handling**: Find valid requests that still make the API fail (`-status 5xx,err`)
- **Rate Limiting**: Test API rate limiting mechanisms
- **Input Sanitization**: Check if the API properly sanitizes user input

---

## Best Practices 🦋

1. **Start Small**: Begin with a low `-n` and a small set of test values and expand gradually
2. **Monitor Responses**: Pay attention to different HTTP status codes (200, 401, 403, 500)
3. **Rate Limiting**: Lower `-c` if needed to avoid overwhelming the target
4. **Legal Considerations**: Only test systems you own or have explicit permission to test
5. **Report Analysis**: Use `-status 5xx,err` to focus the report on what went wrong

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

- 🔴 high impact features or improvements or bug fixes should be prioritized.
- 🟠 medium-high general improvements to code quality, usability, performance and testing.
- 🟡 medium features, improvement or refactoring.
- 🟢 low urgency features, minor tweaks, documentation updates.
- 🔵 funny little things.

## Roadmap 🦉
- 🟠 Generalize and modularize code for easier extension
- 🟡 Implement result export to CSV/JSON formatsI keys


### To-Do Functionality List 🦞
- 🔴 Add support for certificates
- 🟠 Implement logging formats (txt, csv, xml, html) with automatic detection of file extension
- 🟠 Add support for more response formats (e.g., XML, HTML)
- 🟡 Add support for more authentication methods (e.g., OAuth, AP )
- 🟡 Implement result export to CSV/JSON formats keys
- 🟢 Add support for XML body for SOAP APIs  
- 🟢 Add file logger

### To-Do Improvement List 🐍
- 🔵 Calculate and optimize code complexity
- 🟠 Include response time measurements
- 🟠 Generalize and modularize code for easier extension
- 🟠 Implement request rate limiting

### To-Do Bug fixes List 🦔


### To-Do Testing List 🦅
 
---

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
