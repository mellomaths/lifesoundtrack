package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mellomaths/lifesoundtrack/bot/internal/config"
)

func TestLoadLocalDotEnv_MissingFileIsOK(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })
	if err := config.LoadLocalDotEnv(); err != nil {
		t.Fatalf("no .env: %v", err)
	}
}

func TestLoadLocalDotEnv_ValidFileSetsVariables(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })

	contents := "TELEGRAM_BOT_TOKEN=fromfile\nDATABASE_URL=postgres://l:p@h:1/db?sslmode=disable\nLOG_LEVEL=DEBUG\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("LOG_LEVEL")
	})

	if err := config.LoadLocalDotEnv(); err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatalf("from env: %v", err)
	}
	if cfg.TelegramBotToken != "fromfile" {
		t.Fatalf("token: want fromfile, got %q", cfg.TelegramBotToken)
	}
	// config.parseLogLevel returns INFO when unknown — DEBUG is set.
	// (If this fails, adjust expectation to match parseLogLevel.)
	if got := cfg.LogLevel.String(); got != "DEBUG" {
		t.Fatalf("log level: want DEBUG, got %s", got)
	}
}

func TestLoadLocalDotEnv_OSEnvironmentTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })

	// FR-002 (002 spec): .env is loaded first in main, then we read.
	// godotenv does not override existing os.Getenv keys by default.
	// We set token in env before load; file has different token; env wins.
	t.Setenv("TELEGRAM_BOT_TOKEN", "from_os")
	t.Setenv("DATABASE_URL", "postgres://l:p@h:1/db?sslmode=disable")
	t.Cleanup(func() {
		_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
		_ = os.Unsetenv("DATABASE_URL")
	})

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("TELEGRAM_BOT_TOKEN=fromfile\nDATABASE_URL=postgres://a:b@h:1/d?sslmode=disable\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := config.LoadLocalDotEnv(); err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatalf("from env: %v", err)
	}
	if cfg.TelegramBotToken != "from_os" {
		t.Fatalf("precedence: want from_os, got %q", cfg.TelegramBotToken)
	}
}

func TestLoadLocalDotEnv_ParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })

	// Unmarshal rejects this in godotenv v1.5.x; load must fail without silently ignoring.
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("lol$wut\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadLocalDotEnv(); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFromEnv_RequiresToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })
	// Inherit a clean state: any token in the parent would satisfy FromEnv.
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("DATABASE_URL", "postgres://l:p@h:1/db?sslmode=disable")
	t.Cleanup(func() { _ = os.Unsetenv("DATABASE_URL") })

	_, err := config.FromEnv()
	if err == nil {
		t.Fatal("expected error without token")
	}
	if !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN") {
		t.Fatalf("error should name the key: %v", err)
	}
}

func TestLoadLocalDotEnv_NoFile_TokenFromOSEnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })
	t.Setenv("TELEGRAM_BOT_TOKEN", "os_only")
	t.Setenv("DATABASE_URL", "postgres://l:p@h:1/db?sslmode=disable")
	t.Cleanup(func() {
		_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
		_ = os.Unsetenv("DATABASE_URL")
	})

	if err := config.LoadLocalDotEnv(); err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatalf("from env: %v", err)
	}
	if cfg.TelegramBotToken != "os_only" {
		t.Fatalf("want os_only, got %q", cfg.TelegramBotToken)
	}
}

func TestLoadLocalDotEnv_CommentAndCRLF(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })
	t.Cleanup(func() {
		_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("LOG_LEVEL")
	})
	// CRLF + # line (library behavior: token still loads)
	contents := "# comment\r\nTELEGRAM_BOT_TOKEN=from_crlf\r\nDATABASE_URL=postgres://l:p@h:1/db?sslmode=disable\r\nLOG_LEVEL=INFO\r\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadLocalDotEnv(); err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg, err := config.FromEnv()
	if err != nil {
		t.Fatalf("from env: %v", err)
	}
	if cfg.TelegramBotToken != "from_crlf" {
		t.Fatalf("token: %q", cfg.TelegramBotToken)
	}
}
