# Fuzzy - HTTP API Fuzzer

> **Fuzzy** — A simple shiny HTTP API fuzzing tool written in **Go** (Golang).
> Designed to test HTTP endpoints by fuzzing request parameters and JSON body fields with different values.

![fuzzy with little bit of anxiety because the job is not done yet](assets/fuzzy.png)

---

Fuzzy is shy, a little anxious when he is among many people, but he works hard because he loves his job.
He is always eager to learn new things and improve his skills, even if it means stepping out of his comfort zone.

## Table of Contents

* [Description](#description)
* [Key Features](#key-features)
* [Installation](#installation)
* [Usage](#usage)
* [Examples](#examples)
* [File Structure](#file-structure)
* [Command Line Options](#command-line-options)
* [Contributing](#contributing)

---

## Description

Fuzzy is a newbie-friendly command-line HTTP API fuzzer that allows you to test web endpoints by systematically varying input parameters. It supports both GET and POST requests and can fuzz URL parameters for GET requests or JSON body fields for POST requests.

The tool reads values from a text file and iterates through them, making HTTP requests for each value and reporting the response status codes.

---

## Key Features

* **GET Request Fuzzing**: Append different values to URL endpoints
* **POST Request Fuzzing**: Replace JSON body fields with different test values
* **Flexible Input**: Use any text file with values to test (one value per line)
* **API Key Support**: Built-in X-API-KEY header support
* **Simple Output**: Clean output showing request details and response status
* **Lightweight**: Single binary with no external dependencies

---

## Requirements

* Go 1.20+ (1.21+ recommended)
* OS: Linux/macOS/Windows (Linux recommended for heavy testing)
* (Optional) Docker for target isolation

---

## Installation

1. **Prerequisites**: Go 1.19 or higher

2. **Clone and build**:
```bash
git clone <repository-url>
cd fuzzy
go build -o fuzzy fuzzy.go
```

3. **Or compile directly**:
```bash
go build fuzzy.go
```

---

## Usage

The fuzzer supports two HTTP methods: GET and POST. The tool automatically determines which field to fuzz based on the filename of the values file (without .txt extension).

### Basic Syntax

```bash
./fuzzy -m <METHOD> -e <ENDPOINT> -fp <VALUES_FILE> [OPTIONS]
```

### GET Request Fuzzing

For GET requests, values are appended directly to the endpoint URL:

```bash
./fuzzy -m GET -e "https://api.example.com/users/" -fp user_ids.txt
```

This will test URLs like:
- `https://api.example.com/users/1`
- `https://api.example.com/users/admin`
- `https://api.example.com/users/test`

### POST Request Fuzzing

For POST requests, you need to provide a JSON body template and specify which field to fuzz:

```bash
./fuzzy -m POST -e "https://api.example.com/login" -bp req/body.json -fp passwords.txt
```

The tool will replace the field named `passwords` (derived from `passwords.txt`) in your JSON body with each value from the file.


---

## Examples

### Example 1: Testing User ID Endpoints

**Values file** (`user_ids.txt`):
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
./fuzzy -m GET -e "https://api.example.com/users/" -fp user_ids.txt
```

**Output**:
```
[+] Endpoint: https://api.example.com/users/1
[+] Response status: 200 OK

[+] Endpoint: https://api.example.com/users/999999
[+] Response status: 404 Not Found

[+] Endpoint: https://api.example.com/users/admin
[+] Response status: 403 Forbidden
```

### Example 2: Testing Login Endpoints

**JSON body template** (`req/body.json`):
```json
{
  "username": "admin",
  "password": "placeholder",
  "remember": true
}
```

**Values file** (`password.txt`):
```
admin
123456
password
qwerty
letmein
```

**Command**:
```bash
./fuzzy -m POST -e "https://api.example.com/auth/login" -bp req/body.json -fp password.txt
```

**Output**:
```
[+] Request body {... 'password':'admin' ... }
[+] Response status: 401 Unauthorized

[+] Request body {... 'password':'123456' ... }
[+] Response status: 401 Unauthorized

[+] Request body {... 'password':'letmein' ... }
[+] Response status: 200 OK
```

---

## File Structure

```
fuzzy/
├── fuzzy.go              # Main fuzzer application
├── go.mod               # Go module dependencies
├── go.sum               # Go module checksums
├── README.md            # This documentation
├── assets/
│   └── fuzzy.png        # Project logo/image
└── req/                 # Request templates and test data
    ├── body.json        # JSON body template for POST requests
    └── parameters.txt   # Example parameter values file
```

---

## Command Line Options

| Flag | Description | Example |
|------|-------------|---------|
| `-m` | HTTP method (GET or POST) | `-m POST` |
| `-e` | Target endpoint URL | `-e "https://api.example.com/login"` |
| `-bp` | Body file path (required for POST) | `-bp req/body.json` |
| `-fp` | Values file path (required) | `-fp passwords.txt` |

### Important Notes

- **API Key**: The tool includes X-API-KEY header support. Set the `apiKey` variable in the code if needed.
- **Field Mapping**: For POST requests, the field name is derived from the values filename (without .txt extension).
- **Content Type**: POST requests automatically set `Content-Type: application/json`.
- **Response Bodies**: Currently commented out in the code, but can be enabled by uncommenting the response reading sections.

---

## Use Cases

This fuzzer is particularly useful for:

- **Authentication Testing**: Test login endpoints with different password combinations
- **Parameter Validation**: Check how APIs handle unexpected parameter values
- **Security Testing**: Test for injection vulnerabilities, XSS, path traversal
- **Error Handling**: Verify proper error responses for invalid inputs
- **Rate Limiting**: Test API rate limiting mechanisms
- **Input Sanitization**: Check if the API properly sanitizes user input

### Common Test Scenarios

**SQL Injection Testing**:
```
' OR 1=1--
'; DROP TABLE users;--
admin'--
' UNION SELECT * FROM users--
```

**XSS Testing**:
```
<script>alert('xss')</script>
<img src=x onerror=alert(1)>
javascript:alert(1)
```

**Path Traversal Testing**:
```
../../../etc/passwd
..\..\..\..\windows\system32\config\sam
/etc/shadow
```

**Authentication Bypass**:
```
admin
administrator
root
test
guest
```

---

## Best Practices

1. **Start Small**: Begin with a small set of test values and expand gradually
2. **Monitor Responses**: Pay attention to different HTTP status codes (200, 401, 403, 500)
3. **Rate Limiting**: Add delays between requests if needed to avoid overwhelming the target
4. **Legal Considerations**: Only test systems you own or have explicit permission to test
5. **Log Analysis**: Enable response body logging to analyze detailed error messages

---

## Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.

### To-Do List

- [ ] Add support for custom HTTP headers
- [ ] Implement request rate limiting
- [ ] Add XML body support for SOAP APIs  
- [ ] Include response time measurements
- [ ] Add support for HTTP authentication methods
- [ ] Implement result export to CSV/JSON formats

---

## License

This project is open source. Please check the license file for details.

---

## Disclaimer

This tool is for educational and authorized testing purposes only. Always ensure you have proper permission before testing any systems. The authors are not responsible for any misuse of this tool.

## Fuzzing strategies and mutators

Fuzzy includes standard mutators:

* `flip_bits` (flip random bits)
* `byte_insert` / `byte_remove` / `byte_replace`
* `splice` (combine two corpus inputs)
* dictionary-based expansions (use provided keywords)
* length scaling (grow/shrink inputs)

Selection strategies:

* Random
* Coverage-guided (if enabled and the target provides coverage hooks)
* Corpus-priority (favors inputs that improved coverage)

For coverage-guided fuzzing: the target must expose an endpoint or hook (e.g., callback) that returns a coverage bitmap or ID.

---

## Logging, crash triage and reproduction

* Crashes are written to `crashes/` with metadata files (`.meta.json`) containing:

  * timestamp
  * seed
  * mutator pipeline
  * stack trace (if available)
  * target stdout/stderr

* Reproduction: `Fuzzy reproduce --input=crashes/crash-0001` runs the target with that input and prints output and exit code.

* Minimization: a `minimize` command attempts to shrink the input while preserving the crash.

Example:

```bash
./Fuzzy minimize --input=crashes/crash-0001 --harness=./examples/unit_target
```

---

## Metrics and monitoring

Fuzzy exports basic metrics (tests/sec, total crashes, corpus size) to:

* stdout (human-readable)
* a JSON file `metrics.json`
* optionally a Prometheus Pushgateway (configurable)

---

## Best practices and security

* Run fuzzing in isolated environments (containers, VMs) to avoid harming real systems.
* Set resource limits (CPU, memory) for the target.
* Do not fuzz sensitive production data.
* Record seeds and metadata to reproduce results.
* Use real-world corpus inputs (telemetry/customer data) to improve effectiveness — anonymize sensitive data first.

---

## Contributing

1. Fork the repo
2. Create a branch `feature/xxx`
3. Add tests for new features
4. Open a PR with description and use case

Guidelines:

* Maintain CLI backward compatibility when possible
* Add unit tests for new mutators
* Document new flags in `docs/` and `configs/`

---

## Quick examples

1. Fuzz HTTP for 5 minutes:

```bash
./Fuzzy fuzz --mode=http --url=http://localhost:8000/parse --corpus=./examples/http_target/corpus --timeout=5s --duration=5m
```

2. Fuzz unit with 10k iterations:

```bash
./Fuzzy fuzz --mode=unit --harness=./examples/unit_target --max-iter=10000
```

---

## Known limitations

* Currently, coverage-guided mode requires the target to provide coverage hooks.
* Some mutators are naive for highly structured formats (e.g., complex binary images): for those cases, write format-specific generative producers.

---

## License

Choose a license (MIT/Apache-2.0). Example:

```
MIT License
Copyright (c) 2025 Your Name
```

---

## Acknowledgements

Inspired by general ideas from modern fuzzers (AFL, libFuzzer) and community tools in the Go ecosystem.

---

If you want, I can:

* generate a ready-to-use `configs/default.yaml`,
* add more advanced harness examples (e.g., coverage handling),
* prepare a `docker-compose` script to run fuzzing in isolation.

Tell me which of these you prefer and I’ll include the files right away.
