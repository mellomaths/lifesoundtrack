# Command: `/help`

## Summary

Lists available bot commands and how to use **`/album`**.

## Triggers

- User sends **`/help`** (optional `@BotUsername` suffix).

## Responses

| Scenario | Reply |
|----------|--------|
| Private chat with text command | Multi-line help including **`/start`**, **`/help`**, and **`/album … - …`** usage. |
| Cannot build a reply | No panic; errors handled by caller / logged. |

## Given / When / Then

| # | Given | When | Then |
|---|-------|------|------|
| 1 | `/help` in private chat | Handler runs | Reply mentions **`LifeSoundtrack`** or core commands and includes **`/album`** with **` - `** delimiter hint. |
| 2 | `/help@OtherBot` | Handler runs | Same intent as row 1. |
