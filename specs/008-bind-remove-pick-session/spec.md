# Feature Specification: Bind remove picks to disambiguation session

**Feature Branch**: `008-bind-remove-pick-session`  
**Created**: 2026-04-26  
**Status**: Draft  
**Input**: User description: Fix remove-album disambiguation so a pick (inline action or follow-up) applies to the same choice list the user was shown, not a newer or different one—preventing the wrong saved album from being removed when multiple flows or stale on-screen options exist.

## Clarifications

### Session 2026-04-26

- Q: **Analysis I1** — US3/FR-005 implied typed text could distinguish two **historical** remove lists, but plain Telegram private text carries no session id. What is in scope? → A: **Session-bound** correctness is **required** for **inline** `rmp:` callbacks. For **typed** `1`–`99` replies, the system uses the **current** `remove_saved` disambiguation row (latest open for that user with `kind: remove_saved`); v1 does **not** require recovering which **on-screen** list the user meant when storage already superseded a prior `remove_saved` row. **Guaranteed** alignment to a specific list when multiple chat messages are visible is via **inline** actions.
- Q: **Analysis C1** — Edge case “album save disambig after remove” → A: Implementation **must** keep `kind` gating: a typed index must not be interpreted as a **remove** pick when the open session is **album** disambiguation (array JSON), consistent with the private-chat message route order. Documented in Edge Cases and [tasks.md](./tasks.md) T009.
- Q: **Analysis C2** — Adapter tests vs only core? → A: Add a dedicated task (T014) to extend `adapter/telegram` tests so `rmp:` wiring is covered, in addition to core/store regressions.
- Q: **Analysis H1/H2** — Plan file drift? → A: Plan is updated: primary code paths are `remove_saved.go`, `store/disambig*.go`, `run.go`; [tasks.md](./tasks.md) is part of the feature directory.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Pick matches the list you chose (Priority: P1)

A listener is offered several matching saved albums to remove (for example, after a search that is not unique). They use the control that corresponds to one of the listed rows (the inline option for that row). The album that is removed is exactly the one that control represented—the title shown next to that option—not some other row that only appeared in a different, newer choice list.

**Why this priority**: Wrong removal is a data-integrity and trust failure; the primary fix is to align each action with the specific disambiguation “round” the user is completing.

**Independent Test**: Can be verified by simulating two different remove-choice lists in sequence and confirming that an action on the first list only affects candidates from that first list, never a candidate that exists only in the second list.

**Acceptance Scenarios**:

1. **Given** a listener was shown a remove disambiguation list (list 1) with at least two options, **When** they start a new remove disambiguation (list 2) so a newer list exists, **Then** using an action that still refers to list 1 must not remove an album that appears only on list 2 in place of a list-1 option.
2. **Given** a listener selects the inline action for row *k* on a specific remove disambiguation list, **When** the system processes that action, **Then** the removal corresponds to row *k* of **that** list’s candidates, not a different list’s row *k*.

---

### User Story 2 - Stale or invalid choice, safe outcome (Priority: P2)

If the choice list the user is acting on is no longer available (for example, it was replaced by a newer flow, or it expired), the listener must not see a different album removed “by mistake.” They get a clear, helpful outcome (such as an explanation that the choice is no longer active and what to do next) instead of a silent removal of the wrong item.

**Why this priority**: Prevents the worst failure mode when the user interface and the “current” server-side choice get out of sync.

**Independent Test**: Can be verified by revoking or superseding an older disambiguation list, then performing the old action, and checking that no incorrect removal occurs and the user is guided appropriately.

**Acceptance Scenarios**:

1. **Given** a remove disambiguation list is no longer valid for completion, **When** the listener uses a control that pointed at that list, **Then** no saved album is removed unless the system can still unambiguously tie the action to a valid, matching list; otherwise the user receives a non-destructive message.
2. **Given** a listener sees an old message with outdated choices, **When** they try to complete that outdated choice, **Then** the outcome does not remove an unrelated album (for example, the “first” album of a completely different, newer list).

---

### User Story 3 - Typed number follow-up stays “kind”-safe and single-session-correct (Priority: P3)

For accessibility and habit, a listener can still type a number to pick a row. Plain text in private chat **cannot** carry a disambiguation session id, so v1 does **not** guarantee matching a **superseded** on-screen list by text alone. The system must (a) only complete a `remove_saved` text pick when the **latest** open disambiguation session is `remove_saved`, (b) not mis-route that index to an **album-save** (or other) disambiguation session, and (c) avoid wrongful deletion when the `kind` does not allow a remove pick. For **guaranteed** “which list I saw” behavior when multiple disambiguation rounds existed, **inline** actions (US1) are the supported path.

**Why this priority**: Improves safety for the typed path without claiming impossible text semantics across superseded server state.

**Independent Test**: With **one** open `remove_saved` session, typing a valid index removes the corresponding candidate; with the latest session being **not** `remove_saved` (e.g. album disambig), the typed number does not remove a save by mistake (`ok` false to remove, host may fall through to other routing).

**Acceptance Scenarios**:

1. **Given** the latest open disambiguation session is **album** search (not `remove_saved`) and the user only sends a bare number, **When** the remove-pick handler runs, **Then** the system does **not** treat it as a `remove_saved` index pick against that session.
2. **Given** exactly one open `remove_saved` session, **When** the listener types a valid index, **Then** the corresponding candidate is removed, consistent with the labels they were shown.

---

### Edge Cases

- A new “save album” or other disambiguation flow starts after a remove list was shown; the listener returns to an old remove message and acts there. **Text** picks must follow **FR-005** (latest session `kind`); **inline** remains authoritative for a specific `remove_saved` instance.
- Several chat messages may show different remove pick lists, but **storage** only retains the current superseded set; for **stale on-screen** rows with no matching DB row, use **US2** / inline id — not US3 v1 for impossible text disambiguation.
- The listener acts while a remove list is still valid in storage but a newer on-screen list exists (ordering and “most recent” ambiguity).
- Expired or time-limited disambiguation lists: action after expiry must not cause an incorrect removal.
- Picks that are out of range for the targeted list (handled with clear feedback, not by applying to a different list).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST associate each remove-disambiguation inline action with exactly one disambiguation instance (the same instance whose candidate rows were shown for that action).
- **FR-002**: For a remove pick that includes a disambiguation **session id** (e.g. all **inline** `rmp:` actions), the system MUST use **that** disambiguation instance to resolve which saved album to delete, not merely the “newest” such instance for the listener when the id differs. (Typed-only picks: see **FR-005**.)
- **FR-003**: If the disambiguation instance for a **keyed** pick is missing, expired, or no longer valid, the system MUST NOT delete a different saved album in its place; it MUST return a user-visible, non-destructive outcome.
- **FR-004**: Removal MUST still be limited to the acting listener’s own saved albums (no change to the expectation that one user cannot remove another’s saves).
- **FR-005**: For typed numeric picks with **no** session id, the system MUST only complete a remove when the **latest** open disambiguation session for that platform user is `remove_saved` and the index is in range; it MUST NOT apply that index to an album search disambiguation (or any non-`remove_saved` payload). It MUST NOT delete a row that is not a candidate in that `remove_saved` session. A typed message cannot, in v1, target a **superseded** `remove_saved` list that no longer exists in storage; users who need a specific historical list use **inline** actions (US1) or a fresh `/remove`.

### Non-functional and product quality *(should align with project constitution)*

- **UX consistency**: Stale or invalid pick outcomes should use clear, action-oriented copy (what happened and what to do next, e.g. start `/remove` again) consistent with existing bot tone.
- **Testing**: A **session-id (inline) regression** with two `remove_saved` disambiguation contexts in sequence (or a superseding flow) MUST be testable in isolation: the wrong album must never be removed on that path. The **typed** path adds tests for `kind` safety and the single-`remove_saved` case (see **US3**), without requiring two-list-by-text disambiguation that plain chat cannot support.
- **Linting**: No special requirement beyond following repository conventions for new or changed code.
- **Logging and monitoring**: Unusual or rejected picks (e.g. stale disambiguation) MAY be logged for diagnostics; not required to change user-facing behavior.
- **Performance**: The feature is not performance-sensitive; additional lookups to resolve a disambiguation instance by id are acceptable if bounded and indexed as appropriate.
- **Containerization**: No change to containers unless implementation touches runnable entrypoints or configuration.
- **Decommissioning**: None; this tightens existing behavior.

### Key Entities

- **Remove disambiguation instance**: A single, time-bounded “round” in which the listener is offered a concrete ordered list of saved albums to choose from; each offered control must point back to this instance.
- **Pick action**: The user’s completion of a choice (inline or, where supported, a typed index) for a specific instance.
- **Saved album (listener-scoped)**: The row to delete; must match the instance’s candidate and the listener.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In scripted scenarios where an older and a newer **`remove_saved`** disambiguation instance both existed, a **session-bound (inline) pick** intended for the older instance must never remove a candidate that exists only in the newer list (100% of covered test cases for that path). This criterion applies to picks that include the disambiguation session id, not to bare typed text (see **US3** and **FR-005**).
- **SC-002**: In scripted scenarios where a disambiguation list is no longer valid, the user never receives a success message for removing an album that was not the intended label for that (invalid) list without a clear explanation that the choice was stale.
- **SC-003**: Listeners can still complete a normal single-list remove disambiguation in one try when only one such list is open (no regression in the common path).

## Assumptions

- The messaging app continues to support embedding a session reference on inline remove actions; **typed** follow-up is limited as in **FR-005** and **US3** (no session id in text).
- “Superseding” a list means a newer disambiguation instance replaces or invalidates a prior one for the same user flow, as the product already does when starting a new search.
- The scope is the remove-saved-album disambiguation path; other product areas (e.g. album *save* disambiguation) are out of scope unless the same class of bug is reported there separately.
