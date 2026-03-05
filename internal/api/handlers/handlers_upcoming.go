package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MarketNews/internal/api/dto"
	"MarketNews/internal/api/helpers"
	"MarketNews/internal/models"
)

type UpcomingReader interface {
	GetUpcomingAnalysisByEventID(ctx context.Context, eventID int64) (*models.UpcomingAnalysisDetail, error)
	ListUpcomingAnalysis(ctx context.Context, from, to time.Time) ([]models.UpcomingAnalysisDetail, error)
	GetEventTypeStats(ctx context.Context, newsID string) (*models.EventTypeStats, error)
	ListAssetStats(ctx context.Context, newsID string) ([]models.EventAssetStats, error)
	ListEventTypeStatsByNewsIDs(ctx context.Context, newsIDs []string) ([]models.EventTypeStats, error)
	ListAssetStatsByNewsIDs(ctx context.Context, newsIDs []string) ([]models.EventAssetStats, error)
	ListRecentEvents(ctx context.Context, newsID string, limit int) ([]models.EventHistoryItem, error)
	ListReturnsByEventIDs(ctx context.Context, eventIDs []int64) ([]models.EventAssetReturn, error)
}

type UpcomingAnalyzer interface {
	GenerateAnalysisForDateAndFFID(ctx context.Context, day time.Time, ffID int64) error
}

type UpcomingHandlers struct {
	reader UpcomingReader
	ai     UpcomingAnalyzer
}

func NewUpcomingHandlers(reader UpcomingReader, ai UpcomingAnalyzer) *UpcomingHandlers {
	return &UpcomingHandlers{reader: reader, ai: ai}
}

func (h *UpcomingHandlers) HandleUpcomingList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		helpers.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.reader == nil {
		helpers.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "event repository is not configured"})
		return
	}

	now := time.Now().UTC()
	from, to, err := helpers.ParseUpcomingRange(r, now)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	events, err := h.reader.ListUpcomingAnalysis(r.Context(), from, to)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "list upcoming failed"})
		return
	}

	statsByNewsID, assetStatsByNewsID, err := h.loadUpcomingStats(r.Context(), events)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "load stats failed"})
		return
	}

	grouped := helpers.GroupUpcomingByDay(events, statsByNewsID, assetStatsByNewsID)
	resp := map[string]interface{}{
		"from": from.Format(time.RFC3339),
		"to":   to.Format(time.RFC3339),
		"days": grouped,
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *UpcomingHandlers) HandleUpcomingAnalysis(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		helpers.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.reader == nil {
		helpers.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "upcoming repository is not configured"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/upcoming/analysis/")
	idStr := strings.Trim(path, "/")
	if idStr == "" {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "event_id is required"})
		return
	}
	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || eventID <= 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "event_id must be a positive integer"})
		return
	}

	rec, err := h.reader.GetUpcomingAnalysisByEventID(r.Context(), eventID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "get analysis failed"})
		return
	}
	if rec == nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "analysis not found"})
		return
	}

	resp, err := h.buildUpcomingAnalysisResponse(r.Context(), rec)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *UpcomingHandlers) HandleUpcomingAnalysisRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		helpers.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.ai == nil || h.reader == nil {
		helpers.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "upcoming analysis is not configured"})
		return
	}

	var req dto.UpcomingAnalysisRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.Date) == "" {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "date is required (YYYY-MM-DD)"})
		return
	}
	if req.FFID <= 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "ff_id must be a positive integer"})
		return
	}
	day, err := time.Parse("2006-01-02", strings.TrimSpace(req.Date))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "date must be YYYY-MM-DD"})
		return
	}

	if err := h.ai.GenerateAnalysisForDateAndFFID(r.Context(), day, req.FFID); err != nil {
		status := http.StatusInternalServerError
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "event not found"):
			status = http.StatusNotFound
		case strings.Contains(errMsg, "ff_id must be positive"):
			status = http.StatusBadRequest
		case strings.Contains(errMsg, "event time is masked"):
			status = http.StatusBadRequest
		case strings.Contains(errMsg, "event parse failed"):
			status = http.StatusBadRequest
		case strings.Contains(errMsg, "news not found"):
			status = http.StatusNotFound
		}
		helpers.WriteJSON(w, status, map[string]string{"error": errMsg})
		return
	}

	rec, err := h.reader.GetUpcomingAnalysisByEventID(r.Context(), req.FFID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "get analysis failed"})
		return
	}
	if rec == nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "analysis not found"})
		return
	}

	resp, err := h.buildUpcomingAnalysisResponse(r.Context(), rec)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *UpcomingHandlers) buildUpcomingAnalysisResponse(ctx context.Context, rec *models.UpcomingAnalysisDetail) (map[string]interface{}, error) {
	typeStats, err := h.reader.GetEventTypeStats(ctx, rec.NewsID)
	if err != nil {
		return nil, fmt.Errorf("get type stats failed")
	}
	assetStats, err := h.reader.ListAssetStats(ctx, rec.NewsID)
	if err != nil {
		return nil, fmt.Errorf("get asset stats failed")
	}
	history, err := h.reader.ListRecentEvents(ctx, rec.NewsID, 3)
	if err != nil {
		return nil, fmt.Errorf("get recent events failed")
	}
	eventIDs := make([]int64, 0, len(history))
	for _, h := range history {
		eventIDs = append(eventIDs, h.EventID)
	}
	returns, err := h.reader.ListReturnsByEventIDs(ctx, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("get returns failed")
	}
	history = helpers.AttachReturns(history, returns)

	resp := map[string]interface{}{
		"event_id":             rec.EventID,
		"news_id":              rec.NewsID,
		"upcoming_id":          rec.FFID,
		"event_time":           helpers.FormatTime(rec.EventTime),
		"country":              rec.Country,
		"currency":             rec.Currency,
		"symbol":               rec.Symbol,
		"importance":           rec.Importance,
		"forecast_value":       rec.ForecastValue,
		"previous_value":       rec.PreviousValue,
		"metadata":             rec.Metadata,
		"analysis_text":        rec.AnalysisText,
		"created_at":           rec.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":           rec.UpdatedAt.UTC().Format(time.RFC3339),
		"news_name":            rec.NewsName,
		"news_key":             rec.NewsKey,
		"forecast_rate":        rec.ForecastRate,
		"sigma_surprise":       helpers.SigmaFromStats(typeStats),
		"mean_surprise":        helpers.MeanFromStats(typeStats),
		"ff_event_asset_stats": helpers.ToAssetStatsPayload(assetStats),
		"recent_events":        helpers.ToRecentEventsPayload(history),
	}
	return resp, nil
}

func (h *UpcomingHandlers) loadUpcomingStats(ctx context.Context, events []models.UpcomingAnalysisDetail) (map[string]*models.EventTypeStats, map[string][]models.EventAssetStats, error) {
	newsIDs := make([]string, 0, len(events))
	seen := map[string]struct{}{}
	for _, ev := range events {
		if ev.NewsID == "" {
			continue
		}
		if _, ok := seen[ev.NewsID]; ok {
			continue
		}
		seen[ev.NewsID] = struct{}{}
		newsIDs = append(newsIDs, ev.NewsID)
	}

	typeStats := map[string]*models.EventTypeStats{}
	if len(newsIDs) == 0 {
		return typeStats, map[string][]models.EventAssetStats{}, nil
	}
	stats, err := h.reader.ListEventTypeStatsByNewsIDs(ctx, newsIDs)
	if err != nil {
		return nil, nil, err
	}
	for i := range stats {
		rec := stats[i]
		typeStats[rec.NewsID] = &rec
	}

	assetStats := map[string][]models.EventAssetStats{}
	assets, err := h.reader.ListAssetStatsByNewsIDs(ctx, newsIDs)
	if err != nil {
		return nil, nil, err
	}
	for _, rec := range assets {
		assetStats[rec.NewsID] = append(assetStats[rec.NewsID], rec)
	}

	return typeStats, assetStats, nil
}
