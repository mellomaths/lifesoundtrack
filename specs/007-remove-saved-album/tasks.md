# Tasks: LifeSoundTrack — `/remove` saved album (007)

**Input**: Design documents from `specs/007-remove-saved-album/` (generated with **`/speckit-tasks`**, refreshed **2026-04-26**).  
**Prerequisites**: [plan.md](plan.md) (Go **1.25**, **`rmp:`** **inline** **keyboard**, **SC-003**), [spec.md](spec.md) (**FR-006** buttons + text alternate), [research.md](research.md) **§4**, **§8**, **§10**, [data-model.md](data-model.md), [contracts/remove-command.md](contracts/remove-command.md), [quickstart.md](quickstart.md)

**Tests**: [spec.md](spec.md) NFR **Testing**, **SC-003** (Telegram **button** path + **text** **regression**). Table tests and routing tests as listed per phase.

**Organization**: Phases **1–9** cover **shipped** **two-tier** **match**, **disambig**, **numeric** **text** **pick**, **help**, **polish**, **partial** **tier** (**T018–T021**). **Phase** **10** adds **FR-006** / **SC-003** **Telegram** **inline** **buttons** and **`rmp:`** **callbacks** (not yet in `bot/` **as** of **task** **generation**).

## Task line format

Each line: `- [ ]` or `- [x]` + **Task ID** (`T001`…) + optional **`[P]`** + optional **`[USn]`** + description **with file paths** (`bot/...`).

- **`[P]`**: Can run in parallel in the same phase (no blocking dep on an incomplete sibling).
- **`[USn]`**: Only on user-story phase tasks (omit on Setup, Foundational, Polish).

## Path Conventions

- **Bot module**: `bot/` at repository root (see [plan.md](plan.md))

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Align on contracts and existing persistence before code changes.

- [x] T001 [P] Skim `specs/007-remove-saved-album/contracts/remove-command.md` and `specs/007-remove-saved-album/data-model.md` against `bot/internal/store/disambig.go` and `bot/migrations/00001_init_listeners_saved_albums_disambig.sql` for `disambiguation_sessions` shape vs **`remove_saved`** JSON object (`kind` + `candidates` array of `{id, label}`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Store reads/deletes and **`ParseRemoveLine`** must exist before **`HandleRemove`**.

**CRITICAL**: User story work starts after **T002–T004** complete.

- [x] T002 [P] Add `ListSavedAlbumRowsForListener` in `bot/internal/store/saved_albums_list.go` returning `id`, `title`, `primary_artist`, `year` for `listener_id` (in-Go match per [research.md](research.md) §2; no new SQL for title filter)
- [x] T003 [P] Add `DeleteSavedAlbumForListener` in `bot/internal/store/saved_albums_delete.go` implementing `DELETE FROM saved_albums WHERE id = $1::uuid AND listener_id = $2::uuid` with boolean/rows affected; extend `bot/internal/store/saved_albums_delete_test.go` (or equivalent) so another listener’s UUID does not delete another listener’s row
- [x] T004 [P] Add `bot/internal/core/parse_remove.go` with `ParseRemoveLine` and `RemovePickIndexFromText` per [contracts/remove-command.md](contracts/remove-command.md) and [research.md](research.md) §5

**Checkpoint**: Foundation ready — **US1+** implementation can start.

---

## Phase 3: User Story 1 — Remove with one exact match (Priority: P1) — MVP core

**Goal**: **`/remove <query>`** when **FR-003(1) exact tier** finds **exactly one** row → delete, success copy ([spec.md](spec.md) **US1**, **FR-005**).

**Independent Test**: Save one album; **`/remove`** with case/spacing variants of the full title; confirm removed via **`/list`** (or DB).

- [x] T005 [US1] Add `bot/internal/core/remove_saved.go` with `(*LibraryService).HandleRemove` for the **one exact-match** path: `NormalizeArtistQuery`, `DeleteSavedAlbumForListener`, success string per [contracts/remove-command.md](contracts/remove-command.md); reject over **`MaxQueryRunes`** with user copy and no delete ([spec](spec.md) **A3**; depends on **T002–T004**)
- [x] T006 [US1] Extend `bot/internal/adapter/telegram/run.go` `handleMessage` with **`/remove`** after **`/album`** and **`/list`**, calling **`HandleRemove`**; numeric **remove** pick before **album** pick is **T011** (depends on **T005**)

**Checkpoint**: Single **exact**-match removal works end-to-end in private chat.

---

## Phase 4: User Story 2 — Not found, empty query, over-length (Priority: P1)

**Goal**: **0** qualifying rows (both tiers) → not found; **empty** remainder → usage; **>3** partials (tier 2 only) after Phase 9 → narrow message; no silent delete ([spec.md](spec.md) **US2**, **FR-004**, Edge cases).

**Independent Test**: **`/remove`** nonsense; bare **`/remove`**; after Phase 9, **`>3`** partials path.

- [x] T007 [US2] Extend `bot/internal/core/remove_saved.go` **HandleRemove** and `bot/internal/core/copy.go` for **not-found**, **empty** query, and **too-long** messages per [contracts/remove-command.md](contracts/remove-command.md)

**Checkpoint**: **US2** base scenarios; **>3** partials completed in **Phase 9** (**T019**).

---

## Phase 5: User Story 3 — Normalization and match helpers (Priority: P1)

**Goal**: Table tests for **`ParseRemoveLine`**, **normalize** equivalence, and **exact**/**partial** match helpers where tested ([spec.md](spec.md) **US3**).

**Independent Test**: `cd bot && go test ./internal/core -run 'Test(ParseRemove|...)'` passes.

- [x] T008 [P] [US3] Add `bot/internal/core/parse_remove_test.go` for `ParseRemoveLine`, **`NormalizeArtistQuery`** pairs, and **`RemovePickIndexFromText`**; align **length** with **`MaxQueryRunes`** per [research.md](research.md) **§2a**

**Checkpoint**: Parsing and normalize equivalence locked by tests. **Partial-tier** match tests live in **T020** (Phase 9) after **T019**.

---

## Phase 6: User Story 4 — Multi-match and numeric pick (Priority: P2)

**Goal**: **≥2** exact matches **or** **1–3** partial matches (no exact) → **`remove_saved`** session + no silent delete; user replies **1…N**; **TryProcessRemovePick** before **album** pick ([spec.md](spec.md) **US4**, **FR-006** text path).

**Independent Test**: Two saves with same normalized title → **`/remove`** → pick **1** or **2**; or one **partial** disambig → pick **1**.

- [x] T009 [US4] Extend `bot/internal/core/remove_saved.go` for **multi–exact-match** and **`FormatSavedAlbumLine`**, `CreateDisambiguationSession` with object JSON, numbered prompt (depends on **T005**)
- [x] T010 [US4] Add `(*LibraryService).TryProcessRemovePick` in `bot/internal/core/remove_saved.go`: **LatestOpen** session, `kind == remove_saved`, delete by **id** with listener scoping, session cleanup (depends on **T009**)
- [x] T011 [US4] Update `bot/internal/adapter/telegram/run.go` so **1–99** decimal-only replies call **`TryProcessRemovePick`** before `OneBasedPickFromText` + **`SaveService.ProcessPickByIndex`** (depends on **T010**)
- [x] T012 [P] [US4] Extend `bot/internal/adapter/telegram/routing_test.go` for **`/remove`** and remove-disambig **numeric** order vs **`/list`** / **`/album`**

**Checkpoint**: **US4** text pick; **inline** **buttons** added in **Phase** **10** (**T022–T027**).

---

## Phase 7: User Story 5 — Help (Priority: P2)

**Goal**: **`helpCopy`** lists **`/remove`** and remains a full inventory (**FR-008**, **SC-004**).

**Independent Test**: Assert **`/remove`** in help text.

- [x] T013 [US5] Update `helpCopy` in `bot/internal/core/copy.go` with **`/remove`** and accurate command list per [spec.md](spec.md) **FR-008**
- [x] T014 [P] [US5] Add test in `bot/internal/core/core_test.go` (or `copy_test.go`) asserting help contains **`/remove`**

**Checkpoint**: **US5** + **SC-004** automated check.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: **README**, **quickstart**, full **`go test`**, optional logging.

- [x] T015 [P] Update root `README.md` to mention **`/remove`** in command inventory where applicable (`.specify/memory/constitution.md` principle **IX**)
- [x] T016 [P] Align `specs/007-remove-saved-album/quickstart.md` with shipped behavior; add manual steps for **not found**, **disambig**, **help**
- [x] T017 Run `cd bot && go test ./...` and fix failures; optional structured remove-outcome logs (not blocking per [spec.md](spec.md) NFR)

---

## Phase 9: Partial match + >3 cap (clarification 2026-04-26)

**Purpose**: **FR-003(2)** partial substring tier and **>3** partials → narrow message only ([spec.md](spec.md) Clarifications; [plan.md](plan.md) Summary).

- [x] T018 [P] Update `specs/007-remove-saved-album/contracts/remove-command.md` and `specs/007-remove-saved-album/research.md` §2 for two tiers + cap
- [x] T019 [US1] Extend `bot/internal/core/remove_saved.go` **HandleRemove**: after **exact** tier, **partial** tier with **`strings.Contains`** on normalized strings; **0** / **1–3** / **>3** branches; `removeTooManyPartialsCopy` in `bot/internal/core/copy.go` (or adjacent helpers); **openRemoveDisambiguation** for **1–3** including single partial ([spec](spec.md) **FR-005**)
- [x] T020 [P] [US3] Add `bot/internal/core/remove_saved_test.go` for **`exactTitleMatches`** / **`partialTitleMatches`** (substring cases, e.g. Abbey Road vs remastered title) — depends on **T019**
- [x] T021 [P] Run `cd bot && go test ./...`; update `specs/007-remove-saved-album/quickstart.md` with **partial** and **>3** manual steps (depends on **T016** baseline + **T019**)

**Checkpoint**: Abbey Road vs Abbey Road (Remastered) flow; no enumeration when **>3** partials.

---

## Phase 10: Telegram inline keyboard + `rmp:` callback (FR-006, SC-003)

**Purpose**: [spec.md](spec.md) **FR-006** / **SC-003** — disambig **must not** be **text-index-only** on **Telegram**; **`rmp:`** **`callback_data`** per [research.md](research.md) **§4**, **§10** and [contracts/remove-command.md](contracts/remove-command.md). **Numeric** **text** **pick** **remains** **required** **alternate**.

**Independent Test**: Two+ matches → message shows **inline** **buttons**; tap **removes**; sending **`1`** still works (**SC-003**). **`callback_data` ≤ 64** **bytes**.

- [x] T022 [US4] Refactor `(*LibraryService).HandleRemove` in `bot/internal/core/remove_saved.go` to return a **structured** result (e.g. **`RemoveReply`** with **text**, **outcome**, **`DisambigSessionID`**, **`InlineLabels`** or **candidate** **count**) instead of **`(string, error)`** alone, so `bot/internal/adapter/telegram/run.go` can attach **ReplyMarkup** when disambig is created; update `openRemoveDisambiguation` to **return** **`CreateDisambiguationSession`** **id** into that struct (per [plan.md](plan.md) Summary)
- [x] T023 [US4] Add `removeDisambigInlineKeyboard(sessionID string, labels []string)` (or equivalent) in `bot/internal/adapter/telegram/run.go` building **`CallbackData`** **`rmp:<uuid>:<1-based-index>`** for each button; **truncate** **button** **text** with existing **`telegramInlineButtonTextMax`** (depends on **T022**)
- [x] T024 [US4] Update `handleMessage` in `bot/internal/adapter/telegram/run.go` for **`/remove`** **disambig** **responses** to set **`SendMessageParams.ReplyMarkup`** from **T023** when **`RemoveReply`** indicates disambig (depends on **T022–T023**)
- [x] T025 [US4] Add **`parseRemovePickCallbackData`** (`rmp:`) and **`handleRemovePickCallback`** in `bot/internal/adapter/telegram/run.go`; extend **`handleCallback`** to dispatch **`rmp:`** after **`lpl:`** (see [research.md](research.md) **§8**); call **`TryProcessRemovePick`** with **index** from **callback**; **`SendMessage`** confirmation (or **edit** if **required** by **UX**); **`AnswerCallbackQuery`** (depends on **T010**, **T024**)
- [x] T026 [P] [US4] Extend `bot/internal/adapter/telegram/routing_test.go` with **`rmp:`** **parse** tests and **branch** **order**; add **table** test for **64-byte** **`callback_data`** **bound** (sample **UUID** + **index**) in `bot/internal/adapter/telegram/run_test.go` or **`routing_test.go`**
- [x] T027 [P] [US4] Update `specs/007-remove-saved-album/quickstart.md` **SC-003** steps (button tap + **text** **`1`** **regression**); align `specs/007-remove-saved-album/contracts/remove-command.md` **if** **implementation** **details** **differ** from **draft** (quickstart **already** **listed** **buttons**; **no** **contract** **edit** **required**)

**Checkpoint**: **FR-006** / **SC-003** satisfied on **Telegram**.

---

## Phase 11: Stale disambig + save path (research §8a) — verify

**Purpose**: Single-candidate **`/album`** **save** clears **prior** **`remove_saved`** / **album** **disambig** ([research.md](research.md) **§8a**).

- [x] T028 [P] Confirm `bot/internal/core/save_album.go` **`persistSave`** calls **`DeleteDisambigForListener`** when **`disambigID == nil`** before **`InsertSavedAlbum`**; **`go test ./internal/core -run Save`** (or full **`go test ./...`**) passes — **regression** **for** **stale** **remove** **session**

---

## Phase 12: Polish after Phase 10

**Purpose**: Full test gate after **T022–T027** land.

- [x] T029 Run `cd bot && go test ./...` and fix failures after **Phase** **10** implementation

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1** → optional skim before **Phase 2**
- **Phase 2** (**T002–T004**) **blocks** user-story phases **3–7** and **9**
- **Phase 10** (**T022–T027**) **depends** on **Phase** **6** (**T009–T011**) **and** **refactors** **`HandleRemove`** **signature** (**T022**)
- **Phase 11** (**T028**) **verification** — can run anytime **`save_album`** **is** **stable**
- **Phase 12** (**T029**) **after** **T022–T027**

### User story dependencies (US4 extension)

| Task block | Dependency |
|------------|------------|
| **T022** | **T009** (disambig **session** **creation** **ids** **available** **to** **return**) |
| **T023–T025** | **T022** |
| **T026** | **T025** |
| **T027** | **T025** |

### Parallel opportunities

- **T026** [P] with **T027** after **T025** is merged
- **T028** independent of **Phase** **10** (save **path**)

### Parallel example: Phase 10

```text
T022: remove_saved.go — RemoveReply + session id in disambig path
T023: run.go — removeDisambigInlineKeyboard
T024: run.go — handleMessage ReplyMarkup
T025: run.go — handleCallback rmp: + TryProcessRemovePick
T026: routing_test.go / run_test.go — parse + 64-byte
T027: quickstart.md + contracts touch-up
```

---

## Implementation Strategy

### MVP first

1. **Phases 1–2** then **3–6** (**T001–T012**) — **text-only** disambig
2. **Phases 7–9** (**T013–T021**) — **help**, **polish**, **partial** **tier**
3. **Phase 10** (**T022–T027**) — **Telegram** **inline** **buttons** + **`rmp:`** (**spec** **compliance** **FR-006**)
4. **T029** — run full `cd bot && go test ./...`

### Suggested delivery order (remaining work)

1. **T022** (core **API** **shape** for **session** **id** + **labels**)
2. **T023** + **T024** (keyboard + **wire** **in** **send**)
3. **T025** (callback **handler**)
4. **T026** + **T027** (tests + **docs**)
5. **T029**

---

## Notes

- **v1** does not parse `Artist - Title` into separate search fields; whole remainder vs stored `title` only ([spec.md](spec.md) **FR-003** / **A2**).
- **No** new DB migration for JSON shape in `disambiguation_sessions` ([data-model.md](data-model.md)).
- **`TryProcessRemovePick`** remains the text pick entry point; **T025** reuses for **index** from **`rmp:`** callback.

**Format validation**: All task lines use `- [ ]` / `- [x]`, **`T###`**, optional **`[P]`**, optional **`[US#]`**, and **concrete** **`bot/...` or `specs/...` paths** in the description.

**Task counts**: **Total** **29** tasks (**T001**–**T029**). **All** **complete** (Phase **10** + **T029** **done** 2026-04-26).

---

## Extension Hooks (optional)

**`before_tasks`**: `speckit.git.commit` — “Commit before task generation?” — run if you want a clean commit before editing **`tasks.md`**.  
**`after_tasks`**: `speckit.git.commit` — “Commit after task generation?”

## Suggested next command

- **`/speckit.implement`** focusing on **Phase 10** tasks **T022**–**T027**, then **T029**.
