# Fuzzy audit (commit `8c8aa59`, 2026-11-03)

**Fuzzy's concurrency is solid, but I found several bugs that break its documented guarantees, and two easy performance wins.** I reproduced everything below by running it, except where marked "from reading the code".

- **Test suite:** it passes 3 times in a row under `-race`. There's no CI, so nothing would flag a break.
- **Checked and clean:**
  - No data races, deadlocks, or goroutine leaks during a run.
  - Bounded backpressure works as designed.
  - The latency histogram stays a fixed size.
  - I found no HTML or CSV injection.
  - The atomic report replace is sound.

## 🟠 High

**4. Fields that share a `$ref` always get the same value.** Enum and dictionary picks use `(iter + offset) % len` ([generator.go:708](internal/generator/generator.go#L708)), and the memo gives both fields the same node and therefore the same offset ([generator.go:102](internal/generator/generator.go#L102)).

- **Evidence:** over 1000 requests, `from` equalled `to` every time, and only 3 of 27 possible combinations were ever sent.
- **Impact:** bugs that need `from != to` can never be found. Fields with separate enums of equal size also move in lockstep, so they can't cover all combinations either (from reading the code).
- **Fix:** derive the offset per field, or shuffle the pick order per request, while still visiting every value.

**5. A large `maxLength` can exhaust memory or crash the run.** When the schema range doesn't overlap the flag bounds, the full schema range is used ([bounds.go:81](internal/generator/bounds.go#L81)).

| Schema (`minLength: 12`) | Result |
|---|---|
| `maxLength: 100000` | 167 KB mean body |
| `maxLength: 50000000` | 83 MB mean body, 10.7s for 5 requests |
| `maxLength: 2^63-1` | panic: `makeslice: len out of range` |

- **Fix:** cap the upper bound at `lo + span`, which is still valid for the schema.

**6. Any panic loses the whole run, and looks like a usage error.** Workers have no `recover` ([fuzzer.go:143](internal/fuzzer/fuzzer.go#L143)).

- **Evidence:** `maxItems: -1` panicked mid-run ([generator.go:591](internal/generator/generator.go#L591)), leaving a **0-byte report** and **exit code 2**, the code documented for invalid arguments.
- **Same crash at load:** `null` in `parameters`, `properties`, `oneOf`, `allOf`, `components.schemas`, or a parameter's `content` causes a nil-pointer panic ([resolve.go:84](internal/openapi/resolve.go#L84), [resolve.go:95](internal/openapi/resolve.go#L95), [resolve.go:191](internal/openapi/resolve.go#L191), [resolve.go:231](internal/openapi/resolve.go#L231)).
- **Fix:** return an error for nulls at load, and recover per job so a panic becomes a reported result.

**7. A server that dies mid-response is reported as `200`.** The read error is discarded at [fuzzer.go:316](internal/fuzzer/fuzzer.go#L316).

- **Evidence:** both an aborted handler and a body that hangs until timeout gave `counts=map[200:1]`.
- **Impact:** `--status 5xx,err` hides exactly the crashes your README's crash-hunting use case looks for.
- **Fix:** keep the status, but record the read error in the result and surface it in the report and filter.

**8. A big dictionary uses huge amounts of memory before any request is sent.** Each schema builds its own boxed copy of the matching values ([generator.go:381](internal/generator/generator.go#L381)).

| 100k-line dictionary | Peak memory | Time |
|---|---|---|
| No dictionary, 1000 schemas | 12 MB | 0.01s |
| 1000 schemas | **4.6 GB** | 10.7s |
| 4000 schemas | **12.4 GB** | 28.4s |

- **Fix:** cache filtered lists by constraint (length, pattern, format, location), and share one list when nothing is filtered out.

**9. Memory grows without limit under the default filter.** Results are kept for every call ([fuzzer.go:221](internal/fuzzer/fuzzer.go#L221)), and most renderers then build the whole report in memory first ([report.go:129](internal/report/report.go#L129), [html.go:21](internal/report/html.go#L21), [text.go:13](internal/report/text.go#L13), [report.go:119](internal/report/report.go#L119)).

- **Evidence:** 300k results reached 408 MB, about 1.3 KB per result, while a filtered run stayed at 19 MB.
- **Formats at 60k results:** JSON 185 MB, HTML 138 MB, text 99 MB, Markdown 84 MB, CSV and XML (which stream) 54 MB.
- **Impact:** without `--incremental`, an out-of-memory kill loses everything.
- **Fix:** render directly to the buffered writer, and add a per-endpoint result cap or spill to disk.

## ⚡ Performance

| # | Issue | Measured | Fix |
|---|---|---|---|
| 10 | **GC runs constantly.** The live heap is 1–2 MB, so a cycle starts about every 3 ms | Filtered 60k-request run: `GOGC=100` about **5.3s**, 1656 cycles, 18 MB; `GOGC=400` about **1.17s**, 38 cycles, 32 MB | Call `debug.SetGCPercent(400)` in `main` unless `GOGC` is set. **About 4.5× faster for about 14 MB more memory** |
| 11 | **Spec loading is slow.** `Schema.UnmarshalJSON` re-decodes each subtree at every level ([schema.go:74](internal/openapi/schema.go#L74)) | 20 MB spec: 5.9s and 1.23 GB allocated, vs 0.88s and 285 MB for plain JSON decoding | Collect property order in one pass, for example with a single token scan |
| 12 | **Input text is built for every call,** even when the filter discards the result ([target.go:183](internal/fuzzer/target.go#L183)) | `toJSON`, which also encodes bodies, is 14.7% of client CPU | Render the input only if the result is kept or logged |
| 13 | **XML rebuilds variants for each element** at request time ([xml.go:273](internal/fuzzer/xml.go#L273), [xml.go:204](internal/fuzzer/xml.go#L204)) | 106 µs and 334 allocations for a 10-item body | Pre-compute variants and order at compile time |
| 14 | **`--incremental` re-renders the whole report 10 times** | 4.14s vs 1.98s | Save on a time interval, or append results |
| 15 | **Schema compile and generation can blow up** on odd cycles ([generator.go:140](internal/generator/generator.go#L140), [generator.go:665](internal/generator/generator.go#L665)) | `oneOf` of 5 self-refs: 4.5s to compile. 3 required self-refs: 6 MB body. 4 required self-refs: 15.9s for one request, **ignoring Ctrl+C** | Share a per-request node budget, and warn about unsatisfiable cycles |

On test 10, the GC trace reports about 21% of CPU in GC, which doesn't fully account for the 4.5× gap. My guess is that frequent cycles also keep emptying `sync.Pool`s, but I didn't verify that.

## 🟡 Medium

- **16. Password leak:** with `--remote http://user:hunter2@host`, the password appears on stdout ([fuzz.go:80](cmd/fuzzy/fuzz.go#L80)) and in the report ([fuzz.go:99](cmd/fuzzy/fuzz.go#L99)). The log already uses `Redacted()`.
- **17. `--remote` with a query string breaks every request:** `http://h/?tenant=a` becomes `/?tenant=a/x`, so everything hits `/` ([target.go:52](internal/fuzzer/target.go#L52)) and still looks like a normal run. Reject it, or merge it properly, in [parseRemote](internal/config/config.go#L228).
- **18. Two `Cookie:` header lines:** multiple cookie parameters are sent separately ([target.go:157](internal/fuzzer/target.go#L157)), but RFC 6265 says there should be one. Join them with `; `.
- **19. Multi-file uploads aren't files:** an array of `format: binary` is sent as one JSON text field ([target.go:114](internal/fuzzer/target.go#L114)). It should be repeated file parts.
- **20. External path-item `$ref`s vanish silently:** `"/b": {"$ref": "other.json#/..."}` is dropped without error ([document.go:50](internal/openapi/document.go#L50)), but CLAUDE.md says these are refused.
- **21. `readOnly` is lost through `allOf`:** `mergeAt` never copies it ([schema.go:272](internal/openapi/schema.go#L272)), so those fields get sent.
- **22. Untrusted text isn't always inert:**
  - A raw terminal escape from a spec path reaches stderr ([document.go:199](internal/openapi/document.go#L199), [main.go:44](cmd/fuzzy/main.go#L44)).
  - A summary of `# Fake section` becomes a real Markdown heading ([report.go:218](internal/report/report.go#L218)).
- **23. `-o` equal to `--spec` overwrites your spec** with the report. The same check exists for `--log` ([config.go:152](internal/config/config.go#L152)).

## 🔵 Low

- **24. No cleanup in `Run`:** `Run` never calls `CloseIdleConnections` ([fuzzer.go:257](internal/fuzzer/fuzzer.go#L257)). Goroutines went from 3 to 123 over 5 runs. It's harmless in the CLI and only matters in tests or as a library.
- **25. Exit code 0 when nothing was reachable:** this matches your documented rule, but it can mislead CI. Consider an opt-in `--fail-on`.
- **26. `--seed 0` gives a random seed:** an explicit 0 can't be reproduced ([config.go:174](internal/config/config.go#L174), from reading the code).
- **27. Build output isn't ignored:** `.gitignore` has `/main`, but your documented build creates `fuzzy`, which isn't listed (from reading the code).

## Suggested order

1. **Today:** add a CI job, #10 GC setting (one line), #5 length cap.
2. **Next:** #4 value coverage, #7 mid-body failures, #6 panics and nulls, #16 password leak.
3. **Then:** #8 dictionary and #9 streaming and result cap, which are your biggest memory wins, followed by #11 and #12.

## Limits of this audit

- **Not run:** `govulncheck` and `staticcheck`, because downloading them was blocked by the permission policy. To run them yourself:

  ```bash
  go run golang.org/x/vuln/cmd/govulncheck@latest ./...
  go run honnef.co/go/tools/cmd/staticcheck@latest ./...
  ```

- **Measurements:** these used loopback against a trivial server on a 12-thread machine. Trust the ratios more than the absolute numbers; a real network will shrink the gain from #10.
- **Not deep-tested:** the terminal drawing in `progress`, Windows behavior, and HTTP/2 under `--insecure`.
