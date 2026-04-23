# Implementation Plan: LifeSoundTrack — save album command, metadata chain, Postgres

**Branch**: `003-save-album-command` | **Date**: 2026-04-23 | **Spec**: [spec.md](spec.md)  
**Input**: Feature **003** + user planning notes: 2–3 free-tier music APIs, **circuit breaker** fallback, **PostgreSQL**, **migrations** for schema. *(Clarification 2026-04-23: **no Redis**—disambiguation state in Postgres and/or in-memory for single-process dev only.)*

**Note**: `setup-plan.sh` may require a **git feature branch**; canonical paths are in [`.specify/feature.json`](../../.specify/feature.json) → **`specs/003-save-album-command/`**. `/speckit.tasks` (Phase 2) is **not** created by this command.

## Summary

Add the **`save_album`** / Telegram **`/album`** flow: free-form user text → **chained** metadata **MusicBrainz → Last.fm → iTunes** (order fixed in v1) with a **per-provider circuit breaker**; **up to 3** candidates; **user pick** (buttons or numbers). Persist **listeners** and **saved_albums** in **PostgreSQL**; store **pending disambiguation** in **`disambiguation_sessions`** (Postgres) for production / multi-replica, or **in-memory** for local single-process dev. Manage schema with **versioned SQL migrations** (e.g. `golang-migrate`). **Docker Compose** adds **Postgres** only (no Redis).

## Technical Context

| Topic | Choice |
|-------|--------|
| **Language/Version** | **Go 1.22+** (existing `bot/go.mod`) |
| **Runtime** | Long-running `bot/cmd/bot`; Telegram long polling (existing adapter). |
| **Metadata** | **MusicBrainz** (keyless, 1 rps) → **Last.fm** `album.search` (needs `LASTFM_API_KEY`) → **iTunes Search API** (keyless, terms-of-use). See [research.md](research.md). |
| **Resilience** | **Circuit breaker** per provider (`gobreaker` or similar); on failure, next provider. |
| **Database** | **PostgreSQL 15+**; driver **`pgx/v5` + `database/sql`** or pure `pgxpool`; no ORM **required** for v1. |
| **Migrations** | **golang-migrate**; files under **`bot/migrations/`** (or `migrations/` at repo root—pick one, document in [quickstart](quickstart.md)). |
| **Disambiguation storage** | **`disambiguation_sessions`** table in Postgres (see [data-model](data-model.md)); optional in-memory cache for **single-replica** dev only. **No** Redis. |
| **Config** | Extend `bot/internal/config` with `DATABASE_URL`, `LASTFM_API_KEY` (optional if chain stops earlier); **no** keys in `Default`. |
| **Testing** | `go test ./...`; fakes for **orchestrator**; integration tests with **testcontainers** (optional) or `docker compose -f` minimal Postgres. |
| **Target platform** | Linux containers for bot + DB; local Windows/macOS dev via Compose. |
| **Performance** | User-visible target **&lt; 15 s** for a full search+pick; metadata **&lt; 10 s** budget per [spec](spec.md). **N/A** for huge throughput in v1. |
| **Constraints** | Comply with provider ToS; respect MusicBrainz **User-Agent** and rate; **no** secrets in logs. |

## Constitution Check

*GATE: **Pass** (with documented exceptions in Complexity if needed).*

- **I — Code quality**: Migrations in VCS; `golangci-lint` already in `bot/`.
- **II — REST API**: N/A (no new public HTTP product API for this feature).
- **III — Testing**: Fakes for orchestrator; DB tests with Docker or in-memory fakes; critical paths in CI.
- **IV — UX**: New `/album` and **disambig** copy in **core**; help updated.
- **V — Monitoring**: Log provider + error **class**; optional metrics for breaker opens (future).
- **VI — Logging**: No API keys, no DB URLs, no PII in default logs.
- **VII — Performance**: Stated in Technical Context; **N/A** for “millions of RPS.”
- **VIII — Containerization**: **Update** [compose.yaml](../../compose.yaml) (repo root) with `postgres:15` + volume (**no** Redis service). **Update** [bot/Dockerfile](../../bot/Dockerfile) only as needed; document env in [quickstart](quickstart.md).
- **IX — Docs**: This plan, [data-model](data-model.md), [quickstart](quickstart.md), and root [README](../../README.md) (small subsection in implementation phase).

**Re-check after Phase 1 design**: Migrations and contracts are listed; no constitution conflict.

## Project Structure

### Documentation (this feature)

```text
specs/003-save-album-command/
├── plan.md                 # this file
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── album-command.md
    └── metadata-orchestrator.md
```

### Source (repository) — **touch points** (intended)

```text
bot/
├── go.mod
├── cmd/bot/main.go
├── internal/
│   ├── config/            # + DATABASE_URL, API keys
│   ├── core/              # + save_album command, AlbumCandidate, orchestrator **port**
│   ├── adapter/telegram/  # + /album, buttons, disambiguation callbacks, pick by number
│   ├── metadata/          # NEW: subpackages musicbrainz, lastfm, itunes, orchestrator, breaker
│   └── store/ or repo/   # NEW: pg listeners + saved_albums + disambiguation_sessions, migrate hook
├── migrations/
│   ├── 000001_init_listeners_saved_albums_disambig.up.sql
│   ├── 000001_init_listeners_saved_albums_disambig.down.sql
```

**Migrations (operator)** — document in [quickstart](quickstart.md):

- Local: `migrate -path bot/migrations -database "$DATABASE_URL" up`
- **CI**: same against ephemeral Postgres, or `docker compose run migrate`

**Make / scripts** (optional in tasks): `scripts/migrate.sh` or `Makefile` with `make migrate-up`.

**Structure decision**: **Hexagon** already used: **new** `internal/core` **domain** types + **orchestrator** **interface**; **adapters** for HTTP (metadata), **Postgres** (persistence), **Telegram** (UI). `godotenv` and config stay as in 002.

## Key implementation notes (for **/speckit.tasks**)

1. **Migrations** run **before** or **on** first boot for dev only—**prod** should run `migrate` as an **init container** or job; document both.
2. **Circuit breaker**: failure counts and cooldown (e.g. 30s) as **const**s or small config struct.
3. **Disambiguation**: `disambiguation_sessions` rows with `expires_at` (~15m); cleanup on read or cron; in-memory fallback only for **single-process** dev.
4. **iTunes** response mapping: `collectionName`, `artistName`, `releaseDate` → year.

## Complexity Tracking

| Item | Why needed | Simpler alternative rejected because |
|------|------------|----------------------------------------|
| **3 providers** + breakers | Free tiers throttle or fail; spec requires not blocking on one vendor | One API only breaks during outages. |
| **Postgres** + **migrations** | Durable `listeners` + `saved_albums` + auditability; constitution **VIII** for real apps | In-memory / SQLite: weak for future multi-replica. |
| **`disambiguation_sessions` in Postgres** | **FR-009** needs shared state across replicas; no Redis per product decision | In-memory only fails for horizontal scale. |

## Phase 2 reminder

`tasks.md` is produced by **`/speckit.tasks`**, not by this plan.

## Related links

- [research.md](research.md) — API comparison, breaker, migration tool.
- [data-model.md](data-model.md) — DDL and constraints.
- [contracts/album-command.md](contracts/album-command.md) — domain + Telegram.
- [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md) — `Search` port and `AlbumCandidate`.
