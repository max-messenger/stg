# STG Development Guide

## Skill Routing

- Use `stg-add-http-endpoint` for a handler, route, DTO, controller, or Swagger operation.
- Use `stg-add-repository` for PostgreSQL, Redis, migrations, and direct service-owned storage access.
- Use `stg-implement-feature` for a feature that spans layers or whose ownership is not yet clear.
- Use `stg-invariants-review` for a read-only review of a change.
- Repository skills live under `.agents/skills/<skill-name>/SKILL.md`.

## Repository Scope

- This repository contains the framework in `pkg/` and a working example application in `internal/app`. Inspect the relevant implementation before treating an example as a reference pattern.
- Framework packages are local: import `stg/pkg/...`, including `stg/pkg/connections/postgres`, `stg/pkg/connections/rediscli`, `stg/pkg/http/client`, and `stg/pkg/marshaler`. Do not introduce an external GIBS dependency for these capabilities.
- Application behavior belongs in `internal/app`; change `pkg/` only when the task requires a framework change, and verify its affected consumers.
- The `webhookctrl` package is a vendor webhook adapter: it exposes the MAX client's handler directly. Do not apply normal business-controller DTO, validation, response-helper, or Swagger requirements to that adapter.
- Start documentation discovery at `guidelines/README.md` and relevant package READMEs. When an example or guideline disagrees with the current code, Makefile, or CI, verify against those implementations.
- `create.sh` copies the repository, removes example components, and replaces the module name `stg` in text files. Module imports and application-specific paths follow the generated name; skill identifiers and directories always remain `stg-*`. Keep skill references compatible with that distinction. Inspect the generated tree before relying on example packages, `testkit/`, `tests/`, or deployment files there.

## Development Conventions

### Tests

- Unit tests use mocks and do not depend on real databases, HTTP services, or other external infrastructure.
- Name new unit tests `TestUnit...`, integration tests `TestIntegration...`, and E2E tests `TestE2E...`. The current `make test_unit` runs all tests under `internal/...`, without a name filter.
- Prefer table-driven tests for cases that exercise the same behavior with different inputs or expected results.
- Declare dependency interfaces in the package that consumes them, add a `//go:generate go tool mockgen ...` directive, and generate mocks with `make generate` or a focused `go generate` command.
- Put tests in the external `<package>_test` package and create mocks with `gomock.NewController(t)`.
- Use `go.uber.org/mock/gomock` for expectations and `testify/require` for assertions.
- For bot behavior, use `testkit/bot` update builders and outbound-call assertions where applicable; see `guidelines/bot-testing.md`.
- Integration suites live under `tests/integrations_*`; E2E tests live under `tests/e2e`. These require their configured infrastructure and, for E2E, a running application.

### Verification

- Run `make test_unit` after changing application code. It runs `./internal/...` with coverage and the race detector.
- For framework or test-helper changes, also run focused tests for the affected `pkg/` or `testkit/` packages, for example `go test -race ./pkg/<package>/...`; `make test_unit` does not cover those packages.
- Run `make generate` after changing mock interfaces or Swagger annotations. Review the generated-file diff; do not manually edit generated mocks or Swagger artifacts.
- Run read-only `golangci-lint run` before a commit or merge request. The current `make lint` runs `golangci-lint run --fix`; use it only when intentionally applying automatic fixes. There is no `lint-fix` target.
- Run focused race tests after changing goroutines, channels, or lifecycle code.
- Use `make test_integration` and `make test_e2e` when the change requires those checks and the infrastructure is available.
- `make test_cover` reports unit coverage only. Derive commands from `Makefile`, rather than assuming the coverage examples in guidelines match them.

### Dependency Injection

- The application uses `go.uber.org/fx`. Every service, client, repository, and controller exposes a package-level `Module` built with a named `fx.Module` and registers constructors with `fx.Provide`.
- Every cross-service dependency is a consumer-owned interface: the consumer declares the local interface and its `fx.go` adapts the concrete provider to it. Provider services do not export interfaces for consumers.
- Constructors receive only the concrete dependencies or local interfaces they need. Business code depends on local interfaces, not concrete implementations from other layers.
- For new application constructors that need a logger, put `log *zap.Logger` first and configuration immediately after it; other dependencies follow. Match the existing configuration type rather than inventing one.
- Fx-provided dependencies and constructor configuration are required. Do not add nil guards, no-op fallbacks, or silent degradation for them; let Fx reject an incomplete graph. Define configuration defaults in the configuration provider, not again in constructors or business code.
- The consuming package owns its adapter in `fx.go`. Receive concrete dependencies through `fx.In`, expose local interfaces through `fx.Out`, and provide the adapter alongside the constructor. Adapter bodies are plain field assignments; use structurally compatible interfaces instead of conversion wrappers.
- Controller modules compose at the application level; other top-level modules reach dependencies through their layer modules.
- Name module loggers with `fx.Decorate` where the layer already does so. Follow the existing service, lifecycle, and configured-client module patterns.

### Layering And Errors

- Preserve the dependency flow `router -> services -> repository/clients/infra`. Keep shared domain models and business errors in `internal/app/domain`.
- For new bounded contexts, give each repository one owning service. Cross-context data flows through consumer-owned service interfaces. The existing todo example has both bot and todolist repository consumers; do not assume exclusive ownership already holds or refactor it as a side effect of an unrelated task.
- Use focused service packages with methods grouped by responsibility. A long-running loop belongs to the owning service lifecycle rather than an unrelated god-service.
- Declare expected business failures as domain errors and check them with `errors.Is`.
- Wrap technical errors with operation-specific context using `fmt.Errorf("operation: %w", err)`; do not expose them to HTTP clients.

### HTTP Controllers

- Create a controller package under `internal/app/router/<feature>ctrl` with a local service interface, controller struct, constructor, `Register(chi.Router)` method, and package `Module`.
- Generate the service interface mock in the consuming controller package. Keep handlers in focused files such as `create_<resource>.go`.
- Register routes in `Register`, pass request contexts to services, validate path/query/body input before service calls, and map domain errors to HTTP responses.
- Add Swagger annotations to every business handler and expose the controller through `fx.Out` as `router.Controller` in the `controller` group.
- Use `router.WriteSuccess`, `router.WriteError`, or `router.WriteResponse` as appropriate for the existing response contract.

### Repositories

- Repository modules provide concrete repositories; services consume local interfaces through adapters in the consuming service's `fx.go`.
- Inside `read.Call`, `write.Call`, or Redis `Call` callbacks, name the error from the database or client call `qerr`, then translate or wrap it.
- Return only technical errors from a `Call` callback. Capture business outcomes (missing row, Redis miss, unique violation) in flags declared before `Call`, return nil from the callback, and translate them to domain errors after `Call` returns. Callback errors mark operations as failed in tracing and metrics.

#### PostgreSQL Repositories

- Each table has one owning repository package. An operation may use joins, aggregates, and several tables when its owning use case requires them.
- Declare table and column names as package-level constants.
- Obtain explicitly named pools from `*postgres.Pool`. Use `read.Call` for reads and `write.Call` for mutations; the example uses `stg-sel` and `stg-upd`.
- Build parameterized SQL with `github.com/huandu/go-sqlbuilder`, following existing repositories and the configured PostgreSQL execution mode.
- Pass `context.Context` into database calls, wrap errors with operation context, close rows, check `rows.Err()`, and use stable operation names for observability.
- Assess atomicity and retry behavior for multi-write operations by inspecting the actual PostgreSQL wrapper implementation, not just a method name or guideline example.

#### Redis Repositories

- Use `stg/pkg/connections/rediscli`; connection settings live under `redis:` as named pools.
- Obtain the configured pool once in the constructor with `GetPool`; use the pool name from the application's configuration.
- Run every operation inside `Call(ctx, "<operation>", func(ctx context.Context, ucl redis.UniversalClient, kf rediscli.KeyFormatter) error { ... })`, with a stable operation name.
- Build keys with `kf.FormatKey(...)` so the configured prefix is applied. Encode structured values with `marshaler`, treat `redis.Nil` as a business miss, and wrap technical errors.

### HTTP Clients And Infrastructure

- Put clients to external systems in dedicated client packages with a package `Module` and configuration provider. Service-facing interfaces belong to consumers, not client packages.
- Use `stg/pkg/http/client` for tracing, metrics, logging, TLS defaults, and timeouts; inspect `pkg/http/client/client.go` for supported options.
- Do not log credentials, bearer tokens, secret request bodies, or encrypted credential blobs.
- Local platform capabilities that talk to no external system belong in infrastructure packages, with a `Module` and configuration provider. Add a layer aggregator only when needed.

### MAX Bot Update Dispatch

- `internal/app/subscriber` is the sole reader of the MAX update stream and currently delegates to `services/bot.ProcessUpdate`. Services must not create competing readers or webhook subscriptions.
- Preserve the subscriber's consumer-owned interfaces. Introduce a dedicated multiplexer only when multiple owning services need update routing.
- The existing subscriber already processes updates; it is not the logging-only template placeholder. Its per-update goroutines and non-waiting `Stop` are not a lifecycle reference: changes to dispatch must provide bounded concurrency, propagate shutdown, and wait for workers.

### JSON

- Use `stg/pkg/marshaler` for application JSON decoding and encoding; do not introduce alternative JSON libraries in application code.
- Decode request bodies with `marshaler.LoadJSONFromReader` and write JSON responses through the router helpers. Framework code may use standard JSON types where required by its API.

### Migrations

- Add a new SQL migration for every schema change with `make migrate-new name=<name>`.
- Do not modify migrations that have already been applied.
- Assess locking, compatibility with old/new application versions, backfills, and rollback or recovery before applying a schema change.

### Background Services

- Use `stg/pkg/bgtasker` for short, bounded background work. Add a named `background.tasks.<name>` section to the relevant configs and obtain `pool.Get(<name>)` while building the Fx graph, before the pool lifecycle starts.
- A consumer declares a local interface with `Run(context.Context, bgtasker.TaskFunc)` and receives the runner through its `fx.go`. Task errors are logged; tasks are not retried and their errors are not returned to the caller.
- Choose `runner_type: block` when backpressure is acceptable; use `drop` only for intentionally lossy work. Do not enqueue mutable request data. The runner preserves context values without cancellation and bounds work with `async_worker_timeout`.
- Long-running pollers and consumers implement `grace.Service` and join the Fx `grace` group. Do not manage process signals or an HTTP server directly in business code.
- Tie loop and child-worker contexts to the shutdown signal. The `Start` context is a startup deadline, not the service lifetime; derive the loop from `context.Background()` and shutdown cancellation.
- `Stop` propagates shutdown and waits for all goroutines started by the service.
