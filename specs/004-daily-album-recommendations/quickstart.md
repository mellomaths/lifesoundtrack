# Quickstart: daily recommendations (004)

**Plan**: [plan.md](plan.md)

## Prerequisites

- Postgres (repo root `compose.yaml`).
- `TELEGRAM_BOT_TOKEN`, `DATABASE_URL`; migrations include **`last_recommended_at`** and **`recommendations`** after implementation.

## Environment

Add to `bot/.env` (and `bot/.env.example` when updated):

```bash
# Daily recommendations — opt-out master switch (unset = on)
# LST_DAILY_RECOMMENDATIONS_ENABLE=true
# LST_DAILY_RECOMMENDATIONS_TZ=UTC
# LST_DAILY_RECOMMENDATIONS_CRON=0 6 * * *
```

Set **`LST_DAILY_RECOMMENDATIONS_ENABLE=false`** to disable the job entirely (**SC-006**).

## Verify (post-implementation)

1. Seed listeners + `saved_albums` (mix of null and non-null `last_recommended_at`).
2. On startup with the feature **on**, confirm **INFO** logs include `daily_recommendations_config` (enabled, **tz**, **cron**) per **FR-018**; after a scheduled time (or a tightened **`LST_DAILY_RECOMMENDATIONS_CRON`** for dev only), confirm **`daily_recommendations_cron_tick`** / **`daily_recommendations_listeners`** appear (**SC-008**, **FR-017**).
3. Trigger at least one run with the feature **on** and confirm **listener discovery** completes (no batch abort with PostgreSQL **`42P10`** / distinct+order error) and processing continues per listener (**SC-007**, **FR-013**).
4. **Dev only**: optionally tighten cron for a test window or add a test-only manual trigger if the implementation provides one.
5. Confirm Telegram: cover when `art_url` set; caption + optional button; sign-off line.
6. Confirm DB: after successful send, `last_recommended_at` and `recommendations` row share one transaction; failed send → no change.
7. With enable flag **false**, confirm no scheduled ticks and no sends over a would-have-fired window.

## Release / UAT (**SC-004**, **SC-005**)

These steps are **outside** the default `go test` loop; run before major releases or as a product gate.

| Criterion | What to do |
|-----------|------------|
| **SC-004** | UAT: sample of eligible users with Spotify URL on the pick; verify ≥ **95%** reach a working album page via button or caption link; file QA sign-off. |
| **SC-005** | Soak: ≥ **two weeks** pre-prod; dashboard or logs show **at most one** successful send per user per daily run; document any anomaly. |

See [plan.md](plan.md) § Release and UAT gates.

## References

- [contracts/feature-flags.md](contracts/feature-flags.md)
- [contracts/daily-recommendations-job.md](contracts/daily-recommendations-job.md)
- [data-model.md](data-model.md)
