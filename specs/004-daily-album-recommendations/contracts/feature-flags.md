# Contract: environment feature flags (daily recommendations)

**Plan**: [plan.md](../plan.md) | **Config**: `bot/internal/config` (intended)

## Variables

| Variable | Required | Default | Semantics |
|----------|----------|---------|-----------|
| `LST_DAILY_RECOMMENDATIONS_ENABLE` | No | **on** (opt-out) | Same parsing as **`LST_METADATA_ENABLE_*`**: unset/empty/non-falsey → **enabled**; **`0`**, **`false`**, **`no`**, **`off`** (case-insensitive after trim) → **disabled**. When disabled, **do not** register the daily cron job. |
| `LST_DAILY_RECOMMENDATIONS_TZ` | No | `UTC` | IANA timezone, e.g. `America/Sao_Paulo`. Invalid TZ → **fail startup** with clear log (preferred over silent UTC). |
| `LST_DAILY_RECOMMENDATIONS_CRON` | No | `0 6 * * *` | Five-field cron, interpreted in **`LST_DAILY_RECOMMENDATIONS_TZ`**. Invalid expression → **fail startup**. |

## Startup logging

- **INFO**: whether daily recommendations are **enabled**, resolved **TZ**, and **cron** string (no secrets).

## Non-goals (v1)

- Remote config / dynamic flags without restart.
