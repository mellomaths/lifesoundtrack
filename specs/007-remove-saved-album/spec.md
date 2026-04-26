# Feature Specification: LifeSoundtrack — remove saved album (`/remove`)

**Feature Branch**: `007-remove-saved-album`  
**Created**: 2026-04-26  
**Status**: Draft  
**Input**: User description: "Add command to the LifeSoundtrack bot that lists the albums saved by the user. Command: \`/remove ALBUM_NAME\`. The user should be able to edit the list of saved albums by removing one of them by using this command. The bot should be smart enough to normalize the query before searching in the database. Make sure to account for when the album is not found."

*Note: The deliverable is the **remove** command and behavior below; the opening phrase referring to "lists" is treated as a wording slip—the user-defined command is \`/remove ALBUM_NAME\`.*

## Clarifications

### Session 2026-04-26

- Q: How should the help surface stay accurate as features change? → A: The user-visible help response (e.g. `/help` where the messaging host uses slash commands, per the existing bot messaging behavior) must list the **current** user-facing features of the bot. When the remove feature ships, that help text must be updated in the same delivery so it includes `/remove` with a one-line description, and the overall help text must remain a complete, accurate inventory of all supported user-facing commands at that release (no supported command omitted; no obsolete command presented as current unless product policy explicitly documents otherwise).
- Q: (Spec cross-artifact **007**) How is “match” defined for v1 so **FR-003** does not conflict with “artist in the same string” in **A2**? → A: **Lookups** use **the** **stored** **`title`** **field** **only** **(see** **FR-003** **tiers**)** — **not** **splitting** a **user** **string** like **"Artist** **-** **Title"** into **separate** **fields** for **search**. **Primary** **artist** and **year** appear **only** in **disambiguation** **display** **lines** when **a** **pick** is **needed**, **not** as **parsed** **search** **fields**.
- Q: (2026-04-26) **Shorter** **query** than **saved** **title** (e.g. user sends **Abbey Road** while the **saved** **title** is **Abbey Road (Remastered)**): **exact** **normalized** **equality** **finds** **no** **row**. **What** **should** **happen**? → A: **Two-phase** **matching** **after** **the** **same** **normalization** **(FR-002)** **on** **the** **query** **and** **each** **stored** **title** **field**: **(1)** **Exact** **match** — the **normalized** **user** **string** **equals** the **normalized** **stored** **title**; **one** **row** **→** **delete** **and** **confirm**; **two** **or** **more** **rows** **→** **disambiguation** **(multi-exact** **flow**)**. **(2)** If **(1)** **finds** **no** **rows**, **partial** **match**: a **row** **qualifies** when the **normalized** **user** **query** **is** **non-empty** **and** **the** **normalized** **stored** **title** **contains** **that** **query** **as** **a** **contiguous** **substring**. **Zero** **partial** **rows** **→** **not** **found** **(User** **Story** **2**)**. **1–3** **partial** **rows** **→** **no** **silent** **delete**; **show** **numbered** **options** and **the** **same** **`remove_saved`** **pick** **flow** **as** **multi-exact** **(User** **Story** **4** / **FR-006**)**. **More** **than** **three** **partial** **rows** **→** **no** **delete** **and** **no** **listing** **of** **all** **matches**; **ask** **the** **user** **to** **be** **more** **specific**. **(Implementation** **note:** a **shipped** **build** **may** **have** **implemented** **(1)** **only**; **(2)** **is** **required** **to** **match** **this** **clarification**.)**
- Q: (NFR) Must v1 emit dedicated log/metric counts for remove, not-found, or disambig? → A: **Not** required for v1. The product **MAY** add redacted structured logs (e.g. outcome **enum** only, no message **body**) in the same or a follow-up change; if omitted, that is not an acceptance **blocker** (lightweight observability for a chat **command**).
- Q: (Assumption **A3**) How is max length of the remove remainder enforced? → A: The remove album query MUST respect the same (or a stricter) rune/character cap as the product’s other user-entered album text commands (e.g. `/album` free text), with a user-facing response if exceeded (see implementation **tasks**).
- Q: (Cross-artifact **007** / **Constitution** **V** + **NFR** **Logging**) How does this feature record “key metrics the team agrees to track” if remove-specific metrics are optional? → A: **Principle** **V** is satisfied for this workload by the **same** **production/staging** **observability** **baselines** **already** **used** for the **Telegram** **bot** (e.g. process **health**/**liveness**, **error** **rate**, **latency**, and **any** **counters** or **SLOs** the **team** **already** **tracks** for that **runnable**). **Remove**-**specific** **event** **counts** or **outcome** **enums** in **logs** **remain** **optional** in **v1** and are **not** a **gating** **increment** to the **agreed** **metric** **set**; they **may** be **added** **without** **replacing** that **baseline**.
- Q: (UX) When several saved albums could match a `/remove` query, must the user confirm the row **only** by typing a number in chat, or may the product use the host’s interactive controls? → A: **Use** **platform-native** **affordances** **when** **available** **(e.g.** **Telegram** **inline** **keyboard** **buttons** **per** **candidate)** **so** **the** **user** **can** **confirm** **which** **album** **to** **remove** **without** **relying** **solely** **on** **a** **typed** **index**. **The** **message** **MAY** **still** **show** **numbered** **lines** **for** **readability** **(as** **in** **tester** **feedback: “**1:** …**”)**. **Where** the **messaging** **host** **does** **not** **support** **such** **controls,** **or** **a** **given** **adapter** **has** **not** **shipped** **button** **handling** **yet,** the **existing** **session-bound** **numeric** **text** **pick** **(1**–**N** **in** **range,** **per** **`remove_saved`**) **remains** **required** **and** **sufficient**; **v1** **does** **not** **remove** that **path**.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Remove a saved album by name (Priority: P1)

A **listener** sends a **remove** command followed by text that identifies an album **they previously saved** (e.g. album title). The product **normalizes** that text and, when **FR-003(1)** **yields** **exactly** **one** **matching** **row**, **removes** it and **confirms** **without** **a** **prior** **pick** **(see** **FR-003**–**FR-005** **and** **partial** **tier** **for** **other** **cases**)**.


**Why this priority**: This is the core value—trimming the personal library when the user no longer wants a title saved.

**Independent Test**: With a listener who has a known saved album, send the remove command with text that should match after normalization; verify the album no longer appears in their saved set and the reply confirms success.

**Acceptance Scenarios**:

1. **Given** a listener with **one** saved album whose stored title **matches** the command text **after normalization**, **When** they send the remove command with that text, **Then** that saved row is **removed** and the reply **confirms** removal **without** suggesting an error.
2. **Given** a listener, **When** they send the remove command with text that **matches** a saved album **despite** differences only in **letter case** or **extra spaces** (see User Story 3), **Then** the same single match is found and **removed** as in scenario 1.

---

### User Story 2 - Nothing to remove (not found) (Priority: P1)

A **listener** sends the remove command with text that **fails** **both** **exact** **and** **partial** **title** **matching** **(see** **FR-003** **and** **Clarifications** **2026-04-26**)** — **or** **they** **face** **too** **many** **partial** **hits** **(see** **FR-004**)**.

**Why this priority**: Prevents false confidence and support confusion; users must know the system did not delete anything—or that they must narrow a query when matches are ambiguous at scale.

**Independent Test**: With a library that does not contain a matching title under **either** tier, send remove; verify no row is deleted and the reply states clearly that **no matching saved album** was found **or** **asks** **for** **a** **more** **specific** **query** **when** **>3** **partials** **apply**.

**Acceptance Scenarios**:

1. **Given** a listener with **no** saved albums that **exactly** or **partially** **(substring)** **match** the normalized query per **FR-003**, **When** they send the remove command, **Then** **no** saved row is removed and the reply **states** that no matching album was found in their list **without** implying a system failure.
2. **Given** a listener who **has** other saved albums but **none** qualifying under **FR-003**, **When** they send the remove command, **Then** the outcome is the same as scenario 1 (no accidental deletion of a non-matching row).
3. **Given** **more** **than** **three** **partial** **matches** **(and** **no** **exact** **match**)** per **FR-003** **(2)**, **When** they send the remove command, **Then** **no** **row** **is** **deleted** **and** **the** **reply** **asks** **them** **to** **be** **more** **specific** **(not** **a** **generic** **500** **tone**)**.

---

### User Story 3 - Normalization matches intent (Priority: P1)

The product applies **normalization** to the user’s free-text **album query** so matching is **forgiving** of casual typing and **predictable** for testing and support, **before** comparing to **stored** saved-album **title** (and disambiguation rules in User Story 4).

**Why this priority**: Same motivation as for list/save flows—avoids false “not found” when the user types the same title slightly differently than stored.

**Independent Test**: Use equivalent queries that differ only by case, leading/trailing spaces, or internal whitespace runs; for a **single** matching saved album, removal succeeds consistently.

**Acceptance Scenarios**:

1. **Given** a stored title in the listener’s library, **When** two user-entered strings differ **only** by **letter case**, **Then** they **match** the same saved album(s) after normalization.
2. **Given** user-entered text, **When** normalization runs, **Then** **leading and trailing** whitespace is removed and **internal** runs of whitespace are **collapsed to a single space** before comparison (see Assumptions).
3. **Where** the product also treats **Unicode space characters** (e.g. non-breaking space) like ordinary spaces for comparison, that behavior is **documented** for implementers in planning artifacts and **covered** by tests (assumption; see Assumptions).

---

### User Story 4 - More than one saved row could match (Priority: P2)

The listener’s library may contain **more than one** saved row that **matches** under **FR-003** — **either** **exact** **duplicates**, **or** **several** **partial** **(substring)** **qualifiers** **with** **no** **single** **exact** **hit** **(1–3** **partials**)**. The product **must not** delete an arbitrary row without the listener’s **explicit** choice when **multiple** qualifying rows exist **within** **the** **rules** **of** **FR-003** **(including** **the** **>3** **partial** **case** **handled** **in** **FR-004**)**.

**Why this priority**: Correctness over speed—wrong deletion is worse than an extra step.

**Independent Test**: Seed **two** distinct saved rows that both match the same normalized query; issue remove; verify **neither** is deleted without confirmation **or** the product **lists** matches and **asks** the user to **narrow** or **pick** (per Assumptions).

**Acceptance Scenarios**:

1. **Given** **two or more** saved albums for the listener that **match** the normalized query, **When** the listener sends the remove command, **Then** the product **does not** remove a row **silently**; it **explains** that several albums match and **offers a clear next step** to **choose** one **row** (including, **on** **platforms** **that** **support** **it,** **interactive** **controls** such as **inline** **buttons** **per** **option** so the user is **not** **limited** to **typing** a **number**; **and** where **supported** or **as** **a** **fallback,** **numbered** or **distinct** **lines** **plus** a **valid** **session** **pick** per **Clarifications**), or **ask** for a **more specific** name **when** **FR-004** **applies**; behavior **aligns** with other bot flows for disambiguation and **Clarifications** (buttons **when** **available**).
2. **Given** a **unique** match **after** applying the same **matching** rules, **When** the user sends the command, **Then** **User Story 1** applies and **one** row is **removed** without a disambiguation step.

---

### User Story 5 - Help shows current features (Priority: P2)

A **listener** opens the product’s **help** (e.g. **`/help`**). The reply **lists** the **current** user-facing **commands** and **short** descriptions so they can **discover** what the bot does **without** **guesswork**.

**Why this priority**: Remove is useless if people never find it; an accurate **help** is the **default** **discovery** path alongside word-of-mouth. **(Clarified 2026-04-26: help must be updated with the full current feature set when this feature ships.)**

**Independent Test**: Trigger help after release; verify **`/remove`** appears with a one-line description, and that other still-supported commands remain listed in line with the rest of the product’s shipped behavior (spot-check against the release checklist or product inventory used by QA).

**Acceptance Scenarios**:

1. **Given** a **release** that **includes** the **remove** feature, **When** a **listener** uses **help**, **Then** the **text** **includes** **`/remove`** **(and** a **one-line** **explanation** **of** **what** **it** **does**)** and **does** **not** **omit** **other** **supported** **commands** still **in** **the** **product**.
2. **Given** **help** is **updated**, **When** a **new** user **reads** it, **then** they can **identify** **at** **least** **one** **way** to **see** **saved** **albums**, **one** **way** to **add** **albums** (or the **product**’s **stated** **path**), **and** **how** to **remove** a **saved** **album** **from** **that** **list** (wording may vary; **must** be **actionable** and **non-technical**).

---

### Edge Cases

- **Empty or whitespace-only** text after the command: the product **does not** perform a delete; it **tells** the user **how to use** the command (e.g. that an album name is **required**).
- **Very long** user input: the product **does** **not** **delete**; it applies **A3** (same or stricter cap as **`/album`** free text) with a **clear** user-facing **limit** message.
- **Listener** has **no** saved albums at all: any remove attempt behaves like **not found** (User Story 2) without implying a different error.
- **Partial** **matches** **exceed** **three** **(no** **exact** **match**)** per **FR-003**/**FR-004**: the product **does** **not** **enumerate** **every** **row**; it **asks** **the** **user** **to** **narrow** **the** **query**.
- **Concurrent** or **rapid** **repeated** remove commands: the product’s behavior remains **predictable** (e.g. second remove of the same title is **not found** after successful removal); no **spec** requirement on real-time **multi-device** **sync** beyond **observable** **consistency** for the same **conversation** (assumption).
- **Help** **drift**: If **the** **code** **adds** **or** **removes** **user**-**visible** **commands** **in** **the** **same** **release** **as** **remove**, **help** **must** **be** **updated** **in** **tandem** so **stale** **or** **missing** **lines** **do** **not** **ship** (see **FR-008**).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST allow an authenticated **listener** to **remove** **one** of their **saved albums** by sending a **remove** command with a **valid** **album** **query**: text that **identifies** the **album**, is **non-empty** after **trim**, and is **not** over the **maximum** **length** in **A3** (same or stricter cap as other album free-text commands; over-length and empty are rejected per Edge cases, not treated as a match or delete).
- **FR-002**: The product MUST **normalize** the **album query** **before** comparing it to **saved** data for that **listener** only; matching MUST use the **same** **normalization** **conventions** as in **User Story 3** (and MUST be **testable** with fixed examples).
- **FR-003**: The product MUST compare the normalized **album query** to each saved row’s **stored** **title** (the **same** user-visible **field** as **in** **list**/**save**). **Lookup** **uses** **the** **title** **field** **only** — **not** **parsing** a **combined** "Artist - Title" **string** into **separate** **search** **fields** (**Clarifications**). **Matching** is **evaluated** **in** **order**: **(1)** **Exact** **tier** — after **normalization** (**FR-002**), the **normalized** **user** **string** **equals** the **normalized** **stored** **title**; use **one**-**row** **delete** **(FR-005)**, or **disambig** **(FR-006)** **when** **≥2** **rows** **qualify** **(multi-exact).** **(2)** **If** **(1)** **qualifies** **no** **rows** **(no** **exact** **match)**, **Partial** **tier** — a **row** **qualifies** when the **normalized** **user** **query** **is** **non**-**empty** **and** **the** **normalized** **stored** **title** **contains** **that** **string** **as** **a** **contiguous** **substring**. **If** **(2)** **yields** **0** **rows** **and** **(1)** **yielded** **0**, **FR-004** **applies** **(not** **found**)**. **If** **(2)** **yields** **1–3** **rows** **and** **(1)** **yielded** **0** **(only** **partials)**, the product MUST **not** **delete** **without** a **pick**; it MUST offer **the** **same** **numbered** **disambiguation** **+** **`remove_saved`** **session** **as** **multi-exact** **(User** **Story** **4**)**. **If** **(2)** **yields** **more** **than** **three** **rows** **and** **(1)** **yielded** **0**, the product MUST **not** **delete** **any** **row** **and** MUST **ask** **the** **user** **to** **be** **more** **specific** **(FR-004** **second** **bullet**)**. **Disambiguation** **lines** **MAY** **include** **primary** **artist** **and** **year** **for** **readability** **only**.
- **FR-004**: When **neither** **exact** **(FR-003(1))** **nor** **partial** **(FR-003(2))** **yields** **a** **qualifying** **row** **(both** **empty)**, the product MUST **not** **delete** **any** **row** **and** MUST **state** that **no** **matching** **saved** **album** **was** **found** **(User** **Story** **2**)**. **When** **(1)** **is** **empty** **and** **(2)** **has** **more** **than** **three** **rows**, the product MUST **not** **delete** **or** **list** **all** **matches**; it **MUST** **ask** **the** **user** **to** **narrow** **or** **be** **more** **specific** **(User** **Story** **2**)**.
- **FR-005**: When **exactly** **one** **saved** **row** **qualifies** **under** **FR-003(1)** **(exact** **match** **tier**), the product MUST **remove** that **row** and **confirm** **success** with **at least** the **title** (or **equivalent** **label**). **Rows** **reached** **only** **via** **the** **partial** **tier** **(FR-003(2))** **require** **disambig** **(including** **the** **single**-**row** **case: one** **numbered** **line,** **then** **pick**)** — **no** **silent** **delete** **on** **partial** **alone**.
- **FR-006**: When **more** than **one** **saved** **row** **qualifies** **under** **the** **active** **tier** **(multi-exact** **or** **1–3** **partials** **per** **FR-003(2))**, the product MUST **not** **remove** **a** **row** **without** **listener** **explicit** **selection**; it MUST follow **User Story 4** **(disambiguation** **and** **an** **explicit** **pick,** **or** **narrow** **the** **query** **per** **FR-004** **when** **>3** **partials**)**. **For** the **disambig** **pick** **step,** the **messaging** **host**’s **interactive** **controls** **MUST** **be** **used** **where** the **host** **API** **and** the **shipped** **adapter** **support** **them** **(e.g.** **Telegram** **inline** **keyboard** **with** **one** **action** **per** **candidate,** **so** **confirming** **removal** **is** **not** **by** **typed** **index** **alone)**; the **message** **MAY** **still** **include** **numbered** **lines** **for** **readability** **(see** **Clarifications** **Session** **2026-04-26**)**. **Where** such **controls** are **unavailable,** a **valid** **session-bound** **numeric** **text** **pick** in **range** **(same** **semantics** as **`remove_saved`**) **MUST** **remain** **available** and **MUST** **suffice** to **complete** the **pick**.

- **FR-007**: The **remove** **command** **MUST** be **discoverable** in **the** **same** **ways** as **other** **bot** **commands**; **at** **minimum** it **MUST** appear in **the** **help** **inventory** with **a** **one-line** **description** (**FR-008**). Exact **messaging** **host** **mechanics** are **out** of **scope** in **this** **spec** beyond **clarity** for **end** **users**.
- **FR-008**: The **product** **MUST** keep **the** **user**-**visible** **help** **response** **(e.g.** **`/help`**) **aligned** with **the** **current** **set** of **shipped** **user**-**facing** **commands** and **primary** **capabilities** **(User** **Story** **5**). **When** **this** **remove** **feature** **is** **released**, **the** **same** **release** **MUST** **update** **help** **so** **`/remove`** **is** **listed** **and** **so** **no** **other** **still**-**supported** **command** **is** **missing** **from** **help** **without** **documented** **exception**. **Obvious** **core** **flows** (how to **add** a **saved** **album**, **see** the **list**, **remove** **from** the **list**) **MUST** be **discernible** **from** **help** **alone** or **from** **help** **plus** **one** **follow**-**up** **message** **in** **the** **same** **style** as **the** **rest** of **the** **product**.

### Non-functional and product quality *(should align with project constitution)*

- **UX consistency**: Wording and **tone** for **success**, **not** **found**, and **disambiguation** should **align** with **other** **album** **commands** (e.g. **save** **/album** disambig, **list**). **On** **hosts** **with** **inline** **or** **reply** **keyboards,** **remove** disambig **should** follow the **same** **principle** as **Clarifications:** **typed**-**only** **index** is **insufficient** **as** the **sole** **path** **when** **buttons** **are** **available;** **Error** and **help** text MUST **avoid** **blame** and **raw** **internal** **codes** in **end-user** **replies**.
- **Testing**: **Acceptance** **scenarios** **above** **plus** **normalization** **equivalence** **classes** (case, **spaces**) **must** be **verifiable** **independently**; **automated** **tests** **should** **cover** **routing** so **this** **command** **does** **not** **collide** with **other** **numeric** or **text** **handlers** in **private** **chats** (same **spirit** as **list**/**album** **routing** in **sibling** **specs**). **Help** **(FR-008)** **must** be **covered** by **at** **least** one **assertion** (or **snapshot** / **string** **check** in **the** **domain** **help** **content**) that **`/remove`** **appears** **and** **that** **help** **includes** **the** **same** **command** **labels** **the** **product** **claims** to **support** in **this** **release** **(exact** **mechanism** **left** to **planning**).
- **Logging and monitoring**: **Principle** **V** (monitoring) for the **bot** **workload** is met by **existing** **agreed** **signals** (see **Clarifications** Session **2026-04-26** — **Constitution** **V** / **NFR**). **Operational** **logs** (if any) should **not** **include** **message** **body** **PII** **beyond** **what** **policy** **already** **allows** for **other** **commands**. **Optional** **v1** **enhancement** (not **required** for **acceptance**): **redacted** **structured** **logs** (e.g. **remove** **outcome** **enum** **only**) **or** **event** **counts** for **removes** / **not** **found** / **disambig**; if **omitted**, **follow** the **same** **Clarifications** **bullets** (NFR **Q** on **dedicated** **metrics** and **V** **baseline**).
- **Performance**: **Not** **performance**-**sensitive** for **v1** beyond **reasonable** **responsiveness** for **a** **single** **user** **action** (e.g. **under** a **few** **seconds** in **typical** **conditions**); **no** **throughput** **SLO** in **this** **spec**.
- **Containerization / decommissioning**: **No** **new** **runnable** **component**; **N/A** **unless** **implementation** **adds** **new** **services**. **Decommissioning**: **N/A** **for** **a** **new** **command**.

### Key Entities *(include if feature involves data)*

- **Listener**: The **end** **user** **identity** used **across** **bot** **commands**; **removal** **always** **scoped** to **their** **saved** **rows** **only**.
- **Saved album (row)**: A **row** in the **listener’s** **library** with **at** **least** a **stored** **title** and **metadata** as **defined** in **existing** **product** **data** **model**; **removal** **deletes** **one** **such** **row** **by** **identifier** after **match** **resolution**.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In **manual** or **scripted** **acceptance** **runs**, a **listener** with **a** **single** **matching** **saved** **title** can **remove** that **row** in **one** **interaction** (command **plus** any **required** **disambiguation** **step** for **non-ambiguous** **cases** **only** the **first** **command**), and **a** **follow-up** **list**-style **check** (or **equivalent**) **shows** the **album** **absent** **in** **100%** of **seeded** **tests**.
- **SC-002**: In **acceptance** **runs**, when **neither** **FR-003(1)** **nor** **FR-003(2)** **qualify** **any** **row**, **zero** **rows** are **deleted** and **100%** of **cases** **show** **“not** **found”** **(or** **equivalent)** **without** **a** **false** **success**; **when** **>3** **partials** **with** **no** **exact** **match**, **100%** **ask** **to** **narrow** **per** **FR-004** **without** **deleting** **or** **listing** **every** **match**.
- **SC-003**: In **acceptance** **runs** **with** **multiple** **qualifying** **rows** **(multi-exact** **or** **1–3** **partials)**, **zero** **silent** **deletions** **occur**; **100%** **receive** **disambiguation** **(pick)** **or** **a** **narrow** **path** **(>3** **partials)** **per** **FR-006** **and** **FR-004** **before** **any** **removal**. **On** **Telegram**, **acceptance** **MUST** **verify** **that** **a** **user** **can** **complete** the **remove** **pick** **using** an **inline** **keyboard** **(or** **documented** **equivalent)**, **not** **by** **text** **index** **alone;** the **session-bound** **numeric** **text** **pick** **MUST** **remain** **a** **supported** **alternate** **(see** **Clarifications**)**.

- **SC-004**: For a **release** that **includes** this **remove** feature, **100%** of **release** **acceptance** **checks** on the **help** (or **equivalent**) **text** **confirm** **`/remove`** is **present** with a **non-empty** **one-line** **description** and **no** **omission** of **other** **still-supported** user-facing **commands** **(FR-008**).

## Assumptions

- **A1**: **Normalization** for **this** **command** **aligns** with **the** **list** feature’s **text** **normalization** **for** **queries**: **trim** **ends**, **collapse** **internal** **whitespace** to **one** **space**, and **case**-**insensitive** **comparison** for **title** **matching** **unless** **planning** **documents** a **different** **rule** (then **this** **spec** **is** **updated** **or** a **clarification** **note** is **added**).
- **A2**: “Album name” is the user-provided string after the command name. **FR-003** **applies** **first** **exact** **whole-title** **equality** **on** **normalized** **strings**, **then** **(if** **no** **exact** **hit**)** **partial** **substring** **on** **the** **normalized** **stored** **title** **(see** **Clarifications** **2026-04-26**)**. A user string such as `Artist - Album Title` is normalized as one string for **search**; **v1** does **not** **split** it into artist vs **title** **fields** for **lookup** — **search** is **title**-**text** **only** **(same** **as** **earlier** **clarifications**)**.

- **A3**: The remove query remainder **MUST** obey the **same** (or a **stricter**) rune/character **limit** as other album text commands in this product (e.g. `/album` free text; **implementation** may use **`MaxQueryRunes`** or a shared cap). If the user exceeds the limit, the product **MUST** not delete any row and **MUST** respond with a clear, user-facing limit message. **v1** **does** **not** **add** new per-user throttling: **if** the **host** or **bot** has **no** **rate** **limits** **today**, **this** **feature** **adds** **none**; **if** **limits** **exist** for **other** **commands**, **remove** **follows** the **same** **policy** (see **Clarifications** Session **2026-04-26**).
- **A4**: **Feature** **depends** on **existing** **persistence** of **saved** **albums**; **no** **change** to **the** **definition** of **“saved”** **beyond** **deleting** **a** **row**.
- **A5**: **Help** text lives in the product’s **shared** user-visible copy (domain or a **documented** single **source**); **updating** help for this feature does **not** require a **separate** help microsite—the in-chat **`/help`** (or **mapped** **equivalent**) is **sufficient** for **SC-004**.
