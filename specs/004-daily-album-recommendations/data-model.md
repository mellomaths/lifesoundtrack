# Data model: daily recommendations (004)

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-24

## Conventions

- **Time**: `TIMESTAMPTZ`, UTC in SQL comparisons unless noted; cron uses **`LST_DAILY_RECOMMENDATIONS_TZ`** for wall clock.
- **IDs**: UUID, `gen_random_uuid()` consistent with existing migrations.

## `saved_albums` (alter)

| Column | Type | Null | Description |
|--------|------|------|-------------|
| `last_recommended_at` | TIMESTAMPTZ | YES | Last **successful** daily recommendation send for this row; **NULL** = never recommended via this feature. |

**Index** (recommended): partial or composite to support per-listener selection, e.g. `(listener_id, last_recommended_at NULLS FIRST)`.

## New table: `recommendations`

| Column | Type | Null | Description |
|--------|------|------|-------------|
| `id` | UUID | NO | Primary key. |
| `run_id` | UUID | NO | Cron invocation id (shared across listeners in one tick). |
| `listener_id` | UUID | NO | FK → `listeners.id` **ON DELETE CASCADE**. |
| `saved_album_id` | UUID | NO | FK → `saved_albums.id` **ON DELETE CASCADE**. |
| `title_snapshot` | TEXT | NO | Shown title. |
| `artist_snapshot` | TEXT | YES | Shown artist. |
| `year_snapshot` | INT | YES | Optional. |
| `spotify_url_snapshot` | TEXT | YES | URL if any was shown. |
| `sent_at` | TIMESTAMPTZ | NO | Commit time / send time for rotation. |

**Constraints**

- `UNIQUE (listener_id, run_id)` — at most one successful row per listener per run.
- Indexes: `(listener_id, sent_at DESC)` for support queries.

## Migration

- New Goose file: add column + create `recommendations` + constraints + indexes.
- **Down**: drop `recommendations`; drop `last_recommended_at` from `saved_albums`.

## Application validation

- Transaction must update **exactly** the picked `saved_albums.id` and insert one **`recommendations`** row with matching snapshots and **`run_id`**.

## Listener discovery (query semantics)

- Load **distinct** listeners who have **≥ 1** `saved_albums` row, with the same **source / channel** filters as the rest of the job (e.g. Telegram).
- The SQL shape MUST satisfy PostgreSQL: avoid **`SELECT DISTINCT`** with **`ORDER BY`** columns that are not part of the distinct select list (**FR-013**–**FR-015**); see [research.md](research.md) §8.
- Result MUST list each eligible listener **at most once** per **`run_id`**.
