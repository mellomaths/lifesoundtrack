# Contract: `album` (save) — domain and Telegram mapping

**Spec**: [spec.md](../spec.md) | **Date**: 2026-04-24

## Domain command: `save_album` (name)

| Field | Type | Description |
|-------|------|-------------|
| `query` | string | Free-form, **non-empty** after trim; no schema enforced. **Max length** (plan): 512 runes, enforced by adapter. |

**Outcomes (abstract)** — map to [metadata-orchestrator.md](metadata-orchestrator.md):

1. **empty_query** — reply with “need search text” (no metadata call, no save).
2. **candidates** (**two** **or** **more** **distinct** **labels** after **orchestrator** **+** **core** **dedupe**; **2** **shown** + **Other**) — present **at most 2** **distinct**-**by**-**label** album options **by** **relevance** and a distinct **Other** path. If **all** **raw** **candidates** **share** one **user**-**visible** **label**, **do** **not** use this outcome—treat as **`single_effective_match`** (same **persistence** path **as** **single** **match** **for** that **kept** **row**). Each album line’s **label** is **`ALBUM_TITLE | ARTIST (YEAR)`** (year optional). **Disambig** **choice**: `pick_album` (index **0** or **1** into the **two** **offered**) or `pick_other` (no save, send refinement help).
2b. **single_effective_match** (optional domain name) — **all** **raw** **candidates** **collapsed** to **one** **distinct** **label**; **kept** **row** is the **highest**-**relevance** **(first)** **among** **equivalents**; **no** **disambig** **list**; **proceed** to **save** (same **as** **single_match** for **persistence** **semantics** **unless** **confidence** **policy** **differs**—align **in** **core**).
3. **single_match** — optional auto-save (when **one** **raw** **result** **or** **plan** **treats** as **unambiguous**).
4. **no_match** — user-facing “not found.”
5. **provider_exhausted** — all **metadata** **feature** **flags** **off** (nothing **to** **call**), all **relevant** **breakers** **open**, or **chain** **exhausted** with **no** **candidates** / **infrastructure** **dead**: user sees a **short** **generic** **“try** **again**” or **“check** **configuration**” **style** **message** (no **raw** **env** **names** in **chat**).
6. **saved** — confirmation line with title / year, **no** full provider JSON.
7. **other_chosen** — user picked **Other**; **no** `saved_albums` row; message includes guidance (e.g. add full **album title**, **artist**, **release year** to the next query).

**Errors**: Must **not** include API keys, connection strings, or other users’ data ([spec FR-007](../spec.md)).

## Telegram mapping (v1)

| Telegram | Domain |
|----------|--------|
| `/album` with **no** text, or only whitespace | `empty_query` |
| `/album Red` (example) with text | `save_album` with `query = "Red"` (trim) |
| Inline / reply: **two** album buttons (label = `Title \| Artist (Year)`) + **Other**; or text: same labels + **Other** line, next message e.g. `1` / `2` / `3` or `other` (adapter normalizes) | `disambiguation_pick` → `pick_album` (0/1) **or** `pick_other` |

**Help**: `/help` text lists `/album` with one-line description (implementation task in core copy).

## Persistence contract

- A **save** is committed **only** after a **definite** `AlbumRef` (see metadata contract) and **listener** are known.
- **No** `saved_albums` row on `no_match`, `provider_exhausted`, or **`other_chosen`**.

## Acceptance

Matches [spec.md](../spec.md) **US1, US1b, US2, US3**; **FR-001**–**FR-009** (including **equivalent**-**label** **collapse** and **SC-007**); and [data-model.md](../data-model.md).
