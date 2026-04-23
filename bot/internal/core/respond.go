package core

import (
	"strings"
)

// ParseTextMessage maps a private chat text line to a domain [Command]. Lines without a
// recognized /command token are [CommandUnknown]. Leading slash commands may include
// @botname; only the base command name is used.
func ParseTextMessage(text string) Command {
	text = strings.TrimSpace(text)
	if text == "" {
		return CommandUnknown
	}
	first := strings.Fields(text)[0]
	if !strings.HasPrefix(first, "/") {
		return CommandUnknown
	}
	name := strings.TrimPrefix(first, "/")
	if i := strings.Index(name, "@"); i >= 0 {
		name = name[:i]
	}
	switch strings.ToLower(name) {
	case "start":
		return CommandStart
	case "help":
		return CommandHelp
	case "ping":
		return CommandPing
	default:
		return CommandUnknown
	}
}

// Reply returns user-visible copy for the command; [CommandUnknown] is the help hint per
// [contracts/messaging-commands.md] fallback.
func Reply(cmd Command) string {
	switch cmd {
	case CommandStart:
		return startCopy
	case CommandHelp:
		return helpCopy
	case CommandPing:
		return pingCopy
	case CommandUnknown:
		return unknownCopy
	default:
		return unknownCopy
	}
}
