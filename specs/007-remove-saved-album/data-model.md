# Data model: `/remove` saved album (007)

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-26

## Conventions

Unchanged from [003 data model](../003-save-album-command/data-model.md): **UUID** ids, **TIMESTAMPTZ** in UTC, **listener** uniqueness on `(source, external_id)`.

## ER overview

```text
listeners 1───* saved_albums
listeners 1───* disambiguation_sessions   (existing; JSON payload extended for remove picks)
```

**No** new tables for v1.

## Table: `saved_albums` (use)

| Use | Description |
|-----|-------------|
| **Match** | Load rows for `listener_id`. **Exact** tier: normalized user string equals normalized `title`. If no row qualifies, **partial** tier: normalized `title` contains normalized query as a contiguous substring ([research.md](./research.md) §2). |
| **Delete** | `DELETE ... WHERE id = $1 AND listener_id = $2` after **match** resolution or **disambig** pick. |

**Indexes**: Existing **`listener_id`** index supports listing a listener’s library for in-process filtering.

## Table: `disambiguation_sessions` (extended use)

| Column | Use for `/remove` |
|--------|-------------------|
| `listener_id` | Same as today; must match the user **before** showing picks or applying delete. |
| `candidates` | **JSON object** with **`kind: "remove_saved"`** and **`candidates`**: `[{ "id", "label" }]` (see [contracts/remove-command.md](./contracts/remove-command.md)). **Not** a bare array (that remains **save-album** candidates). |
| `expires_at` | Same **TTL** as save flow (`store.DefaultSessionTTL`) unless planning tightens. |

**Cleanup**: Delete session row after **successful** **delete** or on **out-of-range** / **expired** session (align with **save** behavior).

## State transitions (listener action)

1. **`/remove <q>`** with no qualifying row under **either** match tier (per **FR-003**) → no DB delete; user sees not-found text.
2. **Exactly one** match on the **exact** tier → delete that `saved_albums` row; no new disambig session (success confirmation only).
3. **Multiple** exact matches, **or** 1–3 partial matches with no exact match → insert `disambiguation_sessions` with `remove_saved` JSON; no delete until pick (including a **single** partial match — no auto-delete on partial alone).
4. No exact match and **>3** partial matches → do not insert a session; user message to narrow the query (see **FR-004**); do not list every match.
5. **Pick `n`** (valid) when a `remove_saved` session is open → delete `saved_albums` with `id = candidates[n-1].id` (with listener check); delete session. **Index** `n` may **originate** from **private** **text** or from **Telegram** **`rmp:`** **callback** **(same** **session** **row,** **same** **semantics**).

## Transport (adapter only; not new DB state)

- **`rmp:<session_uuid>:<index>`** — **refers** to **`disambiguation_sessions.id`**; **index** is **1-based** **into** that **row’s** **`candidates`** **JSON**. **No** **extra** **columns** (see [research.md](./research.md) **§10**).

## Validation rules

- **User** **query** **length** ([spec.md](spec.md) **A3**): same (or stricter) **rune/character** **limit** as **`/album`** free text; **on** **exceeding**, **no** **delete** — see [research.md](./research.md) §2a and [contracts/remove-command.md](./contracts/remove-command.md).
- **Delete**: Must not execute without **`listener_id`** **ownership** of the row. **Store** **tests** must **cover** **wrong**-**listener** **delete** **attempts** (see [tasks.md](./tasks.md) **T003**).
