package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"MarketNews/internal/ingest"
	"MarketNews/internal/models"
	"MarketNews/internal/scraper"
)

type UpcomingAnalysisService struct {
	repo     UpcomingRepository
	ai       AIClient
	scraper  ScraperClient
	newsRepo NewsRepository
	nowUTC   func() time.Time
	dayDelay time.Duration
}

func NewUpcomingAnalysisService(repo UpcomingRepository, ai AIClient, scraper ScraperClient, newsRepo NewsRepository) *UpcomingAnalysisService {
	return &UpcomingAnalysisService{
		repo:     repo,
		ai:       ai,
		scraper:  scraper,
		newsRepo: newsRepo,
		nowUTC:   func() time.Time { return time.Now().UTC() },
		dayDelay: 750 * time.Millisecond,
	}
}

func (s *UpcomingAnalysisService) GenerateAnalysisForDateAndFFID(ctx context.Context, day time.Time, ffID int64) error {
	if s.repo == nil {
		return fmt.Errorf("upcoming repo is required")
	}
	if s.ai == nil {
		return fmt.Errorf("ai client is required")
	}
	if s.scraper == nil {
		return fmt.Errorf("scraper client is required")
	}
	if s.newsRepo == nil {
		return fmt.Errorf("news repo is required")
	}
	if ffID <= 0 {
		return fmt.Errorf("ff_id must be positive")
	}

	day = dateOnlyUTC(day)
	url := scraper.DayURL(day)
	statusCode, body, reqErr := s.scraper.Fetch(ctx, url)
	if reqErr != nil {
		return fmt.Errorf("fetch day failed | status=%d | err=%w", statusCode, reqErr)
	}

	days, parseErr := s.scraper.ParseDays(body)
	if parseErr != nil {
		return fmt.Errorf("parse day failed | status=%d | err=%w", statusCode, parseErr)
	}

	ev, ok := findEventByFFID(days, ffID)
	if !ok {
		return fmt.Errorf("event not found for date=%s ff_id=%d", day.Format("2006-01-02"), ffID)
	}
	if ev.Dateline == 0 && !scraper.TimeLabelHasDigits(ev.TimeLabel) {
		return fmt.Errorf("event time is masked for ff_id=%d", ffID)
	}

	parsed, ok := buildUpcomingEvent(ev)
	if !ok {
		return fmt.Errorf("event parse failed for ff_id=%d", ffID)
	}

	newsRec, ok := s.resolveNews(ctx, parsed)
	if !ok {
		return fmt.Errorf("news not found for ff_id=%d", ffID)
	}
	parsed.News = *newsRec

	if err := s.analyzeUpcomingEvent(ctx, parsed); err != nil {
		return fmt.Errorf("analyze failed for ff_id=%d: %w", ffID, err)
	}
	return nil
}

func (s *UpcomingAnalysisService) GenerateAnalysesByRange(ctx context.Context, start, end time.Time) error {
	if s.repo == nil {
		return fmt.Errorf("upcoming repo is required")
	}
	if s.ai == nil {
		return fmt.Errorf("ai client is required")
	}
	if s.scraper == nil {
		return fmt.Errorf("scraper client is required")
	}
	if s.newsRepo == nil {
		return fmt.Errorf("news repo is required")
	}
	if end.Before(start) {
		return fmt.Errorf("end must be >= start")
	}

	startDay := dateOnlyUTC(start)
	endDay := dateOnlyUTC(end)
	errCount := 0
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return err
		}
		url := scraper.DayURL(d)
		_, body, reqErr := s.scraper.Fetch(ctx, url)
		if reqErr != nil {
			errCount++
			time.Sleep(s.dayDelay)
			continue
		}

		days, parseErr := s.scraper.ParseDays(body)
		if parseErr != nil {
			errCount++
			time.Sleep(s.dayDelay)
			continue
		}

		for _, day := range days {
			for _, ev := range day.Events {
				if !scraper.TimeLabelHasDigits(ev.TimeLabel) {
					continue
				}

				parsed, ok := buildUpcomingEvent(ev)
				if !ok {
					continue
				}

				newsRec, ok := s.resolveNews(ctx, parsed)
				if !ok {
					continue
				}
				parsed.News = *newsRec

				if err := s.analyzeUpcomingEvent(ctx, parsed); err != nil {
					errCount++
					continue
				}
			}
		}
		time.Sleep(s.dayDelay)
	}

	if errCount > 0 {
		return fmt.Errorf("upcoming analysis errors=%d", errCount)
	}
	return nil
}

func (s *UpcomingAnalysisService) TruncateUpcomingAnalysis(ctx context.Context) error {
	if s.repo == nil {
		return fmt.Errorf("upcoming repo is required")
	}
	return s.repo.TruncateUpcomingAnalysis(ctx)
}

func (s *UpcomingAnalysisService) analyzeUpcomingEvent(ctx context.Context, ev models.UpcomingEvent) error {
	if ev.EventTime == nil {
		return fmt.Errorf("event_time is required")
	}
	if ev.News.ID == "" {
		return fmt.Errorf("news_id is required")
	}
	typeStats, err := s.repo.GetEventTypeStats(ctx, ev.News.ID)
	if err != nil {
		return fmt.Errorf("type stats: %w", err)
	}
	assetStats, err := s.repo.ListAssetStats(ctx, ev.News.ID)
	if err != nil {
		return fmt.Errorf("asset stats: %w", err)
	}

	history, err := s.repo.ListRecentEvents(ctx, ev.News.ID, 3)
	if err != nil {
		return fmt.Errorf("recent events: %w", err)
	}

	eventIDs := make([]int64, 0, len(history))
	for _, h := range history {
		eventIDs = append(eventIDs, h.EventID)
	}
	returns, err := s.repo.ListReturnsByEventIDs(ctx, eventIDs)
	if err != nil {
		return fmt.Errorf("returns: %w", err)
	}

	history = attachReturns(history, returns)

	prompt := buildUpcomingPromptASCII(ev, typeStats, assetStats, history)
	analysis, err := s.ai.GenerateAnalysis(ctx, prompt)
	if err != nil {
		return fmt.Errorf("ai: %w", err)
	}

	rec := models.UpcomingAnalysisRecord{
		EventID:       ev.FFID,
		NewsID:        ev.News.ID,
		FFID:          ev.FFID,
		EventTime:     ev.EventTime,
		Country:       ev.News.Country,
		Currency:      ev.News.Currency,
		Symbol:        assetSymbolForCurrency(ev.News.Currency),
		Importance:    ev.Impact,
		ForecastValue: ev.ForecastValue,
		PreviousValue: ev.PreviousValue,
		Metadata:      ev.Metadata,
		AnalysisText:  analysis,
	}

	if err := s.repo.UpsertUpcomingAnalysis(ctx, rec); err != nil {
		return fmt.Errorf("upsert: %w", err)
	}
	return nil
}

func (s *UpcomingAnalysisService) resolveNews(ctx context.Context, ev models.UpcomingEvent) (*models.NewsRecord, bool) {
	if ev.News.NewsKey != nil && strings.TrimSpace(*ev.News.NewsKey) != "" {
		rec, err := s.newsRepo.GetByKey(ctx, *ev.News.NewsKey)
		if err == nil && rec != nil {
			return rec, true
		}
	}
	rec, err := s.newsRepo.GetByNatural(ctx, ev.News.Name, ev.News.Country, ev.News.Currency)
	if err == nil && rec != nil {
		return rec, true
	}
	return nil, false
}

func buildUpcomingEvent(ev scraper.Event) (models.UpcomingEvent, bool) {
	dt := scraper.FormatEventDatetime(ev)
	if dt == "" {
		return models.UpcomingEvent{}, false
	}
	t, err := time.Parse(time.RFC3339, dt)
	if err != nil {
		return models.UpcomingEvent{}, false
	}

	meta := scraper.BuildMeta(ev)
	metaJSON, _ := json.Marshal(meta)
	var metaStr *string
	if len(metaJSON) > 0 {
		s := string(metaJSON)
		metaStr = &s
	}

	name := ingest.NormalizeName(ev.Name)
	country := ingest.NormalizeField(ev.Country)
	currency := ingest.NormalizeField(ev.Currency)
	impact := ingest.NormalizeField(ev.ImpactName)

	var newsKey *string
	if metaStr != nil {
		newsKey = ingest.DeriveNewsKey(*metaStr)
	}

	return models.UpcomingEvent{
		EventID:       ev.ID,
		FFID:          ev.ID,
		EventTime:     &t,
		Impact:        impact,
		ForecastValue: ingest.ParseValueFloat(ev.Forecast),
		PreviousValue: ingest.ParseValueFloat(ev.Previous),
		Metadata:      metaStr,
		News: models.NewsRecord{
			Name:     name,
			Country:  country,
			Currency: currency,
			NewsKey:  newsKey,
		},
	}, true
}

func attachReturns(history []models.EventHistoryItem, returns []models.EventAssetReturn) []models.EventHistoryItem {
	if len(history) == 0 || len(returns) == 0 {
		return history
	}
	index := make(map[int64]int, len(history))
	for i, h := range history {
		index[h.EventID] = i
	}
	for _, r := range returns {
		if idx, ok := index[r.EventID]; ok {
			history[idx].Returns = append(history[idx].Returns, r)
		}
	}
	for i := range history {
		sort.Slice(history[i].Returns, func(a, b int) bool {
			if history[i].Returns[a].AssetSymbol == history[i].Returns[b].AssetSymbol {
				return history[i].Returns[a].DeltaMinutes < history[i].Returns[b].DeltaMinutes
			}
			return history[i].Returns[a].AssetSymbol < history[i].Returns[b].AssetSymbol
		})
	}
	return history
}

func findEventByFFID(days []scraper.Day, ffID int64) (scraper.Event, bool) {
	for _, day := range days {
		for _, ev := range day.Events {
			if ev.ID == ffID {
				return ev, true
			}
		}
	}
	return scraper.Event{}, false
}

func buildUpcomingPrompt(
	ev models.UpcomingEvent,
	typeStats *models.EventTypeStats,
	assetStats []models.EventAssetStats,
	history []models.EventHistoryItem,
) string {
	var b strings.Builder

	b.WriteString("You are a macro news financial analyst.\n")
	b.WriteString("Write an analytical overview for the upcoming event.\n")
	b.WriteString("Important: do not give numeric forecasts or precise predictions, only analysis of metrics and potential impact.\n\n")

	b.WriteString("UPCOMING_EVENT:\n")
	b.WriteString(fmt.Sprintf("- event_id: %d\n", ev.EventID))
	b.WriteString(fmt.Sprintf("- upcoming_id (ff_id): %d\n", ev.FFID))
	b.WriteString(fmt.Sprintf("- name: %s\n", ev.News.Name))
	b.WriteString(fmt.Sprintf("- country: %s\n", ev.News.Country))
	b.WriteString(fmt.Sprintf("- currency: %s\n", ev.News.Currency))
	b.WriteString(fmt.Sprintf("- symbol: %s\n", assetSymbolForCurrency(ev.News.Currency)))
	b.WriteString(fmt.Sprintf("- importance: %s\n", ev.Impact))
	if ev.EventTime != nil {
		b.WriteString(fmt.Sprintf("- event_time_utc: %s\n", ev.EventTime.UTC().Format(time.RFC3339)))
	}
	b.WriteString(fmt.Sprintf("- forecast_value: %s\n", fmtFloatPtr(ev.ForecastValue)))
	b.WriteString(fmt.Sprintf("- previous_value: %s\n", fmtFloatPtr(ev.PreviousValue)))
	b.WriteString(fmt.Sprintf("- metadata_json: %s\n", nullOrValue(ev.Metadata)))
	b.WriteString(fmt.Sprintf("- forecast_rate: %s\n", fmtFloatPtr(ev.News.ForecastRate)))
	b.WriteString("\n")

	b.WriteString("FIELD_EXPLANATIONS:\n")
	b.WriteString("- all aggregated metrics are computed on historical data imported into the DB at the time of analysis\n")
	b.WriteString("- surprise = actual_value - forecast_value\n")
	b.WriteString("- z_score = surprise / sigma_surprise (if sigma_surprise is available)\n")
	b.WriteString("- forecast_rate = share of cases where actual_value == forecast_value historically\n")
	b.WriteString("- asset_stats are aggregated on historical events for this news and capture the surprise->return relationship\n")
	b.WriteString("- beta/alpha: linear regression of return_ln on z_score, r2: fit quality\n")
	b.WriteString("- p_pos_given_zpos: probability of positive return_ln when z_score > 0\n")
	b.WriteString("- p_neg_given_zneg: probability of negative return_ln when z_score < 0\n")
	b.WriteString("- p_dir: share of cases where sign(beta*z_score) matches actual return_ln sign\n")
	b.WriteString("- mean_abs_return: mean absolute return_ln\n")
	b.WriteString("- return_ln: log return after delta_minutes from event time\n\n")

	b.WriteString("TYPE_STATS:\n")
	if typeStats == nil {
		b.WriteString("- none\n")
	} else {
		b.WriteString(fmt.Sprintf("- sigma_surprise: %s\n", fmtFloatPtr(typeStats.SigmaSurprise)))
		b.WriteString(fmt.Sprintf("- mean_surprise: %s\n", fmtFloatPtr(typeStats.MeanSurprise)))
		b.WriteString(fmt.Sprintf("- n_samples: %d\n", typeStats.NSamples))
		b.WriteString(fmt.Sprintf("- updated_at_utc: %s\n", typeStats.UpdatedAt.UTC().Format(time.RFC3339)))
	}
	b.WriteString("\n")

	b.WriteString("ASSET_STATS:\n")
	if len(assetStats) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, s := range assetStats {
			b.WriteString(fmt.Sprintf("- asset: %s | delta_minutes: %d | beta: %s | alpha: %s | r2: %s | n_samples: %d | p_pos_given_zpos: %s | p_neg_given_zneg: %s | p_dir: %s | mean_abs_return: %s\n",
				s.AssetSymbol,
				s.DeltaMinutes,
				fmtFloatPtr(s.Beta),
				fmtFloatPtr(s.Alpha),
				fmtFloatPtr(s.R2),
				s.NSamples,
				fmtFloatPtr(s.PPosGivenZPos),
				fmtFloatPtr(s.PNegGivenZNeg),
				fmtFloatPtr(s.PDir),
				fmtFloatPtr(s.MeanAbsReturn),
			))
		}
	}
	b.WriteString("\n")

	b.WriteString("RECENT_EVENTS (last 3):\n")
	if len(history) == 0 {
		b.WriteString("- none\n")
	} else {
		for i, h := range history {
			b.WriteString(fmt.Sprintf("- #%d event_id=%d | time_utc=%s | actual=%s | forecast=%s | previous=%s | surprise=%s | z_score=%s\n",
				i+1,
				h.EventID,
				h.EventTime.UTC().Format(time.RFC3339),
				fmtFloatPtr(h.ActualValue),
				fmtFloatPtr(h.ForecastValue),
				fmtFloatPtr(h.PreviousValue),
				fmtFloatPtr(h.Surprise),
				fmtFloatPtr(h.ZScore),
			))
			if len(h.Returns) == 0 {
				b.WriteString("  returns: none\n")
				continue
			}
			b.WriteString("  returns:\n")
			for _, r := range h.Returns {
				b.WriteString(fmt.Sprintf("  - asset=%s | delta_minutes=%d | price_0=%s | price_delta=%s | return_ln=%s\n",
					r.AssetSymbol,
					r.DeltaMinutes,
					fmtFloatPtr(r.Price0),
					fmtFloatPtr(r.PriceDelta),
					fmtFloatPtr(r.ReturnLn),
				))
			}
		}
	}

	b.WriteString("\nANALYSIS_TASK:\n")
	b.WriteString("Analyze what impact the event could have on the asset across different time deltas (delta_minutes) for different ranges of the actual value.\n")
	b.WriteString("Describe scenarios: below expectations (forecast), near expectations, above expectations. If forecast is missing, use previous.\n")
	b.WriteString("Focus on interpreting metrics, surprise vs return_ln relationships, direction probabilities, and magnitude expectations.\n")
	b.WriteString("Do not use a prediction format and do not provide exact numbers, only analytical conclusions.\n")

	return b.String()
}

func buildUpcomingPromptASCII(
	ev models.UpcomingEvent,
	typeStats *models.EventTypeStats,
	assetStats []models.EventAssetStats,
	history []models.EventHistoryItem,
) string {
	var b strings.Builder

	b.WriteString("You are a macro news analyst.\n")
	b.WriteString("Write an analytical overview for the upcoming event.\n")
	b.WriteString("Important: no numeric forecasts or precise predictions. Only analysis of metrics and potential impact.\n\n")

	b.WriteString("UPCOMING_EVENT:\n")
	b.WriteString(fmt.Sprintf("- event_id: %d\n", ev.EventID))
	b.WriteString(fmt.Sprintf("- upcoming_id (ff_id): %d\n", ev.FFID))
	b.WriteString(fmt.Sprintf("- name: %s\n", ev.News.Name))
	b.WriteString(fmt.Sprintf("- country: %s\n", ev.News.Country))
	b.WriteString(fmt.Sprintf("- currency: %s\n", ev.News.Currency))
	b.WriteString(fmt.Sprintf("- symbol: %s\n", assetSymbolForCurrency(ev.News.Currency)))
	b.WriteString(fmt.Sprintf("- importance: %s\n", ev.Impact))
	if ev.EventTime != nil {
		b.WriteString(fmt.Sprintf("- event_time_utc: %s\n", ev.EventTime.UTC().Format(time.RFC3339)))
	}
	b.WriteString(fmt.Sprintf("- forecast_value: %s\n", fmtFloatPtr(ev.ForecastValue)))
	b.WriteString(fmt.Sprintf("- previous_value: %s\n", fmtFloatPtr(ev.PreviousValue)))
	b.WriteString(fmt.Sprintf("- metadata_json: %s\n", nullOrValue(ev.Metadata)))
	b.WriteString(fmt.Sprintf("- forecast_rate: %s\n", fmtFloatPtr(ev.News.ForecastRate)))
	b.WriteString("\n")

	b.WriteString("FIELD_EXPLANATIONS:\n")
	b.WriteString("- all aggregated metrics are computed on historical data currently imported into the DB\n")
	b.WriteString("- surprise = actual_value - forecast_value\n")
	b.WriteString("- z_score = surprise / sigma_surprise (if sigma_surprise exists)\n")
	b.WriteString("- forecast_rate = share of cases where actual_value == forecast_value historically\n")
	b.WriteString("- asset_stats are aggregated for this news and capture the surprise->return relationship\n")
	b.WriteString("- beta/alpha: linear regression of return_ln on z_score, r2: fit quality\n")
	b.WriteString("- p_pos_given_zpos: probability of positive return_ln when z_score > 0\n")
	b.WriteString("- p_neg_given_zneg: probability of negative return_ln when z_score < 0\n")
	b.WriteString("- p_dir: share of cases where sign(beta*z_score) matches actual return_ln sign\n")
	b.WriteString("- mean_abs_return: mean absolute return_ln\n")
	b.WriteString("- return_ln: log return after delta_minutes from event time\n\n")

	b.WriteString("TYPE_STATS:\n")
	if typeStats == nil {
		b.WriteString("- none\n")
	} else {
		b.WriteString(fmt.Sprintf("- sigma_surprise: %s\n", fmtFloatPtr(typeStats.SigmaSurprise)))
		b.WriteString(fmt.Sprintf("- mean_surprise: %s\n", fmtFloatPtr(typeStats.MeanSurprise)))
		b.WriteString(fmt.Sprintf("- n_samples: %d\n", typeStats.NSamples))
		b.WriteString(fmt.Sprintf("- updated_at_utc: %s\n", typeStats.UpdatedAt.UTC().Format(time.RFC3339)))
	}
	b.WriteString("\n")

	b.WriteString("ASSET_STATS:\n")
	if len(assetStats) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, s := range assetStats {
			b.WriteString(fmt.Sprintf("- asset: %s | delta_minutes: %d | beta: %s | alpha: %s | r2: %s | n_samples: %d | p_pos_given_zpos: %s | p_neg_given_zneg: %s | p_dir: %s | mean_abs_return: %s\n",
				s.AssetSymbol,
				s.DeltaMinutes,
				fmtFloatPtr(s.Beta),
				fmtFloatPtr(s.Alpha),
				fmtFloatPtr(s.R2),
				s.NSamples,
				fmtFloatPtr(s.PPosGivenZPos),
				fmtFloatPtr(s.PNegGivenZNeg),
				fmtFloatPtr(s.PDir),
				fmtFloatPtr(s.MeanAbsReturn),
			))
		}
	}
	b.WriteString("\n")

	b.WriteString("RECENT_EVENTS (last 3):\n")
	if len(history) == 0 {
		b.WriteString("- none\n")
	} else {
		for i, h := range history {
			b.WriteString(fmt.Sprintf("- #%d event_id=%d | time_utc=%s | actual=%s | forecast=%s | previous=%s | surprise=%s | z_score=%s\n",
				i+1,
				h.EventID,
				h.EventTime.UTC().Format(time.RFC3339),
				fmtFloatPtr(h.ActualValue),
				fmtFloatPtr(h.ForecastValue),
				fmtFloatPtr(h.PreviousValue),
				fmtFloatPtr(h.Surprise),
				fmtFloatPtr(h.ZScore),
			))
			if len(h.Returns) == 0 {
				b.WriteString("  returns: none\n")
				continue
			}
			b.WriteString("  returns:\n")
			for _, r := range h.Returns {
				b.WriteString(fmt.Sprintf("  - asset=%s | delta_minutes=%d | price_0=%s | price_delta=%s | return_ln=%s\n",
					r.AssetSymbol,
					r.DeltaMinutes,
					fmtFloatPtr(r.Price0),
					fmtFloatPtr(r.PriceDelta),
					fmtFloatPtr(r.ReturnLn),
				))
			}
		}
	}

	b.WriteString("\nANALYSIS_TASK:\n")
	b.WriteString("Analyze potential impact across time deltas (delta_minutes) for different ranges of the actual value.\n")
	b.WriteString("Describe scenarios: below expectations (forecast), near expectations, above expectations. If forecast is missing, use previous.\n")
	b.WriteString("Focus on interpreting metrics, surprise vs return_ln relationships, direction probabilities, and magnitude expectations.\n")
	b.WriteString("Do not format as a prediction and do not provide exact numbers. Provide analytical conclusions only.\n")

	return b.String()
}

func nullOrValue(v *string) string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return "null"
	}
	return *v
}






