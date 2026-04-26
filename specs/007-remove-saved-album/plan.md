# Implementation Plan: LifeSoundtrack — remove saved album (`/remove`)

**Branch**: `007-remove-saved-album` | **Date**: 2026-04-26 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `specs/007-remove-saved-album/spec.md`

**Note**: Filled by `/speckit.plan`. Design artifacts: [research.md](./research.md), [data-model.md](./data-model.md), [contracts/remove-command.md](./contracts/remove-command.md), [quickstart.md](./quickstart.md). Implementation tasks: [tasks.md](./tasks.md) (separate `/speckit.tasks` flow; refresh after this plan if needed).

## Summary

Deliver **`/remove <name>`** so a **listener** can **delete** one of their **saved** **album** rows. Matching is **title-only** (**FR-003**): **(1)** **exact** **normalized** **equality**; **(2)** if none, **partial** **substring** on normalized **title**; **0** **→** not found; **1** **exact** **→** **delete**; **≥2** **exact** **or** **1–3** **partials** **(no** **exact)** **→** **disambig** via **`remove_saved`** JSON in existing **`disambiguation_sessions`**. **>3** **partials** **(no** **exact)** **→** **narrow** **copy** only. On **Telegram**, **disambig** **MUST** show **inline** **keyboard** **buttons** **(one** **per** **candidate)** so **picking** is **not** **text-only**; **session-bound** **numeric** **text** **(1**–**N**)** **remains** a **required** **alternate** (**SC-003** / [spec.md](./spec.md) **FR-006**). **Domain** logic lives in `bot/internal/core`; **buttons** and **`callback_data`** in `bot/internal/adapter/telegram` (same spirit as **`lpl:`** list pagination, **`apick:`** **album**). **Help** and **stale**-**session** behavior align with [research.md](./research.md) (single-save **save** must clear **prior** disambig — see **§10**).

## Technical Context

**Language/Version**: Go **1.25** (`bot/go.mod`)  
**Primary Dependencies**: [go-telegram/bot](https://github.com/go-telegram/bot), **pgx/v5**, **goose**, **google/uuid**  
**Storage**: **PostgreSQL** — existing `listeners`, `saved_albums`, `disambiguation_sessions` (no new tables; see [data-model.md](./data-model.md))  
**Testing**: `cd bot && go test ./...`; table tests for parse/normalize/match; store tests for `DELETE` scoping; **core** + **adapter** tests for **routing** and **callback** **parse** (see [research.md](./research.md) **§9**)  
**Target Platform**: Long-lived **Telegram** bot process (Linux or any host)  
**Project Type**: **Messaging** bot (Telegram adapter + domain core)  
**Performance Goals**: **N/A** (spec: interactive chat, sub-second to few seconds) — per constitution **VII** with justification for low-risk command work  
**Constraints**: **`callback_data` ≤ 64 bytes** on Telegram; prefix + **UUID** **session** `id` + **index** must fit (see [research.md](./research.md) **§10**); **inline** **button** **text** **≤** **64** **runes**/chars (Telegram) — **truncate** labels like **/list** **FR-010** style if needed; **in-memory** list of **all** **saved** **rows** for a listener for **match** in **Go** (acceptable scale per **006** NFR)  
**Scale/Scope**: One command + **remove** **pick** **transport**; listener-scoped **only**

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Confirm alignment with [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md):

- **Code quality (I)**: Scoped changes under `bot/`; existing **gofmt** / **lint** / CI; no unrelated refactors
- **REST API (II)**: **Not in scope** (latent)
- **Testing (III)**: Unit + store + adapter tests per [research.md](./research.md) **§9**; **SC-003** **Telegram** **acceptance** in [quickstart.md](./quickstart.md)
- **User experience (IV)**: **FR-006** **inline** **buttons** on **Telegram**; **text** **pick** **alternate**; help **FR-008**; copy tone consistent with **/album** **/list**
- **Monitoring (V)**: No new SLO; optional outcome logs — per **spec** **Clarifications** (not blocking)
- **Logging (VI)**: No **message** **body** in **logs** beyond existing policy; **error** + **outcome** only
- **Performance (VII)**: Stated **N/A** in Technical Context
- **Containerization (VIII)**: Existing **`bot`** **Dockerfile** / **compose**; **no** new **runnable** **component**
- **Documentation accuracy (IX)**: This **plan**, **spec**, **quickstart**, **contracts**, and **.cursor** **rules** **SPECKIT** **block** **reference** same **paths**; **README** **operator** blurb for **`/remove`** as already shipped in repo or **follow-up**

**Post-design re-check**: Phase 0–1 artifacts updated; no **NEEDS** **CLARIFICATION**; gates **PASS**.

## Project Structure

### Documentation (this feature)

```text
specs/007-remove-saved-album/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── remove-command.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
bot/
├── cmd/bot/
│   └── main.go
├── migrations/
│   └── (no new migration for 007 — reuse disambig + saved_albums)
├── internal/
│   ├── core/           # ParseRemoveLine, remove_saved, match tiers, help copy
│   ├── adapter/telegram/
│   │   ├── run.go      # /remove, RemovePick, numeric pick, handleCallback: rmp: (TBD)
│   │   └── routing_test.go
│   └── store/          # List rows, delete by id+listener, disambig JSON
└── go.mod
```

**Structure Decision**: Single **`bot`** module; **LibraryService**-style **remove** **flow**; **Telegram** **only** first **adapter** — **inline** **keyboard** + **CallbackQuery** **handler** **alongside** **`lpl:`** and **`apick:`**.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations requiring exceptions for this feature.
