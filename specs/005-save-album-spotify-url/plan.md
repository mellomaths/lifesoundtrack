# Implementation Plan: Save album via Spotify album link

**Branch**: `005-save-album-spotify-url` | **Date**: 2026-04-24 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/005-save-album-spotify-url/spec.md`

**Note**: Filled by `/speckit-plan`. Phase 0 → [research.md](./research.md); Phase 1 → [data-model.md](./data-model.md), [contracts/](./contracts/), [quickstart.md](./quickstart.md). Implementation tasks: [tasks.md](./tasks.md) (**T001**–**T017**).

## Summary

The **`/album`** command serves **two parameter families** on one entry point (**FR-000**, **Save-album command: two argument types** in [spec.md](./spec.md)):

1. **FREE_TEXT** — Same as feature **003**: the full argument goes through **`Search`** on the metadata chain (Spotify → iTunes → Last.fm → MusicBrainz per flags), with multi-album disambiguation when needed.
2. **SPOTIFY_URL** | **SHORT_URL** — When the argument is **FR-008**-eligible (including **one** **unambiguous** album or short link **embedded** in short surrounding text per **User Story 1** scenario 5 and **Edge Cases**), classify as a direct Spotify album reference: **bounded** redirect resolution for share hosts, extract album id, **`GET /v1/albums/{id}`** via client credentials, return **one** `AlbumCandidate`, **no** disambiguation step for that turn. **Failed** direct-link attempts **must not** fall back to **`Search`(raw URL)**. **Generic** non-Spotify `https://` input stays **FREE_TEXT** (full string to **`Search`**).

No new persistence schema for 005; reuse `saved_albums` and existing save flow after candidate resolution.

## Technical Context

**Language/Version**: Go **1.25** (`bot/go.mod`)  
**Primary Dependencies**: `go-telegram/bot`, `pgx/v5`, `goose`, `cron`, `gobreaker`, Spotify Web API (existing metadata chain)  
**Storage**: PostgreSQL (existing); **no** new migrations for 005 per [data-model.md](./data-model.md)  
**Testing**: `go test ./...` under `bot/`; table-driven unit tests for URL parsing, redirect policy, orchestrator branches ([research.md](./research.md) §7)  
**Target Platform**: Linux container / local process (Telegram long-polling bot)  
**Project Type**: messaging bot service (`bot/`)  
**Performance Goals**: Short-link resolution uses a **tight** client timeout (e.g. **≤ ~4s** per [research.md](./research.md) §2) so redirect work **does not** consume the whole user-visible budget; **end-to-end** link save aligns with [spec.md](./spec.md) (**about fifteen seconds** typical when dependencies are healthy). The **4s** figure bounds **one** sub-step; the **~15s** figure bounds **perceived** command completion.  
**Constraints**: Redirect cap (e.g. **5** hops), HTTPS + host allowlist; no secrets in logs; honest non-success when Spotify disabled or lookup fails on direct path  
**Scale/Scope**: Single-command extension; `internal/core` + `internal/metadata` + copy/help/tests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

Alignment with [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md):

| Principle | Status |
|-----------|--------|
| **I. Code quality** | Scoped changes to save-album path, parsers, metadata; follow existing Go rules in `.cursor/rules/`. |
| **II. REST API** | **N/A** — no new public HTTP API (Spotify client only). |
| **III. Testing** | Plan [research.md](./research.md) §7 matrix; `go test` in CI when present; **SC-005** release review in [tasks.md](./tasks.md) **T017** / [checklists/requirements.md](./checklists/requirements.md). |
| **IV. UX** | Help/copy names **free text** and **Spotify album / share links**; distinct failure classes (bad Spotify link vs free-text no-match). |
| **V. Monitoring** | **N/A** with note — no new production metrics requirement beyond existing bot ops. |
| **VI. Logging** | Structured messages; no tokens; correlate with listener/chat as today. |
| **VII. Performance** | Short-link timeout and redirect bounds stated above; overall UX per spec. |
| **VIII. Containerization** | `bot/Dockerfile` exists; repo `compose.yaml` for local stack; 005 does not require a new runnable. |
| **IX. Documentation** | Plan/spec/contracts/quickstart paths consistent with tree; [specify-rules.mdc](../../.cursor/rules/specify-rules.mdc) references this plan. |

**Post-design re-check**: [data-model.md](./data-model.md), [contracts/spotify-album-url.md](./contracts/spotify-album-url.md), and [research.md](./research.md) encode **FR-000** routing and embedded-link behavior; no constitution violations requiring **Complexity Tracking**.

## Project Structure

### Documentation (this feature)

```text
specs/005-save-album-spotify-url/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── spotify-album-url.md
├── checklists/
└── tasks.md
```

### Source Code (repository root)

```text
bot/
├── cmd/
├── internal/
│   ├── core/           # SaveService, orchestrator interface, URL classification, copy
│   ├── metadata/       # Chain, Spotify album-by-id, redirect resolution
│   └── adapter/        # Telegram handlers
├── migrations/         # unchanged for 005
└── go.mod
```

**Structure Decision**: Implement in **`bot/internal/core`** (routing, `ProcessAlbumQuery`) and **`bot/internal/metadata`** (HTTP to Spotify, redirects); keep Telegram adapter thin.

## Complexity Tracking

> No constitution exceptions required for this feature.

## Phase 2 (implementation planning)

Executable work is tracked in [tasks.md](./tasks.md) (**T001**–**T017**). `/speckit-plan` stops after design artifacts above; run **`/speckit-implement`** or execute tasks manually for code delivery.
