package core

// User-visible English strings: single product tone; same copy for all adapters in a build.

const productName = "LifeSoundtrack"

var (
	startCopy = "Welcome to " + productName + ". I'm glad you're here — this is a private space for your music journey. " +
		"Use /help to see what I can do, or /ping if you just want a quick liveness check."

	helpCopy = productName + "\n\n" +
		"Here is what I support right now (same in any chat client we connect):\n" +
		"• /start — short welcome and where you are\n" +
		"• /help — this list, with " + productName + " in context\n" +
		"• /ping — a tiny liveness line so you know I'm responding\n"

	pingCopy = "pong — " + productName + " is up."

	unknownCopy = "I did not understand that. Try /help to see the commands I support right now."
)
