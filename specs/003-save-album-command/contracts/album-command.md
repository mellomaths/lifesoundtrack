# Contract: `album` (save) — domain and Telegram mapping

**Spec**: [spec.md](../spec.md) | **Date**: 2026-04-23

## Domain command: `save_album` (name)

| Field | Type | Description |
|-------|------|-------------|
| `query` | string | Free-form, **non-empty** after trim; no schema enforced. **Max length** (plan): 512 runes, enforced by adapter. |

**Outcomes (abstract)** — map to [metadata-orchestrator.md](metadata-orchestrator.md):

1. **empty_query** — reply with “need search text” (no metadata call, no save).
2. **candidates** (2–3 items) — present **up to 3** items ordered by **relevance**; wait for `pick_index` (1-based or 0-based: **1-based in user copy**, adapter normalizes to domain).
3. **single_match** — optional auto-save (when confidence / policy allows).
4. **no_match** — user-facing “not found.”
5. **provider_exhausted** — all breakers open or all returned empty: “try again later.”
6. **saved** — confirmation line with title / year, **no** full provider JSON.

**Errors**: Must **not** include API keys, connection strings, or other users’ data ([spec FR-007](../spec.md)).

## Telegram mapping (v1)

| Telegram | Domain |
|----------|--------|
| `/album` with **no** text, or only whitespace | `empty_query` |
| `/album Red` (example) with text | `save_album` with `query = "Red"` (trim) |
| Inline button **1..3** or next message `1`–`3` | `disambiguation_pick` with 1-based index (adapter converts) |

**Help**: `/help` text lists `/album` with one-line description (implementation task in core copy).

## Persistence contract

- A **save** is committed **only** after a **definite** `AlbumRef` (see metadata contract) and **listener** are known.
- **No** `saved_albums` row on `no_match` or `provider_exhausted`.

## Acceptance

Matches [spec.md](../spec.md) **US1, US1b, US2, US3**; **FR-001**–**FR-009**; and [data-model.md](../data-model.md).
