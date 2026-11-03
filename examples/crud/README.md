# CRUD Example

A small in-memory CRUD API to try Fuzzy on. It runs locally and keeps its items in memory, so fuzzing it can't damage anything: restart the server and the data is gone.

The server has no planted bugs. A healthy run reports no `5xx` and no `err`, which makes it a good first run and a quick way to check that Fuzzy works.

## Quick Start

Run these commands from the repository root.

**1. Start the API:**

```bash
go run ./examples/crud
```

It listens on `127.0.0.1:8000` and writes its spec to `examples/crud/artifacts/openapi.json`. To use another address, run `go run ./examples/crud -addr 127.0.0.1:9000`.

**2. Build Fuzzy and fuzz the API, in another terminal:**

```bash
go build -o fuzzy ./cmd/fuzzy
./fuzzy -remote http://localhost:8000 -spec examples/crud/artifacts/openapi.json -seed 42
```

**3. Read the report**, which is written to `fuzzy-report.md`. To use another path, add `-o <path>`.

## What to Expect

Fuzzy sends 100 requests per endpoint by default, so 500 in total. The summary at the top of the report should look like this:

| Endpoint | Status codes | Why |
|---|---|---|
| `GET /items` | All `200` | Listing always succeeds |
| `POST /items` | All `201` | Every generated body is valid |
| `GET /items/{item_id}` | Mostly `404`, a few `200` | Random IDs from 1 to 1000 rarely match an existing item |
| `PUT /items/{item_id}` | Mostly `404`, a few `200` | Same as above |
| `DELETE /items/{item_id}` | Mostly `404`, a few `204` | Same as above |

- **A `422` is worth a look.** Fuzzy sends valid requests, so a `422` means the generated value and the API's validation disagree.
- **Counts change between runs, even with the same seed.** A seed reproduces the inputs, but the status of each request depends on which items exist when it arrives. Items also build up between runs until you restart the server, so later runs get more `200` and `204` responses.
- **`ERR` with `connection refused` on every result** means the server wasn't running or is on another address.

To see only the failures, filter the report:

```bash
./fuzzy -remote http://localhost:8000 -spec examples/crud/artifacts/openapi.json -status 5xx,err
```

On a healthy run this reports 0 results. The summary still shows the status codes of every request.

## API

| Method | Path | Description | Success |
|---|---|---|---|
| `POST` | `/items` | Create an item | `201` |
| `GET` | `/items` | List items in creation order | `200` |
| `GET` | `/items/{item_id}` | Get an item | `200` |
| `PUT` | `/items/{item_id}` | Replace an item | `200` |
| `DELETE` | `/items/{item_id}` | Delete an item | `204` |
| `GET` | `/openapi.json` | Get the spec | `200` |

`/openapi.json` isn't listed in the spec, so Fuzzy doesn't call it.

### Validation

| Field | In | Type | Required | Rules |
|---|---|---|---|---|
| `name` | body | string | yes | 1 to 64 characters |
| `description` | body | string or `null` | no | Up to 256 characters |
| `price` | body | number | yes | Greater than 0, up to 10000 |
| `quantity` | body | integer | no | 0 to 1000, default 0 |
| `item_id` | path | integer | yes | Greater than 0 |
| `limit` | query | integer | no | 1 to 100, default 20 |
| `offset` | query | integer | no | 0 or more, default 0 |

### Responses

Invalid requests get a `422` that lists every problem:

```json
{"detail":[{"type":"string_too_short","loc":["body","name"],"msg":"String should have at least 1 character","input":"","ctx":{"min_length":1}},{"type":"greater_than","loc":["body","price"],"msg":"Input should be greater than 0","input":0,"ctx":{"gt":0}}]}
```

A valid request for an item that doesn't exist gets a `404`. The spec doesn't declare this response. If a request is invalid and its item doesn't exist, the `422` wins.

```json
{"detail":"item 999 not found"}
```

### Trying It by Hand

```bash
# Create an item
curl -X POST http://localhost:8000/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"mug","price":9.5}'
# {"name":"mug","description":null,"price":9.5,"quantity":0,"id":1}

# List items
curl 'http://localhost:8000/items?limit=5'

# Get an item
curl http://localhost:8000/items/1

# Replace an item
curl -X PUT http://localhost:8000/items/1 \
  -H 'Content-Type: application/json' \
  -d '{"name":"mug","description":"blue","price":12,"quantity":3}'

# Delete an item
curl -X DELETE http://localhost:8000/items/1
```

Keep the `Content-Type` header: `curl -d` sends a form content type by default, and the API answers that with a `422`. Bodies are accepted as `application/json`, `application/*+json` or with no `Content-Type` at all.

IDs are never reused, even after a delete.

## Files

| Path | Description |
|---|---|
| `main.go` | The server and the spec |
| `main_test.go` | Tests, including a check that Fuzzy's parser accepts the spec |
| `artifacts/openapi.json` | The generated spec, rewritten at every startup |

The routes and the bounds in `main.go` feed both the handlers and the spec, so the two stay in sync. Change them there rather than editing `openapi.json`.

## Tests

```bash
go test -race ./examples/crud
```
