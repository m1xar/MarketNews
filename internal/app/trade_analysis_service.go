package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"MarketNews/internal/models"
)

type TradeAnalysisService struct {
	repo     TradeAnalysisRepository
	upcoming UpcomingRepository
	ai       AIClient
}

func NewTradeAnalysisService(repo TradeAnalysisRepository, upcoming UpcomingRepository, ai AIClient) *TradeAnalysisService {
	return &TradeAnalysisService{
		repo:     repo,
		upcoming: upcoming,
		ai:       ai,
	}
}

type TradeAnalysisRequest struct {
	TradeID      int64
	PairName     string
	EntryPrice   float64
	Amount       float64
	Asset        string
	Direction    string
	StopLoss     *float64
	TakeProfit   *float64
	OpenDate     time.Time
	CurrentDate  time.Time
	CurrentPrice float64
	EventsDate   time.Time
}

type TradeAnalysisEvent struct {
	Upcoming    models.UpcomingAnalysisDetail
	TypeStats   *models.EventTypeStats
	AssetStats  []models.EventAssetStats
	AllHistory  []models.EventHistoryItem
	AllReturns  []models.EventAssetReturn
	HistoryWith []models.EventHistoryItem
}

func (s *TradeAnalysisService) GenerateTradeAnalysis(ctx context.Context, req TradeAnalysisRequest) (*models.TradeAnalysisRecord, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("trade analysis repo is required")
	}
	if s.upcoming == nil {
		return nil, fmt.Errorf("upcoming repo is required")
	}
	if s.ai == nil {
		return nil, fmt.Errorf("ai client is required")
	}
	if req.TradeID <= 0 {
		return nil, fmt.Errorf("trade_id must be positive")
	}
	if strings.TrimSpace(req.PairName) == "" {
		return nil, fmt.Errorf("pair_name is required")
	}
	if strings.TrimSpace(req.Asset) == "" {
		return nil, fmt.Errorf("asset is required")
	}
	dir := strings.ToLower(strings.TrimSpace(req.Direction))
	if dir != "long" && dir != "short" {
		return nil, fmt.Errorf("direction must be long or short")
	}
	if req.OpenDate.IsZero() || req.CurrentDate.IsZero() {
		return nil, fmt.Errorf("open_date and current_date are required")
	}
	if req.EventsDate.IsZero() {
		return nil, fmt.Errorf("events_date is required")
	}

	start := dateOnlyUTC(req.EventsDate)
	end := start.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	events, err := s.upcoming.ListUpcomingAnalysis(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("list upcoming analysis: %w", err)
	}

	filtered := filterUpcomingByPair(events, req.PairName)

	newsIDs := uniqueNewsIDs(filtered)
	typeStatsByNewsID, err := loadTypeStats(ctx, s.upcoming, newsIDs)
	if err != nil {
		return nil, err
	}
	assetStatsByNewsID, err := loadAssetStats(ctx, s.upcoming, newsIDs)
	if err != nil {
		return nil, err
	}

	eventDetails := make([]TradeAnalysisEvent, 0, len(filtered))
	for _, ev := range filtered {
		history, err := s.upcoming.ListEventsByNewsID(ctx, ev.NewsID)
		if err != nil {
			return nil, fmt.Errorf("list events by news_id: %w", err)
		}

		eventIDs := make([]int64, 0, len(history))
		for _, h := range history {
			eventIDs = append(eventIDs, h.EventID)
		}
		returns, err := s.upcoming.ListReturnsByEventIDs(ctx, eventIDs)
		if err != nil {
			return nil, fmt.Errorf("list returns: %w", err)
		}

		history = attachReturnsToHistory(history, returns)

		eventDetails = append(eventDetails, TradeAnalysisEvent{
			Upcoming:    ev,
			TypeStats:   typeStatsByNewsID[ev.NewsID],
			AssetStats:  assetStatsByNewsID[ev.NewsID],
			AllHistory:  history,
			AllReturns:  returns,
			HistoryWith: history,
		})
	}

	prompt := buildTradeAnalysisPromptASCII(req, eventDetails)
	analysis, err := s.ai.GenerateAnalysis(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("ai: %w", err)
	}

	rec := models.TradeAnalysisRecord{
		TradeID:      req.TradeID,
		PairName:     req.PairName,
		EntryPrice:   req.EntryPrice,
		Amount:       req.Amount,
		Asset:        req.Asset,
		Direction:    dir,
		StopLoss:     req.StopLoss,
		TakeProfit:   req.TakeProfit,
		OpenDate:     req.OpenDate,
		CurrentDate:  req.CurrentDate,
		CurrentPrice: req.CurrentPrice,
		EventsDate:   start,
		AnalysisText: analysis,
	}
	if err := s.repo.UpsertTradeAnalysis(ctx, rec); err != nil {
		return nil, fmt.Errorf("upsert trade analysis: %w", err)
	}
	return &rec, nil
}

func (s *TradeAnalysisService) GetTradeAnalysisByID(ctx context.Context, tradeID int64) (*models.TradeAnalysisRecord, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("trade analysis repo is required")
	}
	if tradeID <= 0 {
		return nil, fmt.Errorf("trade_id must be positive")
	}
	return s.repo.GetTradeAnalysisByID(ctx, tradeID)
}

func filterUpcomingByPair(events []models.UpcomingAnalysisDetail, pair string) []models.UpcomingAnalysisDetail {
	p := normalizePair(pair)
	if p == "" {
		return nil
	}
	out := make([]models.UpcomingAnalysisDetail, 0, len(events))
	for _, ev := range events {
		symbol := normalizePair(ev.Symbol)
		if symbol == "" {
			continue
		}
		if strings.Contains(p, symbol) {
			out = append(out, ev)
		}
	}
	return out
}

func normalizePair(v string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(v), " ", ""))
}

func uniqueNewsIDs(events []models.UpcomingAnalysisDetail) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, ev := range events {
		if strings.TrimSpace(ev.NewsID) == "" {
			continue
		}
		if _, ok := seen[ev.NewsID]; ok {
			continue
		}
		seen[ev.NewsID] = struct{}{}
		out = append(out, ev.NewsID)
	}
	return out
}

func loadTypeStats(ctx context.Context, repo UpcomingRepository, newsIDs []string) (map[string]*models.EventTypeStats, error) {
	out := map[string]*models.EventTypeStats{}
	if len(newsIDs) == 0 {
		return out, nil
	}
	stats, err := repo.ListEventTypeStatsByNewsIDs(ctx, newsIDs)
	if err != nil {
		return nil, fmt.Errorf("list event type stats: %w", err)
	}
	for i := range stats {
		rec := stats[i]
		out[rec.NewsID] = &rec
	}
	return out, nil
}

func loadAssetStats(ctx context.Context, repo UpcomingRepository, newsIDs []string) (map[string][]models.EventAssetStats, error) {
	out := map[string][]models.EventAssetStats{}
	if len(newsIDs) == 0 {
		return out, nil
	}
	stats, err := repo.ListAssetStatsByNewsIDs(ctx, newsIDs)
	if err != nil {
		return nil, fmt.Errorf("list asset stats: %w", err)
	}
	for _, rec := range stats {
		out[rec.NewsID] = append(out[rec.NewsID], rec)
	}
	return out, nil
}

func attachReturnsToHistory(history []models.EventHistoryItem, returns []models.EventAssetReturn) []models.EventHistoryItem {
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

func buildTradeAnalysisPromptASCII(req TradeAnalysisRequest, events []TradeAnalysisEvent) string {
	var b strings.Builder

	b.WriteString("You are a trading analyst.\n")
	b.WriteString("Provide a scenario-based analysis for the current position around upcoming macro events.\n")
	b.WriteString("Do not give exact price targets or numeric predictions. Focus on conditions and risk scenarios.\n\n")

	b.WriteString("TRADE:\n")
	b.WriteString(fmt.Sprintf("- trade_id: %d\n", req.TradeID))
	b.WriteString(fmt.Sprintf("- pair: %s\n", req.PairName))
	b.WriteString(fmt.Sprintf("- entry_price: %s\n", fmtFloat(req.EntryPrice)))
	b.WriteString(fmt.Sprintf("- amount: %s\n", fmtFloat(req.Amount)))
	b.WriteString(fmt.Sprintf("- asset: %s\n", req.Asset))
	b.WriteString(fmt.Sprintf("- direction: %s\n", req.Direction))
	b.WriteString(fmt.Sprintf("- stop_loss: %s\n", fmtFloatPtr(req.StopLoss)))
	b.WriteString(fmt.Sprintf("- take_profit: %s\n", fmtFloatPtr(req.TakeProfit)))
	b.WriteString(fmt.Sprintf("- open_date_utc: %s\n", req.OpenDate.UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- current_date_utc: %s\n", req.CurrentDate.UTC().Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- current_price: %s\n", fmtFloat(req.CurrentPrice)))
	b.WriteString(fmt.Sprintf("- events_date_utc: %s\n", dateOnlyUTC(req.EventsDate).Format("2006-01-02")))
	b.WriteString("\n")

	if len(events) == 0 {
		b.WriteString("UPCOMING_EVENTS: none matched for this pair and date.\n\n")
	} else {
		b.WriteString("UPCOMING_EVENTS:\n")
		for i, ev := range events {
			b.WriteString(fmt.Sprintf("- #%d event_id=%d | news_id=%s | name=%s | country=%s | currency=%s | symbol=%s | importance=%s | event_time_utc=%s\n",
				i+1,
				ev.Upcoming.EventID,
				ev.Upcoming.NewsID,
				ev.Upcoming.NewsName,
				ev.Upcoming.Country,
				ev.Upcoming.Currency,
				ev.Upcoming.Symbol,
				ev.Upcoming.Importance,
				fmtTime(ev.Upcoming.EventTime),
			))
			b.WriteString(fmt.Sprintf("  forecast_value=%s | previous_value=%s | forecast_rate=%s | metadata_json=%s\n",
				fmtFloatPtr(ev.Upcoming.ForecastValue),
				fmtFloatPtr(ev.Upcoming.PreviousValue),
				fmtFloatPtr(ev.Upcoming.ForecastRate),
				nullOrValue(ev.Upcoming.Metadata),
			))
			if ev.TypeStats == nil {
				b.WriteString("  TYPE_STATS: none\n")
			} else {
				b.WriteString(fmt.Sprintf("  TYPE_STATS: sigma_surprise=%s | mean_surprise=%s | n_samples=%d | updated_at_utc=%s\n",
					fmtFloatPtr(ev.TypeStats.SigmaSurprise),
					fmtFloatPtr(ev.TypeStats.MeanSurprise),
					ev.TypeStats.NSamples,
					ev.TypeStats.UpdatedAt.UTC().Format(time.RFC3339),
				))
			}
			if len(ev.AssetStats) == 0 {
				b.WriteString("  ASSET_STATS: none\n")
			} else {
				b.WriteString("  ASSET_STATS:\n")
				for _, s := range ev.AssetStats {
					b.WriteString(fmt.Sprintf("  - asset=%s | delta_minutes=%d | beta=%s | alpha=%s | r2=%s | n_samples=%d | p_pos_given_zpos=%s | p_neg_given_zneg=%s | p_dir=%s | mean_abs_return=%s\n",
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

			b.WriteString("  HISTORICAL_EVENTS_WITH_RETURNS (all):\n")
			if len(ev.HistoryWith) == 0 {
				b.WriteString("  - none\n")
			} else {
				for j, h := range ev.HistoryWith {
					b.WriteString(fmt.Sprintf("  - #%d event_id=%d | time_utc=%s | actual=%s | forecast=%s | previous=%s | surprise=%s | z_score=%s\n",
						j+1,
						h.EventID,
						h.EventTime.UTC().Format(time.RFC3339),
						fmtFloatPtr(h.ActualValue),
						fmtFloatPtr(h.ForecastValue),
						fmtFloatPtr(h.PreviousValue),
						fmtFloatPtr(h.Surprise),
						fmtFloatPtr(h.ZScore),
					))
					if len(h.Returns) == 0 {
						b.WriteString("    returns: none\n")
						continue
					}
					b.WriteString("    returns:\n")
					for _, r := range h.Returns {
						b.WriteString(fmt.Sprintf("    - asset=%s | delta_minutes=%d | price_0=%s | price_delta=%s | return_ln=%s\n",
							r.AssetSymbol,
							r.DeltaMinutes,
							fmtFloatPtr(r.Price0),
							fmtFloatPtr(r.PriceDelta),
							fmtFloatPtr(r.ReturnLn),
						))
					}
				}
			}
		}
	}

	b.WriteString("\nANALYSIS_TASK:\n")
	b.WriteString("Describe favorable and unfavorable scenarios for the current position.\n")
	b.WriteString("Explain when it is better to close the position before the news versus hedge during the initial reaction window.\n")
	b.WriteString("Tie the recommendations to the given metrics and historical reaction patterns.\n")
	b.WriteString("Clarify which trader expectations or risk preferences change the decision.\n")

	return b.String()
}
