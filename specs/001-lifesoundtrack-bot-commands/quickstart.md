# Quickstart: LifeSoundtrack (sandbox) — first adapter (Telegram)

**Plan**: [plan.md](plan.md)  
**Date**: 2026-04-23

The **product rules** in [spec.md](spec.md) and [contracts/messaging-commands.md](contracts/messaging-commands.md) are **platform-agnostic**. This runbook uses the **first** supported adapter: **Telegram** (private chat, slash-style commands in the app). A different adapter would have different setup steps; **domain** behavior (start / help / ping) stays the same.

## Prerequisites

- **Go 1.22+**
- A **Telegram** bot token from [@BotFather](https://t.me/BotFather) (dedicated **sandbox** token, not production)
- (Optional) **Docker** and **Docker Compose**

## 1. Configure environment

Create **`bot/.env`** (or export in your shell). **Do not commit secrets.** For **load order**, **`.env` location**, and optional **air** hot reload, see [002 — env & dev reload](../002-env-file-config/quickstart.md).

```bash
# Required for the Telegram adapter; see [bot/.env.example](../../bot/.env.example)
export TELEGRAM_BOT_TOKEN="your-sandbox-token"
# Optional: DEBUG, INFO (default), WARN, ERROR
export LOG_LEVEL=INFO
```

**Never** log or paste the token.

## 2. Run locally (no Docker)

```bash
cd bot
go run ./cmd/bot
```

- Stop with **Ctrl+C** (clean shutdown; see [research.md](research.md)).
- In **Telegram**, open a **private** chat with your bot: **`/start`**, then **`/help`**, then **`/ping`**.

**Unit-test the domain without Telegram:**

```bash
go test ./internal/core/... -count=1
```

## 3. Run with Docker (`bot/Dockerfile` + root `compose.yaml`)

From the **repository root** (where [compose.yaml](../../compose.yaml) lives):

```bash
docker compose build bot
docker compose up bot
```

Set `TELEGRAM_BOT_TOKEN` in the environment (e.g. `export` before `up`) or use a non-committed env file; **never** commit tokens.

## 4. Runbook (acceptance)

1. **Environment ready** — token set, process up, no startup errors in logs.
2. **`/start`** — welcome includes **LifeSoundtrack** (domain string from `internal/core`).
3. **`/help`** — all three command labels and LifeSoundtrack context.
4. **`/ping`** — short reply within a few seconds.

For **SC-004** (“easy for a new team member”), run the above once in your **sandbox** and record a yes/no plus a short note outside this repo if needed.

## 5. Quality gates

```bash
cd bot
go test ./...
go vet ./...
```

Optional: `golangci-lint run ./...` if configured. Verify **`internal/core`** has **no** imports of `github.com/go-telegram` (or any adapter path).

## 6. Documentation alignment

Update root [**README.md**](../../README.md) when `bot/` ships: describe **`internal/core` + `internal/adapter/...`** and link here (per constitution **IX**).
