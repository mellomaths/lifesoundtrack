# Feature Specification: LifeSoundtrack — save albums to a personal list

**Feature Branch**: `003-save-album-command`  
**Created**: 2026-04-23  
**Status**: Draft  
**Input**: User description: "Add a new command to the bot to allow users to save music albums that they want to listen to. Command usage example: `/album Abbey Road by The Beatles`. The command must accept any input from the user and should not validate or expect a specific format. Using an external API, the bot will search for the album metadata (e.g. title, artists, genres, year, and other helpful information). The bot will save the album data captured in a database. The bot will also save the user information in the database: name, username (e.g. Telegram username), external_id (e.g. Telegram user_id), source (e.g. Telegram), and other useful data points."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Save an album with a free-form message (Priority: P1)

A **listener** in a private chat with LifeSoundTrack sends a **save-album** command with **any** text they choose after the command (e.g. a title, a title plus artist, or informal wording). The product **resolves** that text to a real album using a **metadata lookup**, then **records** the album and the **user** so the item appears on the listener’s saved list and can be used in later product features.

**Why this priority**: This is the core value—turning a casual message into a structured, retrievable “want to listen” item without forcing the user to type a strict pattern.

**Independent Test**: In isolation, a user (or test double) issues the command with a single free-form string; the system either confirms success with a short, **non-technical** message or returns a **clear** failure that does not expose secrets. Data rows exist for the user and the album association when the flow succeeds.

**Acceptance Scenarios**:

1. **Given** a listener who is identified by the host platform, **When** they send the save-album command with a **non-empty** free-form text (any characters the platform allows, no required template), **and** the metadata lookup yields **exactly one** strong match, **Then** the product persists **album** fields returned or derived from the provider (e.g. title, primary artists, genre(s), release year) together with a link to the **user** who issued the command, and the user sees a **success** confirmation.
2. **Given** the same, **When** the free-form text is **empty** (only the command, no user query), **Then** the product responds with a **helpful** message explaining that a **search text** is required, and **does not** run metadata lookup or create a new stored album.
3. **Given** a successful save, **When** the same listener runs the command again on **another** free-form text, **Then** the product can store **another** album association for that user (v1 does **not** require merging duplicates unless specified later).

---

### User Story 1b - Choose among multiple search results (Priority: P1)

A **listener** issues the save command with a **vague** query (e.g. a single common album title) and the metadata source returns **several** plausible albums. The product **does not** guess silently: it **shows a short, ordered list** of candidates so the listener can **pick the correct one**; only the **chosen** album is stored.

**Why this priority**: Ambiguous queries are common; auto-picking the wrong album would break trust. This story is on the **critical path** to a correct “save” for many real messages.

**Independent Test**: With a **stub** or **sandbox** provider that returns at least two distinct albums for one query, the user always sees a **disambiguation** step (unless they cancel), and after choosing an option, **exactly** that album is persisted and linked to the user.

**Acceptance Scenarios**:

1. **Given** metadata lookup returns **two or more** relevant candidates, **When** the listener has just sent a non-empty free-form query, **Then** the product presents **up to three (3)** options **ordered by relevance** (most relevant first). If fewer than three candidates exist, the product shows **all** returned candidates; if more than three exist, the product shows only the **top three** by relevance. **No** album row is **finally** written until the user **confirms** one of the options.
2. **Given** a disambiguation prompt, **When** the host platform supports **inline or reply buttons** for this interaction, **Then** the product **prefers** that UI for each listed option. **When** buttons are not available, **Then** the product uses a **numbered text** list (e.g. `1: …`, `2: …`) and accepts the listener’s next message as a **choice** of index (e.g. replying `2` to select the second line), as in: query `/album Red`, two albums returned—`1: Red — Taylor Swift (2012)` and `2: Red — Gil Scott-Heron (1971)`; the user sends `2` and the second album is saved.
3. **Given** a single high-confidence match, **When** the provider returns one clear result (or a defined single-match path in planning), **Then** the product **may** skip the disambiguation list and **save** directly (per **FR-009** and plan).

---

### User Story 2 - When lookup fails, the user is not left guessing (Priority: P1)

The listener’s text may be ambiguous, the metadata service may be unavailable, or no **reasonable** match may exist. The product must respond in a **human-readable** way and must **not** leak internal keys, stack traces, or other users’ data.

**Why this priority**: Reliability and trust; failed saves are common when people type creatively or when services are down.

**Independent Test**: Trigger or simulate lookup failure, empty result set, and service error; every path shows a **short** user-facing message appropriate to the case, with **no** secret material in the reply. Verify persistence rules: **no orphan album record** is created for a **failed** end-to-end save (policy: do not store a “success” album row without a resolved match unless explicitly changed in a later spec).

**Acceptance Scenarios**:

1. **Given** a metadata lookup that returns **no** suitable match, **When** the user sent valid non-empty text, **Then** the user sees a **clear** “not found” (or similar) message and **no** new **album** row is created for a resolved release.
2. **Given** a **temporary** failure of the external metadata service, **When** the user issues the command, **Then** the user sees a **generic** failure or “try again” message, and the system does **not** claim success; **no** false confirmation.

---

### User Story 3 - The listener’s identity is stored for the feature to work (Priority: P1)

The product must **create or update** a **listener profile** for the chat platform so that saved albums are tied to a **stable identity** (who saved what). This includes a **display name**, **platform handle** (where applicable), **external identifier** from the host, **source** of that identity, and any other **non-secret** fields needed for support and future features.

**Why this priority**: Without user persistence, the “my saved albums” story cannot exist across sessions.

**Independent Test**: After one successful save, a stored user record contains at minimum: **name** (or best available), **handle/username** (or null if absent on host), **external_id**, and **source**; re-invoking the command for the same platform user **updates** profile fields that may change (e.g. display name) per **FR** below.

**Acceptance Scenarios**:

1. **Given** a first-time listener who completes a **successful** save, **When** the flow finishes, **Then** a **user** (listener profile) record exists with **name**, **username** (if available), **external_id**, and **source** (e.g. the chat product name).
2. **Given** a returning listener whose **name** or **handle** changed on the host, **When** they save again, **Then** the product **updates** the stored profile for that **external_id** and **source** with current values, without duplicating a second “same person” record.

---

### User Story 4 - Operators and contributors can discover the feature (Priority: P2)

A **new team member** can read the product documentation, understand the command shape (free text), environment needs (e.g. credentials for the metadata service and **durable storage**), and how to test safely without affecting production data.

**Why this priority**: Reduces time-to-first-successful integration and support burden.

**Independent Test**: Follow the **feature quickstart** (to be created in planning) with test credentials: command works end-to-end in a **sandbox** environment.

**Acceptance Scenarios**:

1. **Given** a reader of the runbook, **When** they configure required secrets and run the app, **Then** they can complete one successful **save** path and one **intentional failure** path (e.g. no match) with documented expectations.

### Edge Cases

- **Extremely long** free-form text: the product may **truncate** for the provider request or return a “too long” user hint; behavior must be **documented** and must not crash the process.
- **Empty** command text: see Story 1; no metadata lookup, no new album.
- **Special characters, emoji, multiple languages** in the query: still **one** free-form string; no format validation beyond platform limits; lookup **may** still fail—Story 2 applies.
- **Metadata provider returns multiple** candidates: see **User Story 1b**; the product shows **up to three** options by **relevance**, **then** requires user confirmation (**buttons** preferred, else **numbered text**). **No** save without an explicit pick when two or more candidates are shown. **Session** timeout, duplicate messages during disambiguation, and cancel semantics are **plan-owned** but must be **user-safe** (no partial save of the “wrong” album without confirmation).
- **Concurrent** commands from the same user: v1 need not guarantee global ordering; each command is processed independently; duplicate rows possible if the user double-taps.
- **Host platform** not Telegram: the **domain** command and payload are still “save album with free text”; **source** and **field mapping** are adapter-specific. Telegram is the first supported host unless planning narrows further.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST expose a **save-album** interaction (see platform mapping in the implementation plan) that accepts a **single** user-supplied string **as-is**; the product MUST **not** require a specific syntax (e.g. `Title | Artist`) for v1.
- **FR-002**: For a **non-empty** query string, the product MUST call an **outbound metadata lookup** (through a pluggable module to be selected in planning) to obtain **one or more** **structured** album **candidates** including at least **title** and **primary artist(s)** per candidate, and SHOULD obtain **genres** and **release year** when the provider supplies them, plus any other fields agreed in planning as **user-visible** or **analytically** useful. The product MUST then apply **FR-009** when **more than one** relevant candidate is returned.
- **FR-003**: On **successful** resolution of a match, the product MUST **persist** the album (or a normalized representation of the provider’s result) in application storage so it can be listed or queried later, and MUST **link** that save to the **listener** who issued the command.
- **FR-004**: The product MUST **create or update** a **listener profile** per **(source, external_id)** (or the equivalent composite key) containing at least: **name** (display name or best available), **username** / **handle** (when the host provides it; nullable otherwise), **external_id**, and **source**; additional non-secret **profile** fields are allowed as documented in planning.
- **FR-005**: If **metadata** lookup does **not** yield an acceptable match, the product MUST **not** create a new stored **album** record for a “successful” discography entry; the user receives an explanatory message (per Story 2).
- **FR-006**: If **metadata** lookup fails for **infrastructure** reasons (network, rate limit, server error), the product MUST return a user-appropriate error and MUST **not** state that the album was saved.
- **FR-007**: User-visible and log output for this feature MUST follow existing **safety and tone** rules for the product: **no** API keys, **no** database connection strings, **no** other users’ PII, and **no** full raw provider payloads in the default user message (summaries or confirmations only).
- **FR-008**: The **first** host adapter will map the product’s **save-album** command to the **Telegram** bot interface (e.g. `/album …` with trailing text) unless a subsequent amendment narrows scope; the **core** name for the feature remains **host-neutral** in domain documentation.
- **FR-009**: When metadata lookup returns **more than one** relevant candidate, the product MUST **not** persist any album for that request until the user **selects** one option. The product MUST present **up to three (3)** options **sorted by relevance** (show **all** candidates when two or three exist; when **more than three** exist, show only the **top three**). The product MUST use **platform-native chooser controls** (e.g. **inline / reply buttons**) when the host supports them; **otherwise** the product MUST use a **numbered list** in chat and treat the user’s next message (or the platform’s callback) as the **selection** of **one** option. A **single** unambiguous match MAY proceed without a disambiguation step, per planning rules for confidence thresholds.

### Non-functional and product quality *(should align with project constitution)*

- **UX consistency**: Confirmation and error messages for **save album** use the same tone and product naming conventions as the rest of LifeSoundTrack (e.g. **LifeSoundtrack** where appropriate); **help**-style text should be updated so listeners discover **/album** (or equivalent) alongside **start** / **help** / **ping** when those exist.
- **Testing**: Automated tests SHOULD cover: empty query, at least one **happy** path (with a **recorded** or **stubbed** provider), **multi-match disambiguation** (stub returns ≥2 options → user selects → correct record saved), no-match, and **provider error**; persistence SHOULD be testable with an **in-memory** or test database in CI when the repository’s testing strategy allows.
- **Linting / static analysis**: New code packages MUST follow repository conventions; any **generated** or **migrated** schema MUST be **reviewed** in PR.
- **Logging and monitoring**: Operations SHOULD log **command outcome** and **error class** (e.g. no match vs provider error) without secret headers or user tokens; optional correlation IDs for support.
- **Performance**: A typical save attempt SHOULD complete from the user’s perspective within a **documented** target (e.g. under 15 seconds) when the provider and database are healthy; if slower, a “still working” pattern is a **P2** enhancement.
- **Containerization / local dev**: This feature is expected to add or modify **persistence**; the plan MUST update or add **Dockerfile** and **Docker Compose** (or the repo’s standard equivalent) so contributors can run the bot and database together locally, per constitution **VIII** where applicable.
- **Decommissioning**: If the feature is later replaced, remove or archive **/album** copy, runbooks, and unused tables/migrations in the same **release window** and update **constitution/IX**-relevant links.

### Key Entities

- **Listener (user profile)**: Represents a person using the product through a given **source**; keyed by **source** + **external_id**; stores **name**, **username** (optional), and other **non-secret** profile fields needed for attribution and support.
- **Saved album (list item)**: Represents one **catalog album** (or provider-normalized set of metadata) **saved by** a listener; holds provider-derived fields (title, artists, year, genre(s), **provider identifiers** for dedup or re-fetch) and a **link** to the **listener**; may include a **captured** original query string for user reference.
- **Metadata lookup (conceptual)**: Not a user-visible entity, but a **port** the plan will bind to a **concrete** external or mock provider for **search-and-resolve** behavior.
- **Disambiguation (pending choice)**: An ephemeral interaction state: **(listener, a bounded list of candidate album summaries from the last search, relevance order, optional message/thread identifiers)**. **v1** uses **PostgreSQL** (or in-process state for single-process dev) only—**no** Redis. Must remain **valid** long enough to accept one user **pick**; timeout and cancel are plan-owned.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In **structured tests** (automated and/or a scripted **sandbox**), **100%** of **happy-path** invocations with **non-empty** text result in a **single** new **saved album** record associated with the **correct** listener, with **at least** title and **primary** artist(s) populated when the provider returns them—**including** paths where the user **first** disambiguates among **2–3** options and **then** saves.
- **SC-002**: In **100%** of **intentional** no-match and provider-error cases in the test suite, the user receives a **non-success** message and **durable storage** does not show a new **“successful”** album save for a resolved discography object for that attempt.
- **SC-003**: In **100%** of reviewed **log** and **user message** samples in acceptance review, there is **no** occurrence of **API keys**, **database credentials**, or **other users’** private identifiers in user-visible or default **Info-level** log lines for this feature.
- **SC-004**: A **new contributor** (or a reviewer) following the post-planning quickstart can complete (a) a successful save and (b) a no-match case in **one** work session, **without** reading application source to guess configuration—evidence: checklist sign-off or PR review.

## Assumptions

- **Private chat** remains the default context, consistent with feature **001**; group chat is out of scope unless a later spec extends it.
- The **first** **metadata** provider, rate limits, and field mapping are **selected in the implementation plan**; the spec does not mandate a **named** commercial API here.
- **v1** does **not** require **de-duplication** of the same discography item across multiple saves by the same user; saving twice may create two rows. **Deduplication** and “already in your list” behavior can be a follow-up.
- **Currency / locale** of release year and genre labels follow **provider** data; the product may display them **as provided** in English or raw form for v1.
- **List / browse** my saved albums is **implied** as a future or companion feature; **v1** focuses on **add**; a minimal **acknowledgment** line in the success message is enough to prove persistence (e.g. “Saved *Album* (Year)”).
- **PII and retention** for listener profiles and saved rows follow the team’s default **privacy** policy and region requirements; the plan may add **deletion** or **export** stories later.
- **Disambiguation list size**: The product shows **at most three (3)** options per prompt; with **two** matches, both are shown; with **one** match, a direct save **may** apply (**FR-009**). *This supersedes any informal “at least three options” phrasing: the cap is three, the minimum set size that triggers a list is two.*
- **No Redis (v1)**: Pending disambiguation state is **not** stored in Redis; use **PostgreSQL** (durable) and/or **in-process** for single-node dev, per **Clarifications** (Session 2026-04-23).

## Clarifications

### Session 2026-04-23

- *(Specification pass.)* No open `[NEEDS CLARIFICATION]` markers; provider choice and schema are plan-owned; reasonable defaults are in **Assumptions**.

- **Q:** When metadata lookup returns **several** matching albums, how should the user select one? **A:** The product must offer **up to three** options **ordered by relevance** (if only two match, show both; if more than three match, show the **top** three). Use **platform buttons** (inline/reply) when the host supports them; **otherwise** use a **numbered text** list and accept the user’s **index** (e.g. `2`) as the choice. **Persist** only the **selected** album; **no** silent auto-pick among multiple results. *Stakeholder example: query “Red” → `1: … Taylor Swift (2012)`, `2: … Gil Scott-Heron (1971)` → user sends `2` → save the second.*

- **Q:** Should disambiguation state use **Redis**? **A:** **No.** **v1** does **not** use Redis. Pending choices are stored in **PostgreSQL** (durable, multi-replica) and/or **in-process** memory for **single-instance** local dev only.
