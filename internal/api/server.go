package api

import (
	"context"
	"net/http"
	"time"

	"MarketNews/internal/api/handlers"
	"MarketNews/internal/api/middleware"
	"MarketNews/internal/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

type Server struct {
	server         *http.Server
	upcomingReader handlers.UpcomingReader
	upcomingAI     handlers.UpcomingAnalyzer
	tradeSvc       handlers.TradeAnalysisService
}

func NewServer(addr string, upcomingReader handlers.UpcomingReader, tradeSvc handlers.TradeAnalysisService, upcomingAI handlers.UpcomingAnalyzer) *Server {
	docs.Register()
	s := &Server{
		upcomingReader: upcomingReader,
		upcomingAI:     upcomingAI,
		tradeSvc:       tradeSvc,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.HandleHealth)
	mux.HandleFunc("/swagger", handlers.HandleSwaggerRedirect)
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	upcomingHandlers := handlers.NewUpcomingHandlers(upcomingReader, upcomingAI)
	tradeHandlers := handlers.NewTradeHandlers(tradeSvc)
	mux.HandleFunc("/api/v1/upcoming", upcomingHandlers.HandleUpcomingList)
	mux.HandleFunc("/api/v1/upcoming/analysis", upcomingHandlers.HandleUpcomingAnalysisRun)
	mux.HandleFunc("/api/v1/upcoming/analysis/", upcomingHandlers.HandleUpcomingAnalysis)
	mux.HandleFunc("/api/v1/trade-analysis", tradeHandlers.HandleTradeAnalysisCreate)
	mux.HandleFunc("/api/v1/trade-analysis/", tradeHandlers.HandleTradeAnalysisGet)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           middleware.Logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
