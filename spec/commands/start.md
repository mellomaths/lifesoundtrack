# Command: `/start`

## Summary

Greets the user and confirms the bot is running. Entry point for new chats.

## Triggers

- User sends a message whose text is exactly `/start` (optionally with `@BotUsername` suffix as normalized by Telegram clients).

## Responses

| Scenario | Reply |
|----------|--------|
| Normal private message | Welcome text including the project name **LifeSoundtrack**. |
| Bot cannot reply (e.g. missing chat) | No panic; failure handled by caller / logged. |

## Given / When / Then

| # | Given | When | Then |
|---|-------|------|------|
| 1 | Private message with text `/start` | Handler runs | Reply text contains `LifeSoundtrack` and reads as a short welcome (non-empty). |
| 2 | Private message with text `/start@OtherBot` | Handler runs | Same welcome behavior as row 1 (suffix ignored for routing if command is `/start`). |
| 3 | Update has no message / no chat | Handler runs | Returns error or no-op without sending (implementation-defined but must not panic). |

## Idempotency

Sending `/start` multiple times is allowed; each time returns the same style of welcome message.
