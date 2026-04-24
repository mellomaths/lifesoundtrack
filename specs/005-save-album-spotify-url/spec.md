# Feature Specification: Save album via Spotify album link

**Feature Branch**: `005-save-album-spotify-url`  
**Created**: 2026-04-24  
**Status**: Draft  
**Input**: User description: "Allow the user the use the save album function with a spotify link to the album page. For example: `/album https://open.spotify.com/intl-pt/album/1fneiuP0JUPv6Hy78xLc2g?si=m5CsZ66-SQ-SLuSgOr1wdA`. The bot should be able to perform the metadata lookup using the Spotify API and save the information accordingly. This mitigates issues with double-check when multiple options are found, in this scenario only one option will be found - of course."

**Amendment (2026-04-24)**: This feature **extends** the existing **save-album** capability (free-form text and metadata resolution). When the user provides a **recognized Spotify album page link**—either a **full** `open.spotify.com` **album** URL or a **supported Spotify short link** that **resolves** to such an album page—the product treats it as a **direct album reference**: it resolves **exactly one** catalog album for that link and **does not** present the **multi-album disambiguation** step used when free-text search returns **several distinct** releases. **Invalid** links, **non-album** Spotify pages, **short links** that **cannot** be **safely** resolved to **one** album, or **missing** albums follow the same **honest failure** expectations as the base feature (no false “saved” confirmation).  
**Amendment (2026-04-24 — analysis)**: **Direct-link eligibility** is **Spotify-only** (**FR-008**). **Generic** web links on **other** hosts follow **normal free-text** save behavior with the **full** user argument—not the failed-link path. **Multiple** qualifying links in one message are handled with a **safe**, **single-link** rule (**Edge Cases**). **SC-004** is satisfied by **in-bot help** plus the **feature quickstart** document operators use for this release.

## Save-album command: two argument types

The **same** save-album entry point (e.g. `/album` with **one** trailing argument) accepts **exactly one** of the following **parameter kinds** per message:

| Kind | Description | Product behavior |
|------|-------------|------------------|
| **FREE_TEXT** | The argument is treated as **ordinary** listener text for lookup—**including** any input that is **not** **Spotify** **direct-link**-**eligible** under **FR-008** (e.g. album titles, artist names, informal phrases, and **generic** non-Spotify **HTTP(S)** URLs). | **Current** behavior, **unchanged**: metadata **search** (ordered catalog chain), **disambiguation** when needed, success and failure rules **as in** the **existing** save-album specification. |
| **SPOTIFY_URL** | A **qualifying** full **`open.spotify.com` album** page address (per planning’s parsing rules). | **New** **direct-link** flow: resolve **that** album **without** the **multi-album** chooser used for **vague** **FREE_TEXT**; details in **FR-001**–**FR-003**, **FR-005**, **FR-007**, **User Story 1**. |
| **SHORT_URL** | A **qualifying** **Spotify-operated** **short** **share** link host documented in planning, resolved safely to **one** album page. | **New** **direct-link** flow (same **outcome** class as **SPOTIFY_URL** after resolution): **FR-001**–**FR-003**, **FR-005**, **FR-007**, **User Story 1**. |

**Routing rule**: The product **classifies** the argument **first**; **FREE_TEXT** vs **SPOTIFY_URL** / **SHORT_URL** is determined by **FR-008** and planning’s parsing rules. **Help** and the **feature quickstart** MUST **name** both **FREE_TEXT** and link paste (**SPOTIFY_URL** and **SHORT_URL**) in **non-technical** wording (**User Story 3**, **SC-004**).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Paste a Spotify album link to save (Priority: P1)

A **listener** uses the same **save-album** entry point they already know (e.g. `/album` plus a single argument) but pastes a **full Spotify album page address** or a **Spotify short link** that points at an album instead of a title or artist phrase. The product **recognizes** the input as an album link, **resolves** the album behind that address (including any **allowed** short-link **resolution** step) through **metadata lookup**, and **records** the album and the listener so the outcome matches a **normal successful save** (confirmation message, durable association).

**Why this priority**: This is the core value—**precise** saves from a shared link **without** choosing among **multiple** search hits.

**Independent Test**: With a **valid** public album URL **or** a **valid** supported short link and a healthy metadata path, one command produces **one** saved album for the issuing listener **without** an intermediate “pick which album” step.

**Acceptance Scenarios**:

1. **Given** a listener on a supported host, **When** they send the save-album command with a **single** argument whose **primary** content is a **standard** `open.spotify.com` **album** page URL (including **locale** path segments such as `intl-pt` and **trailing** query parameters such as `si=…`), **Then** the product **identifies** the **album** from that link, **fetches** structured metadata for **that** release, **persists** the save **linked** to the listener, and shows a **short** **success** confirmation consistent with the rest of LifeSoundTrack.
2. **Given** the same listener, **When** they send the save-album command with a **single** argument whose **primary** content is a **supported Spotify short link** that **resolves** to **exactly one** `open.spotify.com` **album** page, **Then** the product **resolves** the short link **safely** (per **FR-007**), **identifies** the **same** **album** as if the full URL had been pasted, **persists** the save **linked** to the listener, and shows a **short** **success** confirmation consistent with the rest of LifeSoundTrack.
3. **Given** a successful save-from-link, **When** the same listener later saves **another** album via **another** link, **Then** both items can exist on their list (same **duplicate** behavior as the base save-album feature unless a future spec adds deduplication).
4. **Given** the save-album command with **only** free-form text (no album URL), **When** the user sends it, **Then** behavior remains **unchanged** from the **existing** save-album specification (search, chain, disambiguation when needed).
5. **Given** a listener pastes **one** **unambiguous** `open.spotify.com` **album** URL (or **supported** short link) **embedded** in short surrounding text (e.g. a brief phrase before the link), **When** they send the save-album command, **Then** the product **detects** that link, follows the **same** **direct-link** success path as a lone URL, and **does not** require the **multi-album** chooser for that command.

---

### User Story 2 - Clear outcomes when the link is wrong or the album cannot be resolved (Priority: P1)

The pasted text may be **malformed**, point to a **different** Spotify page type (not an album), point to an album that **no longer exists** or cannot be read, or metadata lookup may **fail** temporarily. The user must get a **plain-language** outcome and the product must **not** claim success when nothing was saved.

**Why this priority**: Trust and parity with the base feature’s failure handling.

**Independent Test**: Exercise **bad** URL shapes, **non-album** URLs, **not found**, and **service** failure; each path yields an appropriate **non-success** message and **no** new **saved** album row for a **resolved** release.

**Acceptance Scenarios**:

1. **Given** an argument that is **not** a recognizable Spotify **album** page URL **or** **supported** short link, **When** it is **clearly** not intended as a URL (e.g. ordinary words with spaces), **Then** the product follows **free-text** save-album behavior (not the direct-link path).
2. **Given** an argument that matches **Spotify direct-link eligibility** (**FR-008**) but is **not** a valid **album** reference after parsing or resolution (including short links that **fail** resolution, land on a **non-album** page, or are malformed), **When** the user sends it, **Then** the product responds with a **helpful** message (e.g. could not use that link) and **does not** create a **saved** album for a **guessed** release and **does not** substitute **free-text** catalog search using the raw URL string as the **whole** query in place of a **failed** direct resolution.
3. **Given** an argument that is a **generic** **HTTP(S)** URL on a **non-Spotify** host (not **FR-008**-qualifying), **When** the user sends it, **Then** the product uses **existing** **free-text** save-album behavior with the **full** argument text (same as any other free-form query), **not** the “could not use that link” path reserved for **failed** **Spotify** direct-link attempts.
4. **Given** a **valid** album link pattern but the **catalog** cannot return metadata for that album (removed, restricted, or empty result), **When** the user sends it, **Then** the user sees a **clear** “could not find” or equivalent message and **no** false success.
5. **Given** a **temporary** metadata or infrastructure failure on the direct-link path, **When** the user sends a valid album **full** URL **or** **supported** short link, **Then** the user sees a **generic** retry-style or unavailable message and **no** false success.

---

### User Story 3 - Listeners discover they can paste links (Priority: P2)

A **listener** who reads **in-app help** or the **feature quickstart** (operator-facing) learns that **paste a Spotify album link** works with the same command as **typing an album name**, and sees a **short** example.

**Why this priority**: Adoption; reduces mistaken “only works with titles” assumptions.

**Independent Test**: **In-bot help** and the **feature quickstart** (operator-facing) **mention** link paste with at least one **example**; reviewer can confirm **without** reading application source.

**Acceptance Scenarios**:

1. **Given** a user opens **help** for the bot, **When** save-album is described, **Then** the description **includes** that listeners may use **either** **free text** (title, artist, informal query—**same as before**) **or** a **Spotify album page link** / **Spotify share short link** (wording may vary; must be **non-technical**).
2. **Given** an operator reads the **feature quickstart** for this release, **When** they look up how listeners save albums, **Then** the document **states** that **full** album URLs and **Spotify share** short links work with **`/album`**, with at least one **illustrative** example (wording may vary; must be **non-technical**).

---

### Edge Cases

- **Extra** **text** around the URL (accidental spaces or words): the product SHOULD still **detect** and use a **well-formed** Spotify album URL **or** **supported** short link **inside** the argument when **unambiguous** (e.g. `/album hey check this https://open.spotify.com/.../album/…` behaves like the **direct-link** path when exactly **one** qualifying link is present and **no** conflicting **primary**). **Structured** tests SHOULD include **at least** one such row (**User Story 1**, scenario 5).
- **Multiple** **qualifying** **Spotify** links (**FR-008**) in **one** argument: if **more than one** distinct **album** or **short-link** target is present and there is **no** single **clear** **primary** link under **plan-defined** rules (e.g. which URL to prefer when two album links appear), the product MUST **not** pick an album **silently**; it MUST respond with a **short**, **non-technical** message asking the listener to send **one** Spotify album link at a time, and MUST **not** create a **saved** album for that message.
- **Spotify** **short** **links**: the product MUST accept **Spotify-operated** short share URLs that **resolve** to **exactly one** `open.spotify.com` **album** page, with the **same** direct-album save behavior as a full link after resolution. **Allowed** hosts, **redirect** **chain** limits, **timeouts**, and **anti-abuse** rules (e.g. **not** following arbitrary third-party shorteners unless they **resolve** to an **allowed** outcome) are **plan-owned**; if resolution **fails**, **times** **out**, or lands on a **non-album** page, the user gets a **clear** failure and **no** **wrong** free-text search **guess**.
- **Same** album saved **twice** via link: **same** as base feature—**duplicate** rows **allowed** in v1 unless a later spec adds deduplication.
- **Metadata** **chain** from the base feature: the direct-link path **must** still respect **operator** controls (e.g. **disabled** integrations) defined for save-album; if the **integration** needed to resolve a Spotify album identifier is **unavailable** or **off**, the user gets a **coherent** failure, **not** a **silent** fallback to a **random** search hit.
- **Privacy** and **safety**: user-visible and log output **must not** expose **secrets**, **tokens**, or **other** users’ data (consistent with the base feature).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-000**: The save-album command MUST support **two** **parameter** **families** on the **same** entry point: **(1)** **FREE_TEXT** — **unchanged** behavior per the **existing** save-album specification; **(2)** **SPOTIFY_URL** or **SHORT_URL** — **new** **direct-link** behavior in this specification (**FR-001**–**FR-003**, **FR-005**, **FR-007**, **FR-008**). Classification MUST follow **FR-008** and **FR-004**.
- **FR-001**: When the argument is classified as **SPOTIFY_URL** or **SHORT_URL**, the product MUST accept it as either **(a)** a **Spotify** **album** **page** URL (`open.spotify.com` with an **album** path per planning’s **exact** parsing rules) **or** **(b)** a **supported** **Spotify** **short** **link** form that **planning** documents (hosts, path patterns, and validation).
- **FR-002**: When **FR-001** applies, the product MUST resolve metadata by **direct** reference to the **album** indicated by that URL or **fully** **resolved** short link (identifier derived from the **final** **trusted** album location), such that **at most one** release is **candidates** for save for that request **before** persistence—**without** applying the **multi-distinct** **disambiguation** **prompt** defined for **vague** free-text queries in the base save-album specification.
- **FR-003**: When **FR-001** applies and metadata resolution **succeeds**, the product MUST **persist** album fields and **associate** them with the **listener** to the **same** completeness and **safety** standards as a **successful** free-text save (title, primary artists, and other fields the product already stores when available).
- **FR-004**: When the argument does **not** meet **Spotify direct-link eligibility** (**FR-008**), the product MUST **not** force the direct-link path; it MUST use **existing** free-text save-album behavior (the **full** user argument as today).
- **FR-005**: When the argument **is** **Spotify** **direct-link**-**eligible** (**FR-008**) but **parsing**, **short-link** **resolution**, or **direct** **album** **metadata** lookup **fails**, the product MUST **not** claim the album was saved, MUST **not** create a **new** stored album row for a **fabricated** match, and MUST **not** use **free-text** catalog search on the **raw** pasted URL string **as a replacement** for that **failed** **direct** attempt (listeners may still send a **new** free-text query in a **follow-up** message).
- **FR-006**: User-visible messages for this feature MUST remain **non-technical**, **consistent** in tone with existing LifeSoundTrack messaging, and MUST **not** include **secrets** or full raw provider dumps.
- **FR-007**: For **short** **links**, the product MUST **only** treat redirects as **trusted** when they follow **plan-defined** **allowlists** and **limits** (e.g. **maximum** hops, **time** budget); the product MUST **not** persist or **echo** **opaque** redirect targets in user-visible text beyond what is **already** normal for confirmations; **resolution** failures MUST map to the **same** **class** of user messaging as other **invalid** links.
- **FR-008**: **Spotify direct-link eligibility** — The direct album-by-link branch applies **only** when the argument contains **(a)** a recognized **`open.spotify.com` album** page address per planning’s parsing rules **or** **(b)** a **supported** **Spotify-operated** **short** **share** link host documented in planning. **Any** other **HTTP(S)** URL (different host) is **not** **direct-link**-eligible: the product MUST apply **FR-004** and treat the input as **ordinary** free-text for save-album.

### Non-functional and product quality *(should align with project constitution)*

- **UX consistency**: Success and error copy for **link** saves matches the **save-album** patterns; **help** text is updated so listeners know **links** are accepted.
- **Testing**: Automated tests SHOULD cover: **happy** path with a **representative** album URL (including **locale** segment and **query** string), **happy** path with a **representative** **supported** **short** **link** that **redirects** to an **album**, **one** case where the **same** URL forms appear with **short** surrounding prose (**unambiguous** embedded link; **Edge Cases**), **non-album** Spotify URL, **short** **link** that **does not** resolve to an **album**, **malformed** URL, **not found** album, **provider** failure, a **control** case proving free-text behavior is **unchanged** when no URL is present, **generic** non-Spotify **HTTP(S)** URLs (**FR-008** / **FR-004**), **multiple** qualifying links (**Edge Cases**), and **at least one** **structured** end-to-end-style check that exercises the **save** path from **short-link** handling through **direct** album resolution to **persistence** using an **isolated** test setup (no **live** third-party services required for CI).
- **Linting / static analysis**: Follow repository conventions for any new parsing or routing logic.
- **Logging and monitoring**: Log **outcome** class (success, bad link, not found, provider error) without **tokens** or **full** URLs if policy treats them as sensitive; keep **correlation** sufficient for support.
- **Performance**: From the user’s perspective, a link save with healthy dependencies SHOULD feel as fast as a **single-match** free-text save under the same conditions; **typical** completion SHOULD stay within **about fifteen seconds** when upstream services and storage are healthy, so listeners are not left wondering whether the bot received the link. **Shorter** per-step time budgets (for example a **tight** budget for **short-link** **resolution** alone) are **implementation** choices that exist to keep the **overall** experience within this **user-visible** window; they are **not** a second, conflicting “maximum command duration” for the whole flow.
- **Containerization / local dev**: No new **runnable** artifact is required by this spec alone; if implementation touches **configuration**, the plan SHOULD update **runbook** env documentation.
- **Decommissioning**: If link support is removed, **help** and **runbook** references MUST be updated in the **same** delivery window.

### Key Entities

- **Listener (user profile)**: Unchanged from the base save-album feature; **keyed** by host **source** and **external_id**.
- **Saved album (list item)**: Same as base feature; **may** optionally retain the **original** pasted URL or **catalog** identifiers as **provenance** if planning adds a field—**not** required for v1 acceptance if persistence already captures provider IDs.
- **Spotify album link (input)**: A user-supplied **HTTP(S)** address that **identifies** a **specific** **album** on Spotify—either a **direct** `open.spotify.com` **album** URL or a **supported** **short** **link** that **resolves** to one—**normalized** in processing to a **single** **album** identity for lookup.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In **structured** tests, **100%** of **valid** Spotify **album** **full** **URL** and **supported** **short-link** cases that **should** resolve produce **exactly one** **successful** save **without** showing a **multi-album** **chooser** for that command—**same** **single-candidate** **outcome** as **FR-002** before persistence.
- **SC-002**: In **structured** tests, **100%** of **invalid** link, **non-album** URL, **not** **found**, and **provider**-**error** cases yield a **non-success** user outcome and **no** **new** **saved** album associated with a **wrong** or **guessed** release.
- **SC-003**: In **structured** tests, inputs that are **not** **Spotify** **direct-link**-**eligible** (**FR-008**), **including** ordinary words **and** **generic** non-Spotify **HTTP(S)** URLs, **behave** per the **existing** free-text save-album specification (**100%** alignment on a **defined** regression suite or **explicit** test matrix row).
- **SC-004**: A **reviewer** can confirm from **in-bot help** **and** the **feature quickstart** document (operator-facing runbook for this feature) that **Spotify album page links** and **Spotify share short links** are supported, **without** reading **application** source.
- **SC-005**: In **acceptance** review of **sample** logs and user messages, **no** **secrets** or **tokens** appear in **default** user-facing text for this feature. **Release** verification SHOULD be recorded in the feature **checklist** so the review is not skipped.

## Assumptions

- **Scope** is **Spotify** `open.spotify.com` **album** URLs in **standard** shapes **and** **Spotify-operated** **short** **share** links that **resolve** to those album pages; **other** streaming services, **arbitrary** third-party URL shorteners, or **deep** mobile-only schemes are **out of scope** unless added later.
- **FREE_TEXT** **parameter** **kind** keeps **all** behavior from the **existing** save-album specification (**metadata** **chain**, **feature** **flags**, **disambiguation**). This feature **adds** the **SPOTIFY_URL** and **SHORT_URL** **kinds** and their **direct-link** branch **only**.
- **Parsing** details (exact path patterns, ID extraction, rules for **primary** link vs **multiple** links, **supported** short-link **hosts**) are **plan-owned**—recorded in this feature’s **research** and **contracts** companion documents in the **same** specification folder—and must satisfy **FR-002**, **FR-008**, and **Edge Cases**.
- **Duplicate** saves and **listener** **profile** **updates** follow the **same** assumptions as the **base** save-album feature.

## Clarifications

### Session 2026-04-24

- *(Specification pass.)* No open `[NEEDS CLARIFICATION]` markers in the **initial** draft; **short-link** behavior was later **promoted** to a **required** capability—see clarification below.
- **Q**: Should the product accept **both** full Spotify album URLs **and** Spotify **short** links? **→ A**: **Yes.** The product **MUST** accept **both** **full** `open.spotify.com` **album** URLs and **supported** **Spotify-operated** **short** links that **resolve** to **one** album page, with **plan-defined** **trust** rules for redirects (**FR-001**, **FR-007**, **Edge Cases**).
- **Analysis remediation (2026-04-24)**: **Direct-link** behavior is **scoped** to **FR-008** **qualifying** links only; **other** URLs use **free-text** save (**FR-004**). **Failed** **qualifying** attempts do **not** fall back to **searching** the raw URL string (**FR-005**). **Multiple** qualifying links use a **safe** **one-link** prompt (**Edge Cases**). **SC-004** explicitly includes the **feature quickstart** as reviewer-visible documentation.
- **Q**: How should the spec **name** the **two** **save-album** **parameter** **modes**? **→ A**: **FREE_TEXT** (**current**, **unchanged**) vs **SPOTIFY_URL** **or** **SHORT_URL** (**new** **direct-link** flow), documented in **Save-album command: two argument types** and **FR-000**.
