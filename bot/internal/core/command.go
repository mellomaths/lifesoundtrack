package core

// Command is a platform-neutral domain action for private chat in v1.
type Command int

const (
	CommandStart Command = iota
	CommandHelp
	CommandPing
	CommandUnknown
)

// String returns a stable name for logging and tracing (not user-visible text).
func (c Command) String() string {
	switch c {
	case CommandStart:
		return "start"
	case CommandHelp:
		return "help"
	case CommandPing:
		return "ping"
	case CommandUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}
