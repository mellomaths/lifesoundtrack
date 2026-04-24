# Implementation Plan: Daily fair album recommendations

**Branch**: `004-daily-album-recommendations` | **Date**: 2026-04-24 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `/specs/004-daily-album-recommendations/spec.md`

**Note**: Filled by `/speckit.plan`. Template: `.specify/templates/plan-template.md`.

## Summary

Ship a **daily scheduled job** (06:00 in an operator-configured timezone, default UTC) that, when enabled via **`LST_DAILY_RECOMMENDATIONS_*`** env flags, sends **each eligible Telegram listener** exactly **one** album recommendation from their **`saved_albums`** using **fair rotation** (never-recommended first, then oldest `last_recommended_at`, uniform random within ties), with **Telegram-first** delivery and a **single post-success transaction** updating **`last_recommended_at`** and inserting **`recommendations`**.

**Defects documented and partially implemented**: (1) **Listener enumeration** — PostgreSQL-safe query (**FR-013**–**FR-015**); implemented as **`store.ListTelegramListenerIDsWithSavedAlbums`**. (2) **Scheduler never ran** — **in-process** **`robfig/cron/v3`** in **`bot/cmd/bot`**, started in a **goroutine** before Telegram long-polling, stopped on process shutdown (**FR-017**); startup log **`daily_recommendations_config`** (**FR-018**); tick logs **`daily_recommendations_cron_tick`** then listener discovery (**SC-008**).

**Remaining product work**: **Per-listener** fair pick (**FR-003**), Telegram **send** (**FR-004**–**FR-006**), **`RecordRecommendationTx`** after success (**FR-007**–**FR-008**), Goose migration **`00002_daily_recommendations.sql`** if not yet in tree, and wiring the cron callback to that runner instead of “list IDs only.”

## Technical Context

**Language/Version**: Go **1.25** (`bot/go.mod`)  
**Primary Dependencies**: `github.com/go-telegram/bot`, `github.com/jackc/pgx/v5`, `github.com/pressly/goose/v3`, `github.com/robfig/cron/v3`, `github.com/joho/godotenv`, `github.com/google/uuid`  
**Storage**: **PostgreSQL** (listeners, `saved_albums`, `recommendations`; migrations under `bot/migrations/`)  
**Testing**: `go test ./...` from `bot/`; config table tests for daily env; store integration test for listener list when **`DATABASE_URL`** set (**SC-007**); manual/staging check for cron tick logs (**SC-008**)  
**Target Platform**: Linux containers / local **Docker Compose** (`compose.yaml`)  
**Project Type**: Single **Telegram bot** service (`bot/cmd/bot`)  
**Performance Goals**: **N/A** with justification — daily batch for small-to-medium listener counts; avoid unnecessary metadata calls when **`saved_albums`** is sufficient.  
**Constraints**: Telegram send **before** DB commit (**FR-007**/**FR-008**); **`UNIQUE (listener_id, run_id)`** on **`recommendations`**; listener SQL must be PostgreSQL-valid (**FR-013**–**FR-015**); scheduler must not block Telegram (**FR-017**).  
**Scale/Scope**: One proactive message per eligible listener per cron tick; single global timezone (**A-002**).

## Environment (operators)

Canonical variable **names**, **defaults**, and **parsing** (opt-out enable, IANA TZ, five-field cron) are defined in [contracts/feature-flags.md](contracts/feature-flags.md). Implementation loads them via `bot/internal/config` (`LST_DAILY_RECOMMENDATIONS_*`); `bot/.env.example` lists the keys for local use.

## Release and UAT gates (**SC-004**, **SC-005**)

These are **manual / release** checkpoints, not automated build tasks:

- **SC-004**: Before production rollout, run **user acceptance** on a staging or pilot cohort: at least **95%** of eligible users with a Spotify URL on the chosen row get a working destination (button or text). Record evidence in release notes or QA sign-off.
- **SC-005**: During **pre-production soak** (≥ **two weeks**), monitor operator-facing metrics: at most **one** successful recommendation counted per user per daily run; investigate any spike without a documented cause.

Details and checklists: [quickstart.md](quickstart.md) § Release / UAT.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

Alignment with [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md):

- **Code quality**: Go conventions per `.cursor/rules/`; scoped changes; no unjustified lint suppressions.
- **REST API**: **Not in scope**; principle **II** N/A.
- **Testing**: Unit tests for config and rotation when implemented; store integration for listener query; `go test ./...` in CI when present.
- **User experience**: Message template and Spotify rules per [contracts/daily-recommendations-job.md](contracts/daily-recommendations-job.md).
- **Monitoring**: Per-run counts (**FR-009**); startup schedule visibility (**FR-018**). **HTTP health / liveness**: **N/A** for the current Telegram **long-poll-only** process (constitution **V** “where applicable”); if an HTTP surface is added later, document a health path then.
- **Logging**: **`run_id`** on batch steps; no secrets; **`daily_recommendations_*`** log keys for operators.
- **Performance**: N/A above; profile if batch grows.
- **Containerization**: **`compose.yaml`** + **`bot/Dockerfile`**; env examples in **`bot/.env.example`**.
- **Documentation accuracy**: This plan, [quickstart.md](quickstart.md), contracts, and `.cursor/rules/specify-rules.mdc` stay aligned with **004** and the repo.

## Project Structure

### Documentation (this feature)

```text
specs/004-daily-album-recommendations/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── daily-recommendations-job.md
│   └── feature-flags.md
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
bot/
├── cmd/bot/main.go                 # Cron goroutine + telegram.Run
├── internal/
│   ├── adapter/telegram/           # Daily send (to complete)
│   ├── config/
│   │   ├── config.go
│   │   └── daily_schedule.go       # LST_DAILY_RECOMMENDATIONS_*
│   ├── core/                       # SaveService today; daily runner + rotation (to add)
│   └── store/
│       ├── daily_recommendations.go
│       └── daily_recommendations_test.go
├── migrations/
go.mod
```

**Structure Decision**: **`main`** owns scheduler lifecycle; **`config`** owns schedule parsing; **`store`** owns listener query and (planned) **`RecordRecommendationTx`**; **`core`** + **`adapter`** own the send-then-persist loop.

## Complexity Tracking

No constitution violations requiring exceptions for this feature.

## Phase 0 & Phase 1 artifacts

Design artifacts for this feature already exist and are maintained with the spec; **tasks** are in [tasks.md](tasks.md) (Phase 2 per Spec Kit naming).

| Artifact | Path | Status |
|----------|------|--------|
| Research | [research.md](research.md) | §8 DISTINCT/ORDER BY; §9 scheduler lifecycle; §10 **SC-003** tolerance |
| Data model | [data-model.md](data-model.md) | `saved_albums` / `recommendations`; listener discovery note |
| Contracts | [contracts/](contracts/) | In-process trigger **FR-017**; flags **FR-010**–**FR-012** |
| Quickstart | [quickstart.md](quickstart.md) | Verify steps; **Release / UAT** for **SC-004** / **SC-005** |

Phase 2 (**tasks.md**) is produced by `/speckit.tasks`, not this command.
