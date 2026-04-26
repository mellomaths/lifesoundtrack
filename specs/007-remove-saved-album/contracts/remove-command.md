# Contract: `/remove` command (domain + Telegram)

**Spec**: [spec.md](../spec.md) | **Plan**: [plan.md](../plan.md) | **Date**: 2026-04-26

## Command grammar (private chat)

| Input pattern | Meaning |
|---------------|---------|
| `/remove` or `/remove@bot` with **no** remainder (or only whitespace) | **Usage** error: album name **required**; **no** delete. |
| `/remove <text>` | **Album query** = entire remainder after the first token; may contain spaces. |

**Notes**:

- **Bot mention**: First token may be `/remove@BotName`; parser strips **`@suffix`** (same pattern as **`ParseListLine`** / **`ParseAlbumLine`**).
- **v1** does not parse an `Artist - Title`-style string into separate fields. The **entire** remainder is normalized and matched in **two** **phases** (see [spec.md](../spec.md) **FR-003**; [research.md](../research.md) §2).

## Normalization (title match)

Use **`NormalizeArtistQuery`** from `internal/core/parse_list.go` on:

1. The **user remainder** (full string after `/remove`).
2. Each **stored** `saved_albums.title` when evaluating a match.

**Phase 1 — exact:** normalized user string **==** normalized stored **title** (both non-empty after trim). **0** **→** go to **phase 2**; **1** **→** delete; **≥2** **→** disambig (numbered pick).

**Phase 2 — partial (only if phase 1 found 0 rows):** normalized stored title **contains** the normalized user string as a **contiguous** **substring** (`strings.Contains` in Go on normalized values). **0** **→** not found; **1–3** **→** disambig (including **1** **row**; **no** auto-delete on partial only); **>3** **→** user message to **narrow** the search **without** **listing** all **matches**.

**Unicode spaces**: As for **`/list`** — `strings.TrimSpace` + `strings.Fields` + join with single ASCII space, then `ToLower`.

## Outcomes (domain → user text)

| Case | DB | User |
|------|----|----|
| Empty query | None | Short **usage** copy (like other empty commands). |
| **0** matches in **both** **phases** (exact then partial) | None | **Not found** — **no** row deleted, non-technical wording. |
| **1** **exact** match | `DELETE` that row | **Success** with **at least** stored **title** (and optional artist/year **like** list lines). |
| **≥2** **exact** **or** **1–3** **partial** (no **exact** **hit** **for** that **search**) | `INSERT` `disambiguation_sessions` with `kind: remove_saved` | Numbered list **(for** **readability)** + **explicit** **pick** **(see** **below)**. **N** as **candidates** **count**; **for** **partial** **tier** **N** **≤** **3** per [spec.md](../spec.md) **FR-003**. **No** **auto-delete** when **only** **partials** **apply**. |
| **>3** **partial** matches (no **exact** **hit**) | None | **Narrow** **search** **copy** **only** (no full list of all rows). |
| Stale/invalid pick | None or optional cleanup | **Out of range** / **no active session** copy consistent with **save** disambig. |

## Explicit pick (after disambig message)

- **Text**: User sends a **line** that is **only** a **decimal** **1**–**N** (implementation: **1**–**99** max **index**; **N** = **len(candidates)**). **Session** = **latest** open **`disambiguation_sessions`** for that **user** with **`kind: remove_saved`**.
- **Telegram (required for SC-003)**: **Inline** **keyboard** with **one** **button** **per** **candidate** (same order as text lines). **`callback_data`**: `rmp:<disambiguation_session_id>:<1-based_index>` (see [research.md](../research.md) **§10**); must fit **≤** **64** **bytes** total. **Tapping** a **button** must **complete** the **same** **remove** as **sending** the **index** in **text**.

## Disambiguation JSON (`disambiguation_sessions.candidates`)

**Shape** (must be a **JSON object** at root, not an array — distinguishes from **save-album** sessions):

```json
{
  "kind": "remove_saved",
  "candidates": [
    { "id": "uuid-string", "label": "Title | Primary Artist (Year)" }
  ]
}
```

- **`label`**: Reuse the same line formatting as **`/list`** / **`FormatSavedAlbumLine`** for readability (button **caption** may **truncate** per Telegram **limits**; keep **unambiguous** where possible).
- **Pick (text)**: **Adapter** / **core** run **remove** **pick** **before** **`/album`** **pick** for **private** **text** **matching** `1`–`N` (see [research.md](../research.md) §4–5).
- **Pick (callback)**: **Adapter** **parses** `rmp:` and **invokes** the **same** **domain** **operation** as **text** **index** (see [research.md](../research.md) §8, **§10**).

## Telegram transport

- **FR-006** / **SC-003**: **Disambig** **message** **MUST** **include** **inline** **keyboard** **buttons** (**one** **per** **candidate**); **typed** **index** **alone** is **not** the **only** path.
- **Callback** **prefix** **`rmp:`** — new **pattern**; **no** **collision** with **`lpl:`**, **`apick:`**, **`aother`**.
- **Numeric** **text** **pick** **remains** **supported** and **MUST** **pass** **acceptance** as **regression** / **accessibility** **alternate** ([spec.md](../spec.md) **Clarifications**).

## Help (`/help`)

Domain **`helpCopy`** must include **`/remove`** with a **one-line** **description** and list **all** other **shipped** **commands** for this release ([spec.md](../spec.md) **FR-008**). Verify with a **string** or **test** **fixture** in CI ([spec.md](../spec.md) NFR **Testing**).

## Testing requirements (traceability)

- **Normalization** table tests: **case** / **whitespace** / **equivalence** pairs.
- **Listener** **A** **cannot** **delete** **listener** **B**’s row (store or integration test).
- **Help** string includes **`/remove`** and **no** **regression** on **`/list`** / **`/album`** **mentions** (spot-check or golden substring).
