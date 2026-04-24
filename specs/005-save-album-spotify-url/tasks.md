# Tasks: Save album via Spotify album link (005)

**Input**: Design documents from `/specs/005-save-album-spotify-url/`  
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/spotify-album-url.md](./contracts/spotify-album-url.md), [quickstart.md](./quickstart.md)

**Tests**: [spec.md](./spec.md) NFR (*Testing*) requires automated coverage; tasks include **unit** and **one e2e-style** test (no mandatory TDD ordering).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no unmet dependencies)
- **[Story]**: User story label ([US1], [US2], [US3]) only on user-story phases
- Paths are relative to repo root unless noted

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm scope and constraints before code changes.

- [X] T001 Confirm no new PostgreSQL migration is required per [data-model.md](./data-model.md) (no changes under `bot/migrations/` for 005 unless you intentionally add optional provenance later)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Orchestrator interface + Spotify HTTP pieces **must** exist before `SaveService` branching.

**⚠️ CRITICAL**: No user story work until this phase is complete.

- [X] T002 Extend `MetadataOrchestrator` in `bot/internal/core/orchestrator.go` with `LookupSpotifyAlbumByID(ctx context.Context, albumID string) ([]AlbumCandidate, error)` and `ResolveSpotifyShareURL(ctx context.Context, shareURL string) (albumID string, err error)` per [contracts/spotify-album-url.md](./contracts/spotify-album-url.md)
- [X] T003 [P] Implement `open.spotify.com` album URL detection and ID extraction in `bot/internal/core/spotify_album_url.go` per [research.md](./research.md) §1: discover **candidate** `http(s)://` URLs across the **full** argument (not “first URL only”—prose may precede the link); locale segments + query strings; **primary** / multi-target rules when **multiple** **FR-008** links appear; export functions used by `SaveService`
- [X] T004 [P] Implement bounded HTTPS redirect resolution with host allowlist (concrete hosts per [contracts/spotify-album-url.md](./contracts/spotify-album-url.md) + [research.md](./research.md) §2, e.g. `spoti.fi` → `open.spotify.com`) in `bot/internal/metadata/spotify_redirect.go` and wire `ResolveSpotifyShareURL` on `*Chain` in `bot/internal/metadata/orchestrator.go`
- [X] T005 [P] Implement Spotify Web API `GET /v1/albums/{id}` (client credentials, breaker, JSON → `AlbumCandidate`) in `bot/internal/metadata/spotify.go` and wire `LookupSpotifyAlbumByID` on `*Chain` in `bot/internal/metadata/orchestrator.go` per [research.md](./research.md) §4 *(**T004** and **T005** both touch `bot/internal/metadata/orchestrator.go`—serialize merge or pair-program; **T003** is safely parallel)*
- [X] T006 Update `fakeSearch` (and any other test doubles) to implement the extended `MetadataOrchestrator` in `bot/internal/core/save_album_test.go` and `bot/internal/metadata/orchestrator_test.go` as needed

**Checkpoint**: `cd bot && go test ./...` compiles (tests may still fail until US1 wiring).

---

## Phase 3: User Story 1 — Paste full URL or short link to save (Priority: P1) 🎯 MVP

**Goal**: `/album` with a valid Spotify album URL or supported short link resolves to **one** save **without** disambiguation (**FR-000**, **FR-001**–**FR-003**, **FR-008**), **including** [spec.md](./spec.md) **User Story 1** scenario 5 (**one** **unambiguous** link **embedded** in short surrounding text).

**Independent Test**: [quickstart.md](./quickstart.md) happy paths (full URL + short link + optional embedded-link example); `saved_albums` row with Spotify provider + album id.

### Implementation for User Story 1

- [X] T007 [US1] Branch `ProcessAlbumQuery` in `bot/internal/core/save_album.go`: after listener upsert, if `spotify_album_url.go` yields an album ID **or** `ResolveSpotifyShareURL` succeeds, call `LookupSpotifyAlbumByID` and `persistSave` with **no** disambiguation session; otherwise keep existing `Search` path; implement **multi-link** / **no clear primary** behavior per [spec.md](./spec.md) Edge Cases and **FR-008**; pass the **full** user argument into classification so embedded links resolve per [research.md](./research.md) §1 and [contracts/spotify-album-url.md](./contracts/spotify-album-url.md)
- [X] T008 [P] [US1] Add table-driven tests for URL parsing in `bot/internal/core/spotify_album_url_test.go` (locale + `si=` query; happy extract; **one** row with **short** surrounding prose and **one** unambiguous embedded album URL per [spec.md](./spec.md) Edge Cases / User Story 1 scenario 5; **one** row where a **non**-**FR-008** URL appears **before** the album URL in the same argument—expect **direct-link** path still selects the Spotify album per [research.md](./research.md) §1)

**Checkpoint**: Valid full URL + representative short-link fixture path save exactly one album; embedded-link case saves without chooser; **FREE_TEXT** still reaches `Search` when not **FR-008**-eligible.

---

## Phase 4: User Story 2 — Clear outcomes when link is wrong or album missing (Priority: P1)

**Goal**: Malformed links, failed redirects, non-album pages, Spotify off/404, and transient errors never produce a false “saved” and **do not** run `Search` on the raw URL string (**FR-005**); generic non-Spotify URLs stay **FREE_TEXT** (**FR-004**, **FR-008**).

**Independent Test**: [quickstart.md](./quickstart.md) failure paths; tests assert no wrongful save / no `Search`-as-fallback for failed **FR-008** attempts.

### Implementation for User Story 2

- [X] T009 [US2] Add user-facing `badLinkCopy()` (or equivalent) in `bot/internal/core/copy.go` and route URL-branch failures in `bot/internal/core/save_album.go` to distinct outcomes vs free-text `noMatchCopy()` per [research.md](./research.md) §3
- [X] T010 [P] [US2] Extend `bot/internal/core/save_album_test.go` with cases: unusable **Spotify**-eligible link, redirect loop / wrong final host, track-only URL, `LookupSpotifyAlbumByID` returns empty/`ErrNoMatch`, Spotify disabled (`ErrAllProvidersExhausted`), **multiple** qualifying links; assert **`https://example.com/...`** invokes **`Search`** with the **full** argument string (**FR-004** / **FR-008**), not the failed-link-only path; add **one** case where the query embeds a valid album URL in surrounding text and **direct-link** path runs (**no** `Search` on full string as **FREE_TEXT**) per [spec.md](./spec.md) scenario 5

**Checkpoint**: User Story 2 acceptance scenarios in [spec.md](./spec.md) covered by tests or documented manual checks.

---

## Phase 5: User Story 3 — Listeners discover links in help (Priority: P2)

**Goal**: `/help` and empty-query hint mention **free text** (unchanged) **and** **Spotify album page** / **share short** links (**SC-004**).

**Independent Test**: Reviewer reads `copy.go` help strings and [quickstart.md](./quickstart.md) without opening save logic.

### Implementation for User Story 3

- [X] T011 [US3] Update help and empty-query strings in `bot/internal/core/copy.go` to mention full Spotify album URLs and share short links alongside free-text; update expectations in `bot/internal/core/core_test.go` if help copy is asserted there
- [X] T012 [US3] Refresh [specs/005-save-album-spotify-url/quickstart.md](./quickstart.md) with non-technical examples; add or extend a short operator blurb in `bot/README.md` or repo `README.md` pointing to this quickstart so **SC-004** is satisfied without reading source

**Checkpoint**: **SC-004** — help + quickstart both document link paste.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Redirect unit tests, logging, full suite green, structured e2e-style test per spec NFR, **SC-005** acceptance review (**T017**).

- [X] T013 [P] Add `httptest` coverage for redirect allowlist and hop limits in `bot/internal/metadata/spotify_redirect_test.go`
- [X] T014 Add structured `slog` fields for link-parse / redirect / `LookupSpotifyAlbumByID` outcomes in `bot/internal/core/save_album.go` and `bot/internal/metadata/spotify.go` (no secrets; avoid full URLs at Info if sensitive) per [plan.md](./plan.md); respect **context** deadlines and HTTP client timeouts so **sub-step** budgets (e.g. short-link client) plus album fetch fit within [spec.md](./spec.md) **~15s** user-visible guideline
- [X] T015 Run `cd bot && go test ./...` and resolve failures; run `gofmt` on touched packages; run repository **lint** command if documented (e.g. `Makefile`, `README`, CI config)
- [X] T016 [P] Add one **isolated** end-to-end-style test in `bot/internal/core/save_album_test.go` (or adjacent `*_test.go`) covering **short-link** resolution + `LookupSpotifyAlbumByID` + successful **persist** using fakes, per [spec.md](./spec.md) Testing NFR
- [X] T017 Complete **SC-005** **acceptance** pass: review **sample** structured logs (link-save paths) and **default** user-facing replies for link success/failure; confirm **no** secrets, tokens, or raw provider dumps; record outcome in [checklists/requirements.md](./checklists/requirements.md) Notes or PR description

---

## Dependencies & Execution Order

### Phase Dependencies

```text
Phase 1 (T001)
    └── Phase 2 (T002–T006)  ← blocks all user stories
            ├── Phase 3 US1 (T007–T008)  ← MVP
            ├── Phase 4 US2 (T009–T010)  ← depends on T007 existing
            ├── Phase 5 US3 (T011–T012)  ← copy/docs; easiest after T007
            └── Phase 6 (T013–T017)  ← after behavior stable
```

### User Story Dependencies

| Story | Depends on | Independent test |
|-------|------------|------------------|
| **US1** (P1) | Phase 2 complete | Quickstart happy paths; single save, no chooser |
| **US2** (P1) | T007 (link branch in `save_album.go`) | Failure matrix; `example.com` → `Search(full)` |
| **US3** (P2) | T007 recommended (accurate product behavior) | Help + quickstart mention links + free text |

US1 and US2 are both P1: deliver **US1** first as MVP, then **US2** hardening. **US3** can follow or overlap copy-only work once **T007** lands.

### Within Each User Story

- **US1**: Routing (`T007`) before or in parallel with parser tests (`T008`) once `spotify_album_url.go` is stable
- **US2**: Copy (`T009`) then or parallel with `save_album_test.go` (`T010`)
- **US3**: In-bot strings (`T011`) then docs (`T012`)

### Parallel Opportunities

| Phase | Parallel tasks |
|-------|----------------|
| 2 | After **T002**: **T003** ∥ (**T004** + **T005** with coordinated `orchestrator.go` merge); **T006** after interface + `*Chain` methods |
| 3 | **T008** ∥ **T007** if parser API is fixed (different files) |
| 4 | **T010** ∥ **T009** (different files) |
| 6 | **T013** ∥ **T016**; **T014** serial or after T007–T010; **T017** after **T014** and representative runs |

---

## Parallel Example: User Story 1

```bash
# After Phase 2: wire save path and parser tests
Task T007 — bot/internal/core/save_album.go
Task T008 — bot/internal/core/spotify_album_url_test.go   # [P] if parser stable
```

## Parallel Example: User Story 2

```bash
Task T009 — bot/internal/core/copy.go
Task T010 — bot/internal/core/save_album_test.go            # [P] vs T009
```

## Parallel Example: User Story 3

```bash
Task T011 — bot/internal/core/copy.go, bot/internal/core/core_test.go
Task T012 — specs/005-save-album-spotify-url/quickstart.md, README path
```

## Parallel Example: Phase 2 (after T002)

```bash
Task T003 — bot/internal/core/spotify_album_url.go
# Coordinate merge order:
Task T004 — bot/internal/metadata/spotify_redirect.go + orchestrator.go
Task T005 — bot/internal/metadata/spotify.go + orchestrator.go
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Complete Phase 1–2 (T001–T006)  
2. Complete Phase 3 (T007–T008)  
3. **STOP and validate**: [quickstart.md](./quickstart.md) happy paths + `go test ./...`

### Incremental Delivery

1. Add Phase 4 (US2) — failure classes and regression tests  
2. Add Phase 5 (US3) — help + operator docs  
3. Add Phase 6 — redirect tests, logging, e2e-style test, suite green, **SC-005** review (**T017**)  

### Parallel Team Strategy

1. Team finishes Phase 2 together (watch `orchestrator.go` conflicts).  
2. After Phase 2: one developer owns **T007** + **T009** + **T011** (save + copy); another owns **T008**, **T010**, **T013**, **T016**, **T017** (tests + release review).  
3. Merge **T007** before broad failure-test expectations that assume the branch exists.

---

## Notes

- **Total tasks**: **17** (T001–T017)  
- **Per story**: US1 → 2 (T007–T008); US2 → 2 (T009–T010); US3 → 2 (T011–T012); Setup 1; Foundational 5; Polish 5 (**T013**–**T017**)  
- **[P]** tasks: T003, T004, T005, T008, T010, T013, T016  
- Avoid vague tasks; every task names concrete files or docs
