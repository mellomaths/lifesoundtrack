# Tasks: Daily fair album recommendations (004)

**Input**: Design documents from `specs/004-daily-album-recommendations/`  
**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md), [contracts/](contracts/), [research.md](research.md), [quickstart.md](quickstart.md)

**Regenerated**: 2026-04-24 (`/speckit-tasks`).

**Tests**: Plan and [spec.md](spec.md) NFR **Testing** call for unit tests on selection, flag parsing, and fakes for send/store; **listener-enumeration** regression on Postgres (**SC-007**, **FR-013**–**FR-015**) via **T026**; **SC-003** tie-break tolerance per [research.md](research.md) §10.

**Organization**: Phases follow **implementation dependency order** (US2 → US3 → US4 → US1). All four stories are **P1** in the spec; US1 (user-visible message) is integrated last after domain, persistence rules, and scheduling flags are in place.

**Status (synced with repo + [plan.md](plan.md))**: **T003**–**T007**, **T008**–**T013**, **T017**–**T021**, **T025**–**T027**, **T001**–**T002**, **T004**–**T005**, **T014**–**T016**, **T020** are **done** in tree (migration **`00002`**, store list/persist, fair pick + tests, runner + fake tests, Telegram daily send + single `*bot.Bot`, cron invokes runner per listener, **429** retry helper). **T022**–**T024** (tick-level aggregate log fields, quickstart final pass, full lint) remain **open**. Manual release gates **SC-004** / **SC-005**: [plan.md](plan.md) § Release and UAT; [quickstart.md](quickstart.md) § Release / UAT.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: `US1`–`US4` maps to user stories in [spec.md](spec.md)

## Path Conventions

- Bot module: `bot/` at repo root (see [plan.md](plan.md))

---

## Phase 1: Setup (shared infrastructure)

**Purpose**: Dependencies and operator-facing env documentation.

- [x] T001 Add `github.com/robfig/cron/v3` to `bot/go.mod` and run `go mod tidy` in `bot/`
- [x] T002 [P] Document `LST_DAILY_RECOMMENDATIONS_ENABLE`, `LST_DAILY_RECOMMENDATIONS_TZ`, and `LST_DAILY_RECOMMENDATIONS_CRON` in `bot/.env.example` per [contracts/feature-flags.md](contracts/feature-flags.md)

---

## Phase 2: Foundational (blocking prerequisites)

**Purpose**: Schema, configuration, and store support **before** any user story wiring.

**⚠️ CRITICAL**: No user story integration until this phase completes.

- [x] T003 Add Goose migration `bot/migrations/00002_daily_recommendations.sql` (`saved_albums.last_recommended_at`, `recommendations` table, indexes, `UNIQUE (listener_id, run_id)`) per [data-model.md](data-model.md)
- [x] T004 Extend `bot/internal/config/config.go` with daily recommendation fields (`DailyRecommendationsEnable`, timezone, cron expression) using the same opt-out boolean semantics as metadata flags per [contracts/feature-flags.md](contracts/feature-flags.md) (see also `bot/internal/config/daily_schedule.go`)
- [x] T005 [P] Add table-driven tests for `LST_DAILY_RECOMMENDATIONS_*` parsing and invalid TZ/cron handling in `bot/internal/config/config_test.go`
- [x] T006 Add store methods in `bot/internal/store/` to load `saved_albums` rows for a listener (for rotation) and `RecordRecommendationTx` updating `last_recommended_at` + inserting `recommendations` in one transaction (**FR-007**); `ListTelegramDailyTargets` + refactored `ListTelegramListenerIDsWithSavedAlbums` in `bot/internal/store/daily_recommendations*.go`
- [x] T007 [P] Extend `bot/internal/store` tests (e.g. `daily_recommendations_test.go` or new file) to apply **T003** migration and assert `recommendations` constraints / `RecordRecommendationTx` against Postgres
- [x] T025 Refactor listener-list SQL in `bot/internal/store/daily_recommendations.go` (or the store module that logs `daily_recommendations_listeners`) so PostgreSQL accepts it: **no** `SELECT DISTINCT` with `ORDER BY` expressions missing from the select list (**SQLSTATE 42P10**); follow [research.md](research.md) §8 and [spec.md](spec.md) **FR-013**–**FR-015**
- [x] T026 Add or extend Postgres-backed test in `bot/internal/store/daily_recommendations_test.go` or `bot/internal/store/migration_listeners_test.go` to run the listener enumeration query with seeded `listeners` + `saved_albums`, assert **no** error, and assert each eligible listener appears **at most once** per call (**SC-007**); depends on **T025**
- [x] T027 Register in-process daily cron in `bot/cmd/bot/main.go` (background goroutine + `robfig/cron/v3` with `LST_DAILY_RECOMMENDATIONS_*` from config) so ticks run while Telegram long-polls (**FR-017**); log `daily_recommendations_config` at startup (**FR-018**); on tick log `daily_recommendations_cron_tick` and run listener discovery (`daily_recommendations_listeners`)

**Checkpoint**: Foundational schema, store, and listener discovery + cron are ready (**T003**–**T007**, **T025**–**T027**).

---

## Phase 3: User Story 2 — Fair rotation (Priority: P1)

**Goal**: Deterministic selection: never-recommended first, then oldest `last_recommended_at`, uniform random within ties ([spec.md](spec.md) **FR-003**).

**Independent Test**: Unit tests with fixed RNG and known `saved_albums` rows prove tier ordering and tie draws.

### Implementation for User Story 2

- [x] T008 [US2] Implement fair rotation picker (pure function or small type) in `bot/internal/core/daily_recommendation.go` (or `pick_saved_album.go`) per [contracts/daily-recommendations-job.md](contracts/daily-recommendations-job.md)
- [x] T009 [P] [US2] Implement Spotify open URL helper from `saved_albums` fields in `bot/internal/core/daily_recommendation.go` per [research.md](research.md) §6
- [x] T010 [P] [US2] Add unit tests for tier ordering, ties, and injectable `rand` in `bot/internal/core/daily_recommendation_test.go`

**Checkpoint**: Selection logic matches **SC-003** harness expectations.

---

## Phase 4: User Story 3 — Only record success when delivery lands (Priority: P1)

**Goal**: After Telegram accepts, one transaction updates `last_recommended_at` and inserts `recommendations`; on failure, no DB writes (**FR-007**, **FR-008**).

**Independent Test**: Tests with fake Telegram client: success → tx called; error → no tx.

### Implementation for User Story 3

- [x] T011 [US3] Define messenger port interface (`DailyMessenger` / `SendDailyPick`) in `bot/internal/core/daily_recommendation.go`
- [x] T012 [US3] Implement `DailyRecommendRunner` in `bot/internal/core/daily_recommendation.go` that: accepts `run_id`, loads pick via store, calls messenger, **only on success** calls `RecordRecommendationTx` (**FR-007**/**FR-008**); on send error, **no** persistence, log failure per **FR-009**; batch continues in `bot/cmd/bot/main.go` (**FR-002**)
- [x] T013 [P] [US3] Add unit tests with fake messenger and stub store in `bot/internal/core/daily_recommendation_test.go` covering success vs send error paths

**Checkpoint**: Persistence ordering matches **SC-001** / **SC-002**.

---

## Phase 5: User Story 4 — Operators toggle via environment (Priority: P1)

**Goal**: Master flag off → no scheduled runs and no job side effects (**FR-010**, **SC-006**); valid schedule from env (**FR-011**, **FR-012**).

**Independent Test**: Integration/manual: disable flag, observe no cron fires; enable flag, observe startup INFO logs for TZ/cron per **FR-009** / plan.

### Implementation for User Story 4

- [x] T014 [US4] Register `cron.Cron` with `cron.WithLocation` in `bot/cmd/bot/main.go` only when `cfg.DailyRecommendationsEnable` is true; parse `LST_DAILY_RECOMMENDATIONS_CRON` into schedule
- [x] T015 [US4] On startup, log at **INFO** whether daily recommendations are enabled, resolved timezone, and cron expression (no secrets) in `bot/cmd/bot/main.go`
- [x] T016 [US4] Ensure when daily recommendations are disabled, no cron goroutine or scheduled callbacks run (no sends, no store calls from this feature) in `bot/cmd/bot/main.go`

**Checkpoint**: **SC-006** observable in staging with flag off.

---

## Phase 6: User Story 1 — Wake up to one pick (Priority: P1) 🎯 MVP slice

**Goal**: User receives one message per run with cover (when available), templated copy, Spotify button or inline URL (**FR-004**–**FR-006**).

**Independent Test**: Manual or sandbox: enabled flag, seeded `saved_albums`, trigger run → one Telegram message matching template.

### Implementation for User Story 1

- [x] T017 [US1] Implement Telegram daily message send (photo + caption and/or text message, inline URL keyboard when URL present) in `bot/internal/adapter/telegram/daily_recommendation.go` using `github.com/go-telegram/bot` per [contracts/daily-recommendations-job.md](contracts/daily-recommendations-job.md)
- [x] T018 [US1] Build caption/body text: `Your pick today: TITLE — ARTIST (YEAR)` and sign-off in `bot/internal/core/daily_recommendation.go` (`FormatDailyPickLine`, `DailySignoff`, `BuildDailyPickMessage`)
- [x] T019 [US1] Reuse the **single** `*bot.Bot` from `telegram.NewBot` / `telegram.Start` in `bot/cmd/bot/main.go`; `DailyMessenger` calls `SendPhoto` / `SendMessage` on that instance; `ListTelegramDailyTargets` supplies `external_id` → `chat_id`
- [x] T020 [US1] Cron callback in `bot/cmd/bot/main.go` iterates targets, `run_id` per tick, invokes `DailyRecommendRunner.RunForListener`; invalid `external_id` logged and skipped (**FR-002**)

**Checkpoint**: End-to-end path satisfies **User Story 1** acceptance scenarios when flag on.

---

## Phase 7: Polish & cross-cutting concerns

**Purpose**: Hardening, rate limits, docs.

- [x] T021 [P] Handle Telegram **429** / transient errors with backoff between listeners in `bot/internal/adapter/telegram/daily_recommendation.go` or runner
- [ ] T022 [P] Add structured log fields for `run_id`, attempted/ok/skipped/failed counts at end of each cron tick in `bot/internal/core/daily_recommendation.go` or `bot/cmd/bot/main.go`
- [ ] T023 [P] Align `specs/004-daily-album-recommendations/quickstart.md` with final end-to-end verification after **T017**–**T020** and confirm **Release / UAT** section covers **SC-004** / **SC-005** sign-off expectations
- [ ] T024 [P] Run `go test ./...` and `golangci-lint run` from `bot/` and fix any new issues

---

## Dependencies & execution order

### Phase dependencies

- **Phase 1** → **Phase 2** (**T003**/**T006**/**T007** after **T025**/**T026**) → **Phase 3 (US2)** → **Phase 4 (US3)** → **Phase 5 (US4)** (done) → **Phase 6 (US1)** → **Phase 7**
- **US1** depends on pick (**US2**), send-then-tx (**US3**), and cron behind flag (**US4**); **T020** depends on **T025** and a working **DailyRecommendRunner** (**T012**).

### User story dependency graph

```text
Foundational (schema, config, store — **T003**/**T006**/**T007** done)
        │
        ▼
   US2 (pick + URL helper)
        │
        ▼
   US3 (runner + fake tests)
        │
        ▼
   US4 (cron + main wiring — **T014**–**T016** done via **T027**)
        │
        ▼
   US1 (Telegram formatting + **T020** full wiring)
```

### Parallel opportunities

- **Phase 1**: T001 vs T002  
- **Phase 2**: T005 and T007 after T004/T006 land (T007 may need T003); **T026** after **T025**  
- **Phase 3**: T009 vs T010 after T008  
- **Phase 7**: T021–T024 mostly independent files

---

## Parallel example: User Story 2

```bash
# After T008 completes:
# T009 Spotify URL helper and T010 unit tests can proceed in parallel (different focus, same package—coordinate if merge conflicts).
```

---

## Parallel example: User Story 1

```bash
# T018 copy builder in core can start alongside T017 adapter skeleton once message DTO shape from T011–T012 is stable.
```

---

## Implementation strategy

### MVP first (smallest shippable slice)

1. **T003**/**T006**/**T007**, **Phase 3–4**, and **Phase 6**/**T020** are complete in tree.  
2. **Next**: validate against [quickstart.md](quickstart.md) (**T023**), add tick-level summary logs if desired (**T022**), run **`golangci-lint run ./...`** in CI or locally (**T024**).

### Incremental delivery

- **US2** alone: provable fair rotation in tests.  
- **+US3**: correct persistence coupling.  
- **+US4**: safe operator kill switch.  
- **+US1**: full product message.

### Listener discovery hotfix (if the job aborts at `daily_recommendations_listeners`)

1. Complete **T025** then **T026** (can ship ahead of any optional polish).  
2. Re-run **quickstart.md** listener-discovery step (**SC-007**).

---

## Notes

- Exact file splits (`daily_recommendation.go` vs existing files) may follow repo naming; tasks name preferred new paths.  
- `[P]` tasks: avoid editing the same merge-hotspot simultaneously.  
- Commit after each task or logical group.  
- **T025**/**T026**/**T027** implement listener discovery + in-process scheduler; **T020** must call the full runner, not only `ListTelegramListenerIDsWithSavedAlbums`.
