package configs

import "testing"

func TestConfig_WebhookFullURL(t *testing.T) {
	t.Parallel()
	c := &Config{
		WebhookBaseURL: "https://example.com",
		WebhookPath:    "/telegram/webhook",
	}
	got, err := c.WebhookFullURL()
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com/telegram/webhook"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestConfig_WebhookFullURL_trimsBaseSlash(t *testing.T) {
	t.Parallel()
	c := &Config{
		WebhookBaseURL: "https://example.com/",
		WebhookPath:    "/hook",
	}
	got, err := c.WebhookFullURL()
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.com/hook"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
