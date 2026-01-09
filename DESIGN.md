# Generic CI Gateway – Design Document

## Overview

This document describes the design of a **Generic CI Gateway**: a small Go service that exposes a stable, provider-agnostic CI API while using **Concourse CI** internally.

The gateway:

- Exposes **generic CI concepts** (job, run, status, logs)
- Hides all Concourse-specific details (teams, pipelines, ATC API)
- Uses **local Concourse auth** internally
- Is designed to be **simple now**, but extensible later (tenants, multiple providers)

---

## Goals

### Primary

- Provide a clean, minimal CI API for internal systems
- Avoid leaking Concourse concepts to clients
- Support trigger, status, logs, cancel
- Reliable, headless auth

### Non-goals (v1)

- No UI
- No multi-tenant API surface
- No pipeline authoring via API
- No raw Concourse passthrough

---

## High-Level Architecture

```
Clients
  |
  v
Generic CI API (Go)
  |
  v
Provider Adapter (Concourse)
  |
  v
Concourse ATC API
```

Key principle: **clients only talk to the Generic CI API**.

---

## Public API (v1)

### Core Concepts

#### Job

A **Job** is a stable, runnable unit defined by this system.

```json
{
  "job_id": "job_payments_build_test",
  "project": "payments",
  "display_name": "Build & Test",
  "environment": "prod"
}
```

#### Run

A **Run** is a single execution of a job.

```json
{
  "run_id": "run_01HZ...",
  "job_id": "job_payments_build_test",
  "status": "running",
  "created_at": "2026-01-08T18:22:11Z",
  "started_at": "2026-01-08T18:22:15Z",
  "finished_at": null
}
```

#### Run Status Enum

```
queued | running | succeeded | failed | canceled | errored | unknown
```

---

### API Endpoints

#### List Jobs

```
GET /v1/jobs
```

Response:

```json
{ "jobs": [ Job ] }
```

---

#### Trigger a Run

```
POST /v1/jobs/{job_id}/runs
```

Request:

```json
{
  "parameters": {
    "git_sha": "abc123"
  },
  "idempotency_key": "optional"
}
```

Response:

```json
{ "run": Run }
```

---

#### Get Run

```
GET /v1/runs/{run_id}
```

Response:

```json
{ "run": Run }
```

---

#### Stream Run Events (SSE)

```
GET /v1/runs/{run_id}/events
```

Example:

```
event: status
data: {"status":"running"}

event: log
data: {"stream":"stdout","line":"building..."}
```

---

#### Cancel Run

```
POST /v1/runs/{run_id}/cancel
```

---

## Internal Data Model

### ci\_job

| field          | type   | notes                         |
| -------------- | ------ | ----------------------------- |
| job\_id        | string | stable public ID              |
| tenant\_id     | string | nullable, default = "default" |
| project        | string | logical grouping              |
| display\_name  | string | UI name                       |
| provider\_kind | string | e.g. "concourse"              |
| provider\_ref  | jsonb  | provider binding              |

Example `provider_ref`:

```json
{
  "team": "main",
  "pipeline": "payments",
  "job": "build-test"
}
```

---

### ci\_run

| field              | type      | notes                         |
| ------------------ | --------- | ----------------------------- |
| run\_id            | string    | public ID                     |
| job\_id            | string    | FK ci\_job                    |
| tenant\_id         | string    | nullable, default = "default" |
| status             | string    | generic enum                  |
| provider\_run\_ref | jsonb     | opaque                        |
| created\_at        | timestamp |                               |
| started\_at        | timestamp |                               |
| finished\_at       | timestamp |                               |

---

## Provider Abstraction

```go
type Provider interface {
  Trigger(ctx, jobRef, req) (ProviderRunRef, error)
  GetRun(ctx, runRef) (ProviderRun, error)
  StreamEvents(ctx, runRef, w) error
  Cancel(ctx, runRef) error
}
```

Only the Concourse adapter knows about teams, pipelines, or ATC endpoints.

---

## Concourse Adapter (Internal)

### Auth

- Uses **local Concourse user**
- Fetches token via: `GET /api/v1/teams/{team}/auth/token`
- Caches Bearer token
- Refreshes on 401

### Mappings

| Generic | Concourse          |
| ------- | ------------------ |
| Job     | pipeline + job     |
| Run     | build              |
| Events  | build event stream |
| Cancel  | abort build        |

---

## Tenancy Strategy

- **No tenant exposed in v1 API**
- Internal `tenant_id` always resolved to `"default"`
- Schema and code are tenant-ready
- Future options:
  - path-based tenant
  - auth-claim-based tenant

---

## Security

- Gateway enforces caller auth (JWT / mTLS / internal auth)
- Concourse credentials never exposed
- Strict provider allowlist
- Audit logs:
  - who triggered what
  - when
  - result

---

## What’s Next (Future)

- Add tenant support
- Add more providers (GitHub Actions, Buildkite)
- Pipeline config management
- Artifacts API
- UI on top of Generic CI API

---

## Summary

This design:

- Keeps the API **clean and stable**
- Avoids CI vendor lock-in
- Is easy to implement incrementally
- Matches your preference for simplicity now, extensibility later

This is a solid foundation for a long-lived CI abstraction layer.


