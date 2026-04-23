# Tasks: LifeSoundTrack — `.env` loading and local hot reload (002)

**Input**: Design documents from `specs/002-env-file-config/`  
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md) (with **US1–US4**), [research.md](research.md), [contracts/env-loading.md](contracts/env-loading.md), [data-model.md](data-model.md), [quickstart.md](quickstart.md)

**Principle**: Add **`github.com/joho/godotenv`** only in **`bot/internal/config`** (and `go.mod`); **`bot/internal/core`** and **`bot/internal/adapter/telegram`** MUST NOT import `godotenv`. Use **`air`** + **`bot/.air.toml`** for watch/restart (dev only; not a `go.mod` import).

**Format**: `- [ ] TNNN [P?] [USn?] Description` with a concrete `bot/...` or `specs/002-...` path.

## Phase 1: Setup (dependency)

**Purpose**: Add the open-source **`.env`** parser dependency per **FR-007** and [plan.md](plan.md).

- [x] T001 Add `github.com/joho/godotenv` to `bot/go.mod` (pin a **released** version, e.g. v1.5.1) and run `go mod tidy` in `bot/`

**Checkpoint**: `go mod verify` from `bot/`; `go list -m all | grep godotenv` shows the module.

---

## Phase 2: Foundational (blocking for all user stories)

**Purpose**: Load optional **`bot/.env`** into the process environment **before** `config.FromEnv()` so **FR-001–FR-003** and [contracts/env-loading.md](contracts/env-loading.md) order hold.

- [x] T002 Add `bot/internal/config/loadenv.go` exporting a function (e.g. `LoadLocalDotEnv() error`) that resolves **`.env`** in the current working directory (documented as **run from `bot/`**), calls `godotenv.Load` for that path, **ignores** “file not found” (`os.IsNotExist`), and returns a **parse** error without including file **contents** in the error value (wrap if needed) per **FR-004**
- [x] T003 In `bot/cmd/bot/main.go`, call the new loader **before** `config.FromEnv()`; on loader error, log with `slog` using only safe fields (no token, no `.env` body) and `os.Exit(1)`

**Checkpoint**: `go build -o nul ./cmd/bot` (or `go build ./cmd/bot`) from `bot/`; with only `TELEGRAM_BOT_TOKEN` in `bot/.env` and an empty shell, config succeeds (manual smoke or later tests).

---

## Phase 3: User Story 1 — Run with a local env file (Priority: P1)

**Goal**: **File-only** config works; **OS overrides file**; **file optional** if OS is complete. Maps to **US1** and [contracts/env-loading.md](contracts/env-loading.md) precedence.

**Independent test**: [spec.md](spec.md) US1; [contracts/env-loading.md](contracts/env-loading.md).

- [x] T004 [US1] Extend `bot/internal/config/config_test.go` (or add `loadenv_test.go` in the same package) with table-driven tests: (a) chdir to temp, write `.env` with `TELEGRAM_BOT_TOKEN`, **clear** that env in `t.Setenv`, assert `FromEnv` succeeds and token matches file; (b) both file and `t.Setenv("TELEGRAM_BOT_TOKEN", ...)` with **different** values—assert **OS** wins; (c) no `.env`, token **only** from `t.Setenv`—assert success (**FR-003**); (d) optional: one fixture with a **`#` comment line** and CRLF to lock **library** behavior per [contracts/env-loading.md](contracts/env-loading.md) (no need to re-spec grammar—assert token still loads)
- [x] T005 [P] [US1] Add a test that a **missing** `.env` and missing token yields the **same** error class as today (string may include “required” / “TELEGRAM_BOT_TOKEN” as name only, not a secret) per [spec.md](spec.md) US1/US2 boundary. *Related: invalid-parse cases are in **T006** (not silent ignore).*

**Checkpoint**: `go test ./internal/config/... -count=1` passes without network or Telegram.

---

## Phase 4: User Story 2 — Safe handling when the file is wrong or missing (Priority: P1)

**Goal**: Missing required keys fail clearly; no secret leakage. **US2**; align with [contracts/env-loading.md](contracts/env-loading.md).

- [x] T006 [US2] If `godotenv.Load` returns a **parse** error (invalid `.env` line), ensure `main` path logs a **generic** message (e.g. “env file invalid”) and does **not** use `%s` with file contents; add a test with a **bad** `.env` in temp that expects failure without full dump. *Complements **T005** (missing token) — different failure class per [spec.md](spec.md) US2 and [contracts/env-loading.md](contracts/env-loading.md).*
- [x] T007 [US2] In `specs/002-env-file-config/quickstart.md` and/or `bot/README.md`, add one short **operator** paragraph: invalid `.env` may fail fast on parse; missing token behavior unchanged; still **no** real secrets in docs

**Checkpoint**: Unit tests for bad file pass; error output reviewed for no token.

---

## Phase 5: User Story 4 — Fast local iteration (hot reload) (Priority: P1)

**Goal**: **FR-008** — documented watch of **`.go`** + **`.env`**; **not** in Docker/CI. [plan.md](plan.md) **air** + [research.md](research.md).

**Independent test**: [spec.md](spec.md) US4; save a `.go` and `.env` and observe restart (manual for **air**).

- [x] T008 [US4] Add `bot/.air.toml` that builds `go build` output for `./cmd/bot`, watches `include_ext` including **`go`** and **`env`**, and excludes `tmp/`, `vendor/`, and binary artifacts; `root` = `.` when run from `bot/`
- [x] T009 [US4] Update `bot/README.md` and [quickstart.md](quickstart.md): install **air** (`go install` or release), run **`air`** from `bot/`; state **Dockerfile** / **compose** / CI **do not** use **air** by default; link [plan.md](plan.md) research choice

**Checkpoint**: From `bot/`, `air` starts the bot (with token); touch `.env` → process restarts (manual); `docker compose` path unchanged in behavior.

---

## Phase 6: User Story 3 — Documentation and discovery (Priority: P2)

**Goal**: **FR-005** — where `.env` lives, keys, precedence, no secrets in VCS. **US3**.

**Independent test**: New reader follows docs only. Can proceed after **Phase 3** (content can mention **air** from Phase 5).

- [x] T010 [US3] Update `bot/.env.example` with comments: load order (**OS** over file), `godotenv`, link to 002 **quickstart**
- [x] T011 [US3] Update [README.md](../../README.md) (repo root) with one subsection: **002** runbook for `.env` + **local dev** (`air`); point to [quickstart.md](quickstart.md) and [001 quickstart](../001-lifesoundtrack-bot-commands/quickstart.md) for Telegram domain steps
- [x] T012 [P] [US3] In [specs/001-lifesoundtrack-bot-commands/quickstart.md](../001-lifesoundtrack-bot-commands/quickstart.md), add a **single** cross-link under “Configure environment” to [002 quickstart](quickstart.md) for file-based and watch-mode details (avoid duplicating long prose)

**Checkpoint**: **FR-005** checklist: path, keys, precedence, no committed secrets.

---

## Phase 7: Polish and cross-cutting

- [x] T013 From `bot/`: `go test ./... -count=1` and `go vet ./...`
- [x] T014 [P] `go list -deps ./internal/core/...` and confirm **no** `joho/godotenv` in **core** (adapter unchanged)
- [x] T015 [P] Log audit: `slog` paths on config/dotenv errors must not log `TELEGRAM_BOT_TOKEN` or raw `.env` (review `bot/cmd/bot` + `bot/internal/config`)
- [x] T016 (optional) [P] If `golangci-lint` is used in the repo, `golangci-lint run ./...` from `bot/` and fix any new issues

**Checkpoint**: All tests green; [contracts/env-loading.md](contracts/env-loading.md) still matches code paths.

---

## Dependencies and execution order

1. **T001** → **T002–T003** (foundational) — required before all stories.
2. **US1** (T004–T005) should complete before **US2** (T006–T007) to lock precedence tests.
3. **US4** (T008–T009) can start after **T003**; prefer completing **T004** first so `air` runs a build that passes unit tests.
4. **US3** (T010–T012) can overlap with **US4** after **T002–T003** (documentation).

## Parallel example

```text
# After T003 and T004:
# Terminal A: T008 (air + .air.toml)
# Terminal B: T010 ( .env.example ) + T012 (001 quickstart link)
```

## Implementation strategy

1. **MVP**: **Phase 1–3** (T001–T005) — `godotenv` + load + tests for US1.
2. **Hardening**: **Phase 4** (US2) — parse error safety + docs line.
3. **DevEx**: **Phase 5** (US4) — `air` + docs.
4. **Polish**: **Phase 6** (US3) — cross-links and root README; **Phase 7** — gates.

## Task summary

| Metric | Value |
|--------|------:|
| **Total tasks** | 16 (T001–T016) |
| **MVP (smallest)** | Through **T005** (Phases 1–3) |
| **US1** | T004, T005 |
| **US2** | T006, T007 |
| **US4** | T008, T009 |
| **US3** | T010, T011, T012 |
| **Polish** | T013–T016 |

**Format validation**: All lines use `- [ ] TNNN`, `bot/...` or `specs/...` paths, and `[US#]` for story phases 3–6.

## Notes

- **Domain**: [spec.md](spec.md) **FR-006** — no edits to `bot/internal/core` copy for start/help/ping except accidental imports; **air** and **godotenv** stay out of **core**.
