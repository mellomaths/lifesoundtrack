# Research: `/list` saved albums

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-26

## 1. Telegram `callback_data` length

**Decision**: Encode list paging as **`lpl:<session_uuid>:<page>`** (1-based page), where `<session_uuid>` is the standard 36-character UUID string. Example length: `4 + 36 + 1 + 2` = **43 bytes** for page ≤99 — under the **64-byte** limit for `callback_data`.

**Rationale**: Artist filters can exceed 64 bytes; storing **normalized filter text** (and listener ownership) in **`album_list_sessions`** keeps callbacks small and **validates** on callback that the session belongs to the clicking user.

**Alternatives considered**:

- Put filter + page entirely in `callback_data` — **rejected** (filter too long; trivial to tamper without server state).
- HMAC-signed blob in callback — **rejected** for v1 (more moving parts; session table already required for filter).

## 2. Artist substring matching in PostgreSQL

**Decision**: After Go-side normalization (**trim**, **collapse internal whitespace** to single ASCII space, **Unicode-aware lower** via **`strings.ToLower`** / Go stdlib for v1), use:

`strpos(lower(coalesce(primary_artist, '')), $1) > 0`

with `$1` the normalized lowercase needle.

**Rationale**: **Avoids `LIKE`** user-supplied `%`/`_` wildcard injection and keeps semantics as **substring** match, which matches user expectations for `/list The Beatles` vs stored `The Beatles`.

**Alternatives considered**:

- `ILIKE '%' || quote_literal(...) || '%'` / escape functions — workable but **`strpos`** is simpler and fast enough with B-tree-friendly predicate on `(listener_id)` pre-filter.
- Full-text / trigram index — **rejected** for v1 (spec scope is modest library sizes).

## 3. When to create `album_list_sessions`

**Decision**: Insert a session row **only** when the list has **more than one page** (`total_count > 5`). Single-page results, **empty library**, and **zero filter matches** **do not** need a session.

**Rationale**: Avoids pointless rows and keeps **`/list next`** from implying more pages when there are none.

## 4. “Latest session” for `/list next` / `/list back`

**Decision**: On **`/list next`** or **`/list back`**, resolve the listener’s **most recently created** **`album_list_sessions`** row where **`expires_at > now()`**, then fetch **page ± 1** within **total page count** (clamp). If **no** session, reply with a **short** hint to run **`/list`** again.

**Rationale**: Satisfies spec **FR-006** text fallback without ambiguous **`n`/`p`** single-letter commands; **`/list`** with **multi-word artist** remains **page 1** for that filter.

**Alternatives considered**:

- Only callbacks (no text paging) — **rejected** (spec explicitly requires documented text fallback).
- Per-message “reply next” without session — **rejected** (cannot identify filter + total pages).

## 5. Update list message vs send a new message (Telegram)

**Decision**: Prefer **`EditMessageText` (+ edit reply markup)** when the host provides **`message_id`** from the **`callback_query`**, so **Back**/**Next** feels like **one** evolving message. If edit fails (e.g. message too old), **fall back** to sending a **new** message with the same content.

**Rationale**: Better UX and matches common bot pagination patterns.

**Alternatives considered**: Always send new messages — simpler but noisier; acceptable as fallback only.

## 6. Empty listener row

**Decision**: If **no** `listeners` row exists for `(source, external_id)`, treat **saved album count** as **0** and show **empty-library onboarding** (same as listener with zero saves).

**Rationale**: **`UpsertListener`** today runs on **save**; **`/list`** should not require a prior save to **create** a listener row unless product later requires it.

## 7. Sort order

**Decision**: **`ORDER BY created_at DESC, id DESC`** for stable ordering when timestamps collide.

**Rationale**: Matches spec assumption (**newest saved first**) and adds **tie-break**.

## 8. Readable list lines (**FR-010**) and button captions

**Decision**: Format each album line using the **same** **`Title | Artist (Year)`** conventions as **`/album`** disambig; when text exceeds **Telegram** message or **inline keyboard** label limits, **truncate or abbreviate** in **core** and/or **`adapter/telegram`** (reuse **`truncateForButton`** / analogous patterns). **Never** mutate **`saved_albums`** rows for display.

**Rationale**: [spec.md](spec.md) **FR-010** and Clarifications (analysis **U1**).

**Alternatives considered**: Wrap-only layout — **rejected** (Telegram messages are not rich layout); storing shortened titles — **rejected** (violates FR-010).

## 9. Automated tests (**SC-002**, **SC-003**, routing)

**Decision**: (1) **Unit/table** tests for **`NormalizeArtistQuery`** / **`ParseListLine`**. (2) **Store** tests: same **ordered `saved_albums.id` list** for several **normalized** filter strings on a **fixture**; assert **listener A** never sees **listener B** rows. (3) **`run_test.go`**: **private handler** order — **`/list`** paths must not consume **`1`/`2`/`3`** meant for **`/album`** disambig and vice versa.

**Rationale**: [spec.md](spec.md) **Testing** NFR, **SC-002**/**SC-003**, Clarifications **U2**/**C1**; aligns with **`tasks.md` T010, T015, T016** (full gate **T017**).
