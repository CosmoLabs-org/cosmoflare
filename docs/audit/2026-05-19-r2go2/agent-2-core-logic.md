# Agent 2: Core Logic Audit Report

**Date**: 2026-05-19
**Scope**: CosmoDev-R2Go2 codebase — correctness, error handling, edge cases, data flow, concurrency safety
**Overall Score**: 78/100

## Sub-Scores

| Category | Score |
|----------|-------|
| Correctness | 80 |
| Error Handling | 82 |
| Edge Cases | 68 |
| Data Flow | 80 |
| Concurrency Safety | 75 |

---

## Critical Bugs

### CRITICAL BUG 1: Multipart Upload Parts Not Sorted (upload.go:253-261)

**Location**: `upload.go:253-261`
**Severity**: Critical — data corruption / upload failure
**Description**: When completing a multipart upload, the `completedParts` slice is passed directly to `CompleteMultipartUpload` without sorting by `PartNumber`. The S3/R2 API specification requires that completed parts be listed in ascending order by part number. When multiple parts are uploaded concurrently, the order in which they finish and are appended to the slice is non-deterministic.

**Impact**: For single-part uploads or uploads where parts happen to finish in order, this works correctly. For multi-part files where parts complete out of order (which is common under concurrent upload), the API will return an `InvalidPartOrder` error and the upload will fail. The user will see a cryptic S3 error with no indication that the client is at fault.

**Fix**: Add `sort.Slice(completedParts, func(i, j int) bool { return *completedParts[i].PartNumber < *completedParts[j].PartNumber })` before calling `CompleteMultipartUpload`.

### CRITICAL BUG 2: All S3 Errors Misclassified as NotFound (storage.go:158, download.go:59)

**Location**: `storage.go:158`, `download.go:59`
**Severity**: Critical — incorrect error semantics
**Description**: In `GetObject`, `HeadObject`, and `Download`, any error returned by the S3 client is wrapped as `R2NotFoundError`. This includes authentication failures (`403 Forbidden`), network errors (`connection refused`), server errors (`500 Internal Server Error`), and rate limiting (`429 Too Many Requests`).

**Impact**: Callers who check `errors.As(&r2NotFoundErr)` to determine if an object does not exist will get false positives for every other kind of failure. This can lead to dangerous behavior: a caller that deletes-then-recreates on "not found" would delete and recreate on an auth error. Retry logic that skips "not found" errors would silently skip transient network failures.

**Fix**: Inspect the underlying S3 error. Only wrap as `R2NotFoundError` when the S3 error code is `NoSuchKey` or the HTTP status is `404`. For all other errors, wrap as the appropriate error type (`R2AuthError` for 403, `R2Error` for 500, etc.).

### CRITICAL BUG 3: Config File Briefly World-Readable (internal/config/config.go:114-120)

**Location**: `internal/config/config.go:114-120`
**Severity**: Critical — security vulnerability
**Description**: The config file is written via `WriteConfigAs` (which creates the file with default permissions, typically `0644` — world-readable) and then `os.Chmod` is called to restrict permissions to `0600`. Between these two calls, the file containing API tokens is readable by any user on the system.

**Impact**: On multi-user systems or systems with monitoring agents, another process could read the config file during the window between write and chmod. The API token would be exposed. This is a classic TOCTOU (time-of-check-time-of-use) vulnerability.

**Fix**: Create the file with restricted permissions from the start. Use `os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)` to create the file with `0600` permissions, then write the config content to the file handle. Alternatively, write to a temporary file with `0600` permissions and atomically rename it into place.

---

## Additional Findings

### Correctness Issues

- **download.go:53 — Range condition fragile**: The range start condition `rangeStart >= 0` always evaluates to true when `rangeStart` is at its default value of `0`. This causes every download to include a `Range` header, even when no range was requested. (Also flagged by Agent 1.)

- **GetBucket/BucketExists O(n)**: Both methods list all buckets in the account and iterate to find the target. This is O(n) on the number of buckets. For accounts with hundreds of buckets, this adds significant latency. The Cloudflare API supports direct bucket retrieval by name.

- **DNSService.Update — no fetch-then-patch**: The `Update` method sends whatever fields the caller provides without first fetching the existing record. If the caller omits a field (e.g., `TTL`), the API may reset it to the default rather than preserving the existing value. A proper update should fetch the current record, merge the caller's changes, and send the merged result.

- **ZoneService.List — no accountID scoping**: The `List` method does not pass the account ID as a filter parameter to the Cloudflare API. On accounts that belong to multiple organizations, this may return zones from all organizations rather than just the target account.

- **CreateBucket uses client timestamp not server**: The bucket creation metadata includes a timestamp generated on the client machine. If the client's clock is skewed, this timestamp will be inaccurate. Server-side timestamps should be preferred for any metadata that will be compared across machines.

### Pagination and Resource Issues

- **WAF ListAccessRules only page 1**: The WAF access rules listing fetches only the first page of results. If the zone has more rules than fit on one page, the remaining rules are silently dropped. There is no auto-pagination or indication to the caller that results are truncated.

- **ListObjects no auto-pagination**: The object listing returns only the first page of results. Buckets with more than 1,000 objects (the default page size) will show a truncated listing. There is no continuation token handling or auto-pagination loop.

- **KV Put reads entire value into memory**: The `Put` method reads the entire value into a byte slice before sending it. For large values (KV supports up to 25MB), this causes memory pressure proportional to the value size. A streaming approach using `io.Reader` would be more memory-efficient.

### API Correctness Issues

- **CORS multi-origin invalid per spec**: The CORS configuration allows specifying multiple origins in a single rule. However, the CORS specification requires that the `Access-Control-Allow-Origin` response header contain exactly one origin (or `*`). Multiple origins in a single rule may produce invalid HTTP headers.

- **ExportProfile shell injection risk**: The `ExportProfile` function generates shell `export` statements by interpolating profile values directly into the output string. If a profile value contains shell metacharacters (e.g., backticks, `$(...)`, semicolons), the exported string could execute arbitrary commands when sourced by the user's shell.

---

## Strengths

### 1. Consistent Error Type Hierarchy

The codebase defines a clear set of error types (`R2NotFoundError`, `R2ValidationError`, `R2AuthError`, `R2BucketError`, etc.) that enable callers to programmatically match on error kinds. Each error type carries contextual fields and implements the `error` interface cleanly. This is well above the average Go project's error handling quality.

### 2. Comprehensive Input Validation

Every public method validates its inputs before making API calls. Bucket names, object keys, zone IDs, namespace IDs, and other parameters are all checked for validity early. Error messages are descriptive and include the invalid value, making debugging straightforward.

### 3. Functional Options Everywhere

The functional options pattern (`WithAccountID`, `WithAPIToken`, `WithEndpoint`, etc.) is used consistently across the client and service constructors. This provides a clean, extensible API that is easy to use correctly and hard to use incorrectly.

### 4. Correct DoctorService Concurrency

The `DoctorService` runs multiple diagnostic probes (DNS, SSL, HTTP, nameservers) concurrently using goroutines and a `sync.WaitGroup`. Results are collected via channels with proper synchronization. There are no data races, no unbounded goroutine spawning, and proper timeout handling.

### 5. Thread-Safe Audit Logger

The audit logging system uses proper mutex protection for concurrent access. Log entries are serialized correctly, and the logger handles file rotation and cleanup without races.

### 6. Clean Service Isolation

Each service (DNS, KV, Workers, SSL, Cache, etc.) is implemented in its own file with its own struct and methods. Services do not reach into each other's internals. This isolation makes it safe to modify one service without risk of breaking another.

### 7. Guardrail System

The codebase includes a guardrail system that validates operations before executing them. This provides a safety net against destructive operations (e.g., deleting a bucket with objects) and gives users clear warnings with actionable guidance.

### 8. Smart Cache Tier Classification

The cache service classifies cache purge operations into tiers (everything, URLs, tags, hosts) and applies appropriate validation and rate limiting for each tier. This prevents accidental full-cache purges and provides clear feedback about the scope of each operation.

---

```json:audit-result
{
  "agent": "agent-2-core-logic",
  "date": "2026-05-19",
  "overall_score": 78,
  "sub_scores": {
    "correctness": 80,
    "error_handling": 82,
    "edge_cases": 68,
    "data_flow": 80,
    "concurrency_safety": 75
  },
  "critical_bugs": [
    "upload.go:253-261 — multipart upload completedParts NOT sorted by PartNumber before CompleteMultipartUpload; S3/R2 requires ascending order; will cause InvalidPartOrder for multi-part files when parts finish out of order",
    "storage.go:158, download.go:59 — all S3 errors (auth, network, server) misclassified as R2NotFoundError in GetObject, HeadObject, Download",
    "internal/config/config.go:114-120 — config file with API tokens briefly world-readable between WriteConfigAs and Chmod"
  ],
  "top_strengths": [
    "Consistent error type hierarchy",
    "Comprehensive input validation",
    "Functional options everywhere",
    "Correct DoctorService concurrency",
    "Thread-safe audit logger",
    "Clean service isolation",
    "Guardrail system",
    "Smart cache tier classification"
  ],
  "top_weaknesses": [
    "download.go:53 range condition fragile",
    "GetBucket/BucketExists O(n)",
    "DNSService.Update no fetch-then-patch",
    "ZoneService.List no accountID scoping",
    "CreateBucket uses client timestamp not server",
    "WAF ListAccessRules only page 1",
    "ListObjects no auto-pagination",
    "KV Put reads entire value into memory",
    "CORS multi-origin invalid per spec",
    "ExportProfile shell injection risk"
  ],
  "recommendations": [
    "Sort completedParts by PartNumber before CompleteMultipartUpload",
    "Inspect S3 error codes and map to correct R2 error types",
    "Create config file with 0600 permissions from the start using os.OpenFile",
    "Add rangeSet bool to download options",
    "Use direct bucket lookup instead of list-and-scan",
    "Implement fetch-then-patch for DNS updates",
    "Add accountID scoping to zone listing",
    "Add auto-pagination to ListObjects and ListAccessRules",
    "Stream large KV values instead of reading into memory",
    "Sanitize ExportProfile output against shell injection"
  ]
}
```
