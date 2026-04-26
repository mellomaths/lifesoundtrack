# Feature Specification: LifeSoundtrack — list saved albums (`/list`)

**Feature Branch**: `006-list-saved-albums`  
**Created**: 2026-04-26  
**Status**: Draft  
**Input**: User description: "Add command to the LifeSoundtrack bot that lists the albums saved by the user. Command: `/list`. The user should be able to query a specific artist as well, for example using the command like `/list The Beatles`. The bot should be smart enough to normalize the query before searching in the database. The response should be paginated with 5 albums per page and buttons `Back` and `Next` (if platform allows) to allow the users change pages. If the user has not registered an album yet, the bot should teach the user how to register one."

## Clarifications

### Session 2026-04-26

- Q: Specification analysis **A1** — artist filter field breadth vs data model → A: For **v1**, the artist filter matches **only** the **primary artist** string stored on each saved album (the same single display field used when the album was saved). **Multi-artist / featured / track-level** credits are **out of scope** for matching unless folded into that field later.
- Q: **U1** — very long titles in list output → A: List lines MUST remain **readable**; when a title or artist string is unusually long, the product MAY **shorten the visible line** using the **same display conventions** as other album labels in the bot, **without** changing stored album data.
- Q: **U2** / **SC-002** — proving identical match sets → A: Validation MUST include checks (automated where practical) that **several normalized variants** of the same filter yield the **same set of matching saved albums** for a fixed fixture library.
- Q: **C1** — `/album` disambig vs `/list` routing → A: Automated tests MUST cover **private-message routing order** so **numeric disambiguation** for **`/album`** and **`/list`** paging do **not** regress each other.
- Q: **I2** — where operators learn text paging → A: **Text fallback** (**`/list next`**, **`/list back`**) MUST be documented in the project’s **primary contributor README** at the repo root **and** in this feature’s **quickstart** runbook.
- Q: Specification analysis **I1** — when to persist **`album_list_sessions`** → A: **Only** when the result set needs **more than one page** at page size **5** — i.e. **`total_count > 5`**, equivalently **`total_pages > 1`**. Do **not** use **`total_pages > 5`** (that would mean **31+** albums). Align **tasks**, **plan**, **research**, and **data-model** on this threshold.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See my saved albums, five at a time (Priority: P1)

A **listener** sends a **list** command with **no** text after the command. The product shows **up to five** of that listener’s **saved albums** on the **current page**, in a **consistent, predictable order** (see assumptions), and indicates **which page** they are on when there is more than one page.

**Why this priority**: This is the core value—surfacing the personal library the product already stores.

**Independent Test**: With a listener who has several saved albums, issue `/list`; verify only five appear per response, order is stable across repeated calls, and paging controls or instructions appear when the total exceeds five.

**Acceptance Scenarios**:

1. **Given** a listener with **more than five** saved albums, **When** they send the list command **with no query text**, **Then** the reply shows **exactly the first five** albums for **page 1** (per ordering assumptions) and offers a way to move to the **next** page (see User Story 4 for platform behavior).
2. **Given** a listener with **between one and five** saved albums, **When** they send the list command **with no query text**, **Then** the reply lists **all** of their saved albums and **does not** imply additional pages.
3. **Given** a listener with **no** saved albums, **When** they send the list command, **Then** the reply **does not** show an empty table; it **explains briefly how to add** their first album using the product’s **existing save/register flow** (wording may vary; must be actionable and non-technical).

---

### User Story 2 - Filter my list by artist (Priority: P1)

A **listener** sends the list command **followed by artist text** (e.g. a band or performer name). The product **normalizes** that text and returns **only** saved albums whose **primary artist** string (**FR-003**, same field as after save) matches the normalized query, still **paginated** at five per page.

**Why this priority**: Users with large libraries need a fast way to find albums by creator without scrolling the full list.

**Independent Test**: Save albums from different artists; call the command with an artist substring; verify results only include matching saved albums, and that harmless variations in user typing (spacing, letter case) still find the same rows after normalization.

**Acceptance Scenarios**:

1. **Given** a listener with saved albums from **multiple** artists, **When** they send the list command with **non-empty** artist text that **matches** at least one saved album (after normalization), **Then** the reply shows **only** matching albums, **up to five** on the current page, with paging when matches exceed five.
2. **Given** the same listener, **When** they send artist text that **matches no** saved album (after normalization), **Then** the reply states clearly that **no saved albums** match **without** suggesting a system error, and **does not** show unrelated albums.
3. **Given** artist text with **leading or trailing spaces** or **irregular spacing**, **When** the listener sends the command, **Then** normalization is applied **before** matching so that the same logical query behaves consistently.

---

### User Story 3 - Normalization behaves predictably (Priority: P1)

**Normative rules**: **FR-004**. The product applies **normalization** to optional artist query text so that matching is **forgiving** of casual typing but **predictable** for testing and support.

**Why this priority**: Avoids false “no results” when the user types the same name slightly differently than stored display text.

**Independent Test**: Use equivalent queries that differ only by case, extra spaces, or Unicode spaces (if applicable); outcomes should match the documented normalization rules.

**Acceptance Scenarios**:

1. **Given** stored artist display values in the listener’s library, **When** two user-entered queries differ **only** by **letter case** (e.g. `the beatles` vs `THE BEATLES`), **Then** matching results are the **same** after normalization.
2. **Given** user-entered artist text, **When** normalization runs, **Then** **leading and trailing** whitespace is removed and **internal** runs of whitespace are collapsed to a **single** space before comparison (assumption; see Assumptions).

---

### User Story 4 - Move between pages (Priority: P2)

When the host **supports** **interactive controls** (e.g. labeled buttons) for bot replies, the product provides **Back** and **Next** (or equivalent labels) so the listener can change pages **without** retyping the command. When the host **does not** support such controls, the product still allows paging through an **accessible fallback** (e.g. a short text instruction or command pattern) documented for operators.

**Why this priority**: Pagination without navigation frustrates users; platform limits require a defined fallback.

**Independent Test**: On a platform with buttons, verify Next/Back appear only when applicable and lead to the correct slice. On a platform without buttons, verify the fallback is explained in the same reply or follow-up guidance.

**Acceptance Scenarios**:

1. **Given** more than one page of results and a **button-capable** host, **When** the user is on **page 1**, **Then** **Next** is available and **Back** is **absent or disabled** (per host conventions), and **Next** shows **page 2**’s items.
2. **Given** the user is on the **last** page, **When** they view controls, **Then** **Next** is **absent or disabled** and **Back** is available when there is a previous page.
3. **Given** a **button-limited** host, **When** results span multiple pages, **Then** the user receives **clear** instructions for how to request the **previous** or **next** page without relying on buttons.

---

### Edge Cases

- Listener sends the list command with **only** whitespace after it → treat as **no artist filter** (same as bare `/list`).
- Listener has albums saved but **zero** match the artist filter → **no-match** message distinct from **empty library** (Story 1 vs Story 2).
- **Exactly five** albums or matches → **one** page; no **Next**.
- **Concurrent** list requests from the same listener → each response reflects **current** saved data; paging state may be **per message** or **session** per product conventions (see Assumptions).
- **Very long** artist or album titles → each list line remains **readable**; the product MAY shorten visible text per **FR-010** (same display conventions as other album labels; no raw error strings).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST expose a **list** command (user-facing invocation: `/list`) that returns **saved albums for the requesting listener only**.
- **FR-002**: When the listener invokes the command **with no artist query** (or only whitespace), the product MUST include **all** of that listener’s saved albums in the result set subject to pagination.
- **FR-003**: When the listener invokes the command **with non-empty artist text** after the command, the product MUST restrict results to saved albums whose **primary artist** string stored for that save (the same field used when displaying the album after save) matches the query **after normalization** (see FR-004). **v1** does **not** match on separate credited, featured, or track-level artist fields unless they are already represented in that **primary artist** string.
- **FR-004**: The product MUST **normalize** artist query text by **trimming** leading/trailing whitespace and **collapsing** internal whitespace to single spaces, and MUST perform **case-insensitive** comparison against stored artist text used for matching.
- **FR-005**: The product MUST show **at most five** saved albums **per page** in list responses.
- **FR-006**: When total results exceed five, the product MUST support **moving to the previous and next page**; on hosts that support **inline controls**, the product MUST offer **Back** and **Next** (or equivalent) consistent with host guidelines; on hosts that do not, the product MUST provide a **documented text fallback** for paging in **operator-facing** materials (**root project README** and this feature’s **quickstart**, per Clarifications) and a **user-facing** hint in the conversation.
- **FR-007**: When the listener has **no** saved albums at all, the product MUST **not** return a blank list alone; it MUST include **brief guidance** to **register/save** a first album using the **existing** user-facing save flow (command name and example may reference current product copy).
- **FR-008**: List responses MUST use **non-technical** language, MUST **not** expose other listeners’ libraries, and MUST **not** leak secrets or internal identifiers unnecessarily.
- **FR-009**: Ordering of albums within the list MUST be **stable** for a given listener until their library changes (see Assumptions for default ordering).
- **FR-010**: For **very long** album titles or artist names, each **visible list line** MUST remain **readable**; the product MAY **abbreviate or shorten** display text using the **same conventions** as other album labels in the bot. Stored album data MUST **not** be altered solely for listing.

### Non-functional and product quality *(should align with project constitution)*

- **UX consistency**: Align command prefix, tone, and help text with existing LifeSoundtrack bot commands; pagination labels (`Back` / `Next`) should match platform norms where translated or localized copy exists.
- **Testing**: Each user story is independently verifiable with a listener fixture and controlled saved-album data; normalization and paging require explicit expected vectors (case, spacing, page boundaries). Where automated tests exist, they SHOULD assert that **multiple normalized variants** of the same artist filter return the **same matching saved-album identities** on a fixed dataset (supports **SC-002**). Automated tests SHOULD also guard **listener isolation** (supports **SC-003**).
- **Linting**: No specific requirement beyond repository defaults for any new code.
- **Logging and monitoring**: Operational logs may record list usage counts and failures **without** logging full album lists or message content if privacy policy restricts it (follow project standards).
- **Performance**: Listing SHOULD complete within **ordinary interactive chat expectations** for typical library sizes (hundreds of saved rows per user); no hard numeric SLA required unless constitution adds one.
- **Containerization**: No change to deployment shape unless implementation introduces a new runnable component (unlikely for a command-only feature).
- **Decommissioning**: If the list command is removed later, remove user-facing help references and operator docs in the same change.

### Key Entities *(include if feature involves data)*

- **Listener**: The person using the chat product; identified the same way as for other bot commands. List results are always scoped to this identity.
- **Saved album (library entry)**: An association between a listener and an album record, including **display fields** needed to list an item (e.g. title, artist(s), optional year) as already stored by the save/register feature.
- **List query (optional)**: Free text after the command, interpreted **only** as an **artist filter** when non-empty; not a full-text search of arbitrary metadata unless expanded in a future spec.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In usability tests or staged checks, **100%** of trial listeners with **10+** saved albums can reach the **second page** of results within **two** interactions when the host supports paging controls (or within **one** extra message when using the text fallback).
- **SC-002**: For **ten** curated text variants of the same artist name (differing only by case and spacing), **normalized** matching yields **identical** result sets in **at least 95%** of cases (allowing rare intentional exceptions only if documented, e.g. special characters).
- **SC-003**: **Zero** list responses in acceptance testing **expose** another user’s album titles or **show** empty content **without** onboarding hints when the listener’s library is empty.
- **SC-004**: Support or pilot feedback: **at least 90%** of surveyed listeners **understand** how to add a first album after seeing the empty-library message (measured via short post-task survey or interview).

## Assumptions

- **Save/register flow exists**: “Register an album” refers to the **already shipped** save-album command and flow; the list feature **does not** redefine how albums are added.
- **Default sort order**: Saved albums are listed in **reverse chronological order of when they were saved** (newest first). If that field is unavailable, **alphabetical by album title** is an acceptable implementation default—either choice MUST remain stable until changed by a future spec.
- **Artist matching field**: Matching uses **only** the **primary artist** string stored on each saved album (**FR-003**). **Featured artists** or **track-level** credits are **out of scope** unless already represented in that string.
- **Paging state**: **Per-message** paging (user taps Next on a specific reply) is sufficient; **cross-session** resume of “page 3” is **not** required for v1.
- **Platforms**: The product may ship on **one or more** chat hosts; button availability follows each host’s capabilities. Fallback text for paging is **required** where buttons are unavailable.
