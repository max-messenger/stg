---
name: stg-invariants-review
description: Review STG changes for correctness and framework invariants. Use when reviewing a diff that changes services, repositories, HTTP routes, Fx wiring, configuration, background work, or template generation.
---

## Review Process

1. Read `AGENTS.md`, inspect the diff and surrounding implementation, then read
   relevant local `pkg/` APIs and existing patterns. Do not infer wrapper
   behavior or dependency ownership from names alone.
2. Report actionable findings ordered by severity. Include file/line references,
   the failure mode, and the smallest safe fix.
3. Do not modify code unless explicitly asked. State when no findings were
   discovered and name remaining verification gaps.

## Required Checks

- Check `AGENTS.md` conventions: layering and ownership, Fx wiring, technical
  errors, secrets, generated files, migrations, and verification. Distinguish
  new regressions from existing example limitations.
- For persistence, check callback outcomes, named read/write pools, wrapper
  usage and actual atomicity/retries, operation names, query resources,
  parameterized `go-sqlbuilder` queries, and Redis key formatting.
- For HTTP, check validation, request contexts, domain-error mapping, route
  registration, and generated API contracts. Exempt vendor webhook adapters
  that directly delegate to their vendor client.
- For asynchronous/MAX changes, check bounded concurrency, shutdown of all
  goroutines, `grace.Service` registration, and one update-stream consumer.
  The current subscriber dispatches to bot; it is not a logging-only scaffold
  and does not require a multiplexer when only one service handles updates.
- For schema changes, check compatibility, locks, backfills, recovery, and that
  applied migrations remain unchanged.
- For framework changes, check affected application consumers and focused
  `pkg/` tests; `make test_unit` covers only `internal/...`.
- For template-generation changes, check module-import replacement, skill names
  and directory paths, and references to components removed by `create.sh`.
  Skill identifiers and directories must remain `stg-*` while application
  imports and paths follow the generated module name.
- Verify changed behavior has focused tests and reported commands match the
  current Makefile and CI rather than stale guideline examples.

## Finding Threshold

Prioritize behavioral defects, consistency and migration risks, security,
Fx failures, observability regressions, lifecycle leaks, public-contract
breakage, and missing tests for changed behavior. Do not report tooling-enforced
formatting preferences unless they obscure a real defect.
