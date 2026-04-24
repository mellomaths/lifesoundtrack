# Feature Specification: Daily fair album recommendations

**Feature Branch**: `004-daily-album-recommendations`  
**Created**: 2026-04-24  
**Status**: Draft  
**Input**: User description: "On a fixed schedule (every day at 6am), the bot sends each eligible user **one** album recommendation from their saved **`saved_albums`**, using **fair rotation** (never-recommended and longest-since-recommended first) and **uniform random** choice within that tier. Each successful send updates **`saved_albums.last_recommended_at`** and appends a row to **`recommendations`** (title/artist snapshots) in **one transaction** after Telegram accepts the message. The recommendation message should include the album art cover image and a button a link that redirects the user to the album's Spotify page, if not possible, the link should be pasted within the message text Example: ALBUM COVER Your pick today: TITLE — ARTIST (YEAR) {LINK - if button not available} Enjoy the listen — LifeSoundtrack. {BUTTON - if available}" — plus: **Implement feature flags using environment variables** so operators can enable or disable the daily job and configure schedule-related settings **without code changes**.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Wake up to one pick from my library (Priority: P1)

An eligible user who has saved albums in LifeSoundtrack receives **exactly one** proactive recommendation per scheduled daily run **while the feature is enabled for that deployment**. The message shows the album cover image, a line in the form **Your pick today: TITLE — ARTIST (YEAR)** (year omitted when unknown), a sign-off **Enjoy the listen — LifeSoundtrack.**, and a way to open the album on **Spotify**—either as an inline link in the text or as a tap target (button) when the messaging channel supports it. If only one presentation mode is possible for a given send, the Spotify URL still appears in the message in the supported form.

**Why this priority**: This is the core product moment: a daily, low-friction nudge back into music the user already cared enough to save.

**Independent Test**: With a test account that has at least one saved album and a known Spotify URL on that row, trigger the scheduled run once **with the feature enabled**; verify one outbound message, correct copy pattern, cover present, and Spotify destination reachable from the link or button.

**Acceptance Scenarios**:

1. **Given** an eligible user with at least one saved album and a resolvable Spotify album page URL for the chosen row, **When** the daily job runs **and** proactive recommendations are **enabled**, **Then** the user receives one message that includes cover art, the formatted pick line with title and artist (and year when known), the sign-off, and Spotify access via button **or** inline URL as supported.
2. **Given** the same user on a subsequent day, **When** the job runs again, **Then** the user again receives at most one recommendation message for that run (no duplicate sends for the same scheduled execution).
3. **Given** a user with saved albums but no Spotify URL available for the album selected by rotation rules, **When** the job runs, **Then** the user still receives the cover and formatted pick line and sign-off, and any link present follows the “paste in text if button not available” rule; if no Spotify URL exists at all, the message does not claim a broken button—text simply omits a URL line or states absence per agreed copy (see Assumptions).

---

### User Story 2 - Fair rotation across everything I saved (Priority: P1)

Over time, recommendations favor albums that have **never** been recommended in this way, then albums whose **last recommendation** lies furthest in the past. Whenever multiple saved albums tie on that priority, the system picks **uniformly at random** among the tied set. The user experiences variety rather than repeated emphasis on a small subset when their library has many rows.

**Why this priority**: Without fair rotation, power users with large libraries feel the product is “stuck” repeating the same picks.

**Independent Test**: Seed a listener with several saved albums with controlled `last_recommended_at` values (including null/never); run the job repeatedly in a test harness and verify selection order and random tie-breaking behave as specified.

**Acceptance Scenarios**:

1. **Given** a user with some albums never recommended and some recommended before, **When** a pick is made, **Then** every never-recommended album is strictly preferred over any previously recommended album.
2. **Given** a user where all albums have been recommended at least once, **When** a pick is made, **Then** an album with the oldest `last_recommended_at` is strictly preferred over one with a more recent timestamp.
3. **Given** two or more albums in the same top priority tier (all never recommended, or equal `last_recommended_at` per tie semantics), **When** a pick is made, **Then** each album in that tier has equal probability of selection for that draw (uniform at random).

---

### User Story 3 - Only record success when delivery actually lands (Priority: P1)

When the messaging platform (**Telegram** in production) **accepts** the outbound recommendation message, the system updates the chosen saved album’s **`last_recommended_at`** and appends one **`recommendations`** row holding **title** and **artist** (and any other agreed snapshot fields) **in a single atomic persistence action** with those two effects. If the message is not accepted (network failure, user blocked the bot, rate limits, etc.), **neither** the timestamp **nor** the history row is written for that attempt.

**Why this priority**: Prevents the rotation logic from skipping albums the user never saw.

**Independent Test**: Simulate acceptance and failure paths; assert database effects match the outcome and that partial updates cannot occur.

**Acceptance Scenarios**:

1. **Given** Telegram returns success for the send, **When** persistence runs, **Then** `last_recommended_at` on the chosen `saved_albums` row and a new `recommendations` row are both committed together.
2. **Given** Telegram does not accept the message, **When** the job finishes handling that user, **Then** no new `recommendations` row is created and `last_recommended_at` is unchanged for that attempt.
3. **Given** two concurrent or overlapping attempts for the same user are possible in deployment, **When** the design is validated, **Then** the product still guarantees at most one successful recommendation per user per scheduled daily run (see Assumptions for timezone and idempotency).

---

### User Story 4 - Operators toggle the daily job with deployment settings (Priority: P1)

Operations and maintainers can **turn off** the daily recommendation behavior for a deployment using **environment variables only** (no code change, no redeploy of different binaries beyond config). When turned off, **no** proactive recommendation messages are sent and **no** `last_recommended_at` or `recommendations` updates occur from this job. When turned on, behavior matches User Stories 1–3. Operators can also set **when** the daily run fires (wall-clock within a configured timezone) using **environment variables**, consistent with how other LifeSoundtrack operator toggles work.

**Why this priority**: Safe rollout, incident response, and cost control require a supported kill switch and schedule control without engineering intervention.

**Independent Test**: In a staging deployment, flip the master enable variable off and observe a full scheduled window with zero sends; flip on and observe the job resume per configured schedule.

**Acceptance Scenarios**:

1. **Given** the master enable flag is **off** via environment, **When** a scheduled run time passes, **Then** no recommendation messages are sent and no rotation-related database updates occur for that run.
2. **Given** the master enable flag is **on** and schedule variables are valid, **When** the configured daily time arrives in the configured timezone, **Then** the job executes User Story 1 for each eligible user.
3. **Given** operators change only environment variables and restart or roll the process per normal release practice, **When** the new values load, **Then** enable/disable and schedule behavior reflect the new settings without code edits.

---

### Edge Cases

- User has **no** saved albums: skip that user for the run; no persistence and no error message required unless operators define otherwise.
- User blocks the bot or chat is unavailable: treat as non-acceptance; no rotation advance.
- **Single** saved album: always that album when eligible; random tier has one element.
- All albums are never-recommended: entire library is tier one; uniform random among all.
- Album has cover URL missing: send text layout still; cover omitted or placeholder behavior per Assumptions.
- Scheduled run overlaps slow sends: processing may span wall-clock time, but each user still receives at most one pick **per run** definition.
- Clock changes (DST, manual time changes): daily boundary follows operator-configured timezone rules.
- **Feature disabled mid-deployment**: after reload, no further scheduled sends until enabled again; partial runs mid-toggle follow “at most one per run” for already-started invocations only.
- **Listener discovery fails** (for example the database rejects the query used to list eligible listeners): the run MUST surface the failure to operators via logs; the product MUST NOT treat the run as having successfully processed users until discovery succeeds.
- **Scheduler not started**: if the daily job never registers a schedule while the feature is enabled (**FR-017**), operators see no scheduled ticks; this is treated as a defect, not an optional enhancement.

## Defect fix: Listener enumeration (P1)

*The narratives in this and the following defect subsection elaborate **FR-013**–**FR-017** for operators and postmortems; normative requirements remain under **Requirements** below.*

**Severity**: P1 — the scheduled job can fail before processing any listener.

**Symptom (operators / debugging)**: The daily run stops during listener discovery (for example log context `daily_recommendations_listeners`), with a database error that indicates **deduplicating** the listener list while **ordering** by expressions that are not part of the deduplicated select list is invalid on the production database. **Example observed** (local manual test, 2026-04-24): `SQLSTATE 42P10` — *for SELECT DISTINCT, ORDER BY expressions must appear in select list*.

**User impact**: No eligible user receives a recommendation for that run even when the feature is enabled and users have saved albums.

**Required outcome**: Listener discovery MUST use a data-access shape accepted by the production database, MUST return the same **distinct** set of listeners as required by **FR-002** and by [Listener iteration](contracts/daily-recommendations-job.md) in the job contract (listeners with at least one `saved_albums` row, subject to the same filters as the rest of the feature, such as messaging source), MUST ensure **at most one** processing pass per listener per **`run_id`**, and MUST keep operator logs for this step correlated with **`run_id`**. If a deterministic processing order is desired, it MUST be expressed without changing who is eligible or duplicating listeners within a run.

**Out of scope for this fix**: Changing fair rotation rules, Telegram payload, or feature-flag semantics beyond what is necessary to meet the requirements below.

## Defect fix: Scheduled job never runs (P1)

**Severity**: P1 — users never receive daily recommendations on the intended schedule even when the feature is enabled and configuration looks correct.

**Symptom (operators / debugging)**: Process logs show Telegram long-polling but **no** recurring “tick” or batch start for the daily recommendation job at the configured wall time; no **`run_id`**-correlated activity for scheduled runs.

**User impact**: Eligible listeners never get proactive picks from the daily job despite **FR-001** and **User Story 4**.

**Required outcome**: While the master enable flag is **on**, the **same long-running process** that serves Telegram MUST **register and drive** the configured daily schedule for the recommendation job (so the schedule is not dependent on a missing in-process hook or an external OS cron that was never set up). The schedule MUST run **concurrently** with messaging (for example long-polling) without one path preventing the other. Startup logs MUST record whether daily recommendations are enabled and the resolved schedule identity (**FR-018**). When the flag is **off**, no scheduler callbacks for this job run (**FR-010**).

**Out of scope for this fix**: Changing the fair-rotation algorithm or message template; only ensuring the schedule actually fires and reaches downstream steps (listener discovery and beyond) as already specified.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST run an automated **daily** job at **06:00** in an **operator-configured timezone** (documented default: UTC) and process **each eligible user** once per run **when the daily recommendation feature is enabled** for that deployment.
- **FR-002**: An **eligible user** MUST be defined as a listener who has **at least one** row in **`saved_albums`**, excluding users the product definition marks as opted out (if no opt-out exists in scope, all listeners with saved albums are candidates for a send attempt). **Reachable for that attempt** means the messaging platform **accepts** the outbound recommendation (**FR-007**); if the platform **does not** accept the message (blocked bot, chat unavailable, rate limits, etc.), that user MUST be treated as not having received the pick for rotation purposes (**FR-008**), the job MUST continue with other listeners, and the outcome MUST be observable per **FR-009** without logging message bodies or secrets.
- **FR-003**: For each eligible user, the system MUST select **exactly one** `saved_albums` row per run using **fair rotation**: first prioritize rows with **no** prior successful recommendation recorded via **`last_recommended_at`** (never-sent tier); among rows not in that tier, prioritize the **smallest** (oldest / null-safe) `last_recommended_at`. **Ties** within the winning tier MUST be broken by **uniform random** selection.
- **FR-004**: The outbound message MUST include **album cover art** when a cover image URL or equivalent is available on the chosen saved album; otherwise the message MUST still send with the text layout intact.
- **FR-005**: The outbound message MUST follow the copy pattern: **Your pick today: TITLE — ARTIST (YEAR)** with year omitted when unknown, and MUST end with **Enjoy the listen — LifeSoundtrack.**
- **FR-006**: When a **Spotify album page URL** can be determined for the pick, the system MUST present it as a **button** (inline keyboard / tap target) **when the Telegram integration supports URL buttons for that send**; otherwise the URL MUST appear **in the message text** in place of the button line. If no Spotify URL can be resolved, the system MUST NOT fabricate a broken link.
- **FR-007**: After **Telegram accepts** the sent message, the system MUST, in **one transaction**: update **`saved_albums.last_recommended_at`** for the chosen row to the send timestamp (or agreed clock source) and **insert** one row into **`recommendations`** containing at least **snapshot** **title** and **primary artist** matching what the user was shown.
- **FR-008**: If Telegram does **not** accept the message, the system MUST **not** update **`last_recommended_at`** and MUST **not** insert a **`recommendations`** row for that attempt.
- **FR-009**: The system MUST log or otherwise observe send outcomes per listener (success vs failure reason bucket) so operators can detect systemic delivery issues without logging message contents or secrets.
- **FR-010**: The daily recommendation job MUST be **governed by feature flags supplied as environment variables** (master enable/disable and schedule-related settings), documented for operators, such that **disabling** the flag prevents **any** scheduled recommendation sends and **any** associated rotation persistence for that job.
- **FR-011**: **Enabling** the master flag MUST restore scheduled behavior according to the configured timezone and daily schedule environment settings, without requiring source code changes.
- **FR-012**: Boolean flag semantics for the master enable switch MUST be **consistent** with existing LifeSoundtrack operator toggles (e.g. metadata catalog enables): **unset or empty means enabled**; only **explicit false-like** values disable, as documented in the implementation plan.
- **FR-013**: The daily job MUST complete **listener discovery** (loading the distinct set of listeners to consider for the current **`run_id`**) without errors that abort the entire run before per-listener work, when executed against the production database and configuration.
- **FR-014**: Listener discovery MUST return exactly the set of **distinct** listeners who satisfy **FR-002** (and the job contract’s listener iteration), including the same eligibility filters (for example messaging channel / source) as the rest of the feature; it MUST NOT omit or double-count an eligible listener for that run.
- **FR-015**: When the implementation applies a **stable or deterministic order** to listener discovery, that ordering MUST remain valid for the production database and MUST NOT introduce duplicate listener rows for a single **`run_id`** or change eligibility versus **FR-002**.
- **FR-016**: Operator logs for listener discovery MUST remain correlated with **`run_id`**, and discovery failures MUST remain visible and actionable (no silent swallowing that hides a batch-level failure).
- **FR-017**: When the master enable flag for daily recommendations is **on**, the primary bot process MUST register and execute the configured **daily** wall-clock schedule for this feature for the lifetime of that process, without requiring a separate external scheduler, and MUST NOT rely solely on user-driven Telegram traffic to trigger runs.
- **FR-018**: At process startup, operators MUST be able to see in logs whether daily recommendations are **enabled**, and the resolved **timezone** and **schedule** identity (not secrets), consistent with **User Story 4** observability.

### Non-functional and product quality *(should align with project constitution)*

- **UX consistency**: Copy matches the agreed template; Spotify presentation follows button-first, text-fallback ordering; failures do not leak internal errors to end users.
- **Testing**: Selection logic (tiers, ties, randomness distribution smoke), persistence coupling to acceptance, message assembly, **listener discovery** (including regression coverage so listener-list loading cannot fail the whole run for the class of error observed in the **Defect fix** section), and **flag-off / flag-on** behavior are independently verifiable; integration tests cover Telegram acceptance vs failure hooks if a test double is used.
- **Linting**: No project-wide rule relaxations; any new modules follow existing Go and repository conventions when implemented.
- **Logging and monitoring**: Per **FR-009** (per-listener outcome buckets, correlate by **`run_id`**) and **FR-018** (startup visibility of enable + timezone + schedule identity, no secrets); aggregate counts as appropriate for operators; no PII beyond what policy already allows.
- **Performance**: The daily batch MUST complete within a window appropriate for a small-to-medium listener base (exact SLA left to plan); per-user work SHOULD avoid unnecessary external calls when data already exists on `saved_albums`.
- **Containerization**: If the scheduler runs inside an existing runnable, plan updates to **Docker Compose** and example env files when new variables are introduced.
- **Decommissioning**: If the feature is retired, remove scheduler entry points and document migration for `recommendations` and `last_recommended_at` semantics.

### Key Entities *(include if feature involves data)*

- **`saved_albums` (existing)**: User’s saved album rows; gains or uses **`last_recommended_at`** (timestamp of last **successful** daily recommendation send for that row).
- **`recommendations` (new or extended)**: Append-only history of successful sends: at minimum **listener** identity, **saved album** reference or stable id, **title** and **artist** snapshots, **sent at** timestamp; may include optional **year**, **Spotify URL** snapshot, and **run** identifier for auditing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For test accounts with known libraries, **100%** of successful sends update **`last_recommended_at`** and create exactly **one** matching **`recommendations`** row in the same persistence action.
- **SC-002**: For test accounts, **0%** of failed Telegram accepts produce a new **`recommendations`** row or advance **`last_recommended_at`**.
- **SC-003**: Across a Monte Carlo or repeated-run harness with tied tiers, observed pick frequencies for tied albums converge to **equal** likelihood within the statistical tolerance defined in [research.md](research.md) §10 (tie-break fairness).
- **SC-004**: At least **95%** of eligible users with a Spotify URL on the chosen row receive a working Spotify destination via button or text in user acceptance testing.
- **SC-005**: Operator-facing metrics show **at most one** counted successful recommendation per user per daily run in a **pre-production soak** of at least **two weeks** (no double-count spikes without documented cause).
- **SC-006**: In a controlled test environment, with the master enable flag **off**, **100%** of observed scheduled windows produce **zero** outbound daily recommendation messages and **zero** new **`recommendations`** rows attributed to that job.
- **SC-007**: In test or staging, when the feature is **on** and at least one **FR-002**-eligible listener exists, **100%** of triggered runs complete listener discovery and proceed to per-listener handling (or documented skips such as no saved albums), with **0%** of runs failing the entire batch solely because listener discovery could not be executed successfully.
- **SC-008**: In test or staging with the feature **on**, over any **24-hour** wall-clock window (or a shorter **documented test schedule** used only for validation), **100%** of windows include **at least one** observable scheduled run start (for example a log correlated with a **`run_id`**) proving the in-process schedule is active—not only Telegram message handling.

## Assumptions

- **A-001**: “Never recommended” means **`last_recommended_at`** is null (or equivalent sentinel agreed in data modeling); any row successfully sent updates this field.
- **A-002**: Daily **06:00** uses a **single** configured timezone for all users in this release; **per-user local 6am** is out of scope unless added later.
- **A-003**: Spotify URLs come from data already stored on **`saved_albums`** or from a deterministic resolver documented in the plan; no requirement to call Spotify on behalf of the end user if metadata is incomplete.
- **A-004**: Missing cover art does not block the send; visual may be omitted rather than failing the job.
- **A-005**: **`recommendations`** may omit **year** in storage if only title/artist are mandatory, but the user-visible line still shows year when known from the saved row.
- **A-006**: Eligible-user opt-out (mute digests) is **out of scope** unless already present; if absent, every listener with saved albums is processed.
- **A-007**: Canonical **names**, defaults, and parsing semantics for daily-recommendation feature flags are in [contracts/feature-flags.md](contracts/feature-flags.md). The [implementation plan](plan.md) **Environment (operators)** subsection references that contract; behavior in **FR-010**–**FR-012** and **User Story 4** is binding.
