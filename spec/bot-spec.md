# LifeSoundtrack Telegram bot — product spec

## Purpose

The bot is the Telegram interface for **LifeSoundtrack** (“the soundtrack for your life”). Commands are documented under [`commands/`](commands/).

## Goals

- Respond to Telegram updates reliably using configurable **long polling** or **webhook**.
- Keep secrets (bot token, webhook secret, database URL) out of source control and logs.
- Add new behavior by updating specs first, then tests, then code.

## Non-goals

- Inline keyboards, payments, or group administration beyond simple command replies.
- Integrating external music catalog APIs for metadata (beyond storing user-entered album/artist strings).

## Data persistence

The bot stores data in **PostgreSQL**:

| Table | Purpose |
|-------|---------|
| **`users`** | Internal profiles; primary key **`id`** is the only key used by **`album_interests`**. Optional **`name`** (display name). |
| **`user_identities`** | Maps external accounts to **`users.id`**: **`source`** (e.g. `telegram`), **`external_id`** (provider user id as text), optional **`username`** (e.g. Telegram handle without `@`). Unique **`(source, external_id)`**. |
| **`album_interests`** | Rows keyed by **`user_id`** → **`users.id`**; **`album_title`**, **`artist`**, optional **`last_recommended_at`** (when last sent as daily pick). Telegram ids are **never** stored as FKs here. |
| **`recommendation_audit`** | Append-only log of each daily recommendation sent: **`user_id`**, optional **`album_interest_id`**, title/artist **snapshots**, **`recommended_at`**. |

Retention: indefinite until a future delete/list feature exists.

### Daily recommendations

At **`DAILY_RECOMMENDATION_HOUR_UTC`** (default **9**), the bot sends one fair-random album per user with saved interests (see [`commands/daily-recommendation.md`](commands/daily-recommendation.md)). Disable with **`DAILY_RECOMMENDATION_ENABLED=false`**.

Domain tables use **internal ids only**. Telegram user ids appear only on **`user_identities`** together with **`source`** `telegram`.

## Privacy

- **Purpose limitation:** Persist **Telegram-derived profile fields** (**`users.name`**, **`user_identities.username`**) and **`album_interests`** only to operate `/album` and related features.
- Do not log raw tokens, **`DATABASE_URL`** credentials, or unrelated message bulk content at info level.

## Command registry

| Command | Status | Spec |
|---------|--------|------|
| `/start` | Implemented | [`commands/start.md`](commands/start.md) |
| `/help` | Implemented | [`commands/help.md`](commands/help.md) |
| `/album` | Implemented | [`commands/album.md`](commands/album.md) |
| `/ping` | **TBD** | — |

## Operational configuration

Environment variables (see also [`bot/.env.example`](../bot/.env.example)):

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Token from [@BotFather](https://t.me/BotFather). |
| `DATABASE_URL` | Yes | PostgreSQL URL (local dev typically via Docker Compose; see README). |
| `TRANSPORT` | Yes | `polling` or `webhook`. |
| `LISTEN_ADDR` | Webhook only | HTTP listen address (e.g. `:8080`). |
| `WEBHOOK_URL` | Webhook only | Public base URL used with `setWebhook` (must be `https` in production). |
| `WEBHOOK_PATH` | Webhook only | URL path for Telegram POSTs (e.g. `/telegram/webhook`). |
| `WEBHOOK_SECRET_TOKEN` | No | If set, sent to Telegram as `secret_token` and validated on incoming requests via `X-Telegram-Bot-Api-Secret-Token`. |
| `DAILY_RECOMMENDATION_ENABLED` | No | If `false`, disables the daily recommendation scheduler (default on). |
| `DAILY_RECOMMENDATION_HOUR_UTC` | No | Hour **0–23** (UTC) for the daily send (default **9**). |

Polling mode does not require `LISTEN_ADDR`, `WEBHOOK_URL`, or `WEBHOOK_PATH`.

## Health checks

When `TRANSPORT=webhook`, an HTTP server exposes **`GET /health`** on `LISTEN_ADDR` for liveness probes. Telegram updates are **`POST`** to `WEBHOOK_PATH`.
