package runner

import (
	"context"
	"fmt"
	"time"

	"MarketNews/internal/app"
	"MarketNews/internal/config"
)

func RunSync(ctx context.Context, cfg config.Config, scrapeService *app.ScrapeService) error {
	start := cfg.SyncStartDate
	if start == nil {
		fallback := dateOnlyUTC(time.Now().UTC()).AddDate(0, 0, -7)
		start = &fallback
	}

	yesterday := dateOnlyUTC(time.Now().UTC()).AddDate(0, 0, -1)
	if start.After(yesterday) {
		start = &yesterday
	}

	syncCfg := app.SyncConfig{
		StartDate:     *start,
		DeltasMinutes: cfg.DeltasMinutes,
	}
	return scrapeService.RunSync(ctx, syncCfg)
}

func RunReimport(ctx context.Context, cfg config.Config, maintenance *app.MaintenanceService, scrapeService *app.ScrapeService) error {
	now := time.Now().UTC()
	startYear := now.Year() - cfg.ReimportYears
	start := time.Date(startYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := dateOnlyUTC(now).AddDate(0, 0, -1)
	if err := maintenance.TruncateAll(ctx); err != nil {
		return fmt.Errorf("truncate all: %w", err)
	}
	scrapeCfg := app.ScrapeConfig{
		StartDate:     start,
		EndDate:       end,
		DeltasMinutes: cfg.DeltasMinutes,
	}
	return scrapeService.RunScrape(ctx, scrapeCfg)
}

func dateOnlyUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
