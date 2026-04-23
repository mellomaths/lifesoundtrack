# Feature Specification: LifeSoundtrack — load configuration from `.env` files

**Feature Branch**: `002-env-file-config`  
**Created**: 2026-04-25  
**Status**: Draft  
**Input**: User description: "Enable environment variables configuration through .env files. Currently the LifeSoundtrack bot does not look for the .env config. We need the ability to run the application with .env files"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run the bot with a local env file (Priority: P1)

A **developer or operator** runs the LifeSoundtrack process from a checkout on their machine. They create a **non-committed** `.env` file in the place described in the product runbook, put **TELEGRAM_BOT_TOKEN** and optional **log level** there, and start the app **without** having to copy those values into the shell for every session.

**Why this priority**: This is the core ask—repeatable local and sandbox setup without ad-hoc `export` steps.

**Independent Test**: With only a `.env` file present and no `TELEGRAM_*` in the current shell, start the app from the documented working directory; configuration MUST succeed if the file is valid and complete for required values.

**Acceptance Scenarios**:

1. **Given** a valid `.env` in the **documented location** with **TELEGRAM_BOT_TOKEN** set (and optional settings as required by the product), **When** the process starts with **no** matching variables exported in the OS environment, **Then** the application loads that configuration and runs as it would when those same values are only in the environment.
2. **Given** a `.env` that sets a variable, **and** the same key is set in the **OS environment** to a different value, **When** the process starts, **Then** the OS environment value **takes precedence** (so CI and production overrides remain predictable).
3. **Given** a `.env` is **absent** or **empty of required keys**, but required keys exist only in the **OS environment**, **When** the process starts, **Then** configuration MUST still be satisfied from the environment alone (file is optional, not a hard dependency of startup).

---

### User Story 2 - Safe handling when the file is wrong or missing (Priority: P1)

A user misplaces the file, uses a bad line, or forgets a required value. The product MUST fail in a **clear, operator-friendly** way (no secrets printed), consistent with how missing configuration is handled when only OS env is used.

**Why this priority**: Prevents “silent” wrong config and avoids leaking tokens in error output.

**Independent Test**: Remove `.env` with nothing in the shell, or provide an invalid file; error behavior MUST be understandable without exposing token values.

**Acceptance Scenarios**:

1. **Given** **TELEGRAM_BOT_TOKEN** is missing from **both** `.env` and the OS environment, **When** the process starts, **Then** startup fails with an error that does **not** include any secret or full file contents.
2. **Given** a `.env` that is **not** parseable as valid `KEY=VALUE` / comment syntax for the **chosen OSS file loader** (e.g. **syntactically** invalid file), **When** the process starts, **Then** startup **fails** with a **generic** operator-visible error (no full file dump, no token) so mis-edits are not silently ignored. *Comment lines (`#` …) and typical CR/LF/whitespace are accepted per that loader; v1 does **not** spec a separate “ignore bad line and continue” mode.*

---

### User Story 3 - Documentation and discovery (Priority: P2)

A new contributor reads the runbook and knows **where** to put `.env`, **which** keys are supported, and that **`.env` must not be committed** to version control.

**Why this priority**: Unlocks Story 1 for people who were not the original author.

**Independent Test**: Follow only the public runbook: create the file, fill documented keys, start the app; success without tribal knowledge.

**Acceptance Scenarios**:

1. **Given** a reader following the product quickstart, **When** they place `.env` as documented, **Then** their configuration is loaded per Story 1.
2. **Given** a reader checking source hygiene rules, **When** they look at repository guidance, **Then** it is clear that real `.env` files with secrets are excluded from the **shared** codebase (per team policy already assumed elsewhere).

---

### User Story 4 - Fast local iteration (reload on source and `.env` changes) (Priority: P1)

A **developer** runs the app **locally** while changing **Go source** and/or the **local `.env` file**. The development workflow **restarts the process** so the next run picks up the latest code and the latest file-based configuration **without** the developer having to remember to kill and restart the process for every change.

**Why this priority**: Shortens the feedback loop for configuration and code changes during implementation and sandbox validation.

**Independent Test**: With the **documented** local development command, change a value in `.env` or a small safe edit in a watched source file, save, and observe a **new** process start and behavior reflecting the new inputs (within a reasonable delay).

**Acceptance Scenarios**:

1. **Given** the developer is using the **documented** local “watch”/reload workflow, **When** they save a change to **the product’s own source files** in scope for that workflow, **Then** the running bot process is **replaced** by a fresh run that includes those changes.
2. **Given** the same workflow, **When** they save a change to the **documented** `.env` file, **Then** a **restarted** process loads the updated environment according to the same precedence rules (OS env over file where applicable) without requiring a full machine reboot.
3. **Given** a **container image** or **production-style** start (not the local dev workflow), **When** the service runs, **Then** the hot-reload behavior is **not** a requirement—plain env injection and a normal single process remain sufficient.

### Edge Cases

- **No `.env` file**: Startup MUST still work if all required values are provided via the OS environment (e.g. containers, CI, production).
- **`.env` present but only partial**: Missing required keys after merging file + environment MUST yield the same class of startup failure as when everything is from env only and incomplete.
- **Line endings and whitespace**: Trimming and comments (lines starting with `#` where applicable) are handled so common editor-generated files work without hand-fixing.
- **Multiple “layers” of config** (e.g. `.env` vs `.env.local`) are **out of scope** for v1 unless added as a follow-up; a **single** documented file path is enough for this feature.
- **Hot reload in production** is **out of scope**; automatic restart is required only for the **local development** experience (Story 4).
- **Unreadable `.env`** (e.g. OS permission denied): v1 does not require a bespoke message; startup may fail with a **generic** error without logging the file path’s **contents** (**FR-004**).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The LifeSoundtrack **runnable** MUST load key-value settings from a **local `.env` file** in a **documented** path relative to how the app is run (e.g. when started from the **bot** module directory for local work), in addition to reading the **OS process environment**.
- **FR-002**: For any configuration name supported by the product (at minimum **TELEGRAM_BOT_TOKEN** and **LOG_LEVEL** or their documented equivalents), a value set in the **OS environment** MUST **override** the same key from the `.env` file when both are present.
- **FR-003**: The `.env` file MUST be **optional**: if it is missing, the process MUST still start successfully whenever all **required** values are supplied only via the OS environment.
- **FR-004**: Startup failure due to **missing** required configuration MUST **not** echo secret values, full `.env` contents, or **TELEGRAM_BOT_TOKEN** in user-visible or log output in a way that would leak credentials.
- **FR-005**: The **quickstart** (or equivalent operator doc) for LifeSoundtrack MUST state **where** to place `.env`, **which** keys it may contain, the **precedence** rule (OS over file), and that committed repos MUST NOT include real secrets in tracked files.
- **FR-006**: This feature is **operational and delivery**-scoped: it MUST NOT change **domain** command behavior, user-visible **copy** for start/help/ping, or the **platform-agnostic domain vs adapter** split described in the existing product specification—only how configuration is **supplied** to the process.
- **FR-007**: The implementation MUST use a **maintained, open-source** library to parse and apply the **`.env` file** into the process environment (or an equivalent effect that honors **FR-002** precedence), in preference to a hand-rolled parser. The choice and version are **documented** in the implementation plan and dependency metadata.
- **FR-008**: The repository MUST document a **local development** command (or small set of commands) that **watches** applicable **Go source** and the **documented** `.env` file and **restarts** the application process on change, for use on developer machines. **CI and production** entrypoints are unchanged except where they also benefit from **FR-001** (plain `.env` or env-injection) without watch mode.

### Non-functional and product quality *(should align with project constitution)*

- **UX consistency**: N/A to end-user messages; any new operator-facing text MUST stay consistent with the rest of the runbook.
- **Testing expectations**: Stakeholders MUST be able to **verify** Story 1 with a file-only setup and Story 2 with missing/invalid file scenarios without a live chat network if the rest of the app allows offline tests; unit-level checks MAY target configuration loading in isolation in planning/implementation.
- **Linting**: N/A to the spec; implementation planning may add patterns for not baking secrets into source.
- **Logging and monitoring**: Configuration errors at startup MUST remain safe for logs (no token dumps); routine logs SHOULD continue to distinguish **configuration errors** from **runtime** events without logging `.env` bodies.
- **Performance**: N/A; one-time read at startup.
- **Containerization**: Running under Docker or Compose MUST remain coherent with **env-injection** from the host; `.env` inside the image is not required for this feature to be satisfied if the runbook shows passing variables via Compose or runtime env. **File watchers / hot reload** are **not** required for the container path unless explicitly added later; local dev uses the documented watch workflow.
- **Decommissioning**: N/A for v1.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: **100%** of runbook document reviewers agree that a new team member can configure **TELEGRAM_BOT_TOKEN** (and any other **required** named settings) using **only** a local `.env` in the **documented** location, without exporting those variables in the shell. *Evidence of meeting this criterion is a **process** artifact (e.g. PR review, release checklist, or team sign-off)—it is not asserted by a single automated test.*
- **SC-002**: In a **structured test** (manual or automated), for a fixed set of keys, the process observes **OS environment values** over **`.env` values** when both are set, in **100%** of trials for those keys.
- **SC-003**: **100%** of “missing required token” failure trials produce output that a reviewer classifies as **non-leaking** (no full token, no full `.env` content in the message).
- **SC-004**: In a **local development** trial using the documented watch workflow, **100%** of controlled saves to the **`.env` file** and to an in-scope **source file** are followed, within a stated window (e.g. one minute), by a **restarted** process that reflects the change (code or effective configuration), as agreed by a tester without ambiguity. *The “one minute” window is a **practical** example for dev machines, not a product-wide latency SLA. Evidence is **manual** or scripted trial notes, not CI.*

## Assumptions

- **Product scope** is the **LifeSoundtrack** bot process (same deployment unit as today’s `go run` / container entry); other services are out of scope unless explicitly extended later.
- **`.env` format** follows common `KEY=VALUE` conventions, optional comments on lines starting with `#`, and is UTF-8 text. **FR-007** requires a **named** maintained open-source **library** in implementation; the **spec** does not fix the module (that lives in the **plan** and `go.mod`)—this avoids duplicating the **FR-007** obligation as a “no library” assumption.
- **Secret storage** in team **Git** remains “do not commit real `.env`”; `.env.example` may list names with empty or placeholder values.
- This feature **builds on** the existing variable names and semantics already described for the Telegram adapter and logging (see feature **001** runbook and contracts) without renaming them unless a separate change amends that documentation.

## Clarifications

### Session 2026-04-25

- Q: If `.env` and OS env conflict, which wins? → **A: OS process environment overrides `.env` for the same key.**
- Q: Is `.env` required for startup? → **A: No; it is optional if OS env alone is complete.**
- Q: One file or many? → **A: Single documented path (e.g. `bot/.env` for local runs) for v1; additional files are a possible follow-up.**

### Session 2026-04-25 (planning input)

- Q: How should `.env` be parsed? → **A: Use a **maintained open-source** library (choice recorded in the implementation plan) rather than a custom parser.**
- Q: Hot reload? → **A: Yes for **local development**—rebuild/restart the process when **source** or the **documented** `.env` changes; not required for production/CI.**
