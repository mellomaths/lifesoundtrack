# Data model: save album + listeners (003)

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-23

## Conventions

- **IDs**: `UUID` (v4), `gen_random_uuid()` when available (extension `pgcrypto` or `uuid-ossp`).
- **Time**: `TIMESTAMPTZ`, UTC in app.
- **Uniqueness**: A **listener** is unique per `(source, external_id)` (Telegram, etc.).

## ER overview

```text
listeners 1───* saved_albums
listeners 1───* disambiguation_sessions   (pending FR-009 picks; TTL via expires_at)
```

## Table: `listeners`

| Column | Type | Null | Description |
|--------|------|------|-------------|
| `id` | UUID | NO | Primary key. |
| `source` | TEXT | NO | e.g. `telegram`. |
| `external_id` | TEXT | NO | Host user id (string for BigInt). |
| `display_name` | TEXT | YES | “First Last” or display name. |
| `username` | TEXT | YES | @handle, null if not set. |
| `metadata` | JSONB | YES | Other non-secret host fields. |
| `created_at` | TIMESTAMPTZ | NO | |
| `updated_at` | TIMESTAMPTZ | NO | |

**Constraints**: `UNIQUE (source, external_id)`.

**Indexes**: `PRIMARY KEY (id)`; unique on `(source, external_id)`.

## Table: `saved_albums`

One row per **user action** that successfully saves a resolved album (spec allows duplicate titles across rows).

| Column | Type | Null | Description |
|--------|------|------|-------------|
| `id` | UUID | NO | Primary key. |
| `listener_id` | UUID | NO | FK → `listeners.id` **ON DELETE CASCADE**. |
| `user_query_text` | TEXT | YES | Original free-form `/album` text (length-capped in app, e.g. 512). |
| `title` | TEXT | NO | Resolved album title. |
| `primary_artist` | TEXT | YES | Main artist as single string. |
| `year` | INT | YES | Release year (if known). |
| `genres` | TEXT[] | YES | Optional genre tags. |
| `provider_name` | TEXT | NO | `musicbrainz` \| `lastfm` \| `itunes` \| … |
| `provider_album_id` | TEXT | YES | Opaque id for re-fetch. |
| `art_url` | TEXT | YES | Optional cover, HTTPS. |
| `extra` | JSONB | YES | Truncated provider payload for analytics (no PII, no full raw dump). |
| `created_at` | TIMESTAMPTZ | NO | |

**Indexes**: `INDEX (listener_id)`; optional `INDEX (created_at DESC)`.

## Table: `disambiguation_sessions`

Stores **pending** candidate lists between “search returned 2–3 albums” and “user picked one” (**FR-009**). **Required** for production / **multi-replica**; local **single-process** dev may use in-memory instead, but the same schema is the source of truth when Postgres is enabled. **No Redis** in v1.

| Column | Type | Null | Description |
|--------|------|------|-------------|
| `id` | UUID | NO | Session id. |
| `listener_id` | UUID | NO | FK → `listeners`. |
| `candidates` | JSONB | NO | Up to 3 **normalized** candidates. |
| `created_at` | TIMESTAMPTZ | NO | |
| `expires_at` | TIMESTAMPTZ | NO | Polling / cron deletes expired. |

**Indexes**: `(listener_id)`; index or BRIN on `expires_at` for cleanup (periodic job or `DELETE` in transaction on session consume).

## Migration file examples (golang-migrate)

File `000001_init_listeners_and_saved_albums.up.sql` (excerpt only—implementations copy verbatim into repo):

- `CREATE EXTENSION IF NOT EXISTS "pgcrypto";`
- `CREATE TABLE listeners (...);`
- `CREATE TABLE saved_albums (...);`
- `CREATE TABLE disambiguation_sessions (...);` — same initial migration for v1 (no Redis).

**Down** migrations **drop** in reverse order with `CASCADE` where safe.

## Validation rules (application layer)

- `source` and `external_id` never empty for persistence.
- `user_query_text` max length and UTF-8 validation before write.
- `genres` array bounded (e.g. max 8 entries) to avoid pathological API payloads.
- `extra` size cap (e.g. 8 KiB) to protect Postgres.

## Future (not in v1)

- `catalog_album` normalized table + FK from `saved_albums` for deduplication and cross-user analytics.
