# Command: `/album`

## Summary

Saves an album the user wants to listen to: **album title** and **artist**, linked to their internal **LifeSoundtrack** user (resolved via Telegram **`Message.From`**).

## Syntax

```
/album <album title> - <artist>
```

Use the **first** occurrence of **` - `** (space, hyphen, space) to separate **title** (left) from **artist** (right). Titles or artist names containing that exact substring cannot be represented without ambiguity in v1.

Telegram passes the remainder of the message as [command arguments](https://core.telegram.org/bots/api#message).

## Identity and storage

- **Canonical user key:** **`Message.From.ID`** (Telegram user id), not **`chat_id`** alone.
- Rows are keyed by **`users.id`** internally. Telegram id is stored under **`user_identities`** with **`source`** `telegram` and **`external_id`** equal to **`From.ID`** as a decimal string.
- **`users.name`** is updated when **`From.FirstName` / `From.LastName`** yield a non-empty display name after trim/join.
- **`user_identities.username`** holds the Telegram **`@username`** **without** the **`@`** when present.

## Responses

| Scenario | Reply |
|----------|--------|
| Valid parse and DB success | Short confirmation including album title and artist (non-empty). |
| Missing arguments or missing ` - ` delimiter | Brief usage hint; no DB write. |
| Empty title or artist after trim | Brief error; no DB write. |
| **`Message.From` missing** (anonymous / edge cases) | Brief error; no DB write. |
| Database error | Generic failure message; **no** stack traces or connection strings to the user. |

## Given / When / Then

| # | Given | When | Then |
|---|-------|------|------|
| 1 | Private message `/album Abbey Road - The Beatles`, **`From`** present | Handler runs | Interest saved; reply confirms (mentions album/artist). |
| 2 | Same as 1 with `/album@BotName` suffix on command | Handler runs | Same as row 1. |
| 3 | `/album` with no arguments | Handler runs | Usage error reply; no DB write. |
| 4 | `/album Foo` (no ` - `) | Handler runs | Usage error reply; no DB write. |
| 5 | Update has **no** `From` | Handler runs | Error reply; no DB write. |

## Idempotency

Saving the same album multiple times creates **multiple** rows unless a future dedupe rule is added.
