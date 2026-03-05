package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"MarketNews/internal/bootstrap"
	"MarketNews/internal/config"
	"MarketNews/internal/logging"
	"MarketNews/internal/runner"
	"MarketNews/internal/scheduler"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		return
	}

	logFile, err := logging.Setup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "log setup failed: %v\n", err)
		return
	}
	defer logFile.Close()

	app, err := bootstrap.Build(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed: %v\n", err)
		return
	}
	defer app.DB.Close()

	logConfig(cfg)
	logImplementations(app)

	switch cfg.Mode {
	case config.ModeSync:
		if err := runner.RunSync(ctx, cfg, app.ScrapeService); err != nil {
			fmt.Fprintf(os.Stderr, "sync failed: %v\n", err)
		}
	case config.ModeReimport:
		if err := runner.RunReimport(ctx, cfg, app.Maintenance, app.ScrapeService); err != nil {
			fmt.Fprintf(os.Stderr, "reimport failed: %v\n", err)
		}
	}

	jobScheduler, err := scheduler.Start(cfg, app.ScrapeService, app.UpcomingSvc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cron schedule failed: %v\n", err)
		return
	}
	if jobScheduler != nil {
		defer jobScheduler.Stop()
	}

	if cfg.UpcomingManualImport && cfg.UpcomingEnabled && app.UpcomingSvc != nil {
		windowStart := dateOnlyUTC(time.Now().UTC())
		windowEnd := windowStart.AddDate(0, 0, cfg.UpcomingWindowDays).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		if err := app.UpcomingSvc.TruncateUpcomingAnalysis(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "upcoming analysis truncate failed: %v\n", err)
		}
		if err := app.UpcomingSvc.GenerateAnalysesByRange(ctx, windowStart, windowEnd); err != nil {
			fmt.Fprintf(os.Stderr, "upcoming analysis failed: %v\n", err)
		}
	}

	log.Printf("api server starting | addr=%s | mode=%s", cfg.HTTPAddr, cfg.Mode)
	if err := app.API.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "api server failed: %v\n", err)
	}

}

func dateOnlyUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func logConfig(cfg config.Config) {
	log.Printf(
		"config loaded | mode=%s | http_addr=%s | deltas=%v | cron_enabled=%t | cron_spec=%s | sync_start=%s | reimport_years=%d | upcoming_enabled=%t | upcoming_days=%d | upcoming_manual_import=%t | openai_model=%s | openai_base_url=%s | openai_max_output_tokens=%d | openai_api_key_set=%t",
		cfg.Mode,
		cfg.HTTPAddr,
		cfg.DeltasMinutes,
		cfg.CronEnabled,
		cfg.CronSpec,
		fmtTime(cfg.SyncStartDate),
		cfg.ReimportYears,
		cfg.UpcomingEnabled,
		cfg.UpcomingWindowDays,
		cfg.UpcomingManualImport,
		cfg.OpenAIModel,
		cfg.OpenAIBaseURL,
		cfg.OpenAIMaxOutputTokens,
		cfg.OpenAIAPIKey != "",
	)
}

func logImplementations(app *bootstrap.App) {
	log.Printf(
		"impls | db=%t | api=%t | event_repo=%t | upcoming_repo=%t | trade_repo=%t | scrape_service=%t | maintenance=%t | upcoming_service=%t | trade_service=%t",
		app.DB != nil,
		app.API != nil,
		app.EventRepo != nil,
		app.UpcomingRepo != nil,
		app.TradeRepo != nil,
		app.ScrapeService != nil,
		app.Maintenance != nil,
		app.UpcomingSvc != nil,
		app.TradeSvc != nil,
	)
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "null"
	}
	return t.UTC().Format(time.RFC3339)
}
