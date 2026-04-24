# Tasks: LifeSoundTrack — save album command (003)

**Regenerated**: 2026-04-24 (`/speckit-tasks`) | **Prerequisite script**: [`.specify/scripts/bash/check-prerequisites.sh --json`](../../.specify/scripts/bash/check-prerequisites.sh) (paths from [`.specify/feature.json`](../../.specify/feature.json) → `specs/003-save-album-command`).

**Input**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md), [contracts/](contracts/), [research.md](research.md), [quickstart.md](quickstart.md)

**Context (authoritative)**: Metadata **Spotify → iTunes → Last.fm → MusicBrainz** with **`LST_METADATA_ENABLE_*`** **flags**; disambig **≤2** **distinct** **`ALBUM_TITLE | ARTIST (YEAR)`** **lines** + **Other** when needed; **no** **Redis**; **PostgreSQL** `disambiguation_sessions` **only** when a **true** two-choice list is shown. **2026-04-24** — **FR-009** / **SC-007**: **collapse** **candidates** **sharing** the **same** **user-visible** **label** before **any** disambig **prompt**; **first** in **relevance** **order** **wins** among **equivalents**; if **one** **distinct** **label** **remains** → **save** **without** **session** (see [contracts/album-command.md](contracts/album-command.md) **single_effective_match**).

**Principle**: **No Redis**; **provider** on `core.AlbumCandidate` per [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md).

**Format**: `- [ ]` or `- [X]` + `TNNN` + optional `[P]` + `[USn]` on story tasks + **concrete** `bot/...` or `specs/...` paths.

**Note**: `setup-plan.sh` can **overwrite** [plan.md](plan.md) — back up before running it.

---

## Phase 1: Setup (dependencies and env surface)

**Purpose**: Tooling; no new `go mod` work expected beyond Spotify stdlib; env surface documented.

- [X] T001 [P] Confirm `bot/go.mod` includes `github.com/sony/gobreaker` and HTTP/JSON stdlib; run `go mod tidy` in `bot/` if any new direct deps are added for Spotify (e.g. none if using stdlib only)

**Checkpoint**: `go build -C bot ./...` passes before orchestrator changes.

---

## Phase 2: Foundational — config, feature flags, Spotify adapter, chain refactor

**Purpose**: **BLOCKS** all user-story work for **FR-002**. Wire **`LST_METADATA_ENABLE_*`**, **Spotify** **Client** **Credentials** **`bot/internal/metadata/spotify.go`**, **refactor** `bot/internal/metadata/orchestrator.go` to **Spotify → iTunes → Last.fm → MusicBrainz**; **if** all flags **off** → **`core.ErrAllProvidersExhausted`** before HTTP.

- [X] T002 [P] Add `parseMetadataBool` (or equivalent) in `bot/internal/config/config.go`: env unset → **true**; `true`/`1`/`yes`/`on` → true; `false`/`0`/`no`/`off` (case-insensitive) → false
- [X] T003 [P] Extend `Config` in `bot/internal/config/config.go` with: `LST` flags as four `bool` fields, `SpotifyClientID` / `SpotifyClientSecret` as `string`
- [X] T004 Update `config.FromEnv()` in `bot/internal/config/config.go` to read `LST_METADATA_ENABLE_SPOTIFY`, `LST_METADATA_ENABLE_ITUNES`, `LST_METADATA_ENABLE_LASTFM`, `LST_METADATA_ENABLE_MUSICBRAINZ`, `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET` per [quickstart.md](quickstart.md)
- [X] T005 [P] Update `bot/.env.example` (or canonical env sample) with all new variables and a one-line pointer to [specs/003-save-album-command/quickstart.md](quickstart.md)
- [X] T006 [P] Implement `bot/internal/metadata/spotify.go`: OAuth **client credentials** + album search, map to `[]core.AlbumCandidate`, set `Provider: "spotify"`, `ProviderRef`, `ArtURL`/`Genres`/`Year` when present; follow Spotify docs for headers/ToS
- [X] T007 Add **four** breakers in `bot/internal/metadata/orchestrator.go` (names `spotify`, `itunes`, `lastfm`, `musicbrainz`); align with [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md)
- [X] T008 Refactor `(*Chain).Search` in `bot/internal/metadata/orchestrator.go`: **(1)** no enabled catalogs → `core.ErrAllProvidersExhausted`; **(2)** each enabled step in order; **(3)** first non-empty `[]core.AlbumCandidate` with `capTop2`; **(4)** fallthrough on empty/flag/skip; **(5)** Spotify enabled but creds missing → log **Warn** (no secret value) and fall through per [plan.md](plan.md)
- [X] T009 Wire new **config** into **`metadata.NewChain`** and **`bot/cmd/bot/main.go`** (and test constructors in `bot/internal/metadata/*_test.go`)

**Checkpoint**: `go test -C bot ./...` passes; all flags false → `ErrAllProvidersExhausted` without HTTP; Spotify off + iTunes on can return candidates.

---

## Phase 3: User Story 1 — Free-form single-match save (Priority: P1)

**Goal**: Non-empty query → **metadata** → **single** strong match → **persist**; empty → help, no `Search` ([spec](spec.md) **US1**).

- [X] T010 [US1] In `bot/internal/core/save_album_test.go`, keep/adjust fakes for `single_match` / **`Provider: "spotify"`** where relevant
- [X] T011 [P] [US1] In `bot/internal/adapter/telegram/`, re-verify **empty** query and **512**-rune cap per [contracts/album-command.md](contracts/album-command.md)

---

## Phase 4: User Story 1b — Disambiguation (≤2 distinct labels + Other) (Priority: P1)

**Goal**: **Two** **or** **more** **distinct** **`ALBUM_TITLE | ARTIST (YEAR)`** **labels** → **at** **most** **2** + **Other**; **no** **save** **until** **pick**; **Other** = refinement only ([spec](spec.md) **US1b**, **FR-009**).

- [X] T012 [US1b] Ensure orchestrator `capTop2` and `bot/internal/adapter/telegram/` use **≤2** **rows** in session JSON for the **offered** list
- [X] T013 [P] [US1b] Adjust `bot/internal/core/save_album_test.go` disambig tests for **Spotify** / **relevance** if assumptions broke

**Checkpoint**: **SC-005** and **disambig** with **two** **different** **labels** still verifiable in tests

---

## Phase 4b: User Story 1b — Equivalent label collapse (SC-007 / FR-009) 🎯

**Goal**: If **all** raw candidates **share** one user-visible label (`formatAlbumLine` in `bot/internal/core/save_album.go`), **do not** show disambig (no two identical lines, no pointless `disambiguation_sessions` row)—**persist** the **first** (highest-relevance) row among equivalents per [spec](spec.md) and [contracts/album-command.md](contracts/album-command.md) **single_effective_match**.

**Independent test**: Fake `Search` returns **≥2** `AlbumCandidate` with **identical** `formatAlbumLine` → **`OutcomeSaved`** (or same path as one raw match) and **no** `CreateDisambiguationSession` / **`OutcomeDisambig` with duplicate labels**; two **different** **labels** → still **`OutcomeDisambig`** with **2** **distinct** **button** **texts**.

- [X] T026 [US1b] In `bot/internal/core/`, implement **deduplication** **by** `formatAlbumLine` **(see** `save_album.go`) **preserving** **relevance** **order** **(first** **occurrence** **wins)**: e.g. new **`dedupeCandidatesByAlbumLine([]AlbumCandidate) []AlbumCandidate`** in `bot/internal/core/candidates.go` (or `save_album.go`). **After** `Search`, run **dedupe** **before** **`len(cands)`** **branching**: if **deduped** **len** **==** **1** **(including** **N**-**raw**-**rows**-**same**-**label)**, call **`persistSave`** like **single**-**result**; if **≥** **2** **distinct** **labels**, use **`deduped[:min(2,len)]` or** walk **to** **first** **two** **distinct** **—** must **not** **slice** **raw** `cands[:2]` **without** **dedupe** first
- [X] T027 [P] [US1b] In `bot/internal/core/save_album_test.go`, add table cases: **(a)** two candidates same `formatAlbumLine` different `ProviderRef` → **no** disambig / **one** **save**; **(b)** two different labels → disambig with **2** **distinct** **labels** in **`AlbumButtonLabels`**
- [X] T028 [US1b] In `bot/internal/adapter/telegram/`, ensure inline/reply buttons (or text list) never show two identical strings for album rows (defensive if core already dedupes); run `go test ./...` covering `bot/internal/adapter/telegram/` as needed

**Checkpoint**: [spec](spec.md) **SC-007** **and** **FR-009** **collapse** **clause** **covered** by **T027** **+** **manual** **smoke** in [quickstart.md](quickstart.md) **§7**

---

## Phase 5: User Story 2 — Safe failure, flags, fallthrough (Priority: P1)

- [X] T014 [US2] In `bot/internal/core/`, map `ErrAllProvidersExhausted` to `OutcomeProviderExhausted` and `tryAgainCopy()`-style messages without env names in chat
- [X] T015 [P] [US2] `bot/internal/metadata/orchestrator_test.go`: all flags false; Spotify empty, iTunes returns candidates; optional breaker
- [X] T016 [US2] In `bot/internal/adapter/telegram/`, no `InsertSavedAlbum` on `OutcomeProviderExhausted` / `no_match`

---

## Phase 6: User Story 3 — Listener profile (Priority: P1)

- [X] T017 [P] [US3] Re-verify `bot/internal/store/` **UpsertListener** on save + disambig **complete** **paths** in `bot/internal/adapter/telegram/`

---

## Phase 7: User Story 4 — Documentation (Priority: P2)

- [X] T018 [P] [US4] `bot/README.md`: **`LST_METADATA_*`**, **`SPOTIFY_*`**, link to [quickstart.md](quickstart.md)
- [X] T019 [P] [US4] Root `README.md`: one line chain order + pointer to **003** **quickstart**
- [X] T020 [US4] Re-read [specs/003-save-album-command/quickstart.md](quickstart.md) against `config.FromEnv()` after config changes; fix drift

---

## Phase 8: Polish and cross-cutting

- [ ] T021 [P] Run `golangci-lint run ./...` in `bot/`; fix or narrow `//nolint` with comment per constitution **I**
- [X] T022 [P] Run `go test ./... -count=1` and `go vet ./...` from `bot/`
- [X] T023 [P] Log audit: no secrets at `slog` **Info** on new paths ([spec](spec.md) **FR-007**)
- [ ] T024 (optional) [P] If CI exists: `migrate` + `go test` on ephemeral Postgres
- [X] T025 [P] Update [specs/001-lifesoundtrack-bot-commands/contracts/messaging-commands.md](specs/001-lifesoundtrack-bot-commands/contracts/messaging-commands.md) if `/album` or env out of date
- [X] T029 [P] [US4] After **T026–T028**, update [specs/003-save-album-command/quickstart.md](quickstart.md) **§5–7** if behavior/copy changed; **confirm** **duplicate**-**label** **smoke** **bullet** **matches** **reality** *(no copy change required; §5/§7 already describe collapse; **verified** **against** **`TestProcessAlbumQuery_DuplicateUserVisibleLabelSavesFirst`**)*

**Checkpoint**: **merge-ready** when **T026–T028** + **T021** (or documented lint waiver) are satisfied

---

## Dependencies and execution order

1. **T001** → **T002**–**T005** (parallel where **[P]**); **T004** after **T002**–**T003**
2. **T006** then **T007**–**T008** → **T009**
3. **T010**–**T020** after **T009** (overlap allowed)
4. **T026** **depends** on **T008**–**T009** (stable `Search` + `AlbumCandidate`); **T027** with **T026**; **T028** after **T026**
5. **T029** after **T026–T028**; **T021** any time in **Phase** **8** after code stable

### User story order

**Foundational (Phase 2)** → **US1** → **US1b (Phases 4+4b)** → **US2–US3** (already done) → **US4** → **Polish**

### Parallel example

```text
# After T009:
T010 + T011 + T014   # different areas

# After T026:
T027 [P]  # tests
T028      # telegram (after core API stable)
```

---

## Task summary

| Metric | Value |
|--------|------:|
| **Total tasks** | **T001–T029** (29 tasks) |
| **Open (default)** | **T021** (lint tool not on dev PATH), **T024** (optional **CI** **`migrate`**) |
| **By user story** | US1: 2, US1b: 6 (incl. **T026–T028**), US2: 3, US3: 1, US4: 4 (incl. **T029**) |
| **MVP for SC-007** | **T026**–**T029** **complete** (see **`save_album.go`** + **`run_test.go`**) |

**Parallel** **[P]**: T001–T002, T005, T006, T011, T013, T015, T017–T021, T023–T025, T027, T029

---

## Implementation strategy

1. **If** **T001–T025** **already** **merged**: **start** at **T026** **(label** **dedupe** **in** **core**)**.  
2. **Run** **T027** **tests** **before** **merge**.  
3. **T021** + **T029** **before** **release** **or** **PR** **ready**.

---

## Extension hooks (`.specify/extensions.yml`)

**`hooks.before_tasks`** (optional, `speckit.git.commit`): to run: `/speckit.git.commit` or commit manually.

**`hooks.after_tasks`** (optional, `speckit.git.commit`): to run: `/speckit.git.commit` after reviewing **tasks.md**.
