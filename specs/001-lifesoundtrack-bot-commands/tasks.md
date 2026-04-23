# Tasks: LifeSoundtrack — core command behavior (sandbox)

**Input**: Design documents from `specs/001-lifesoundtrack-bot-commands/`  
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md) (includes **FR-007** and **Clarifications**)  
**Optional context**: [research.md](research.md), [data-model.md](data-model.md), [contracts/messaging-commands.md](contracts/messaging-commands.md), [quickstart.md](quickstart.md)

**Principle**: **`bot/internal/core`** holds **all** product strings and command semantics (platform-agnostic). **`bot/internal/adapter/telegram`** (first adapter) may import `github.com/go-telegram/bot` only. **`internal/core` MUST NOT** import any adapter or vendor SDK.

**Format**: `- [ ] TNNN [P?] [USn?] Description` with a concrete `bot/...` path.

## Phase 1: Setup (shared infrastructure)

**Purpose**: `bot/` module layout, tools, and adapter dependency in the **adapter** path only (not in `core`).

- [x] T001 Create directory layout per [plan.md](plan.md): `bot/cmd/bot/`, `bot/internal/core/`, `bot/internal/adapter/telegram/`, `bot/internal/config/`, and note `bot/Dockerfile` for Phase 6
- [x] T002 Initialize `bot/go.mod` and run `go get github.com/go-telegram/bot@latest` from `bot/` so the **module**’s go.sum includes the Telegram library for **`internal/adapter/telegram` only** (no `core` import of it)
- [x] T003 [P] Add `bot/.gitignore` and optional `bot/.golangci.yml` (or `bot/README.md` with `gofmt` / `go vet` / lint notes) per [plan.md](plan.md) and [research.md](research.md) §6

**Checkpoint**: `cd bot && go mod verify` succeeds; `go list -deps ./internal/core/...` must **not** list `github.com/go-telegram/bot` once `core` is populated.

---

## Phase 2: Foundational (blocking prerequisites)

**Purpose**: `internal/core` **Respond** API (domain), `internal/adapter/telegram` **Run** loop, `cmd/bot` composition, private-chat filter in **adapter**, config, logging.

**Critical**: Phases 3–6 require **T004–T009** (or equivalent) so **core** is testable without Telegram.

- [x] T004 Add `bot/internal/core/respond.go` (or `core.go`) exposing functions like `HandlePrivateCommand(domainName string) (reply string, ok bool)` mapping **start** / **help** / **ping** / **unknown** to static copy per [contracts/messaging-commands.md](contracts/messaging-commands.md) and [spec.md](spec.md) **FR-001**–**FR-006**; **no** third-party imports
- [x] T005 [P] Add `bot/internal/core/core_test.go` with table tests for all four domain paths (no network)
- [x] T006 Implement `bot/internal/config/config.go` (e.g. `TELEGRAM_BOT_TOKEN`, `LOG_LEVEL`) and `bot/.env.example` (names only) per [quickstart.md](quickstart.md) / [research.md](research.md) §3
- [x] T007 Add `bot/internal/adapter/telegram/run.go` (or `bot.go`) that creates the Telegram `bot`, uses **long polling**, maps incoming **private** messages to a parsed domain command, calls `internal/core` for reply text, and sends the reply; ignore non-private chats per [data-model.md](data-model.md)
- [x] T008 Wire `bot/cmd/bot/main.go`: load config, `slog`, `signal.Notify` + root `context`, start **adapter/telegram** `Run(ctx, cfg)` per [research.md](research.md) §5
- [x] T009 In `bot/internal/adapter/telegram`, map **unmapped** private text to **core**’s **unknown** path and send the **try help** reply (per [spec.md](spec.md) Edge cases)

**Checkpoint**: `go build ./cmd/bot` from `bot/`; with a valid **sandbox** token, all three domain behaviors work through Telegram; `go test ./internal/core/...` passes **without** network.

---

## Phase 3: User Story 1 – First contact (Priority: P1)

**Goal**: **start** domain path returns **LifeSoundtrack** welcome (exercised on Telegram as `/start`).

**Independent test**: [spec.md](spec.md) US1; contract [contracts/messaging-commands.md](contracts/messaging-commands.md) row **start**.

- [x] T010 [US1] Ensure `bot/internal/core` welcome text for the **start** case meets US1/contract; keep strings in one place (e.g. unexported consts in `core` or a dedicated `bot/internal/core/copy.go`)
- [x] T011 [US1] In `bot/internal/adapter/telegram`, map Telegram **/start** (and native start) to the **start** domain call into `core`
- [x] T012 [US1] Add or extend `bot/internal/core/*_test.go` for **start** substrings and brand name

**Checkpoint**: Sandbox: `/start` only → welcome including **LifeSoundtrack**.

---

## Phase 4: User Story 2 – Help (Priority: P1)

**Goal**: **help** path lists the three actions with **LifeSoundtrack** in context.

- [x] T013 [US2] Add **help** string assembly in `bot/internal/core` per contract; **no** new imports outside stdlib
- [x] T014 [US2] Map Telegram **/help** in `bot/internal/adapter/telegram` to **help** in `core`
- [x] T015 [US2] Extend `bot/internal/core` tests for help content (all three command names, LifeSoundtrack)

**Checkpoint**: `/help` only → full help text (same process, independent of `/start` order).

---

## Phase 5: User Story 3 – Ping (Priority: P1)

**Goal**: **ping** liveness string from **core**; adapter sends it.

- [x] T016 [US3] Add **ping** reply in `bot/internal/core`
- [x] T017 [US3] Map **/ping** in `bot/internal/adapter/telegram` to `core` **ping**
- [x] T018 [US3] Add `core` test for **ping** line stability and non-emptiness

**Checkpoint**: `/ping` only → short liveness reply.

---

## Phase 6: User Story 4 – Sandbox, packaging, docs (Priority: P1)

**Goal**: `Dockerfile`, Compose, **README** alignment; runbook for **first** (Telegram) adapter.

- [x] T019 [US4] Add `bot/Dockerfile` and root `compose.yaml` `bot` service with `TELEGRAM_BOT_TOKEN` from env; document in [quickstart.md](quickstart.md) if paths differ
- [x] T020 [US4] Update root [README.md](../../README.md): **`internal/core`** (domain) and **`internal/adapter/telegram`** (first platform); link [quickstart.md](quickstart.md) (per constitution **IX**)
- [x] T021 [US4] Reconcile [quickstart.md](quickstart.md) and any `.env.example` variable names with actual `config` and adapter wiring ([spec.md](spec.md) **SC-004**)
- [x] T022 [US4] Manually run runbook in **Telegram** sandbox: start → help → ping; record gaps only in **docs** if code is correct

**Checkpoint**: Docs and tree match; domain contract unchanged if adapter is swapped.

---

## Phase 7: Polish and cross-cutting

- [x] T023 [P] From `bot/`: `go test ./...`, `go vet ./...` — confirm **`internal/core`** does not import `github.com/go-telegram/bot` (use `go list` / grep in CI)
- [x] T024 [P] Log audit: no tokens or full private message bodies in `slog` per [spec.md](spec.md) and [plan.md](plan.md)

---

## Dependencies and execution order

- **Phases 1–2** are blocking. **T004**/**T005** (core + tests) can be done **before** T007 (adapter) if you want domain-first: reorder within Phase 2 to **T004, T005, T006, T007, T008, T009** (already close).
- **US1–US3** extend `core` then **telegram** mapping; keep **one merge point** in the adapter to reduce conflicts.
- **US4** after domain + adapter are feature-complete for the three commands.

## User story map (all P1 in [spec.md](spec.md))

| Story | Tasks |
|-------|--------|
| US1 | T010–T012 |
| US2 | T013–T015 |
| US3 | T016–T018 |
| US4 | T019–T022 |

## Parallel example (User Story 1)

```text
# Domain-first (no Telegram needed for tests):
# 1) bot/internal/core/ — start text + tests
# 2) bot/internal/adapter/telegram/ — map /start
go test ./internal/core/ -count=1
```

## Implementation strategy

1. **MVP**: Phases 1–2 (core+adapter skeleton) + Phase 3 (**start** end-to-end in Telegram).
2. Add **help** and **ping** in **core** first, then **telegram** mappers, then **unknown** in Phase 2 if not already done (T009).

## Task summary

| Metric | Value |
|--------|------:|
| **Total tasks** | 24 (T001–T024) |
| **MVP (smallest)** | Phases 1–3 (through T012) |

**Format validation**: All tasks use `- [ ] TNNN`, `bot/...` paths, and `[US*]` in story phases.

## Notes

- Adding **WhatsApp** or **Discord** later: new `bot/internal/adapter/<name>/` that calls the **same** `internal/core` functions; **out of** this task list.
- Old paths `internal/handlers/*` and `contracts/telegram-bot-commands.md` are **superseded** by `internal/core` and [contracts/messaging-commands.md](contracts/messaging-commands.md).
