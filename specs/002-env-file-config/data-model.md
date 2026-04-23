# Data model: LifeSoundtrack (002) — environment configuration

**Spec**: [spec.md](spec.md)  
**Date**: 2026-04-25

## Scope

No database. This document names **process environment keys** and **file** location as **concepts** for testing and code review—not SQL tables.

## Named configuration (same as feature 001)

| Name | Required | Source | Notes |
|------|----------|--------|--------|
| `TELEGRAM_BOT_TOKEN` | Yes (for running with Telegram) | OS env and/or `bot/.env` via `godotenv` | Never log value |
| `LOG_LEVEL` | No | OS env and/or `bot/.env` | e.g. `INFO`, `DEBUG` |

**Precedence (required by spec)**: if a name exists in the **OS environment**, it **wins** over the value in `.env` for the same name.

## File artifact

| Concept | Path (local dev) | Committed? |
|--------|-------------------|------------|
| `.env` (secrets) | `bot/.env` | **No** (gitignored) |
| Example template | `bot/.env.example` | **Yes** (names only) |

## State transitions (conceptual)

1. **Process start** → (optional) load `.env` into empty env keys → read `config.FromEnv()` from `os` package.
2. **Air file change** → kill child process → rebuild → new process → step 1 again (developer loop only).

## Versioning

Adding new keys: amend **.env.example** + this doc + [contracts/env-loading.md](contracts/env-loading.md) in the same change as code.
