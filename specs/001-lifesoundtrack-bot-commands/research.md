# Research: LifeSoundTrack bot (Go, `bot/`) — domain and adapters

**Feature**: [spec.md](spec.md)  
**Date**: 2026-04-23

## 0. Platform-agnostic domain (required by FR-007)

**Decision**: Implement **core product behavior** in `bot/internal/core/`: domain command type (`Start` | `Help` | `Ping` | `Unknown` or similar), all **user-visible strings** (LifeSoundtrack copy), and the rule “unknown → hint to help.” **No** imports from Telegram, Discord, or other vendor SDKs in this package.

**Rationale** (clarification 2026-04-23): The same **domain** must be **reused** for any host platform; only **adapters** change.

**Port (interface) shape** (suggested, exact names are implementation details):

- **Inbound**: normalized event such as `PrivateMessage { Text string }` and optional `IsCommand` / `Command` after adapter parsing.
- **Outbound**: one or more `Reply` strings the adapter sends on the right channel.
- `internal/adapter/telegram` (first milestone) **maps** Telegram updates → core, and core results → `SendMessage` (or library equivalent).

**Alternatives considered**:

| Option | Outcome |
|--------|---------|
| All logic in Telegram handlers | Rejected: violates **FR-007**; hard to add a second platform |
| Generating different copies per platform in v1 | Rejected: spec **SC-002** expects same product copy for the same build |

## 1. First adapter: Telegram — client library

**Decision (Telegram only)**: In **`bot/internal/adapter/telegram`**, use **`github.com/go-telegram/bot`** for long polling, handler registration, and types.

**Rationale**: One dependency **inside the adapter**; `internal/core` stays vendor-free. Same comparison table as the original plan for “why this library in the adapter.”

**Follow-up (non-blocking)**: Second adapter (e.g. another Bot API) as a new subfolder under `internal/adapter/`.

## 2. Long polling vs webhook (Telegram adapter, sandbox)

**Decision**: **Long polling** for the **Telegram** adapter in sandbox and v1.

**Rationale**: No public HTTPS server required for local runs.

**Note**: A future Discord/Slack-style adapter may use **websockets** or **webhooks**; that is **per-adapter** and does not change [contracts/messaging-commands.md](contracts/messaging-commands.md).

## 3. Configuration

**Decision**: Environment variables names may be **adapter-specific** (e.g. `TELEGRAM_BOT_TOKEN` for the Telegram adapter) and a generic `LOG_LEVEL`. Optional `ADAPTER=telegram` (or compile-time default) to select which `main` wires.

**Rationale**: Token shape differs by vendor; keep `.env.example` names accurate for the **shipped** adapter; document in [quickstart.md](quickstart.md).

## 4. Unknown input (domain vs adapter)

**Decision**: `internal/core` returns the **“try help”** text for `Unknown` **once**; the **Telegram** adapter calls `core` when no command pattern matches, then sends the reply. Same text on any other adapter.

## 5. Concurrency and shutdown

**Decision**: `cmd/bot` cancels a root `context` on **SIGINT/SIGTERM**; the active adapter’s run loop should respect that context.

**Rationale**: Shared lifecycle; adapter implements shutdown details.

## 6. Linting and formatting

**Decision**: **`gofmt`**, **`go vet`**, and optional **`golangci-lint`** under `bot/`; enforce **import boundaries** in CI if desired (e.g. `core` must not import `adapter`).

**Rationale**: Prevents **FR-007** regression via accidental vendor imports in `core`.
