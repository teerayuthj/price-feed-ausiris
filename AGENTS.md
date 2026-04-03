# AGENTS.md

Instructions for AI coding assistants working in this repository.

## Project Overview

Ausiris Bullion Price Feed - realtime market data and gold price delivery service for the Ausiris bullion platform. This repo owns WebSocket broadcasting, upstream price ingestion, scheduler-driven refresh, and realtime payload delivery.

**Tech Stack:** Go, WebSocket, SFTP, Redis, Docker, Make

## Multi-Repo Architecture

This service is one of three repos in the same product. In new Codex sessions, assume these repos work together unless the user explicitly limits scope.

| | Frontend | Backend API | Price Feed (this repo) |
|---|---|---|---|
| **Current location** | `/Users/teerayutht/WorkSpace/ausiris-bullion-web` | `/Users/teerayutht/WorkSpace/ausiris-bullion-api` | `/Users/teerayutht/WorkSpace/ausiris-bullion-price-feed` |
| **Repo name** | `ausiris-bullion-web` | `ausiris-bullion-api` | `ausiris-bullion-price-feed` |
| **Role** | UI, local state, rendering live prices | REST API, business logic, persistence, auth | WebSocket broadcast, market price ingestion, realtime delivery |
| **Rule** | Never duplicate backend business logic | Single source of truth for pricing, inventory, orders | Single source of truth for realtime price events |

### Repo Responsibility Map

- **Frontend (`ausiris-bullion-web`)**: pages, components, hooks, WebSocket consumers, live price display
- **Backend API (`ausiris-bullion-api`)**: routes, validation schema, business rules, persistence, REST responses
- **Price Feed (`ausiris-bullion-price-feed`)**: WebSocket channels, market data ingestion, broadcast logic, reconnect behavior, realtime payload format

### Cross-Repo Rules For New Sessions

- If the user asks to change `web`, `api`, and `price-feed`, treat it as one coordinated task.
- Keep realtime delivery logic in this repo; do not move it into `web` or `api`.
- If websocket payloads change, update the frontend WebSocket consumer and any API/docs that depend on the payload shape.
- If a change only affects persistence or business rules, keep it in `ausiris-bullion-api`.

### Best Session Setup

For reliable multi-repo edits in one prompt, open Codex from the shared workspace root:

```bash
cd /Users/teerayutht/WorkSpace
```

### Session Kickoff Template

```text
This task may touch 3 repos in the same product:
- /Users/teerayutht/WorkSpace/ausiris-bullion-web (web)
- /Users/teerayutht/WorkSpace/ausiris-bullion-api (api)
- /Users/teerayutht/WorkSpace/ausiris-bullion-price-feed (price-feed)

Please inspect the relevant repos first and implement the change in the correct layer.
Rules:
- web handles UI only
- api owns business logic and schema
- price-feed owns realtime gold price events
```

## Git Workflow

- Long-lived branches for this product are `main` and `uat` only.
- `main` is the production-ready branch. Production deploys must come from `main`.
- `uat` is the integration branch. UAT deploys must come from `uat`.
- Start new work from `uat` using short-lived branches named `feature/<topic>`.
- Start urgent production fixes from `main` using `hotfix/<topic>`.
- Merge `feature/*` into `uat` only via PR. Promote to production via PR from `uat` to `main`.
- Treat `uat` and `main` as protected branches: no direct pushes, require PRs, keep branches up to date, and require passing checks plus at least one review.
- After merging a `hotfix/*` branch into `main`, back-merge the same fix from `main` to `uat` immediately.
- Do not create long-lived branches for `local`, `development`, `test`, or `deploy`. Environment differences belong in env files, Compose files, CI, and deploy scripts.
- `wholesale/v2`, `wholesale/v3`, and similar legacy long-lived branches are deprecated and must not be used as active merge targets.
- For coordinated multi-repo work, use the same branch suffix in all touched repos, for example `feature/payment-batch-v2`, and link the related PRs together.

## Core Architecture

### Project Structure

```text
cmd/                 # Application entrypoints
internal/config/     # Environment and config loading
internal/sftp/       # SFTP download and validation logic
internal/parser/     # Price and market data parsing
internal/websocket/  # Hub, clients, and websocket handlers
internal/scheduler/  # Periodic refresh and broadcast trigger
internal/redis/      # Optional scaling and pub/sub
internal/api/        # REST handlers and routes
pkg/models/          # Shared output models
web/static/          # Lightweight static interface
deployments/         # Docker, nginx, systemd
raw-data/            # Downloaded source files
```

### Data Flow

```text
Upstream source/SFTP -> parser -> local data files -> websocket hub -> connected clients
```

## Essential Commands

```bash
make build
make run
make run-download
make run-continuous
make test
make lint
make fmt
make local-dev
make local-dev-down
```

## Critical Rules

- Keep websocket payloads backward compatible unless the user explicitly requests a breaking change.
- If payload shape changes, document the new fields and verify the frontend consumer path.
- Do not hard-code frontend business rules here.
- Redis is optional; the service must still run without Redis unless the task explicitly requires it.
- Prefer small, focused changes in `internal/` packages over large edits in `cmd/`.
