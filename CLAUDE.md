# CLAUDE.md

Fuzzy is a concurrent, fast, reliable and efficient HTTP API fuzzer written in Go. It takes an OpenAPI 3.0 spec and sends semantically valid requests to every endpoint, using random or dictionary values, then writes a Markdown report.

`README.md` is the user documentation: full flag table, examples, known limitations and roadmap. Update it when flags or behavior change.

## Commands

```bash
go build -o fuzzy ./cmd/fuzzy
go vet ./...
go test -race ./...
./fuzzy -remote http://localhost:8080 -spec testdata/api.json -status 5xx,err -seed 42
```

Requires Go 1.25.1+. Sample inputs are `testdata/api.json` (spec) and `testdata/name.txt` (dictionary). `examples/crud` is a local API that is safe to fuzz.

## Architecture

The entry point is `cmd/fuzzy`. Data flows through the packages in `internal/` in this order:

| Package | Responsibility |
|---|---|
| `config` | Flags and their validation |
| `openapi` | Loads the spec, resolves `$ref` and `allOf` |
| `generator` | Turns schemas into valid values, including dictionary handling |
| `fuzzer` | Encodes requests, runs the worker pool, filters statuses |
| `report` | Writes the Markdown report |

A **result** is the tuple `(endpoint, status code, message, input)`. Each endpoint in the report shows its spec (HTTP verb, parameters, parameter types) and its results.

## Behavior to preserve

- **Requests stay valid.** Values respect types, formats, ranges, lengths, patterns, enums, required fields, `$ref`, `allOf`, `oneOf`, `anyOf` and discriminators, in path, query, header and cookie parameters and in JSON, form, multipart, text and binary bodies.
- **The spec beats the bounds.** Values come from the intersection of the flag bounds and the schema's range. If the two don't overlap, the schema's range is used.
- **Random defaults.** Strings have 1 to 10 characters, integers go from 0 to 1000, floats from 0 to 1000.0. The charset is `unicode` or `alnum`.
- **Dictionary with a fallback.** A dictionary value is used only where the spec allows it. Anything it doesn't cover gets a random valid value.
- **Seeds reproduce inputs at any concurrency.** Each job's random stream derives only from the seed and the job, never from scheduling.
- **`-status` only filters the report.** Filtered results still count in each endpoint's totals. `err` means a request got no response.
- **The host always comes from `-remote`.** Fuzzy never calls the server named in the spec, and redirects are reported, not followed.
- **Ctrl+C still writes the report**, marked as partial.
- **Memory stays low on big specs**, using a bounded worker pool on a shared connection pool.
- **No external dependencies.** `go.mod` has no requirements, and the output is a single binary.
- **Only OpenAPI 3.0.x in JSON is accepted.** Swagger 2.0, OpenAPI 3.1, YAML and `$ref`s pointing outside the document are refused with an error.

## Contributing rules

- Keep the CLI backward compatible when possible.
- Add tests for new features.
- Use `feature/xxx` branches. PRs need a description and a use case.

## Roadmap priorities

The roadmap in `README.md` uses these markers:

- 🔴 High-impact features, improvements or bug fixes; do these first
- 🟠 Medium-high: general improvements to code quality, usability, performance and testing
- 🟡 Medium: features, improvements or refactoring
- 🟢 Low urgency: minor tweaks, documentation updates
- 🔵 Funny little things

## Safety

Fuzzy calls every operation with valid values, including `DELETE` and `PUT`, so it really changes or deletes data. Only run it against disposable or local targets that you have permission to test, never production.
