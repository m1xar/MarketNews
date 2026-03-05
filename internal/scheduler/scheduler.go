package scheduler

import (
	"context"
	"log"
	"time"

	"MarketNews/internal/app"
	"MarketNews/internal/config"
	"MarketNews/internal/runner"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
}

func Start(cfg config.Config, scrapeService *app.ScrapeService, upcomingService *app.UpcomingAnalysisService) (*Scheduler, error) {
	if !cfg.CronEnabled {
		return nil, nil
	}
	c := cron.New(cron.WithLocation(time.UTC))
	_, err := c.AddFunc(cfg.CronSpec, func() {
		ctx := context.Background()
		weeklyCfg := cfg
		start := dateOnlyUTC(time.Now().UTC()).AddDate(0, 0, -7)
		weeklyCfg.SyncStartDate = &start
		if err := runner.RunSync(ctx, weeklyCfg, scrapeService); err != nil {
		}
		if cfg.UpcomingEnabled && upcomingService != nil {
			windowStart := dateOnlyUTC(time.Now().UTC())
			windowEnd := windowStart.AddDate(0, 0, cfg.UpcomingWindowDays).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			if err := upcomingService.TruncateUpcomingAnalysis(ctx); err != nil {
			}
			if err := upcomingService.GenerateAnalysesByRange(ctx, windowStart, windowEnd); err != nil {
			}
		}
	})
	if err != nil {
		return nil, err
	}
	c.Start()
	log.Printf("cron started | spec=%s | upcoming_enabled=%t", cfg.CronSpec, cfg.UpcomingEnabled)
	return &Scheduler{cron: c}, nil
}

func (s *Scheduler) Stop() {
	if s == nil || s.cron == nil {
		return
	}
	s.cron.Stop()
}

func dateOnlyUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
