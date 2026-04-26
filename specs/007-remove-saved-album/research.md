# Research: `/remove` saved album

**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Date**: 2026-04-26

## 1. Title query normalization (aligned with `/list`)

**Decision**: Reuse **`NormalizeArtistQuery`** from [`parse_list.go`](../../bot/internal/core/parse_list.go) (trim Unicode space, `strings.Fields` to collapse internal whitespace, **`strings.ToLower`**) for **both** the user’s **`/remove`** remainder and each stored **`saved_albums.title`** before comparison.

**Rationale**: [spec.md](spec.md) **A1** and **FR-002** require the same conventions as the list feature’s query normalization; a single function avoids drift between commands.

**Alternatives considered**:

- Duplicate normalization in SQL only — **rejected** (harder to match Go’s Unicode/Fields behavior; tests are clearer in Go).

## 2. Matching rule (v1) — two phases + partial cap

**Decision**: **Same** **normalization** for **all** **comparisons** (see **§1**). **(1) Exact:** **`NormalizeArtistQuery(row.Title) == NormalizeArtistQuery(userRemainder)`** — **0** **rows** **→** **(2)**; **1** **row** **→** **DELETE**; **≥2** **→** **disambig** **pick**. **(2) Partial (substring):** only if **(1)** **returned** **no** **rows** — a row **matches** if **`strings.Contains(NormalizeArtistQuery(row.Title), NormalizeArtistQuery(userRemainder))`**. **0** **→** not found; **1–3** **→** **disambig** **pick** (including a **single** **partial**; **no** **auto-delete** on **partials**); **>3** **→** **user** **message** **to** **narrow** **(no** **list** of **all** **rows**). Remainder **empty** after trim **→** **usage** (not a match phase). **v1** does **not** **parse** `Artist - Title` into **separate** **search** **fields** ([spec.md](spec.md) **A2** / **Clarifications** 2026-04-26).

**Rationale**: **Exact** **tier** **keeps** **predictable** **one-shot** **removes**; **partial** **covers** **shorter** **user** **queries** **(e.g.** **Abbey** **Road** **vs** **Abbey** **Road** **(Remastered)**). **>3** **partials** **avoids** **spamming** **a** **long** **list** and **nudges** **a** **clearer** **query**.

**Alternatives considered**:

- Substring **only** — **rejected** (too many false positives); **current** design **uses** **exact** **first**.
- List **all** **partial** **matches** when **>3** — **rejected** by **Clarifications** 2026-04-26.
- Relying **on** **SQL** for **filtering** **only** with **no** in-process **list** — **rejected** for **(2)** **partials** **(need** to **re-run** same **Go** **normalization** as **exact** **tier;** list **all** **rows** for **listener** is **OK** for **v1** **scale** per **list** spec).

## 2a. Query length (spec **A3**)

**Decision**: Enforce the **same** (or a **stricter**) **rune** **ceiling** on the **`/remove`** **remainder** as **`/album`** free-text (`MaxQueryRunes` in `internal/core`, aligned with the save flow). If over limit: do **not** run **DELETE**; return user-visible limit copy (see [tasks.md](tasks.md) **T005** / **T007**).

**Rationale**: [spec.md](spec.md) **A3** and **Clarifications** (Session 2026-04-26); avoids abuse and matches **/album** expectations.

**Alternatives considered**: Separate higher limit for **remove** — **rejected** for v1 (spec points at **stricter of** other **commands**).

## 3. Reusing `disambiguation_sessions` for remove picks

**Decision**: **Insert** rows into existing **`disambiguation_sessions`** with **`candidates`** JSON **object**:

```json
{
  "kind": "remove_saved",
  "candidates": [
    { "id": "<uuid>", "label": "Title | Artist (Year)" }
  ]
}
```

**Legacy** save-album disambig remains a **JSON array** at the root (`[]AlbumCandidate` — no `kind` field) so the dispatcher can **tell** session types apart without a migration.

**Rationale**: No new table; **TTL** and **LatestOpen** behavior already match “pick a row” flows.

**Alternatives considered**:

- Separate `remove_disambig_sessions` table — **rejected** for v1 (duplicates infrastructure).
- In-memory only — **rejected** (spec expects multi-replica–safe session storage like save flow).

## 4. Remove disambig: Telegram inline buttons + numeric text (spec **FR-006**, **SC-003**)

**Decision**: **(A)** For **disambig** replies, **Telegram** **MUST** attach an **inline** **keyboard** with **one** **button** **per** **candidate** (same **order** as **numbered** **message** **lines**). **Button** **labels** are **user-visible** and **MUST** respect Telegram’s **short** **caption** **limit** (truncate **like** list **/album** **line** **display** if needed; see **code** `telegramInlineButtonTextMax`). **(B)** **`callback_data`** **uses** a **dedicated** **prefix** (plan: **`rmp:`**) + **`disambiguation_sessions.id`** (UUID) + **`:`** + **1-based** **index** — same **length** **pattern** as **`lpl:`** **+** **uuid** **+** **page** (fits **≤** **64** **bytes**). **Handler** **calls** the **same** **domain** **path** as **text** **pick** (delete **by** **candidate** **id** with **listener** **check,** **then** **delete** **session**). **(C)** **Private** **text** that **matches** **`RemovePickIndexFromText`** (1…99) **still** **tries** **`TryProcessRemovePick`** before **`/album`** **pick**; **unchanged** **semantics** so **E2E** and **text-only** **clients** pass **SC-003** **alternate** path. **(D)** Reuse or mirror **`TryProcessRemovePick`** for **index** from **callback** (same **session** **+** **index**).

**Rationale**: [spec.md](spec.md) **Clarifications** 2026-04-26 (tester **feedback:** **index-only** **UX** is **not** **sufficient** on **Telegram** **when** **buttons** **are** **available**); **dual** path matches **/album** **(buttons** **+** **send** **1/2/3**).

**Alternatives considered**:

- **Text-only** **v1** — **rejected** (superseded by spec **FR-006** / **SC-003**).
- **Button-only** (drop **numeric** **text**) — **rejected** ([spec.md](spec.md) requires **text** **pick** **as** **alternate**).

## 5. `OneBasedPickFromText` vs `12` and multi-digit

**Decision**: **`OneBasedPickFromText`** remains for **single-digit** `1`/`2`/`3` only (**album** UX). For **remove**, use **`strconv.Atoi`** on **trimmed** text when it matches **`^[0-9]{1,2}$`** and **1…99**, **only** inside the **remove** dispatcher (after **`/remove`** disambig was shown). **“12”** is **not** a valid **album** pick today (len ≠ 1), so it **won’t** route to **`ProcessPickByIndex`**; it will be tried as **remove** pick first.

**Rationale**: Supports **>3** duplicate saves without changing **album** button model.

**Alternatives considered**:

- Extend `OneBasedPickFromText` to 1–9 only — **rejected** (insufficient if **N > 9**).

## 6. Delete statement

**Decision**: `DELETE FROM saved_albums WHERE id = $1::uuid AND listener_id = $2::uuid` (returning whether a row was deleted).

**Rationale**: **Idempotent** safety: never delete another listener’s row even if id leaked into JSON.

## 7. Help copy (**FR-008**, **SC-004**)

**Decision**: Update **`helpCopy`** in [`copy.go`](../../bot/internal/core/copy.go) to add a **`/remove`** bullet (one line). Keep **`/start`**, **`/help`**, **`/ping`**, **`/album`**, **`/list`** lines accurate with **this** release.

**Rationale**: Single source of truth in domain copy; [spec.md](spec.md) **A5**.

**Alternatives considered**:

- Adapter-only help — **rejected** (violates domain-first messaging **001**).

## 8. Routing order (Telegram private chat)

**Decision**: In **`handleMessage`**, after **`/album`** and **`/list`**, add **`/remove`** parsing + **`HandleRemove`** before **any** **numeric** pick handling. **Numeric** path: **(1)** try **remove** pick if latest session is **`remove_saved`**; **(2)** else existing **`OneBasedPickFromText` + ProcessPickByIndex** for **album**. In **`handleCallback`**, after **`lpl:`** handling, if **`q.Data`** **hasPrefix** **`rmp:`**, **parse** and **invoke** the **same** **library** **remove** **pick** as **text** (then **answer** **callback** **query**).

**Rationale**: Matches [spec.md](spec.md) Testing NFR and **006** spirit (**C1**-style): **`/list`** and **`/album`** and **`/remove`** must not **starve** each other; extend **`routing_test.go`**; **callback** **is** **not** **a** **private** **message** — **dedicated** **branch** in **`handleCallback`**.

## 8a. Stale disambig after single-candidate `/album` save

**Decision**: **Single-result** **save** paths that **call** **`persistSave`** with **`disambigID == nil`** **MUST** **invoke** **`DeleteDisambigForListener`** before **insert,** so a **stale** **`remove_saved`** (or **abandoned** **album** **disambig**) is **not** **left** as the **latest** open session when the **user** **later** **sends** **`1`**. **Documented** in [plan.md](./plan.md) **Summary**; **implementation** in **`save_album`**. (If **already** **implemented,** keep **regression** **tests** around **`DeleteDisambigForListener`**.)

**Rationale**: **Prevents** **accidental** **remove** or **wrong** **album** **pick** when **new** **activity** **supersedes** an **old** **session** (see `persistSave` in core).

## 9. Automated tests

**Decision**: (1) **Table tests** for **`ParseRemoveLine`** and **normalize** equivalence classes. (2) **Store** test: **delete** only within listener. (3) **Core** test: **0 / 1 / 2+** match outcomes, **partial** **tier** **(1**–**3,** **>3)**. (4) **`routing_test.go`**: order of branches for **`/remove`**, **numeric** remove pick vs **`/album`** pick. (5) **Assert** **`helpCopy`** **contains** **`/remove`**. (6) **Unit** test **`parseRemovePickCallbackData`** (or **shared** **parse** **with** list **callback** **style**) for **`rmp:`** **payloads** under **64** **bytes**. (7) **Adapter** or **integration** test that **remove** **disambig** **SendMessage** includes **non-nil** **ReplyMarkup** (inline **keyboard** when **N** **≥** **1**).

**Rationale**: [spec.md](spec.md) **SC-001**–**SC-004** and NFR **Testing**; **SC-003** **Telegram** **button** path.

## 10. `callback_data` format (`rmp:`)

**Decision**: Use **`rmp:`** (remove pick) as **prefix** to **avoid** collision with **`lpl:`** and **`apick:`** / **`aother`**. **Format**: `rmp:<disambiguation_session_uuid>:<1-based_index>` with **index** in **1…99** (string of **1** or **2** **digits**). **Parse** with **split** on **`:`** after **prefix**; **validate** **uuid** and **index** **range** against **session** in **core** (same as **text** **pick**).

**Rationale**: **Telegram** **64-byte** cap; **session** **id** **binds** **callback** to **stored** **candidates** **JSON**; **index** **selects** **row** without **embedding** **album** **uuid** in **callback** (shorter, **one** **source** of **truth** in **DB**).
