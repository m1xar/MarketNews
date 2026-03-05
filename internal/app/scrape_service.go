package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"MarketNews/internal/ingest"
	"MarketNews/internal/models"
	"MarketNews/internal/scraper"
)

type ScrapeConfig struct {
	StartDate     time.Time
	EndDate       time.Time
	Names         []string
	Countries     []string
	Currencies    []string
	Impacts       []string
	DeltasMinutes []int
	Delay         time.Duration
}

type SyncConfig struct {
	StartDate     time.Time
	Names         []string
	Countries     []string
	Currencies    []string
	Impacts       []string
	DeltasMinutes []int
	Delay         time.Duration
}

type ScrapeService struct {
	txBeginner TxBeginner
	newsRepo   NewsRepository
	eventRepo  EventRepository
	scraper    ScraperClient
	stats      StatsRunner
}

func NewScrapeService(
	txBeginner TxBeginner,
	newsRepo NewsRepository,
	eventRepo EventRepository,
	scraper ScraperClient,
	stats StatsRunner,
) *ScrapeService {
	return &ScrapeService{
		txBeginner: txBeginner,
		newsRepo:   newsRepo,
		eventRepo:  eventRepo,
		scraper:    scraper,
		stats:      stats,
	}
}

func (s *ScrapeService) RunScrape(ctx context.Context, cfg ScrapeConfig) error {
	if cfg.StartDate.IsZero() || cfg.EndDate.IsZero() {
		return fmt.Errorf("start and end dates are required")
	}
	if cfg.EndDate.Before(cfg.StartDate) {
		return fmt.Errorf("end date must be >= start date")
	}
	if cfg.Delay == 0 {
		cfg.Delay = 750 * time.Millisecond
	}
	if len(cfg.DeltasMinutes) == 0 {
		return fmt.Errorf("deltas_minutes is required for stats")
	}

	filter := newFilterSet(cfg)

	var failures []models.Failure
	successDays := 0
	totalDays := 0

	for d := cfg.StartDate; !d.After(cfg.EndDate); d = d.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return err
		}
		totalDays++
		url := scraper.DayURL(d)

		statusCode, body, reqErr := s.scraper.Fetch(ctx, url)
		if reqErr != nil {
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: reqErr.Error()})
			time.Sleep(cfg.Delay)
			continue
		}

		days, parseErr := s.scraper.ParseDays(body)
		if parseErr != nil {
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: parseErr.Error()})
			time.Sleep(cfg.Delay)
			continue
		}

		tx, err := s.txBeginner.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		eventsWritten := 0
		dayErr := false
		for _, day := range days {
			for _, ev := range day.Events {
				if !scraper.TimeLabelHasDigits(ev.TimeLabel) {
					continue
				}

				meta := scraper.BuildMeta(ev)
				metaJSON, _ := json.Marshal(meta)

				dt := scraper.FormatEventDatetime(ev)
				row := models.EventRow{
					ID:       ev.ID,
					DateTime: &dt,
					Country:  ingest.NormalizeField(ev.Country),
					Currency: ingest.NormalizeField(ev.Currency),
					Impact:   ingest.NormalizeField(ev.ImpactName),
					Name:     ingest.NormalizeName(ev.Name),
					Actual:   strings.TrimSpace(ev.Actual),
					Forecast: strings.TrimSpace(ev.Forecast),
					Previous: strings.TrimSpace(ev.Previous),
					Metadata: nil,
					NewsKey:  nil,
				}
				if len(metaJSON) > 0 {
					metaStr := string(metaJSON)
					row.Metadata = &metaStr
					row.NewsKey = ingest.DeriveNewsKey(metaStr)
				}

				if !filter.matches(row) {
					continue
				}

				row.ActualF = ingest.ParseValueFloat(row.Actual)
				row.ForecastF = ingest.ParseValueFloat(row.Forecast)
				row.PreviousF = ingest.ParseValueFloat(row.Previous)

				newsID, err := s.newsRepo.EnsureID(ctx, tx, row.Name, row.Country, row.Currency, row.NewsKey)
				if err != nil {
					dayErr = true
					break
				}
				if err := s.eventRepo.Upsert(ctx, tx, newsID, row); err != nil {
					dayErr = true
					break
				}
				eventsWritten++
			}
			if dayErr {
				break
			}
		}

		if dayErr {
			_ = tx.Rollback()
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: "db write failed"})
			time.Sleep(cfg.Delay)
			continue
		}
		if err := tx.Commit(); err != nil {
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: err.Error()})
		} else {
			successDays++
		}

		time.Sleep(cfg.Delay)
	}

	_ = failures

	if s.stats != nil {
		statsErr := s.stats.ComputeStats(ctx, StatsConfig{
			DeltasMinutes: cfg.DeltasMinutes,
		})
		if statsErr != nil {
			return fmt.Errorf("compute stats: %w", statsErr)
		}
	}
	return nil
}

func (s *ScrapeService) RunSync(ctx context.Context, cfg SyncConfig) error {
	if cfg.StartDate.IsZero() {
		return fmt.Errorf("start date is required")
	}
	if cfg.Delay == 0 {
		cfg.Delay = 750 * time.Millisecond
	}
	if len(cfg.DeltasMinutes) == 0 {
		return fmt.Errorf("deltas_minutes is required for stats")
	}

	start := dateOnlyUTC(cfg.StartDate)
	end := dateOnlyUTC(time.Now().UTC()).AddDate(0, 0, -1)
	if end.Before(start) {
		return fmt.Errorf("start date must be <= yesterday (end=%s)", end.Format("2006-01-02"))
	}

	filter := filterSet{
		names:      normalizeSet(cfg.Names, ingest.NormalizeName),
		countries:  normalizeSet(cfg.Countries, ingest.NormalizeField),
		currencies: normalizeSet(cfg.Currencies, ingest.NormalizeField),
		impacts:    normalizeSet(cfg.Impacts, ingest.NormalizeField),
	}

	var failures []models.Failure
	successDays := 0
	totalDays := 0

	newsIDs := map[string]struct{}{}

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return err
		}
		totalDays++
		url := scraper.DayURL(d)

		statusCode, body, reqErr := s.scraper.Fetch(ctx, url)
		if reqErr != nil {
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: reqErr.Error()})
			time.Sleep(cfg.Delay)
			continue
		}

		days, parseErr := s.scraper.ParseDays(body)
		if parseErr != nil {
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: parseErr.Error()})
			time.Sleep(cfg.Delay)
			continue
		}

		tx, err := s.txBeginner.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		eventsWritten := 0
		dayErr := false
		dayNewsIDs := map[string]struct{}{}
		for _, day := range days {
			for _, ev := range day.Events {
				if !scraper.TimeLabelHasDigits(ev.TimeLabel) {
					continue
				}

				meta := scraper.BuildMeta(ev)
				metaJSON, _ := json.Marshal(meta)

				dt := scraper.FormatEventDatetime(ev)
				row := models.EventRow{
					ID:       ev.ID,
					DateTime: &dt,
					Country:  ingest.NormalizeField(ev.Country),
					Currency: ingest.NormalizeField(ev.Currency),
					Impact:   ingest.NormalizeField(ev.ImpactName),
					Name:     ingest.NormalizeName(ev.Name),
					Actual:   strings.TrimSpace(ev.Actual),
					Forecast: strings.TrimSpace(ev.Forecast),
					Previous: strings.TrimSpace(ev.Previous),
					Metadata: nil,
					NewsKey:  nil,
				}
				if len(metaJSON) > 0 {
					metaStr := string(metaJSON)
					row.Metadata = &metaStr
					row.NewsKey = ingest.DeriveNewsKey(metaStr)
				}

				if !filter.matches(row) {
					continue
				}

				row.ActualF = ingest.ParseValueFloat(row.Actual)
				row.ForecastF = ingest.ParseValueFloat(row.Forecast)
				row.PreviousF = ingest.ParseValueFloat(row.Previous)

				newsID, err := s.newsRepo.EnsureID(ctx, tx, row.Name, row.Country, row.Currency, row.NewsKey)
				if err != nil {
					dayErr = true
					break
				}
				if err := s.eventRepo.Upsert(ctx, tx, newsID, row); err != nil {
					dayErr = true
					break
				}
				dayNewsIDs[newsID] = struct{}{}
				eventsWritten++
			}
			if dayErr {
				break
			}
		}

		if dayErr {
			_ = tx.Rollback()
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: "db write failed"})
			time.Sleep(cfg.Delay)
			continue
		}
		if err := tx.Commit(); err != nil {
			failures = append(failures, models.Failure{Date: d.Format("2006-01-02"), StatusCode: statusCode, Err: err.Error()})
		} else {
			for id := range dayNewsIDs {
				newsIDs[id] = struct{}{}
			}
			successDays++
		}

		time.Sleep(cfg.Delay)
	}

	_ = failures

	if s.stats != nil && len(newsIDs) > 0 {
		ids := make([]string, 0, len(newsIDs))
		for id := range newsIDs {
			ids = append(ids, id)
		}
		statsErr := s.stats.ComputeStatsForNewsIDs(ctx, ids, StatsConfig{
			DeltasMinutes: cfg.DeltasMinutes,
		})
		if statsErr != nil {
			return fmt.Errorf("compute stats for news ids: %w", statsErr)
		}
	}
	return nil
}

type filterSet struct {
	names      map[string]struct{}
	countries  map[string]struct{}
	currencies map[string]struct{}
	impacts    map[string]struct{}
}

func newFilterSet(cfg ScrapeConfig) filterSet {
	return filterSet{
		names:      normalizeSet(cfg.Names, ingest.NormalizeName),
		countries:  normalizeSet(cfg.Countries, ingest.NormalizeField),
		currencies: normalizeSet(cfg.Currencies, ingest.NormalizeField),
		impacts:    normalizeSet(cfg.Impacts, ingest.NormalizeField),
	}
}

func normalizeSet(values []string, normalize func(string) string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = normalize(v)
		v = strings.ToLower(v)
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func dateOnlyUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (f filterSet) matches(row models.EventRow) bool {
	if f.names != nil {
		name := strings.ToLower(ingest.NormalizeName(row.Name))
		if _, ok := f.names[name]; !ok {
			return false
		}
	}
	if f.countries != nil {
		country := strings.ToLower(ingest.NormalizeField(row.Country))
		if _, ok := f.countries[country]; !ok {
			return false
		}
	}
	if f.currencies != nil {
		currency := strings.ToLower(ingest.NormalizeField(row.Currency))
		if _, ok := f.currencies[currency]; !ok {
			return false
		}
	}
	if f.impacts != nil {
		impact := strings.ToLower(ingest.NormalizeField(row.Impact))
		if _, ok := f.impacts[impact]; !ok {
			return false
		}
	}
	return true
}
