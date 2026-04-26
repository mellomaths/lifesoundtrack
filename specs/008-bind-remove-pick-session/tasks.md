# Tasks: Bind remove picks to disambiguation session

**Input**: Design documents from `specs/008-bind-remove-pick-session/`  
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/](./contracts/)

**Tests**: Included where [spec.md](./spec.md) NFR **Testing** requires an isolatable **inline (session id)** regression for two `remove_saved` lists ([SC-001](./spec.md)), `kind`-safe text behavior ([US3](./spec.md)), and [quickstart.md](./quickstart.md) verification; see **T014** for adapter coverage.

**Organization**: Tasks are grouped by user story (P1 → P3) for independent delivery and test criteria.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no blocking dependencies on incomplete work in the same critical path)
- **[Story]**: `US1`, `US2`, `US3` for user story phases only

## Path Conventions

All implementation paths are under the Go module `bot/`: `bot/internal/core`, `bot/internal/store`, `bot/internal/adapter/telegram`.

---

## Phase 1: Setup (read before code)

**Purpose**: Lock scope to existing files and the 008 contract.

- [X] T001 Skim [specs/008-bind-remove-pick-session/research.md](./research.md) and [specs/008-bind-remove-pick-session/contracts/remove-pick-binding.md](./contracts/remove-pick-binding.md) and confirm current `rmp:` flow in [bot/internal/adapter/telegram/run.go](../../bot/internal/adapter/telegram/run.go) and [bot/internal/core/remove_saved.go](../../bot/internal/core/remove_saved.go)

---

## Phase 2: Foundational (blocking prerequisites)

**Purpose**: Keyed disambiguation lookup by session id and listener; required before user stories.

**⚠️ CRITICAL**: User story work must not start until the store API exists (or in parallel, stub returns `pgx.ErrNoRows` and tests are skipped with `t.Skip` only if the plan explicitly allows—prefer completing T002–T003 first).

- [X] T002 Add `OpenDisambiguationSessionForListener` (or the final exported name) in [bot/internal/store/disambig.go](../../bot/internal/store/disambig.go) to return `candidates` JSON for `id = $sessionID`, `listener_id = $listenerID`, and `expires_at > now()` (return no row for missing, wrong listener, or expired, per [data-model.md](./data-model.md))
- [X] T003 [P] Add table-driven tests in [bot/internal/store/disambig_test.go](../../bot/internal/store/disambig_test.go) for the new method (isolation, expiry, valid row), reusing the project’s real-Postgres test pattern from [bot/internal/store/saved_albums_delete_test.go](../../bot/internal/store/saved_albums_delete_test.go) if present

**Checkpoint**: Store can resolve an open disambiguation row by id scoped to a listener.

---

## Phase 3: User Story 1 — Pick matches the list you chose (Priority: P1) 🎯 MVP

**Goal**: Inline `rmp:` callback applies index *k* to the **disambiguation session embedded in the callback**, not the listener’s latest session ([spec.md](./spec.md) US1, FR-001, FR-002).

**Independent Test**: Two sequential remove disambiguation lists; action targeting list 1’s session id does not remove a candidate that exists only on list 2 (simulated in tests; aligns with [SC-001](./spec.md)).

### Implementation

- [X] T004 [US1] Extend `TryProcessRemovePick` in [bot/internal/core/remove_saved.go](../../bot/internal/core/remove_saved.go) to accept `sessionID string` (or equivalent); when non-empty, load via T002, parse `remove_saved` JSON, delete only that candidate, clear that session on success; when `sessionID` is empty, keep a clearly documented text path for later (US3)
- [X] T005 [US1] In [bot/internal/adapter/telegram/run.go](../../bot/internal/adapter/telegram/run.go), pass the parsed `sessionID` from `parseRemovePickCallbackData` into `TryProcessRemovePick` in `handleRemovePickCallback` (do not discard with `_`)

### Tests (regression, spec NFR)

- [X] T006 [P] [US1] Add regression test in [bot/internal/core/remove_saved_test.go](../../bot/internal/core/remove_saved_test.go) (or fakes) proving callback session id S1 with index 1 does not delete a row that belongs only to a newer S2 `remove_saved` payload (per [contracts/remove-pick-binding.md](./contracts/remove-pick-binding.md))

**Checkpoint**: [SC-001](./spec.md) green for the covered test; [SC-003](./spec.md) still achievable for the common single-session callback path.

---

## Phase 4: User Story 2 — Stale or invalid choice, safe outcome (Priority: P2)

**Goal**: Missing, expired, or wrong-kind disambiguation for a **keyed** pick yields a non-destructive user message ([spec.md](./spec.md) US2, FR-003, [SC-002](./spec.md)).

**Independent Test**: Supersede or delete the session row, invoke pick with the old `rmp:` id, assert no `DELETE` on `saved_albums` for another candidate and user-facing copy is safe.

- [X] T007 [US2] In [bot/internal/core/remove_saved.go](../../bot/internal/core/remove_saved.go) and, if new strings are needed, [bot/internal/core/copy.go](../../bot/internal/core/copy.go), return clear copy when keyed lookup finds no open session or `kind` is not `remove_saved` (no fall-through to “latest” for callback path; per [FR-003](./spec.md))
- [X] T008 [P] [US2] Add test in [bot/internal/core/remove_saved_test.go](../../bot/internal/core/remove_saved_test.go) or [bot/internal/store/disambig_test.go](../../bot/internal/store/disambig_test.go) for stale or missing session id: no `DeleteSavedAlbumForListener` on unrelated id

**Checkpoint**: [SC-002](./spec.md) supported by tests and user-visible outcomes.

---

## Phase 5: User Story 3 — Typed number follow-up stays “kind”-safe and single-session-correct (Priority: P3)

**Goal**: When `sessionID` is empty, plain text `1`..`99` only completes a `remove_saved` pick against the **latest** open disambiguation session with `kind: remove_saved`; never against album disambiguation or other payloads ([FR-005](./spec.md), [US3](./spec.md)). v1 does **not** disambiguate two *historical* on-screen remove lists from text alone; **inline** (US1) is the path for that guarantee (see [spec.md](./spec.md) **Clarifications**).

**Independent Test**: One open `remove_saved` + valid index works; latest session **not** `remove_saved` → remove pick returns not handled for remove path; matches [US3](./spec.md) **Independent Test** and acceptance scenarios.

- [X] T009 [US3] Finalize `TryProcessRemovePick` when `sessionID` is empty in [bot/internal/core/remove_saved.go](../../bot/internal/core/remove_saved.go) (`LatestOpenDisambiguationSession` + `kind` check; avoid applying index to a non-`remove_saved` session; do not pick “newest” for album-save JSON). Cross-check with private-chat route order in [bot/internal/adapter/telegram/run.go](../../bot/internal/adapter/telegram/run.go) so an album disambig open before a number does not get misrouted as a **remove** pick ([spec.md](./spec.md) Edge Cases, **FR-005**)
- [X] T010 [P] [US3] Add focused test in [bot/internal/core/remove_saved_test.go](../../bot/internal/core/remove_saved_test.go) for the text path with latest `remove_saved` only, and for “not remove_saved latest session” → `ok` false for remove path; extend mocks on [bot/internal/core/save_album_test.go](../../bot/internal/core/save_album_test.go) `memStore` if `LibraryService` now calls new store methods from tests

**Checkpoint**: P3 behavior documented in code comments where subtle; aligns with [SC-003](./spec.md) for the common case.

---

## Phase 6: Polish & cross-cutting concerns

- [X] T011 Run `go test ./...` from [bot/](../../bot/) and fix failures
- [X] T012 [P] Optional: add one-line “superseded by 008” pointer in [specs/007-remove-saved-album/contracts/remove-command.md](../../specs/007-remove-saved-album/contracts/remove-command.md) to [contracts/remove-pick-binding.md](./contracts/remove-pick-binding.md) (only if maintainers want traceability; skip if 007 is frozen)
- [X] T013 Follow [specs/008-bind-remove-pick-session/quickstart.md](./quickstart.md) for a manual smoke after automated tests pass
- [X] T014 [P] Add or extend tests in [bot/internal/adapter/telegram/routing_test.go](../../bot/internal/adapter/telegram/routing_test.go) (or a small new test file in `bot/internal/adapter/telegram/`) so the `rmp:` callback path passes the disambiguation session id into core after `TryProcessRemovePick` gains a `sessionID` parameter (per [plan.md](./plan.md) Testing and [spec.md](./spec.md) **SC-001** inline path), without duplicating full core coverage

---

## Dependencies & execution order

### Phase dependencies

- **Phase 1** → can run anytime.
- **Phase 2 (T002–T003)** → **blocks** Phase 3–5 for real implementations.
- **Phase 3 (US1)** → T004–T005 **depend** on T002; T006 can be drafted after T002 if interfaces are agreed, but must **pass** after T004–T005.
- **Phase 4 (US2)** → depends on **Phase 3** callback path and keyed lookup behavior.
- **Phase 5 (US3)** → depends on `TryProcessRemovePick` shape from T004 and behavior from T007.
- **Phase 6** → after desired user story phases are complete; **T014** can run after **T005** once `TryProcessRemovePick` signature is stable, but is batched in polish for fewer churn.

### User story order

- **US1 (P1)** is MVP: callback session binding.
- **US2 (P2)** extends messaging for stale/invalid keyed picks.
- **US3 (P3)** finalizes the text path policy.

US2 and US3 can overlap in the same files after US1’s signature and store are stable.

### Parallel opportunities

- **T003** and **T006** (tests in different files) in parallel only after the method signature for T002 and `TryProcessRemovePick` for T004 are fixed.
- **T008** and other story tests marked **[P]** when they touch different `*_test.go` files and do not require unfinished production APIs.

---

## Parallel example: User Story 1

```bash
# After T002 returns (Session, []byte) shape:
# Parallel: T003 (store tests) and T006 (core regression) once T004–T005 are merged
# Sequential: T004 → T005 → T006 green
```

---

## Implementation strategy

### MVP first (User Story 1 + foundation)

1. T001 → T002 → T003  
2. T004, T005, T006 (tests green)  
3. Stop: validate [SC-001](./spec.md) / [SC-003](./spec.md) for callback flows  

### Incremental delivery

1. Add T007–T008 (P2) for stale safe outcomes.  
2. Add T009–T010 (P3) for text.  
3. T011–T014 polish (including adapter test **T014**).  

### Suggested MVP scope

**T001 through T006** = **User Story 1** + **foundation** (addresses the high-severity wrong deletion for inline buttons).

---

## Notes

- Every task line uses the required checklist form: `- [ ] Tnnn …` with at least one concrete file path.  
- Prefer small, reviewable PRs: foundation + US1 first, then US2, then US3.  
- If `TryProcessRemovePick` signature changes, update all call sites in [bot/internal/adapter/telegram/run.go](../../bot/internal/adapter/telegram/run.go) for both callback and text branches.

## Format validation

- All tasks use: `- [ ]` + Task ID (T001–T014) + optional `[P]` + optional `[USn]` (only in user story phases) + description + file path.
