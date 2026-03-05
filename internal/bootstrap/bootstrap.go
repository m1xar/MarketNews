package bootstrap

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"MarketNews/internal/ai"
	"MarketNews/internal/api"
	"MarketNews/internal/app"
	"MarketNews/internal/config"
	"MarketNews/internal/db"
	"MarketNews/internal/marketdata"
	"MarketNews/internal/repository"
	"MarketNews/internal/scraper"
)

type App struct {
	DB            *sql.DB
	API           *api.Server
	EventRepo     *repository.EventRepository
	UpcomingRepo  *repository.UpcomingRepository
	TradeRepo     *repository.TradeAnalysisRepository
	ScrapeService *app.ScrapeService
	Maintenance   *app.MaintenanceService
	UpcomingSvc   *app.UpcomingAnalysisService
	TradeSvc      *app.TradeAnalysisService
}

func Build(ctx context.Context, cfg config.Config) (*App, error) {
	dbConn, err := db.OpenFromEnv()
	if err != nil {
		return nil, err
	}
	if err := db.EnsureSchema(ctx, dbConn); err != nil {
		_ = dbConn.Close()
		return nil, err
	}

	httpClient := &http.Client{Timeout: 2 * time.Minute}
	openAIHTTPClient := &http.Client{Timeout: 2 * time.Minute}
	newsRepo := repository.NewNewsRepository(dbConn)
	eventRepo := repository.NewEventRepository(dbConn)
	statsRepo := repository.NewStatsRepository(dbConn)
	upcomingRepo := repository.NewUpcomingRepository(dbConn)
	maintenanceRepo := repository.NewMaintenanceRepository(dbConn)
	tradeRepo := repository.NewTradeAnalysisRepository(dbConn)

	tdClient := marketdata.NewTwelveDataClient(cfg.TwelveDataAPIKey, httpClient)
	scraperClient := scraper.NewClient(httpClient)

	statsService := app.NewStatsService(dbConn, statsRepo, tdClient)
	scrapeService := app.NewScrapeService(dbConn, newsRepo, eventRepo, scraperClient, statsService)
	maintenanceService := app.NewMaintenanceService(maintenanceRepo)

	var openAIClient *ai.OpenAIClient
	if strings.TrimSpace(cfg.OpenAIAPIKey) != "" {
		openAIClient = ai.NewOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIBaseURL, openAIHTTPClient, cfg.OpenAIMaxOutputTokens)
	}
	upcomingService := app.NewUpcomingAnalysisService(upcomingRepo, openAIClient, scraperClient, newsRepo)
	tradeService := app.NewTradeAnalysisService(tradeRepo, upcomingRepo, openAIClient)

	apiServer := api.NewServer(cfg.HTTPAddr, upcomingRepo, tradeService, upcomingService)

	return &App{
		DB:            dbConn,
		API:           apiServer,
		EventRepo:     eventRepo,
		UpcomingRepo:  upcomingRepo,
		TradeRepo:     tradeRepo,
		ScrapeService: scrapeService,
		Maintenance:   maintenanceService,
		UpcomingSvc:   upcomingService,
		TradeSvc:      tradeService,
	}, nil
}
