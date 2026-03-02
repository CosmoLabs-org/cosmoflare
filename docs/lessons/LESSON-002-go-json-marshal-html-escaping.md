# LESSON-002: Go json.Marshal Escapes HTML Characters

**Category**: Go
**Session**: 014 (2026-03-02)
**Occurrences**: 1

## Lesson

Go's `json.Marshal` converts `<`, `>`, and `&` to Unicode escape sequences (`\u003c`, `\u003e`, `\u0026`) by default. This breaks pattern matching on raw JSON strings.

## Context

Discovered while testing input validation/injection detection. Tests checking for `<script>` in JSON output failed because `json.Marshal` had already escaped the angle brackets to `\u003c` and `\u003e`.

## Pattern

```go
// json.Marshal output: {"input":"\u003cscript\u003ealert('xss')\u003c/script\u003e"}
// NOT:                 {"input":"<script>alert('xss')</script>"}

// For injection detection, check the DECODED value, not raw JSON
var result map[string]string
json.Unmarshal(body, &result)
if strings.Contains(result["input"], "<script>") { // works correctly
```

## Key Insight

When testing for dangerous content in JSON responses, always unmarshal first and check the decoded string values. Raw JSON pattern matching will miss HTML-escaped content.
