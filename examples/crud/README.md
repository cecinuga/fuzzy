# CRUD Example

A small in-memory CRUD invoice API to try Fuzzy on. It runs locally, loads a few sample invoices from `db.json` and only changes its copy in memory, so fuzzing it can't damage anything: restart the server and it's back to the contents of `db.json`.

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
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json --seed 42
```

**3. Read the report**, which is written to `fuzzy-report.md`. To use another path, add `-o <path>`.

## What to Expect

Fuzzy sends 100 requests per endpoint by default, so 600 in total. The summary at the top of the report should look like this:

| Endpoint | Status codes | Why |
|---|---|---|
| `GET /invoices` | All `200` | Listing always succeeds |
| `POST /invoices` | About half `201`, half `409` | Dates are random, so about half the bodies have a due date before the issue date |
| `GET /invoices/{invoice_id}` | Mostly `404`, a few `200` | Random IDs from 1 to 1000 rarely match an existing invoice |
| `PUT /invoices/{invoice_id}` | Mostly `404`, a few `200` or `409` | Same as above, and only drafts can be changed |
| `DELETE /invoices/{invoice_id}` | Mostly `404`, a few `204` or `409` | Same as above, and only drafts can be deleted |
| `PUT /invoices/{invoice_id}/status` | Mostly `404`, a few `200` or `409` | Same as above, and most status changes aren't allowed |

- **A `409` is expected.** It means a business rule refused a well-formed request. The spec can't express these rules, so Fuzzy can't avoid them.
- **A `422` is worth a look.** Fuzzy sends valid requests, so a `422` means the generated value and the API's validation disagree.
- **Counts can change between runs, even with the same seed.** A seed reproduces the inputs, but the status of each request depends on which invoices exist, and in which status, when it arrives. Invoices also build up between runs until you restart the server, so later runs get more `200` and `204` responses.
- **`ERR` with `connection refused` on every result** means the server wasn't running or is on another address.

To reach the existing invoices more often, and so the business rules, narrow the random integers:

```bash
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json --int-max 10
```

About half the calls that take an ID then find an invoice, and most of those get a `409`. The bound applies to every integer the spec allows it for, including `quantity`, `unit_price` and `limit`.

To see only the failures, filter the report:

```bash
./fuzzy --remote http://localhost:8000 --spec examples/crud/artifacts/openapi.json --status 5xx,err
```

On a healthy run this reports 0 results. The summary still shows the status codes of every request.

## API

| Method | Path | Description | Success |
|---|---|---|---|
| `POST` | `/invoices` | Create a draft invoice | `201` |
| `GET` | `/invoices` | List invoices in creation order, optionally by status | `200` |
| `GET` | `/invoices/{invoice_id}` | Get an invoice | `200` |
| `PUT` | `/invoices/{invoice_id}` | Replace a draft invoice | `200` |
| `DELETE` | `/invoices/{invoice_id}` | Delete a draft invoice | `204` |
| `PUT` | `/invoices/{invoice_id}/status` | Change an invoice's status | `200` |
| `GET` | `/openapi.json` | Get the spec | `200` |

`/openapi.json` isn't listed in the spec, so Fuzzy doesn't call it.

### Business Rules

- **Amounts are computed by the server.** `subtotal` is the sum of `quantity × unit_price` over the lines, `tax` is `tax_rate` percent of it, rounded to the nearest cent, and `total` is the two together. All amounts are integers in cents, and any `subtotal`, `tax` or `total` in a request is ignored.
- **An invoice can't be due before it's issued.** `due_date` can be the same day as `issue_date`, but not earlier.
- **New invoices are drafts.** Any `status` or `number` in a create or replace request is ignored.
- **Only drafts can be replaced or deleted.** An invoice that has been sent is cancelled by voiding it.
- **The status follows a fixed path.** Paid and void are final, and nothing goes back to draft:

  | From | Can go to |
  |---|---|
  | `draft` | `sent` |
  | `sent` | `paid`, `void` |
  | `paid` | Nothing |
  | `void` | Nothing |

- **Sending assigns the number.** `number` stays `null` on a draft and is set when the invoice is sent, continuing from the highest one in use. Deleted drafts therefore leave no gaps, and later status changes keep the number.

### Sample Data

`db.json` holds the invoices the server starts with, one for each status. New invoices get IDs from 6 and numbers from 4.

| ID | Customer | Status | Number | Total (cents) |
|---|---|---|---|---|
| 1 | Acme Srl | `draft` | | 76250 EUR |
| 2 | Northwind Ltd | `sent` | 1 | 43200 GBP |
| 3 | Globex Inc | `paid` | 2 | 108034 USD |
| 4 | Initech GmbH | `void` | 3 | 178500 EUR |
| 5 | Umbrella Srl | `draft` | | 99999 EUR |

The server never writes to the file. It's embedded when the server is built, so an edit takes effect at the next `go run`. Amounts aren't stored in it, since they're always computed. Each invoice goes through the same validation as a request, and the server refuses to start if one breaks a rule.

### Validation

| Field | In | Type | Required | Rules |
|---|---|---|---|---|
| `customer` | body | string | yes | 1 to 64 characters |
| `currency` | body | string | yes | `EUR`, `USD` or `GBP` |
| `issue_date` | body | string | yes | Date as `YYYY-MM-DD` |
| `due_date` | body | string | yes | Date as `YYYY-MM-DD` |
| `tax_rate` | body | number | no | 0 to 100, default 0 |
| `notes` | body | string or `null` | no | Up to 256 characters |
| `lines` | body | array | yes | 1 to 20 lines |
| `lines[].description` | body | string | yes | 1 to 64 characters |
| `lines[].quantity` | body | integer | yes | 1 to 1000 |
| `lines[].unit_price` | body | integer | yes | 1 to 1000000, in cents |
| `status` | status body | string | yes | `sent`, `paid` or `void` |
| `invoice_id` | path | integer | yes | Greater than 0 |
| `status` | query | string | no | `draft`, `sent`, `paid` or `void` |
| `limit` | query | integer | no | 1 to 100, default 20 |
| `offset` | query | integer | no | 0 or more, default 0 |

### Responses

Invalid requests get a `422` that lists every problem:

```json
{"detail":[{"type":"string_too_short","loc":["body","customer"],"msg":"String should have at least 1 character","input":"","ctx":{"min_length":1}},{"type":"enum","loc":["body","currency"],"msg":"Input should be one of: EUR, USD, GBP","input":"JPY","ctx":{"expected":["EUR","USD","GBP"]}}]}
```

A valid request for an invoice that doesn't exist gets a `404`. The spec doesn't declare this response.

```json
{"detail":"invoice 999 not found"}
```

A valid request that breaks a business rule gets a `409`:

```json
{"detail":"invoice 3 is paid: only drafts can be changed or deleted"}
```

If a request could get more than one of these, the `422` wins over the `404`, and the `404` over the `409`.

### Trying It by Hand

```bash
# Create an invoice
curl -X POST http://localhost:8000/invoices \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Acme Srl","currency":"EUR","issue_date":"2026-06-01","due_date":"2026-06-30","tax_rate":22,"lines":[{"description":"Consulting day","quantity":2,"unit_price":7500}]}'
# {"customer":"Acme Srl","currency":"EUR","issue_date":"2026-06-01","due_date":"2026-06-30","tax_rate":22,"notes":null,"lines":[{"description":"Consulting day","quantity":2,"unit_price":7500}],"id":6,"status":"draft","number":null,"subtotal":15000,"tax":3300,"total":18300}

# List invoices
curl 'http://localhost:8000/invoices?status=draft&limit=5'

# Get an invoice
curl http://localhost:8000/invoices/6

# Replace a draft
curl -X PUT http://localhost:8000/invoices/6 \
  -H 'Content-Type: application/json' \
  -d '{"customer":"Acme Srl","currency":"EUR","issue_date":"2026-06-01","due_date":"2026-07-15","tax_rate":22,"notes":"Extended terms","lines":[{"description":"Consulting day","quantity":3,"unit_price":7500}]}'

# Send it, which assigns its number
curl -X PUT http://localhost:8000/invoices/6/status \
  -H 'Content-Type: application/json' \
  -d '{"status":"sent"}'

# It can't be deleted any more: this gets a 409
curl -X DELETE http://localhost:8000/invoices/6

# Delete a draft
curl -X DELETE http://localhost:8000/invoices/5
```

Keep the `Content-Type` header: `curl -d` sends a form content type by default, and the API answers that with a `422`. Bodies are accepted as `application/json`, `application/*+json` or with no `Content-Type` at all.

IDs are never reused, even after a delete.

## Files

| Path | Description |
|---|---|
| `main.go` | The server and the spec |
| `main_test.go` | Tests, including a check that Fuzzy's parser accepts the spec |
| `db.json` | The invoices the server starts with |
| `artifacts/openapi.json` | The generated spec, rewritten at every startup |

The routes and the bounds in `main.go` feed both the handlers and the spec, so the two stay in sync. Change them there rather than editing `openapi.json`.

## Tests

```bash
go test -race ./examples/crud
```
