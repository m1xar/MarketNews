package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"MarketNews/internal/api/dto"
	"MarketNews/internal/api/helpers"
	"MarketNews/internal/app"
	"MarketNews/internal/models"
)

type TradeAnalysisService interface {
	GenerateTradeAnalysis(ctx context.Context, req app.TradeAnalysisRequest) (*models.TradeAnalysisRecord, error)
	GetTradeAnalysisByID(ctx context.Context, tradeID int64) (*models.TradeAnalysisRecord, error)
}

type TradeHandlers struct {
	svc TradeAnalysisService
}

func NewTradeHandlers(svc TradeAnalysisService) *TradeHandlers {
	return &TradeHandlers{svc: svc}
}

func (h *TradeHandlers) HandleTradeAnalysisCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		helpers.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.svc == nil {
		helpers.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "trade analysis service is not configured"})
		return
	}

	var req dto.TradeAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	openDate, err := time.Parse(time.RFC3339, strings.TrimSpace(req.OpenDate))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "open_date must be RFC3339"})
		return
	}
	currentDate, err := time.Parse(time.RFC3339, strings.TrimSpace(req.CurrentDate))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "current_date must be RFC3339"})
		return
	}
	eventsDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.EventsDate))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "events_date must be YYYY-MM-DD"})
		return
	}

	analysis, err := h.svc.GenerateTradeAnalysis(r.Context(), app.TradeAnalysisRequest{
		TradeID:      req.TradeID,
		PairName:     req.PairName,
		EntryPrice:   req.EntryPrice,
		Amount:       req.Amount,
		Asset:        req.Asset,
		Direction:    req.Direction,
		StopLoss:     req.StopLoss,
		TakeProfit:   req.TakeProfit,
		OpenDate:     openDate,
		CurrentDate:  currentDate,
		CurrentPrice: req.CurrentPrice,
		EventsDate:   eventsDate,
	})
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]interface{}{
		"trade_id":      analysis.TradeID,
		"pair":          analysis.PairName,
		"entry_price":   analysis.EntryPrice,
		"amount":        analysis.Amount,
		"asset":         analysis.Asset,
		"direction":     analysis.Direction,
		"stop_loss":     analysis.StopLoss,
		"take_profit":   analysis.TakeProfit,
		"open_date":     analysis.OpenDate.UTC().Format(time.RFC3339),
		"current_date":  analysis.CurrentDate.UTC().Format(time.RFC3339),
		"current_price": analysis.CurrentPrice,
		"events_date":   analysis.EventsDate.UTC().Format("2006-01-02"),
		"analysis_text": analysis.AnalysisText,
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *TradeHandlers) HandleTradeAnalysisGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		helpers.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.svc == nil {
		helpers.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "trade analysis service is not configured"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/trade-analysis/")
	path = strings.Trim(path, "/")
	if path == "" {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	analysisOnly := false
	if strings.HasPrefix(path, "analysis/") {
		analysisOnly = true
		path = strings.TrimPrefix(path, "analysis/")
		path = strings.Trim(path, "/")
	}
	if path == "" {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "trade_id is required"})
		return
	}
	tradeID, err := strconv.ParseInt(path, 10, 64)
	if err != nil || tradeID <= 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "trade_id must be a positive integer"})
		return
	}

	rec, err := h.svc.GetTradeAnalysisByID(r.Context(), tradeID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "get trade analysis failed"})
		return
	}
	if rec == nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "trade analysis not found"})
		return
	}

	if analysisOnly {
		resp := map[string]interface{}{
			"trade_id":      rec.TradeID,
			"analysis_text": rec.AnalysisText,
			"updated_at":    rec.UpdatedAt.UTC().Format(time.RFC3339),
		}
		helpers.WriteJSON(w, http.StatusOK, resp)
		return
	}

	resp := map[string]interface{}{
		"trade_id":      rec.TradeID,
		"pair":          rec.PairName,
		"entry_price":   rec.EntryPrice,
		"amount":        rec.Amount,
		"asset":         rec.Asset,
		"direction":     rec.Direction,
		"stop_loss":     rec.StopLoss,
		"take_profit":   rec.TakeProfit,
		"open_date":     rec.OpenDate.UTC().Format(time.RFC3339),
		"current_date":  rec.CurrentDate.UTC().Format(time.RFC3339),
		"current_price": rec.CurrentPrice,
		"events_date":   rec.EventsDate.UTC().Format("2006-01-02"),
		"analysis_text": rec.AnalysisText,
		"created_at":    rec.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":    rec.UpdatedAt.UTC().Format(time.RFC3339),
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}
