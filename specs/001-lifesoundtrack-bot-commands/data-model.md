# Data model: LifeSoundtrack (v1) — domain only

**Spec**: [spec.md](spec.md)  
**Date**: 2026-04-23

## Scope

No **database** and no **persistence** of customers or message history in v1. **State** in this document refers to **in-memory domain types** in `bot/internal/core/`, not SQL tables.

## Domain concepts (platform-neutral)

| Concept | Description | Persisted? |
|---------|-------------|------------|
| **Domain command** | One of the logical actions: `start`, `help`, `ping`, or an **unmapped** / **unknown** user turn | No |
| **Reply text** | Determined entirely in **core** from [contracts/messaging-commands.md](contracts/messaging-commands.md) | No (computed) |
| **Conversation context** | “Private 1:1 with end user” — enforced in **adapters** (e.g. filter non-private in Telegram) | No (per incoming event only) |

## Platform payloads

- **User / chat / message** rows from a vendor (Telegram, etc.) are **not** part of a persistent **data model** in v1; they exist as **adapter** types converted to a minimal `PrivateMessage` (or similar) for **core** if needed, then discarded.

## Validation (conceptual)

- **Adapters** MUST drop or ignore **non–private-1:1** traffic for v1 to match the spec, without changing **core** strings.

## Versioning

Persistence or cross-session state requires a new spec/plan; **out of scope** for `001`.
