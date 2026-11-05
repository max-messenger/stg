---
name: stg-add-repository
description: Add or change an STG repository. Use when implementing PostgreSQL or Redis persistence, a migration, database query, keyspace, or service-owned storage access.
---

## Goal

Implement direct service-owned storage access. `AGENTS.md` defines shared
ownership, Fx, error, testing, and verification conventions.

## Workflow

1. Read `AGENTS.md` and inspect the consuming service, domain errors, local
   `pkg/connections/` APIs, configuration, and comparable repositories.
2. Identify the storage, owning service, data model, domain outcomes, migration,
   and configuration. Ask for confirmation before changing a schema or
   introducing a new persistent data store.
3. Give each table one owning repository package and each new repository one
   owning service. Operations may use joins, aggregates, or several tables for
   that use case. Consumers declare local interfaces and provide concrete
   adapters in their `fx.go`; do not export provider-owned interfaces. Respect
   existing example ownership exceptions described in `AGENTS.md`.
4. For PostgreSQL, declare table/column constants, build parameterized queries
   with `github.com/huandu/go-sqlbuilder`, and use explicitly named pools from
   `stg/pkg/connections/postgres`. Use `read.Call` for reads and `write.Call` for
   mutations with stable operation names. Close rows and check `rows.Err()`.
   Inspect wrapper atomicity and retries before implementing multi-write work.
5. For Redis, acquire the configured pool once in the constructor through
   `stg/pkg/connections/rediscli`. Run operations through `Call`, build keys with
   `kf.FormatKey`, and encode structured values with `stg/pkg/marshaler`.
6. Return only technical errors from `Call` callbacks. Capture missing rows,
   Redis misses, and uniqueness outcomes in flags, return nil from the callback,
   and map flags to domain errors after `Call`. Wrap technical errors with
   operation context; name nested database/client errors `qerr`.
7. Create a migration with `make migrate-new name=<name>` for every schema
   change. Do not alter an applied migration.
8. Test outcomes and the owning service behavior. Use mocks for unit tests and
   existing `tests/integrations_*` harnesses for infrastructure-backed checks.
   Follow generation and verification conventions in `AGENTS.md`.

## Boundaries

- Do not expose raw database or Redis clients to business code.
- Do not introduce shared repositories for new bounded contexts. If data is
  owned by an existing service, consume its capability through a local interface.
- Do not create an HTTP controller unless the feature has an HTTP contract.
- For S3-backed storage, inspect the existing client and file repository;
  PostgreSQL/Redis pool rules do not apply to object storage.
