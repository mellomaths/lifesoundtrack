# Contract: Environment bootstrap and `.env` file (v1)

**Spec**: [spec.md](../spec.md)  
**Date**: 2026-04-25

## Purpose

Behavioral contract for **how** the process obtains `TELEGRAM_BOT_TOKEN` and `LOG_LEVEL` before the Telegram adapter runs. This is an **ops/bootstrap** contract, not a user-visible messaging contract (see [001 messaging-commands.md](../../001-lifesoundtrack-bot-commands/contracts/messaging-commands.md) for product copy).

## Load order (MUST)

1. **Optionally** apply key-value pairs from a **local `.env` file** at the **documented path** (see [quickstart.md](../quickstart.md)), using the chosen OSS loader such that **existing** OS environment variables for the same key are **not** replaced.
2. **Read** final values from the **os** environment in application config (`config.FromEnv` or successor).

**MUST NOT**: Log full `.env` file contents, or `TELEGRAM_BOT_TOKEN`, on success or on failure.

## Precedence (MUST)

For any key `K` present in both **OS environment** and **`.env` file**, the **OS environment** value of `K` is the **effective** value.

## File presence (MUST)

- If **`.env` is missing** and **all required** keys exist in the OS environment only, **startup MUST succeed** (per **FR-003**).
- If **required** keys are **missing** after step 1 and 2, **startup MUST fail** with a **non-leaking** error (per **FR-004**).

## Invalid or unreadable `.env` (MUST for v1)

- If the file **exists** but the **OSS loader** reports a **parse** error (file is not valid per that library’s rules), **startup MUST fail** with a **generic** error string that does **not** include the full file body or token values (**FR-004**). *This matches [spec.md](../spec.md) US2 — v1 does **not** require “skip bad lines and continue.”*
- **Comment lines** (`#` …), normal **line endings** (LF/CRLF), and **whitespace trimming** around keys/values follow the **library’s** behavior; tests **MAY** pin that behavior via `testdata` fixtures rather than re-specifying grammar here.
- If the OS reports **permission denied** or another open error, the process **MUST NOT** log file **contents**; a **generic** failure is acceptable for v1 (see [spec.md](../spec.md) Edge cases).

## Local development reload (MUST, dev only)

A **separate** documented command (e.g. **air**) **MAY** watch **`.go` files** and **`.env`** and restart the built binary; this behavior is **out of** scope for container **CMD** in default production images.

## Acceptance

Maps to [spec.md](../spec.md) **FR-001**–**FR-008** and **SC-001**–**SC-004** where applicable.
