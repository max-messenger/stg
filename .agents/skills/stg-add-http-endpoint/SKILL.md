---
name: stg-add-http-endpoint
description: Add or change an STG HTTP endpoint. Use when implementing a handler, route, request or response DTO, controller, or Swagger operation.
---

## Goal

Deliver the smallest complete HTTP vertical slice. `AGENTS.md` defines shared
layering, Fx, test, error, and verification conventions.

## Workflow

1. Read `AGENTS.md`, inspect the router, existing controllers, domain package,
   and owning service before choosing paths or types. Reuse established
   package names and response helpers; framework APIs live in local `pkg/`.
2. State the route and HTTP contract, validation, service method, domain errors,
   Swagger impact, and whether another skill is required. Ask for confirmation
   before an incompatible public-contract or schema change.
3. Keep HTTP concerns in `internal/app/router/<feature>ctrl`. The controller
   owns its local service interface, exposes itself as `router.Controller` in
   the `controller` Fx group, and registers routes in `Register`.
4. Validate path, query, and body input before service calls. Pass the request
   context. Decode bodies with `stg/pkg/marshaler` and use `router.WriteSuccess`,
   `router.WriteError`, or `router.WriteResponse` for the existing contract.
5. Keep business rules and expected business errors in services/domain. Map
   domain errors with `errors.Is`; keep contextual technical errors internal.
6. Add Swagger annotations for every business handler and update DTO
   annotations. Never edit generated Swagger artifacts.
7. Add focused tests for validation, domain-error mapping, and successful
   responses. Follow test and generation conventions in `AGENTS.md`.
8. Run applicable verification from `AGENTS.md`. Report the route, contract
   changes, commands run, and anything not verified.

## Boundaries

- Reuse existing service capabilities; do not add a repository merely because
  an endpoint was requested.
- Do not use `encoding/json`, call repositories from controllers, or bypass
  consumer-owned service interfaces with concrete cross-layer dependencies.
- For persistence or a new service capability, apply `stg-add-repository` and
  `stg-implement-feature` instead of duplicating their decisions here.
- Vendor webhook adapters may delegate directly to vendor handlers; keep
  authentication, validation, and contract ownership with that integration.
