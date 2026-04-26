# Implementation Plan: LifeSoundtrack — list saved albums (`/list`)

**Branch**: `006-list-saved-albums` | **Date**: 2026-04-26 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `specs/006-list-saved-albums/spec.md`

**Note**: Filled by `/speckit.plan`. Design artifacts: [research.md](./research.md), [data-model.md](./data-model.md), [contracts/list-command.md](./contracts/list-command.md), [quickstart.md](./quickstart.md). Implementation tasks: [tasks.md](./tasks.md) (separate `/speckit.tasks` flow).

## Summary

Add **`/list`** so a **listener** sees their **saved albums** only, **5 per page**, **newest first** (`created_at DESC, id DESC`). Optional **`/list <artist>`** applies **FR-004** normalization and **case-insensitive substring** match on **`primary_artist`** only (**FR-003**). **Multi-page** results use **`album_list_sessions`** (created only when **`total_count > 5`**, i.e. **`total_pages > 1`**) plus Telegram **`lpl:<session_id>:<page>`** callbacks and text **`/list next`** / **`/list back`** (**FR-006**). **Empty library** and **no filter matches** get distinct copy; **FR-010** display truncation matches existing album label conventions. **Private-message routing** must preserve **`/album`** numeric disambig vs **`/list`** (see Clarifications **C1**).

## Technical Context

**Language/Version**: Go **1.25** (`bot/go.mod`)  
**Primary Dependencies**: [go-telegram/bot](https://github.com/go-telegram/bot), **pgx/v5**, **goose**, **google/uuid**  
**Storage**: **PostgreSQL** — existing `listeners`, `saved_albums`; new **`album_list_sessions`** ([data-model.md](./data-model.md))  
**Testing**: `go test` from `bot/` (`cd bot && go test ./...`); table tests for parse/normalize; store + `run_test.go` routing per [research.md](./research.md) §9  
**Target Platform**: Linux (or any host) running the **Telegram** bot long-lived process  
**Project Type**: **Messaging bot** (Telegram adapter + domain core)  
**Performance Goals**: **N/A** with justification — interactive chat latency; spec targets “ordinary” expectations for **hundreds** of rows per listener ([spec.md](./spec.md) NFR)  
**Constraints**: Telegram **`callback_data` ≤ 64 bytes** → session id in callback, not raw filter ([research.md](./research.md) §1); **no** `LIKE` with user wildcards — use **`strpos(lower(...), needle)`** (§2)  
**Scale/Scope**: Single feature command; listener-scoped queries only (**SC-003**)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Confirm alignment with [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md):

- **Code quality**: Changes scoped to `bot/`; follow repo **gofmt** / lint in CI (**I**).
- **REST API**: **Not in scope** for this feature (**II** latent).
- **Testing**: Unit/table tests for normalization and parsing; store + adapter tests for pagination, isolation, and handler order (**III**); manual steps in [quickstart.md](./quickstart.md).
- **User experience**: **`/list`**, **`/list next`**, **`/list back`**, help/copy aligned with existing commands (**IV**); root README + quickstart for operator paging docs (Clarifications **I2**).
- **Monitoring**: No new SLA; optional usage/error logs without leaking library content (**V**, **VI**).
- **Logging**: No secrets; avoid logging full list payloads (**VI**).
- **Performance**: Stated **N/A** above (**VII**).
- **Containerization**: Existing **`bot`** Dockerfile / Compose unchanged unless migrations/env docs require a note (**VIII** — no new runnable).
- **Documentation accuracy**: This plan and [spec.md](./spec.md) paths match `specs/006-list-saved-albums/`; **tasks.md** references same artifacts (**IX**).

**Post-design re-check**: Phase 0–1 artifacts present; no unresolved **NEEDS CLARIFICATION** in Technical Context; gates **PASS**.

## Project Structure

### Documentation (this feature)

```text
specs/006-list-saved-albums/
├── plan.md              # This file
├── research.md          # Phase 0
├── data-model.md        # Phase 1
├── quickstart.md        # Phase 1
├── contracts/
│   └── list-command.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit.tasks (not produced by this command)
```

### Source Code (repository root)

```text
bot/
├── cmd/bot/
│   └── main.go
├── migrations/
│   ├── 00001_init_listeners_saved_albums_disambig.sql
│   ├── 00002_daily_recommendations.sql
│   └── 00003_album_list_sessions.sql   # new (per data-model)
├── internal/
│   ├── core/           # LibraryService, parse_list.go, list_saved.go, copy.go
│   ├── adapter/telegram/
│   │   ├── run.go      # handlers, lpl: callbacks, EditMessageText
│   │   └── run_test.go
│   └── store/          # saved_albums list/count, album_list_sessions
└── go.mod
```

**Structure Decision**: Single **`bot`** module; **domain** in `internal/core`, **persistence** in `internal/store`, **Telegram** in `internal/adapter/telegram` — consistent with existing save-album and daily recommendation features.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations requiring exceptions for this feature.
