# Tasks: LifeSoundTrack — save album command (003)

**Last validated** (`/speckit-tasks`): 2026-04-23 — structure and IDs unchanged; design alignment re-checked against [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md), [contracts/](contracts/).

**Input**: Design documents from `specs/003-save-album-command/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md) (stories **US1**, **US1b**, **US2**, **US3**, **US4**), [research.md](research.md), [data-model.md](data-model.md), [contracts/album-command.md](contracts/album-command.md), [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md), [quickstart.md](quickstart.md)

**Principle**: **No Redis**; disambiguation in **PostgreSQL** `disambiguation_sessions` and/or in-process memory for single-process dev. **godotenv** / config patterns stay in `bot/internal/config`; new work in `bot/internal/core`, `bot/internal/metadata/`, `bot/internal/store/`, `bot/internal/adapter/telegram/`.

**Format**: Every task is `- [ ] TNNN [P?] [USn?] Description` with a concrete `bot/...` or `specs/003-...` path.

**Tests** (per [spec NFR — Testing](spec.md)): include unit/integration tasks for **orchestrator**, **DB store**, and **disambig** paths; optional **testcontainers** for Postgres.

**Note**: `check-prerequisites.sh --json` succeeds on feature branch **`003-save-album-command`**. Canonical dir: [`.specify/feature.json`](../../.specify/feature.json) → `specs/003-save-album-command/`.

---

## Phase 1: Setup (dependencies & local stack)

**Purpose**: Add Go modules and Docker Compose so Postgres-backed development is possible; align with [plan.md](plan.md) and constitution **VIII** in [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md).

- [X] T001 Add dependencies in `bot/go.mod`: e.g. `github.com/jackc/pgx/v5`, `github.com/sony/gobreaker`, `github.com/golang-migrate/migrate/v4` (and Postgres driver for migrate); run `go mod tidy` in `bot/`
- [X] T002 [P] Extend root `compose.yaml` with **PostgreSQL 15+** (named volume, `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB`, healthcheck); **no** Redis service; document port and env in `specs/003-save-album-command/quickstart.md` in the same change or when completing **T030**–**T031** (US4 docs)
- [X] T003 [P] Add or extend `bot/.env.example` with `DATABASE_URL`, `LASTFM_API_KEY` (optional), and a note on **MusicBrainz** `User-Agent` / rate policy per [research.md](research.md)

**Checkpoint**: `docker compose up -d` starts Postgres; `bot/go.mod` contains new `require` blocks.

---

## Phase 2: Foundational (migrations, config, ports, store skeleton)

**Purpose**: **Blocking** for all user stories: schema, config, `MetadataOrchestrator` **port**, store access, and chained **orchestrator** per [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md).

- [X] T004 Add `golang-migrate` SQL `bot/migrations/000001_init_listeners_saved_albums_disambig.up.sql` and matching `.down.sql` implementing [data-model.md](data-model.md) tables `listeners`, `saved_albums`, `disambiguation_sessions` (incl. `pgcrypto` / `gen_random_uuid()`)
- [X] T005 Extend `bot/internal/config` to load `DATABASE_URL`, `LASTFM_API_KEY` (optional), and a **fixed app** `User-Agent` string for **MusicBrainz** (e.g. `LifeSoundTrack/1.0 (+repo-url)`) with **no** secrets in `Default`
- [X] T006 [P] Create `bot/internal/store/`: `OpenPool(ctx, databaseURL)` with `pgxpool.Pool` and a helper to run `migrate` **up** from `bot/migrations` (document in `bot/README.md` or quickstart: dev auto-migrate vs. prod init job)
- [X] T007 [P] In `bot/internal/core/`, add domain types: `AlbumCandidate` and save-flow result kinds per [contracts/album-command.md](contracts/album-command.md) (`empty_query`, `candidates`, `single_match`, `no_match`, `provider_exhausted`, `saved`) and wire names to [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md)
- [X] T008 Define `MetadataOrchestrator` **interface** in `bot/internal/core` (or `bot/internal/metadata`) as in [contracts/metadata-orchestrator.md](contracts/metadata-orchestrator.md)
- [X] T009 [P] Implement `bot/internal/metadata/musicbrainz`: `release` search, **1 rps** throttle, JSON parse to `[]AlbumCandidate`, **User-Agent** on every request
- [X] T010 [P] Implement `bot/internal/metadata/lastfm` for `album.search` when `LASTFM_API_KEY` is set; map to `AlbumCandidate` (or skip this ring if no key)
- [X] T011 [P] Implement `bot/internal/metadata/itunes` (Search API) mapping `collectionName`, `artistName`, `releaseDate` → `AlbumCandidate` per [plan.md](plan.md)
- [X] T012 Implement `bot/internal/metadata/orchestrator` chaining **MusicBrainz → Last.fm → iTunes** with **gobreaker** per provider, cooldown, and `ErrNoMatch` / `ErrAllProvidersExhausted` per [metadata-orchestrator.md](contracts/metadata-orchestrator.md)
- [X] T013 [P] Store methods: `UpsertListener`, `InsertSavedAlbum`, `CreateDisambiguationSession`, `GetDisambiguationSession`, `DeleteDisambiguationSession` in `bot/internal/store/` per [data-model.md](data-model.md)
- [X] T014 Wire `bot/cmd/bot/main.go`: after config load, open store pool; optional **dev-only** `migrate` up; pass pool + `MetadataOrchestrator` (and disambig / save deps) into `telegram.Run` (signature change) or a small `internal/app` wire package

**Checkpoint**: `go build ./...` in `bot/`; `migrate up` creates tables; `Search("test")` returns candidates in a unit test with HTTP **mocked** or a fake `MetadataOrchestrator`.

---

## Phase 3: User Story 1 — Free-form single-match save (Priority: P1) — MVP

**Goal**: Non-empty free-form text → metadata → one **plan-defined** high-confidence match → persist **listener** + `saved_albums`; **empty** query → help, **no** provider call ([spec](spec.md) **US1**).

**Independent test**: `/album <query>` returns one top candidate; row in `saved_albums`; `/album` alone — no `Search` call, no new album row.

- [X] T015 [US1] In `bot/internal/core/`, implement the **save** path: `Search`, if **exactly one** top candidate (policy: single-match threshold), `UpsertListener` + `InsertSavedAlbum` and return `saved` with a short user-facing line per [spec](spec.md) and [data-model.md](data-model.md)
- [X] T016 [US1] In `bot/internal/adapter/telegram/`, register `/album` **handler**; **trim** query; on **empty** return core outcome `empty_query` and do **not** call `Search`
- [X] T017 [US1] Map Telegram `User` → `UpsertListener` fields (`source=telegram`, `external_id` as string, `display_name`, `username`) per [spec](spec.md) **US3** (minimal wiring; full “every path” upsert in **T028**)
- [X] T018 [US1] Enforce `query` **max length** (e.g. **512 runes** per [contracts/album-command.md](contracts/album-command.md)) in the Telegram adapter and/or core; on exceed, return a short **“too long”** hint and **no** `Search` (or truncated search if spec/docs explicitly choose truncation — must match [quickstart.md](quickstart.md) / contract)
- [ ] T019 [P] [US1] Unit tests: `TestEmptyQuery`, `TestSingleMatchSaves` (and optional over-long query) with **fake** `MetadataOrchestrator` and store fake or `sqlmock` in `bot/internal/core/*_test.go` and/or `bot/internal/store/*_test.go`

**Checkpoint**: With real bot + Postgres: one successful single-match save; empty `/album` and over-limit query do not call live metadata inappropriately.

---

## Phase 4: User Story 1b — Disambiguation (2–3 options) (Priority: P1)

**Goal**: **2+** candidates → show **up to 3** by relevance; **inline buttons** or **numbered** follow-up; **no** `InsertSavedAlbum` until **pick**; state in `disambiguation_sessions` / Postgres ([FR-009](spec.md), [plan.md](plan.md)).

**Independent test**: Fake orchestrator returns **≥2** candidates; user picks **2**; one correct `saved_albums` row; **no** row before pick.

- [X] T020 [US1b] In `bot/internal/core/`, when `Search` returns **2+** candidates, return `candidates` (bounded, sorted); **no** `InsertSavedAlbum` until a **Pick** with 1-based index
- [X] T021 [US1b] Persist disambig in `disambiguation_sessions` with `candidates` JSONB and `expires_at` (e.g. 15m) in `bot/internal/store/`
- [X] T022 [US1b] In `bot/internal/adapter/telegram/`, send **inline keyboard** (1..N) when available; else **numbered** message body per [contracts/album-command.md](contracts/album-command.md)
- [X] T023 [US1b] Handle **callback** and **next message** digits **1**–**3**: load session, `InsertSavedAlbum` for that candidate, `DeleteDisambiguationSession`, confirm; expired/unknown session: safe user copy
- [ ] T024 [P] [US1b] Test: two candidates, pick second → persisted row matches; **assert no** `InsertSavedAlbum` before pick

**Checkpoint**: “Red”-style disambig in sandbox; **FR-009** satisfied.

---

## Phase 5: User Story 2 — Safe failure (no leak, no false save) (Priority: P1)

**Goal**: **no_match**, **provider_exhausted**, and transient errors → short, safe messages; **no** new `saved_albums` on failure ([spec](spec.md) **US2**, **SC-002**).

**Independent test**: Fakes for empty search, open breakers, 503; **no** new `saved_albums` row; logs show **class**, not full HTTP body at default level.

- [X] T025 [US2] In core, map `ErrNoMatch` / `ErrAllProvidersExhausted` and transient errors to user copy; log **error class** + `provider=`, not raw secrets or full JSON body ([FR-007](spec.md))
- [X] T026 [US2] In `bot/internal/adapter/telegram/`, **never** `InsertSavedAlbum` on `no_match` / `provider_exhausted` / error paths; align with [contracts/album-command.md](contracts/album-command.md)
- [ ] T027 [P] [US2] Tests: no match, all providers failing; assert **no** new `saved_albums` (store fake or test DB)

**Checkpoint**: Spot-check: `LOG_LEVEL=INFO` has no token/DSN; matches **SC-002** / **SC-003** intent.

---

## Phase 6: User Story 3 — Listener profile updates (Priority: P1)

**Goal**: `UpsertListener` on every successful listener touch; **one** row per `(source, external_id)`; **update** `display_name` / `username` on change ([spec](spec.md) **US3**).

**Independent test**: Two saves with different mocked **names**; **one** `listeners` row, **updated** `updated_at`.

- [X] T028 [US3] On every `/album` and disambiguation resolution path, pass **current** Telegram user into `UpsertListener` in `bot/internal/adapter/telegram/` + `bot/internal/store/`
- [ ] T029 [P] [US3] Test or integration check: `UNIQUE (source, external_id)` and **update** path

**Checkpoint**: No duplicate `listeners` rows for the same Telegram user.

---

## Phase 7: User Story 4 — Documentation and discovery (Priority: P2)

**Goal**: New contributors can run the feature using docs; link **001** and **002** as needed ([spec](spec.md) **US4**).

**Independent test**: Another developer follows [quickstart.md](quickstart.md) with a sandbox token and local Postgres and completes a **save** and a **no-match** path.

- [X] T030 [P] [US4] Update `bot/README.md` with `/album`, `DATABASE_URL`, `migrate` command, and link to [specs/003-save-album-command/quickstart.md](quickstart.md)
- [X] T031 [P] [US4] Add a short subsection to root `README.md` for feature **003** and link to [quickstart.md](quickstart.md) and [001 quickstart](specs/001-lifesoundtrack-bot-commands/quickstart.md) where useful
- [X] T032 [US4] Update [specs/001-lifesoundtrack-bot-commands/contracts/messaging-commands.md](specs/001-lifesoundtrack-bot-commands/contracts/messaging-commands.md) to document domain **`save_album` / `album` →** `/album` to avoid **001** vs **003** drift (add an amendment line if 001 v1 “out of scope” text applies)

**Checkpoint**: [quickstart.md](quickstart.md) steps 1–4 are reproducible with **T002** compose and **T030**–**T031** doc updates.

---

## Phase 8: Polish and cross-cutting

- [X] T033 [P] Update `bot/internal/core/copy.go` and `command.go` (or adjacent files) so **/help** and **/start** mention **`/album`** per constitution **IV** in [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md)
- [X] T034 [P] Update `bot/Dockerfile` and root `compose.yaml` **bot** service to pass `DATABASE_URL` and copy `bot/migrations` as needed
- [ ] T035 [P] Run `golangci-lint run ./...` in `bot/` and fix or narrowly document `nolint`
- [X] T036 From `bot/`, run `go test ./... -count=1` and `go vet ./...` clean
- [ ] T037 (optional) [P] If repo adds CI, add a job: `migrate -path bot/migrations -database "$DATABASE_URL" up` then `go test` against an ephemeral Postgres
- [X] T038 [P] Log audit: `slog` in metadata, store, and adapter paths does not log `TELEGRAM_BOT_TOKEN`, `DATABASE_URL` password, or full provider JSON at **Info** (per [FR-007](spec.md))

**Checkpoint**: Tests green; contracts still match the code.

---

## Dependencies and execution order

1. **T001**–**T003** (Setup) before **T004**+.
2. **T004**–**T014** (Foundational) before **all** user stories.
3. **US1 (T015–T019)** before or overlapping **US1b**; **T020**–**T024** need **T015** core result types in place.
4. **US2 (T025–T027)** after search + adapter paths exist; can overlap **US3 (T028–T029)**.
5. **US4 (T030–T032)** and **Polish (T033–T038)** after main flows are stable.

---

## Parallel example

```text
# After Foundational:
# Dev A: T022–T023 (Telegram disambig)   Dev B: T019 (US1 tests) + T010 (Last.fm) [once interfaces exist]
# After T015:
# T018 (query cap) can pair with T017 (different files) if T015 is done first.
```

---

## Task summary

| Metric | Value |
|--------|------:|
| **Total tasks** | 38 (T001–T038) |
| **MVP (smallest shippable slice)** | Phases 1–3 through **T019** (US1) + migrations |
| **US1** | T015–T019 |
| **US1b** | T020–T024 |
| **US2** | T025–T027 |
| **US3** | T028–T029 |
| **US4** | T030–T032 |
| **Polish** | T033–T038 |

**Format validation**: All task lines use `- [ ] TNNN`, include `bot/...` or `specs/...` in the description, and use `[US#]` on user-story phase tasks (none on Setup, Foundational, or Polish—except T037 optional CI—per spec kit convention: Polish has no [US#]).

---

## Implementation strategy

### MVP first (US1 only)

1. **Phase 1** + **Phase 2** (through **T014**)
2. **Phase 3** (through **T019**)
3. **Stop**: confirm single-match + empty + over-long query behavior before disambig (Phase 4)

### Incremental delivery

1. Add **Phase 4** (US1b) → E2E disambig
2. Add **Phase 5** (US2) if not already covered by **T015**+ error mapping
3. **Phase 6** (US3) — tighten `UpsertListener` everywhere
4. **Phase 7** (US4) + **Phase 8** (polish, lint, logs)

### Parallel team (after Phase 2)

- One dev: `bot/internal/metadata/*` and **T012**
- Another: `bot/internal/store/*` + **T004**
- Another: `bot/internal/core/*` + **T007** / **T008** — align on `AlbumCandidate` and errors first

---

## Notes

- Migrations: **versioned** only; do not edit an applied `up` in place after deploy; add `000002_...` for schema changes.
- **MusicBrainz** commercial use: [research.md](research.md) **non-commercial** assumption.
- **iTunes** Search API: [Apple’s terms](https://www.apple.com/legal/internet-services/); **metadata** only.
