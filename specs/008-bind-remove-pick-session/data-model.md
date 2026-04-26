# Data model: Bind remove picks to disambiguation session

**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Existing entities (unchanged schema)

### `disambiguation_sessions`

| Field | Meaning |
|-------|---------|
| `id` (UUID) | Primary key; embedded in `rmp:<id>:<n>` for remove picks. |
| `listener_id` (UUID) | Owner; must match the acting user’s listener when resolving a pick. |
| `candidates` (JSONB) | For remove flow: object with `kind: "remove_saved"` and `candidates[]` of `{id, label}`. |
| `expires_at` | Session invalid after this time; picks must not apply after expiry. |
| `created_at` | Ordering for “latest” *only* when no session id is available (text path). |

**Validation rules for picks:**

- A pick is **valid** only if the row exists, `expires_at > now()`, and `listener_id` matches the authenticated listener.
- **No migration** required for this feature: identity and ownership already present.

### `saved_albums`

Deletion remains `DELETE … WHERE id = $album AND listener_id = $listener` (existing `DeleteSavedAlbumForListener`).

## State transitions (remove disambiguation)

```text
[Created] — open disambig message with rmp:THIS_ID:…
     │
     ├─ User picks (callback with THIS_ID) → load by THIS_ID → delete candidate or safe message
     │
     ├─ Superseded: DeleteDisambigForListener or new session
     │       → old THIS_ID row may be deleted; callback with old id → "stale" / no wrongful delete
     │
     └─ Expires → same as missing row for keyed lookup
```

## New / updated conceptual operations (not new tables)

| Operation | Purpose |
|-----------|---------|
| **Open disambiguation by id + listener** | Satisfies [FR-001](./spec.md#functional-requirements)–[FR-002](./spec.md#functional-requirements) for callback picks. |
| **Latest open remove_saved (text path)** | Unchanged table use; stricter *interpretation* documented in [research.md](./research.md). |
