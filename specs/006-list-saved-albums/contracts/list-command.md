# Contract: `/list` command (domain + Telegram)

**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md) | **Date**: 2026-04-26

## Command grammar (private chat)

| Input pattern | Meaning |
|---------------|---------|
| `/list` or `/list@bot` | Page **1**, **all** saved albums (no artist filter). |
| `/list` + only whitespace after command | Same as bare **`/list`**. |
| `/list <text>` | Page **1**, **artist filter** = entire remainder after command (may contain spaces). |
| `/list next` (case-insensitive token) | **Next** page using **latest non-expired** `album_list_sessions` for this listener. |
| `/list back` | **Previous** page (same). |

**Notes**:

- **`/list next`** as artist name: edge case — if product needs artist literally **`next`**, user may use a **leading** workaround in a future spec; v1 accepts **`next`**/**`back`** only when the remainder is **exactly** that token (after trim).
- **Bot mention**: First token may be `/list@BotName`; parser strips **`@suffix`** like **`ParseAlbumLine`**.

## Normalization (artist filter)

**Matching scope (v1)**: The filter is a **case-insensitive substring** against the **primary artist** string stored for each saved album (same display field as the save flow). See [spec.md](../spec.md) **FR-003** / **FR-010**.

Applied to **filter text** before persistence in session and SQL:

1. Trim leading/trailing Unicode spaces.
2. Replace **internal** runs of whitespace (any Unicode space per `strings.Fields` behavior) with a **single ASCII space**.
3. For **SQL** substring test, use **`strings.ToLower`** on the normalized string as **`$needle`**.

**Not** in v1: diacritic folding, transliteration, “The ” article stripping.

## Page size

**5** albums per page (constant in core, e.g. `ListPageSize = 5`).

## Display format

Each album line **should** mirror save/disambig readability:

- **`Title | Primary Artist (Year)`** with **year omitted** when `year` IS NULL or zero — **same rules** as disambiguation labels (reuse **`formatAlbumLine`** / shared helper in **`internal/core`** with **`save_album`**).

**Header/footer** (non-technical copy):

- Indicate **page** `current` **of** `total` when `total_pages > 1`.
- **Do not** show raw UUIDs or internal IDs.

## Telegram transport

### Inline keyboard (when `total_pages > 1`)

- Row: **`Back`** | **`Next`** (or two rows if UX prefers — default **one** row).
- **`Back`** omitted or **disabled** on page 1 — Telegram has no native disabled; **omit** button or use callback that no-ops with toast “Already on first page” (**prefer omit**).
- **`Next`** omitted on last page.

### `callback_data`

- Pattern: **`lpl:`** + **session UUID** + **`:`** + **1-based page** integer.
- Handler **must** verify: session **`listener_id`** matches clicking user’s listener, session **not expired**, page within **1..total_pages**.

### Callback UX

- **AnswerCallbackQuery** (empty or subtle toast on noop).
- Prefer **edit** the **same** message’s text and markup when possible.

## Outcomes (core → adapter)

| Outcome | User-visible behavior |
|---------|------------------------|
| **Empty library** | No list lines; **onboarding** to **`/album`** with example (reuse tone from [copy.go](../../../bot/internal/core/copy.go)). |
| **No matches** (filter) | Clear **“no saved albums match”** message; **not** the empty-library text. |
| **Page of results** | Formatted list + page indicator + keyboard if needed. |
| **Session missing** (`next`/`back`) | Short hint: start with **`/list`** or repeat filter. |

## Security

- List queries **must** include **`listener_id`** from resolved identity — **never** accept listener id from user text.
- Log **no** full result payloads at **Info** in production.

## Compatibility

- **`/album`** disambiguation still consumes plain-text **`1`**, **`2`**, **`3`** — list paging **must not** use those digits as page navigation.
