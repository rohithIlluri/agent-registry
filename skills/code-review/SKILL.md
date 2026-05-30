---
name: code-review
description: "Systematic code review: correctness bugs, security (OWASP Top 10), performance, and style with severity-rated inline comments."
version: "1.2.0"
license: MIT
keywords: [code-review, security, bugs, quality, owasp]
category: security
allowed-tools: [Read, Bash]
user-invocable: true
---

# Code Review

Perform a thorough code review of the specified files or the current diff.

## When to invoke

Invoke this skill when the user asks for a code review, asks you to "review" or "audit" code, or when a PR review is requested.

## Process

1. **Read the diff or files** — use `git diff` for staged/unstaged changes, or read specified files directly.
2. **Correctness** — logic errors, off-by-one, race conditions, unhandled errors, incorrect assumptions.
3. **Security** — OWASP Top 10: injection (SQL, command, XSS), broken auth, sensitive data exposure, SSRF, insecure deserialization, path traversal, hardcoded secrets.
4. **Performance** — N+1 queries, unnecessary allocations, blocking I/O in hot paths.
5. **Maintainability** — naming, complexity, dead code, missing tests for critical paths.

## Output format

For each finding:
```
[SEVERITY] Category — Short title
File: path/to/file:line
Issue: concise description of the problem
Fix: specific remediation
```

Severities: **CRITICAL** / **HIGH** / **MEDIUM** / **LOW** / **INFO**

Conclude with a **summary table**:

| Severity | Count |
|----------|-------|
| CRITICAL | N     |
| HIGH     | N     |
| MEDIUM   | N     |
| LOW      | N     |

And a **verdict**: APPROVE / REQUEST CHANGES / COMMENT.

## Constraints

- Do not flag style issues as HIGH or above.
- Do not fabricate line numbers — always read the actual file.
- If no issues found, say so explicitly with a green LGTM.
