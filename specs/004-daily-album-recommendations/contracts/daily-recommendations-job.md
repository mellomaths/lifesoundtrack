# Contract: daily recommendations job

**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md)

## Trigger

- An **in-process** wall-clock scheduler in the **same** bot process as Telegram (see **FR-017** in [spec.md](../spec.md)) fires per [feature-flags.md](feature-flags.md) when enabled; it MUST **not** depend on OS-level cron or on inbound Telegram messages to start a run.
- Generate **`run_id`** (UUID) once per invocation; pass to all listeners in that run.

## Eligibility

- Listener has ≥ 1 `saved_albums` row.
- **A-006**: no per-user mute filter unless added later.

## Selection (per listener)

1. Let **Tier A** = rows with `last_recommended_at IS NULL`. If **Tier A** non-empty, candidates = **Tier A**.
2. Else let **t** = minimum `last_recommended_at` among rows; candidates = all rows with `last_recommended_at = t`.
3. Pick **uniformly at random** one row from candidates (injectable `rand` in tests).

## Spotify URL

- `provider_name = spotify` and `provider_album_id` non-empty → `https://open.spotify.com/album/{provider_album_id}`.
- Else optional string in `extra` if implemented.
- Else **no** URL.

## Telegram payload

- If `art_url` is non-empty HTTPS: **`sendPhoto`** with caption; else **`sendMessage`** (or photo-less policy documented in implementation).
- Caption/body includes:
  - `Your pick today: {title} — {primary_artist}` and ` ({year})` only if year present.
  - Spotify URL in text if not using URL button for that send.
  - Trailing `Enjoy the listen — LifeSoundtrack.`
- **Inline keyboard**: one **URL** button (e.g. “Open in Spotify”) when URL exists and adapter uses button path.

## Persist (after Telegram OK)

Single transaction:

- `UPDATE saved_albums SET last_recommended_at = $sent_at WHERE id = $id`
- `INSERT INTO recommendations (...)` including **`run_id`**, snapshots, **`sent_at`**

On Telegram error: **no** UPDATE/INSERT for that listener.

## Listener iteration

- Query listeners that have saved albums (distinct `listener_id` from `saved_albums` join or subquery). The query MUST be valid on the production database: deduplicating listeners MUST NOT be combined with a sort key that the database rejects for that query shape (see **FR-013**–**FR-015** and the **Defect fix: Listener enumeration** section in [spec.md](../spec.md)).
- Telegram `chat_id` from `listeners.external_id` for `source = telegram` (or existing mapping convention in adapter).

## Rate limiting

- On **429**, backoff / throttle between sends; log **`run_id`** and error class.
