# Research: Bind remove picks to disambiguation session

**Date**: 2026-04-26 | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## 1. Root cause (documented)

**Decision:** Treat “which session?” as a **data identity** problem: the `rmp:` callback already carries `disambiguation_sessions.id` and 1-based index, but the handler used `ORDER BY created_at DESC LIMIT 1` for the listener, dropping the id.

**Rationale:** Superseding flows (`DeleteDisambigForListener` or new session insert) can delete or replace older rows; older Telegram messages still show old inline keyboards. Only **keyed** lookup ties the user action to the same JSON payload they were shown.

**Alternatives considered:** Rely on “only one open session per listener” — **rejected** because the UI can still show multiple messages with different keyboards over time; DB may only have the newest row but the user may interact with an old control.

## 2. Store API: load session by id and listener

**Decision:** Add a method such as `OpenDisambiguationSessionForListener(ctx, sessionID, listenerID) ([]byte, error)` (exact name per codebase conventions) that:

- Returns the `candidates` bytea **only if** `id = $1`, `listener_id = $2`, and `expires_at > now()`.
- Returns a typed “not found” outcome (e.g. `pgx.ErrNoRows` or `(nil, nil)`) for missing/expired/wrong-listener, without leaking other listeners’ data.

**Rationale:** One round-trip, uses primary key on `disambiguation_sessions.id` (UUID); enforces **FR-004** (listener-scoped).

**Alternatives considered:** `SELECT` by id only, then check listener in Go — **rejected** as weaker (extra leak surface in logs if mis-handled); join in SQL is standard.

## 3. Core: `TryProcessRemovePick` entry points

**Decision:** Split behavior:

1. **Callback path** (has non-empty `sessionID` from `rmp:`): load via §2; unmarshal `removeDisambigRoot`; apply `oneBased`; on missing session → **FR-003** user copy and **no delete**.
2. **Text path** (no `sessionID`): use `LatestOpenDisambiguationSession` + **must** unmarshal to `kind: remove_saved` before applying an index; **never** use album-disambig (array) JSON for a remove index. v1 does **not** disambiguate which of two **on-screen** remove lists a bare number referred to if storage already has only one (newest) `remove_saved` — that is **inline’s** job per [spec.md](./spec.md) **Clarifications** and **US3**.

**Rationale:** Callback path fixes the high-severity wrong deletion. Plain text cannot carry a session id; the spec (post-clarify) no longer requires impossible two-list-by-text behavior.

**Alternatives considered:** Reject all text picks — too harsh. Reply-to-message binding — **deferred** (Telegram `reply_to_message`); optional follow-up.

## 4. Adapter: pass session id into core

**Decision:** `parseRemovePickCallbackData` already returns `(sessionID, oneBased, ok)`; `handleRemovePickCallback` must pass `sessionID` into `TryProcessRemovePick` (signature extended) or a dedicated `TryProcessRemovePickBySessionID` to avoid conflating paths.

**Rationale:** Minimal surface change; unit tests on parser already exist.

**Alternatives considered:** Resolve session in adapter — **rejected**; domain + ownership belong in `core` + `store`.

## 5. User-visible stale outcomes

**Decision:** Reuse the same family of copy as [007 remove command](../007-remove-saved-album/) “no active session” / not found, extended or clarified for “this list is no longer available” if needed, without duplicating three different help essays.

**Rationale:** Constitution UX consistency; spec [FR-003](./spec.md).

## 6. Tests

**Decision:** Add store-level test: session S1 and S2 for same listener (or sequential insert after delete) + core test that **callback targeting S1** does not delete S2’s candidate. Integration style with real `*store.Store` in existing test patterns, or fakes on the persistence interface for `core`.

**Rationale:** Constitution testing; [SC-001](./spec.md).

**Alternatives considered:** E2E only — **rejected**; too slow for regression on this logic.
