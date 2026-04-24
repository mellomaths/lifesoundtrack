# Contract: LifeSoundtrack — core messaging commands (v1, private conversation)

**Spec**: [spec.md](../spec.md)  
**Date**: 2026-04-23

This is the **behavioral contract** for **domain** behavior. It is **not** bound to a single vendor (e.g. Telegram, WhatsApp, Discord). A **platform adapter** maps each host’s **triggers** (slash commands, buttons, “get started,” etc.) onto the same **command names** and outcomes below.

## Environment

- **Conversation**: private, one end user and the **LifeSoundtrack** assistant.
- **Language (user-visible)**: English.
- **Product name in copy**: **LifeSoundtrack** (same spelling in **start**-equivalent and **help**-equivalent responses).

## Domain commands (platform-neutral names)

| Domain command | User intent | User-visible outcome (summary) | Must include |
|----------------|-------------|----------------------------------|--------------|
| **start** | First contact / (re)open session | One welcome message | The **LifeSoundtrack** name (or a documented short form) and a friendly welcome |
| **help** | List supported actions | List **start**, **help**, and **ping** with a one-line description each | **LifeSoundtrack** in context; plain language |
| **ping** | Liveness check | One short line (e.g. “pong”-style) | No secrets; minimal latency feel |

**Adapter note**: The **host** may show these as `/start`, `/help`, `/ping` or with different prefixes or UI. The **domain** must still recognize the three behaviors above through the adapter mapping.

**Amendment (003 / save album)**: The product adds domain **`save_album`** (free-form text query), mapped in Telegram to **`/album <query>`** and related pick flows. See [specs/003-save-album-command/contracts/album-command.md](../../003-save-album-command/contracts/album-command.md) for outcomes and host mapping. **Implementation (2026-04-24)**: album metadata is resolved in order **Spotify → iTunes → Last.fm → MusicBrainz**; each source is toggled with **`LST_METADATA_ENABLE_*`** (see [003 quickstart](../../003-save-album-command/quickstart.md)). v1 of **this** 001 table remains about **start / help / ping**; the 003 spec is the source of truth for the album feature.

## Fallback (unknown input)

When the user’s message does not map to a known domain command, send a **short hint to use help** (or the platform-appropriate “help” action), if the host allows a reply in that context.

## Out of scope (v1)

- Group, channel, or non-private contexts (unless a future spec amends this).
- Non-English copy.
- Persistent storage of users or messages.

## Acceptance

Maps to [spec.md](../spec.md) **FR-001** through **FR-007** and user stories 1–4. Platform SDK details are **out of** this contract.
