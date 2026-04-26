# Tasks: LifeSoundtrack — `/list` saved albums (006)

**Input**: Design documents from `specs/006-list-saved-albums/`  
**Prerequisites**: [plan.md](plan.md) (Go **1.25**, `bot/` module), [spec.md](spec.md) (incl. **Clarifications** 2026-04-26, **FR-003**, **FR-010**), [research.md](research.md) (§8 display, §9 tests), [data-model.md](data-model.md), [contracts/list-command.md](contracts/list-command.md), [quickstart.md](quickstart.md)

**Tests**: Spec **Testing** NFR + **SC-002** / **SC-003** + Clarifications require **automated** coverage: **`parse_list_test.go`**, store variant/isolation tests, **`run_test.go`** routing — see Phase 5–7.

**Organization**: Phases follow spec user-story priorities (P1 → P2). User Story 3 is **normalization verification** (table tests).

## Task line format

Each task is a single checklist line: `- [ ]` + **Task ID** (`T001`…) + optional **`[P]`** + optional **`[USn]`** + description **including concrete file paths**.

- **`[P]`**: Can run in parallel (different files, no blocking dependency on incomplete tasks in the same wave)
- **`[Story]`**: `[US1]` … `[US4]` on user-story phase tasks only (omit on Setup, Foundational, Polish)

## Path Conventions

- **Bot module**: `bot/` at repository root (see [plan.md](plan.md))

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Align with existing `bot/` layout before code changes.

- [x] T001 [P] Skim `specs/006-list-saved-albums/contracts/list-command.md` and `specs/006-list-saved-albums/data-model.md` against `bot/internal/store/saved_albums.go` and `bot/migrations/00001_init_listeners_saved_albums_disambig.sql` for **`primary_artist`**, listener scoping, and **FR-010** (display vs persistence)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Migration, store queries, and command parsing **must** exist before any user story can ship.

**⚠️ CRITICAL**: No user story phase starts until **T002–T004** are done.

- [x] T002 Add PostgreSQL migration `bot/migrations/00003_album_list_sessions.sql` creating `album_list_sessions` per `specs/006-list-saved-albums/data-model.md`
- [x] T003 Implement `bot/internal/store/saved_albums_list.go`: listener lookup by `(source, external_id)`, `CountSavedAlbumsForListener`, `ListSavedAlbumsPage` (`ORDER BY created_at DESC, id DESC`, limit/offset page size 5, optional **`primary_artist`** filter only per **FR-003** via `strpos(lower(coalesce(primary_artist,'')), $1) > 0`), and `album_list_sessions` insert + fetch-by-id + latest open session per `specs/006-list-saved-albums/research.md` §3–§4 (depends on T002)
- [x] T004 [P] Add `bot/internal/core/parse_list.go` with `ParseListLine` (`/list`, `/list@bot`, `next`/`back` tokens, multi-word artist remainder) and `NormalizeArtistQuery` per `specs/006-list-saved-albums/contracts/list-command.md`

**Checkpoint**: Foundation ready — user stories can begin.

---

## Phase 3: User Story 1 — See my saved albums, five at a time (Priority: P1) 🎯 MVP

**Goal**: `/list` with **no artist filter** shows up to **5** albums, **newest first**, **FR-007** empty-library onboarding, **FR-010** readable lines, and page 2+ via **`/list next`** / **`/list back`** + session when `total_count > 5` ([research.md](research.md) §3–§4).

**Independent Test**: Private chat: 0 saves → onboarding; 3 saves → all shown, no pager; 6+ saves → first 5 + text paging (buttons in US4).

- [x] T005 [P] [US1] Add `bot/internal/core/list_saved.go` with `LibraryService` (`*store.Store`), page size **5**, lines consistent with `bot/internal/core/save_album.go` (`Title | Artist (Year)`), **FR-007** messaging; **FR-010** abbreviate display only ([research.md](research.md) §8, [data-model.md](data-model.md) display vs persistence)
- [x] T006 [P] [US1] Update `bot/internal/core/copy.go` (`startCopy`, `helpCopy`) for `/list` per plan constitution **IV**
- [x] T007 [US1] Wire `LibraryService` in `bot/cmd/bot/main.go` and extend `bot/internal/adapter/telegram/run.go` (`NewBot` / handlers): `ParseListLine` for bare `/list`, whitespace-only remainder, **`/list next`** / **`/list back`**; **create `album_list_sessions` row only when `total_count > 5`** (equivalently **`total_pages > 1`** at page size 5) per `specs/006-list-saved-albums/research.md` §3; handler order preserves `bot/internal/core/parse_album.go` / `OneBasedPickFromText` (**plan** disambig note)

**Checkpoint**: US1 acceptance 1–3 (text paging satisfies “way to move” until US4).

---

## Phase 4: User Story 2 — Filter my list by artist (Priority: P1)

**Goal**: `/list <artist>` — **normalized** substring on **`primary_artist`** only (**FR-003**); **no-match** ≠ empty library.

**Independent Test**: Mixed artists; `/list Beatles` only matching saves; `/list zznone` no-match; case/spacing variants same ids (**SC-002**).

- [x] T008 [US2] Extend `bot/internal/core/list_saved.go` to pass normalized needle into store list/count; **no saved albums match** copy per `specs/006-list-saved-albums/spec.md` User Story 2
- [x] T009 [US2] Extend `bot/internal/adapter/telegram/run.go` for `/list <remainder>` through `LibraryService`; session rules unchanged for filtered multi-page lists

**Checkpoint**: US2 independently testable.

---

## Phase 5: User Story 3 — Normalization behaves predictably (Priority: P1)

**Goal**: Table tests for trim, internal whitespace, case (**FR-004**).

**Independent Test**: From repo root, `cd bot && go test ./internal/core -run …` (or `cd bot && go test ./...`) passes; mirrors User Story 3 + whitespace-only edge case.

- [x] T010 [P] [US3] Add `bot/internal/core/parse_list_test.go` table tests for `NormalizeArtistQuery` and `ParseListLine` per `specs/006-list-saved-albums/spec.md` User Story 3 and Edge Cases

**Checkpoint**: Normalization locked by tests.

---

## Phase 6: User Story 4 — Move between pages (Priority: P2)

**Goal**: Inline **Back**/**Next**, **`lpl:<uuid>:<page>`**, **EditMessageText** + fallback; footer **FR-006** text hints.

**Independent Test**: 6+ albums; buttons when `total_pages > 1`; callbacks edit message; tampered callback rejected.

- [x] T011 [US4] List pagination keyboard + `lpl:` parsing in `bot/internal/adapter/telegram/run.go`; `handleCallback` dispatches list before `apick:` / `aother` per [contracts/list-command.md](contracts/list-command.md); **inline-button** truncation per [research.md](research.md) §8 / **FR-010**
- [x] T012 [US4] `EditMessageText` + reply markup on page change in `bot/internal/adapter/telegram/run.go` with **SendMessage** fallback ([research.md](research.md) §5)
- [x] T013 [US4] `bot/internal/core/list_saved.go` footer: **`/list next`** / **`/list back`** when `total_pages > 1` (**FR-006**)

**Checkpoint**: US4 satisfied on Telegram.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Operator docs, store + routing tests, full test gate, manual quickstart.

- [x] T014 [P] Update `bot/README.md` and repository **root** `README.md` with `/list` and text paging (**FR-006**, Clarifications 2026-04-26)
- [x] T015 [P] Add `bot/internal/store/saved_albums_list_test.go` (or extend store tests): Postgres list/count; **same ordered `saved_albums` ids** for multiple normalized filter strings on a fixture (**SC-002**); **listener isolation** (**SC-003**) per [research.md](research.md) §9
- [x] T016 [P] Add or extend `bot/internal/adapter/telegram/run_test.go`: private message **routing order** — `/list` vs `/album` and `OneBasedPickFromText` (`1`/`2`/`3`) per Clarifications **C1** and constitution **III**
- [x] T017 Run `cd bot && go test ./...` and `golangci-lint` (or repo lint) until clean — must run after **T010**, **T015**, **T016** are implemented
- [x] T018 [P] Manual pass: `specs/006-list-saved-albums/quickstart.md`; note gaps

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1** → **2** → **3 (US1)** → **4 (US2)** → **5 (US3)** → **6 (US4)** → **7**
- **Phase 7**: complete **T014–T016** before **T017**; **T018** after **T017** or in parallel documentation pass

### User Story Dependencies

| Story | Depends on |
|-------|------------|
| US1 | T002–T004 |
| US2 | T005–T007 |
| US3 | **T004** (for `parse_list.go`); **T010** can start after **T004** in parallel with US1–2, or after US2 for tighter ordering |
| US4 | US1 sessions; benefits from US2 |

### Parallel Opportunities

- After **T002**: **T003** and **T004** **[P]**
- After **T004**: **T005** and **T006** **[P]**; **T010** **[P]** can start here (tests only touch `parse_list.go`)
- **T014**, **T015**, **T016**, **T018** **[P]** in Phase 7 before/with **T017**

### Parallel Example: Foundational wave

```bash
# After T002:
# Developer A: T003 bot/internal/store/saved_albums_list.go
# Developer B: T004 bot/internal/core/parse_list.go
```

### Parallel Example: User Story 1 (after T004)

```bash
# Developer A: T005 bot/internal/core/list_saved.go
# Developer B: T006 bot/internal/core/copy.go
# Then: T007 bot/cmd/bot/main.go + bot/internal/adapter/telegram/run.go
```

### Parallel Example: User Story 3 (table tests)

```bash
# After T004 (or anytime parse_list API is stable):
# Developer: T010 bot/internal/core/parse_list_test.go
# Run: cd bot && go test ./internal/core -count=1
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Phases 1–2  
2. Phase 3 (US1) — text paging before US4 buttons  
3. Validate [quickstart.md](quickstart.md) scenarios 1–3 + multi-page text paging  

### Incremental Delivery

1. US2 → US3 tests → US4 → Polish (T014–T018)

### Parallel Team Strategy

After **T002**: Dev A **T003**, Dev B **T004**. After US1: Dev A US4, Dev B US2 + US3.

---

## Notes

- **Task IDs**: **T001–T018** sequential; story phases use **[USn]**.
- **`formatAlbumLine`** in `bot/internal/core/save_album.go` may need export or shared helper during **T005**.
