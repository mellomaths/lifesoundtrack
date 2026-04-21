// Package configs loads and validates bot environment configuration.
package configs

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Transport selects how Telegram updates are received.
type Transport string

const (
	TransportPolling Transport = "polling"
	TransportWebhook Transport = "webhook"
)

// Config holds runtime settings from the environment.
type Config struct {
	Token                      string
	DatabaseURL                string
	Transport                  Transport
	ListenAddr                 string
	WebhookBaseURL             string
	WebhookPath                string
	WebhookSecretToken         string
	DailyRecommendationEnabled bool
	DailyRecommendationHourUTC int
}

// Load reads and validates configuration from environment variables.
func Load() (*Config, error) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	raw := strings.ToLower(strings.TrimSpace(os.Getenv("TRANSPORT")))
	var tr Transport
	switch raw {
	case "polling":
		tr = TransportPolling
	case "webhook":
		tr = TransportWebhook
	default:
		return nil, fmt.Errorf("TRANSPORT must be polling or webhook")
	}

	cfg := &Config{
		Token:              token,
		DatabaseURL:        dbURL,
		Transport:          tr,
		ListenAddr:         strings.TrimSpace(os.Getenv("LISTEN_ADDR")),
		WebhookBaseURL:     strings.TrimSpace(os.Getenv("WEBHOOK_URL")),
		WebhookPath:        strings.TrimSpace(os.Getenv("WEBHOOK_PATH")),
		WebhookSecretToken: strings.TrimSpace(os.Getenv("WEBHOOK_SECRET_TOKEN")),
	}

	if tr == TransportWebhook {
		if cfg.ListenAddr == "" {
			return nil, fmt.Errorf("LISTEN_ADDR is required when TRANSPORT=webhook")
		}
		if cfg.WebhookBaseURL == "" {
			return nil, fmt.Errorf("WEBHOOK_URL is required when TRANSPORT=webhook")
		}
		if cfg.WebhookPath == "" {
			return nil, fmt.Errorf("WEBHOOK_PATH is required when TRANSPORT=webhook")
		}
		if !strings.HasPrefix(cfg.WebhookPath, "/") {
			return nil, fmt.Errorf("WEBHOOK_PATH must start with /")
		}
	}

	dailyEnabled := true
	if v := strings.TrimSpace(os.Getenv("DAILY_RECOMMENDATION_ENABLED")); v != "" {
		switch strings.ToLower(v) {
		case "false", "0", "no", "off":
			dailyEnabled = false
		case "true", "1", "yes", "on":
			dailyEnabled = true
		default:
			return nil, fmt.Errorf("DAILY_RECOMMENDATION_ENABLED must be true or false")
		}
	}
	dailyHour := 9
	if s := strings.TrimSpace(os.Getenv("DAILY_RECOMMENDATION_HOUR_UTC")); s != "" {
		h, err := strconv.Atoi(s)
		if err != nil || h < 0 || h > 23 {
			return nil, fmt.Errorf("DAILY_RECOMMENDATION_HOUR_UTC must be an integer 0-23")
		}
		dailyHour = h
	}
	cfg.DailyRecommendationEnabled = dailyEnabled
	cfg.DailyRecommendationHourUTC = dailyHour

	return cfg, nil
}

// WebhookFullURL joins WEBHOOK_URL and WEBHOOK_PATH for setWebhook.
func (c *Config) WebhookFullURL() (string, error) {
	base := strings.TrimRight(c.WebhookBaseURL, "/")
	path := c.WebhookPath
	u, err := url.Parse(base + path)
	if err != nil {
		return "", fmt.Errorf("parse webhook url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("WEBHOOK_URL must use http or https scheme")
	}
	return u.String(), nil
}
