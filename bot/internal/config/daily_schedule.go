package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// loadDailyRecommendationsSchedule parses LST_DAILY_RECOMMENDATIONS_* per contracts/feature-flags.md.
// When the feature is disabled, timezone and cron are not validated and location is nil.
func loadDailyRecommendationsSchedule() (enable bool, tzName string, loc *time.Location, cronExpr string, err error) {
	enable = parseMetadataFeatureFlag("LST_DAILY_RECOMMENDATIONS_ENABLE")
	if !enable {
		return false, "", nil, "", nil
	}
	tzName = strings.TrimSpace(os.Getenv("LST_DAILY_RECOMMENDATIONS_TZ"))
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err = time.LoadLocation(tzName)
	if err != nil {
		return false, "", nil, "", fmt.Errorf("LST_DAILY_RECOMMENDATIONS_TZ: %w", err)
	}
	cronExpr = strings.TrimSpace(os.Getenv("LST_DAILY_RECOMMENDATIONS_CRON"))
	if cronExpr == "" {
		cronExpr = "0 6 * * *"
	}
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, perr := p.Parse(cronExpr); perr != nil {
		return false, "", nil, "", fmt.Errorf("LST_DAILY_RECOMMENDATIONS_CRON: %w", perr)
	}
	return true, tzName, loc, cronExpr, nil
}
