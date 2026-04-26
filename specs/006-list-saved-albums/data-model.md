# Data model: `/list` + `album_list_sessions`

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-26

## Existing tables (unchanged schema)

Listing reads **`listeners`** and **`saved_albums`** as defined in [specs/003-save-album-command/data-model.md](../003-save-album-command/data-model.md). Relevant columns for `/list`:

| Table | Columns used |
|-------|----------------|
| `listeners` | `id`, `source`, `external_id` |
| `saved_albums` | `id`, `listener_id`, `title`, `primary_artist`, `year`, `created_at` |

**Indexes** (existing): `saved_albums_listener_id`, `saved_albums_created_at` — sufficient for **listener-scoped** list + sort.

## New table: `album_list_sessions`

Short-lived **server-side** context for **pagination** and **optional artist filter** (Telegram callback size + multi-replica).

| Column | Type | Null | Description |
|--------|------|------|-------------|
| `id` | UUID | NO | Primary key; exposed in `callback_data` as **`lpl:<id>:<page>`**. |
| `listener_id` | UUID | NO | FK → `listeners.id` **ON DELETE CASCADE**. |
| `artist_filter_norm` | TEXT | YES | Normalized lowercase substring filter; **`NULL`** means **all** albums for the listener. |
| `current_page` | INT | NO | **1-based** cursor for **`/list next`** / **`/list back`** and callback navigation; updated on each page display. |
| `created_at` | TIMESTAMPTZ | NO | Default `now()`. |
| `expires_at` | TIMESTAMPTZ | NO | e.g. `now() + 15 minutes` (align with disambig session TTL style). |

**Indexes**:

- `INDEX (listener_id, created_at DESC)` — resolve “latest open session” for **`/list next`** / **`/list back`**.
- `INDEX (expires_at)` — optional cleanup job / batch delete.

**Validation (application)**:

- Only **one logical “active”** session per listener is required for text paging; implementation **may** insert a **new** row on each **`/list`** (page 1) and rely on **latest row** for **`next`/`back`**, or **upsert** — plan prefers **insert on each `/list`** + **latest non-expired** for simplicity.

## Queries (conceptual)

1. **Count** matching rows: `WHERE listener_id = $1` and optional `strpos(lower(coalesce(primary_artist,'')), $norm) > 0`.
2. **Page slice**: same filters, **`ORDER BY created_at DESC, id DESC`**, **`LIMIT 5 OFFSET ($page-1)*5`**.
3. **Session insert**: **only** when **`total_pages > 1`** (so callbacks and **`/list next`**/**`back`** have a use). For **empty library**, **zero matches**, or **a single page** of results, **omit** a new session (or do not refresh paging state).

**Listener resolution**: `SELECT id FROM listeners WHERE source = $1 AND external_id = $2` — if **no row**, **count = 0** without insert.

## Migration

- **File**: `bot/migrations/00003_album_list_sessions.sql` (name may follow repo’s migrate naming convention).
- **Down**: `DROP TABLE IF EXISTS album_list_sessions;`

## Privacy

Session rows contain **no** chat text — only **normalized filter** and **foreign keys**. Expire and delete like **`disambiguation_sessions`**.

## Display vs persistence (**FR-010**)

List messages may show **abbreviated** title or artist strings for readability. **Application layer** only — **`saved_albums.title`** and **`saved_albums.primary_artist`** in Postgres are **not** updated for listing.
