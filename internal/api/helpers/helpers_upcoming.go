package helpers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"MarketNews/internal/api/dto"
	"MarketNews/internal/models"
)

func ParseUpcomingRange(r *http.Request, now time.Time) (time.Time, time.Time, error) {
	fromStr := strings.TrimSpace(r.URL.Query().Get("from"))
	toStr := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromStr == "" && toStr == "" {
		start := dateOnlyUTC(now)
		end := start.AddDate(0, 0, 7).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return start, end, nil
	}

	if fromStr == "" || toStr == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("both from and to are required in YYYY-MM-DD")
	}
	fromDate, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from date")
	}
	toDate, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to date")
	}
	start := dateOnlyUTC(fromDate)
	end := dateOnlyUTC(toDate).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("to must be >= from")
	}
	return start, end, nil
}

func GroupUpcomingByDay(
	events []models.UpcomingAnalysisDetail,
	typeStatsByNewsID map[string]*models.EventTypeStats,
	assetStatsByNewsID map[string][]models.EventAssetStats,
) []dto.UpcomingDay {
	grouped := map[string][]map[string]interface{}{}
	for _, ev := range events {
		if ev.EventTime == nil {
			continue
		}
		dateKey := ev.EventTime.UTC().Format("2006-01-02")
		stats := typeStatsByNewsID[ev.NewsID]
		grouped[dateKey] = append(grouped[dateKey], map[string]interface{}{
			"event_id":             ev.EventID,
			"news_id":              ev.NewsID,
			"name":                 ev.NewsName,
			"time_utc":             ev.EventTime.UTC().Format(time.RFC3339),
			"importance":           ev.Importance,
			"currency":             ev.Currency,
			"symbol":               ev.Symbol,
			"sigma_surprise":       SigmaFromStats(stats),
			"mean_surprise":        MeanFromStats(stats),
			"ff_event_asset_stats": ToAssetStatsPayload(assetStatsByNewsID[ev.NewsID]),
		})
	}
	dates := make([]string, 0, len(grouped))
	for d := range grouped {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	out := make([]dto.UpcomingDay, 0, len(dates))
	for _, d := range dates {
		out = append(out, dto.UpcomingDay{Date: d, Events: grouped[d]})
	}
	return out
}

func SigmaFromStats(stats *models.EventTypeStats) interface{} {
	if stats == nil || stats.SigmaSurprise == nil {
		return nil
	}
	return *stats.SigmaSurprise
}

func MeanFromStats(stats *models.EventTypeStats) interface{} {
	if stats == nil || stats.MeanSurprise == nil {
		return nil
	}
	return *stats.MeanSurprise
}

func ToAssetStatsPayload(stats []models.EventAssetStats) []map[string]interface{} {
	if len(stats) == 0 {
		return []map[string]interface{}{}
	}
	out := make([]map[string]interface{}, 0, len(stats))
	for _, s := range stats {
		out = append(out, map[string]interface{}{
			"asset_symbol":     s.AssetSymbol,
			"delta_minutes":    s.DeltaMinutes,
			"beta":             s.Beta,
			"alpha":            s.Alpha,
			"r2":               s.R2,
			"n_samples":        s.NSamples,
			"p_pos_given_zpos": s.PPosGivenZPos,
			"p_neg_given_zneg": s.PNegGivenZNeg,
			"p_dir":            s.PDir,
			"mean_abs_return":  s.MeanAbsReturn,
			"updated_at":       s.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func ToRecentEventsPayload(history []models.EventHistoryItem) []map[string]interface{} {
	if len(history) == 0 {
		return []map[string]interface{}{}
	}
	out := make([]map[string]interface{}, 0, len(history))
	for _, h := range history {
		event := map[string]interface{}{
			"event_id":       h.EventID,
			"time_utc":       h.EventTime.UTC().Format(time.RFC3339),
			"actual_value":   h.ActualValue,
			"forecast_value": h.ForecastValue,
			"previous_value": h.PreviousValue,
			"surprise":       h.Surprise,
			"z_score":        h.ZScore,
			"returns":        []map[string]interface{}{},
		}
		if len(h.Returns) > 0 {
			ret := make([]map[string]interface{}, 0, len(h.Returns))
			for _, r := range h.Returns {
				ret = append(ret, map[string]interface{}{
					"asset_symbol":  r.AssetSymbol,
					"delta_minutes": r.DeltaMinutes,
					"price_0":       r.Price0,
					"price_delta":   r.PriceDelta,
					"return_ln":     r.ReturnLn,
				})
			}
			event["returns"] = ret
		}
		out = append(out, event)
	}
	return out
}

func AttachReturns(history []models.EventHistoryItem, returns []models.EventAssetReturn) []models.EventHistoryItem {
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
