# Feature: daily album recommendation

## Summary

On a fixed **UTC schedule**, the bot sends each eligible Telegram user **one** album recommendation from their saved **`album_interests`**, using **fair rotation** (never-recommended and longest-since-recommended first) and **uniform random** choice within that tier. Each successful send updates **`album_interests.last_recommended_at`** and appends a row to **`recommendation_audit`** (title/artist snapshots) in **one transaction** after Telegram accepts the message.

## Eligibility

- User has at least one **`album_interests`** row and a **`user_identities`** row with **`source`** `telegram`.
- Outbound chat uses **`external_id`** as the Telegram user id (private chat).

## Selection rules

1. If any album for the user has **`last_recommended_at IS NULL`**, the candidate pool is exactly those rows.
2. Otherwise the pool is rows whose **`last_recommended_at`** equals **`MIN(last_recommended_at)`** for that user.
3. Pick **uniformly at random** one row from the pool (`ORDER BY random() LIMIT 1`).

## Message

User receives a short line including album title and artist (see implementation). Failures to send do **not** update the database.

## Audit

Every recorded send inserts into **`recommendation_audit`** with **`album_title`** / **`artist`** snapshots and **`recommended_at`**.

## Failure handling

List/pick/send/record errors are logged; one user’s failure does not stop others.
