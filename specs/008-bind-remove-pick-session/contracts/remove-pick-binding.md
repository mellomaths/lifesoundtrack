# Contract: Remove pick ↔ disambiguation session binding

**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md) | **Date**: 2026-04-26

## Purpose

Amends the behavior described in [007 `remove-command.md`](../007-remove-saved-album/contracts/remove-command.md) for **how** a pick resolves the disambiguation session. Feature **008** makes session identity **authoritative** for **inline** picks.

## Telegram `rmp:` callback

**Format (unchanged):** `rmp:<disambiguation_session_id>:<1-based_index>` (index 1..99, total `callback_data` length ≤ 64 bytes).

**Processing (revised):**

1. **Parse** `disambiguation_session_id` and `1-based_index` (existing parser).
2. **MUST** load `disambiguation_sessions` by **`id` = disambiguation_session_id** and **`listener_id`** = the acting user’s listener (derived from `source` + `external_id`), and **`expires_at` > now()**.
3. If no row: **MUST NOT** delete any saved album; **MUST** return a **non-destructive** user-visible message (stale / expired / no longer valid pick).
4. If row exists: `candidates` **MUST** be JSON with `kind: "remove_saved"` (same shape as 007). If not, treat as invalid — no delete.
5. **MUST NOT** use “latest session for user” for this path when a session id is present.

## Typed numeric pick (private chat)

**When** the user sends a line that is only `1`..`99` (existing `RemovePickIndexFromText`):

- **MUST** resolve a remove pick only if the **latest** open disambiguation session for that user unmarshals to `kind: remove_saved`; **MUST NOT** use album-save (array) JSON for a remove index.
- v1 does **not** require inferring a **superseded** on-screen `remove_saved` list from plain text (see [spec.md](../spec.md) **Clarifications**, **US3**, **FR-005**); **inline** actions carry the session id for that guarantee.

## Domain API expectation

Adapters call into `internal/core` with:

- `TryProcessRemovePick(ctx, source, externalID, oneBased)` — text path (no session id), **or**
- A variant with **explicit `sessionID`** for callbacks, **or** a single function with `sessionID` as optional `""` — behavior as above.

## Testing obligations

- **Regression:** Two sequential `remove_saved` disambiguation contexts (or superseded first session) + **inline** callback for the **first** session id **must not** remove a candidate that exists **only** in a **second** list ([SC-001](../spec.md), session-bound path only).
