# Implementation Plan: LifeSoundTrack — save album command, metadata chain, Postgres

**Branch**: `003-save-album-command` | **Date**: 2026-04-24 | **Spec**: [spec.md](spec.md)  
**Input**: Feature **003** + planning notes: **chained** metadata **(Spotify → iTunes → Last.fm → MusicBrainz)** with **per-catalog** **env** **flags** and **circuit** **breaker** **fallthrough**; **PostgreSQL** + **migrations**; **no** **Redis**—disambiguation state in Postgres and/or in-memory for single-process dev only. **2026-04-24**: **equivalent**-**label** **collapse** before disambiguation UI (**FR-009** / **SC-007**).

**Note**: `setup-plan.sh` may require a **git feature branch**; canonical paths are in [`.specify/feature.json`](../../.specify/feature.json) → **`specs/003-save-album-command/`**. `/speckit.tasks` (Phase 2) is **not** created by this command.

## Summary

Add the **`save_album`** / Telegram **`/album`** flow: free-form user text → **fixed-order** **chained** metadata **Spotify** → **iTunes** → **Last.fm** → **MusicBrainz** (per **FR-002**), with **per-catalog** **feature** **flags** and **per-provider** **circuit** **breakers**; **fallthrough** on skip / empty / recoverable failure. **Disambiguation** shows **at** **most** **two** **distinct** **user-visible** **labels** **`ALBUM_TITLE | ARTIST (YEAR)`** **plus** **Other** **only** when **two** **distinct** **labels** **need** a **user** **choice**. **Before** any **disambig** **prompt**, **collapse** **candidates** that **share** the **same** **label** and **keep** the **first** **by** **relevance**; **if** only **one** **distinct** **label** **remains** (including the **all-duplicates** case), **do** **not** **prompt**—**save** (or follow single-match policy) per **FR-009** / spec **amendment** **equivalent** **labels**. Persist **listeners** and **saved_albums** in **PostgreSQL**; store **pending** **disambiguation** in **`disambiguation_sessions`** when a **true** **two**-**choice** (or **Other**) **step** is **required**. **Versioned** **SQL** **migrations** (e.g. `golang-migrate`). **Docker** **Compose** adds **Postgres** only (no Redis).

## Technical Context

| Topic | Choice |
|-------|--------|
| **Language/Version** | **Go 1.22+** (existing `bot/go.mod`) |
| **Runtime** | Long-running `bot/cmd/bot`; Telegram long polling (existing adapter). |
| **Metadata** | **Spotify** Web API (Client Credentials) → **iTunes** Search API → **Last.fm** `album.search` → **MusicBrainz** JSON — **fixed** order; **per** **`LST_METADATA_ENABLE_*`** **flags**; **Spotify** needs `SPOTIFY_CLIENT_ID` / `SPOTIFY_CLIENT_SECRET` when enabled. See [research.md](research.md). |
| **Resilience** | **Circuit** **breaker** per provider; on failure or throttle (policy), **try** **next** **enabled** **ring**; **4xx**/client errors per orchestrator policy. |
| **Database** | **PostgreSQL 15+**; **`pgx/v5` + `database/sql`** or `pgxpool`; no ORM **required** for v1. |
| **Migrations** | **golang-migrate**; files under **`bot/migrations/`** (see [quickstart](quickstart.md)). |
| **Disambiguation storage** | **`disambiguation_sessions`** when **user** must **pick** **among** **distinct** **labels**; **no** **session** **row** **required** for **auto**-**resolved** **same**-**label** **collapse** path. In-memory only for **single-replica** dev. **No** Redis. |
| **Config** | `bot/internal/config`: `DATABASE_URL`, **`LST_METADATA_ENABLE_*`**, `SPOTIFY_*`, `LASTFM_API_KEY`, `MUSICBRAINZ_USER_AGENT`; **no** secrets in `Default` or user chat. |
| **Testing** | `go test ./...`; fakes for **orchestrator**; **table** **cases** for: **all** **flags** **off**; **duplicate** **labels** **→** **no** **disambig** **UI**; **two** **distinct** **labels** **→** **session** + **pick**. |
| **Target platform** | Linux containers for bot + DB; local Windows/macOS dev via Compose. |
| **Performance** | User-visible **&lt; 15 s** for search + pick; metadata budget **&lt; 10 s** per [spec](spec.md). **N/A** for huge throughput in v1. |
| **Constraints** | Provider **ToS**; MusicBrainz **1 rps** + **User-Agent**; **no** secrets in **logs** ([spec](spec.md) **FR-007**). |

## Constitution Check

*GATE: **Pass** (with documented exceptions in Complexity if needed).*

- **I — Code quality**: Migrations in VCS; `golangci-lint` already in `bot/`.
- **II — REST API**: N/A (no new public HTTP product API for this feature).
- **III — Testing**: Fakes for orchestrator + **label**-**dedup** **cases**; DB tests with Docker or in-memory fakes; critical paths in CI.
- **IV — UX**: `/album`, **disambig** **(two** **distinct** **lines** **max)**, and **no** **duplicate** **identical** **labels**; help updated.
- **V — Monitoring**: Log provider + error **class**; optional metrics for breaker opens (future).
- **VI — Logging**: No API keys, no DB URLs, no PII in default logs.
- **VII — Performance**: Stated in Technical Context; **N/A** for “millions of RPS.”
- **VIII — Containerization**: [compose.yaml](../../compose.yaml) with `postgres:15` + volume (**no** Redis). [bot/Dockerfile](../../bot/Dockerfile) as needed; [quickstart](quickstart.md) for env.
- **IX — Docs**: This plan, [data-model](data-model.md), [quickstart](quickstart.md), contracts, root [README](../../README.md) as implementation proceeds.

**Re-check after Phase 1 design**: Contracts include **label** **collapse**; no constitution conflict.

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
│   ├── config/            # + DATABASE_URL, LST_METADATA_ENABLE_*, SPOTIFY_*, LASTFM_*, etc.
│   ├── core/              # save_album, AlbumCandidate, FormatAlbumLabel, collapse-by-label, orchestrator **port**
│   ├── adapter/telegram/  # /album, buttons, disambiguation callbacks, pick by number; never two identical label buttons
│   ├── metadata/          # spotify, itunes, lastfm, musicbrainz, orchestrator, breaker
│   └── store/             # pg listeners + saved_albums + disambiguation_sessions, migrate hook
├── migrations/
│   ├── 000001_init_listeners_saved_albums_disambig.up.sql
│   ├── 000001_init_listeners_saved_albums_disambig.down.sql
```

**Migrations (operator)** — document in [quickstart](quickstart.md):

- Local: `migrate -path bot/migrations -database "$DATABASE_URL" up`
- **CI**: same against ephemeral Postgres, or `docker compose run migrate`

**Make / scripts** (optional in tasks): `scripts/migrate.sh` or `Makefile` with `make migrate-up`.

**Structure decision**: **Hexagon** as today: **domain** in `internal/core` (including **one** **shared** **function** to **format** **`ALBUM_TITLE | ARTIST (YEAR)`** and **deduplicate** **by** that **string** before **offering** **UI**); **adapters** for HTTP metadata, **Postgres**, **Telegram**.

## Key implementation notes (for **/speckit.tasks**)

1. **Label formatting + collapse (FR-009)**: After `Search` returns candidates (per orchestrator policy): map each to the user-visible **`ALBUM_TITLE | ARTIST (YEAR)`** string; order by relevance; **deduplicate by identical label** keeping the **first** (highest-relevance) row per label. If **only one distinct label** remains, **do not** open a disambiguation prompt or `disambiguation_sessions` row—**persist** the kept candidate like a single strong match. If **two distinct labels** remain, cap to top two by relevance and proceed with session + UI + **Other**.
2. **Migrations** run **on** **boot** **only** where **`AUTO_MIGRATE`** = dev pattern; **prod** **init** **job** / **sidecar**; document both.
3. **Circuit** **breaker**: failure counts and cooldown (e.g. 30s) as **const**s or small config struct.
4. **Disambiguation** **session**: create **only** when **UI** **shows** **two** **distinct** **labels** + user must pick; `expires_at` (~15m); cleanup on read or cron.
5. **iTunes** / **Spotify** **mapping**: title, **primary** **artist**, **year** for **label** and **persistence** fields.

## Complexity Tracking

| Item | Why needed | Simpler alternative rejected because |
|------|------------|----------------------------------------|
| **4** **catalogs** + breakers + flags | Spec **FR-002**; **throttle** / **outage** on one **vendor** | One API only breaks when that API fails. |
| **Postgres** + **migrations** | Durable `listeners` + `saved_albums`; constitution **VIII** | In-memory / SQLite: weak for multi-replica. |
| **`disambiguation_sessions` in Postgres** | **FR-009** shared state for **true** two-choice **steps**; no Redis | In-memory only fails horizontal scale. |
| **Label**-**level** **dedup** in **core** | **SC-007** / **duplicate** **button** **bug**; spec **amendment** | Raw-only **UI** without **dedup** **reopens** **regression**. |

## Phase 2 reminder

`tasks.md` is produced by **`/speckit.tasks`**, not by this plan.

## Related links

- [research.md](research.md) — API order, **flags**, breaker, **label** **dedup** note.
- [data-model.md](data-model.md) — DDL; **session** when **2** **distinct** **offers**.
- [contracts/album-command.md](contracts/album-command.md) — domain + Telegram, **single**-**path** after **collapse**.
- [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md) — `Search` port, **cap** **&** **dedup** **rules**.
