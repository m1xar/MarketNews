package app

import (
	"context"
	"database/sql"
	"time"

	"MarketNews/internal/marketdata"
	"MarketNews/internal/models"
	"MarketNews/internal/scraper"
)

type TxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type NewsRepository interface {
	GetByID(ctx context.Context, id string) (*models.NewsRecord, error)
	GetByKey(ctx context.Context, key string) (*models.NewsRecord, error)
	GetByNatural(ctx context.Context, name, country, currency string) (*models.NewsRecord, error)
	Create(ctx context.Context, rec models.NewsRecord) (*models.NewsRecord, error)
	Update(ctx context.Context, rec models.NewsRecord) (*models.NewsRecord, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, q models.NewsQuery) ([]models.NewsRecord, error)
	EnsureID(ctx context.Context, tx *sql.Tx, name, country, currency string, newsKey *string) (string, error)
}

type EventRepository interface {
	GetByID(ctx context.Context, id int64) (*models.EventRecord, error)
	Create(ctx context.Context, rec models.EventRecord) (*models.EventRecord, error)
	Update(ctx context.Context, rec models.EventRecord) (*models.EventRecord, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, q models.EventQuery) ([]models.EventWithNews, error)
	Upsert(ctx context.Context, tx *sql.Tx, newsID string, row models.EventRow) error
}

type StatsRepository interface {
	ListEventsForSurprise(ctx context.Context, from, to *time.Time, names, countries, currencies []string) ([]models.SurpriseEvent, error)
	ListEventsForSurpriseByNewsID(ctx context.Context, newsID string) ([]models.SurpriseEvent, error)
	UpdateEventSurprise(ctx context.Context, tx *sql.Tx, id int64, surprise, zscore *float64) error
	UpsertEventTypeStats(ctx context.Context, tx *sql.Tx, newsID string, sigma, mean *float64, n int) error
	UpdateNewsForecastRate(ctx context.Context, from, to *time.Time, names, countries, currencies []string) (int64, error)
	UpdateNewsForecastRateByNewsIDs(ctx context.Context, newsIDs []string) (int64, error)
	ListEventsForAssetStats(ctx context.Context, from, to *time.Time, names, countries, currencies []string) ([]models.AssetEvent, error)
	ListAssetEventsByNewsID(ctx context.Context, newsID string) ([]models.AssetEvent, error)
	UpsertEventAssetStats(ctx context.Context, tx *sql.Tx, newsID string, assetSymbol string, deltaMinutes int, beta, alpha, r2 *float64, nSamples int, pPosGivenZPos, pNegGivenZNeg, pDir, meanAbsReturn *float64) error
	UpsertEventAssetReturn(ctx context.Context, tx *sql.Tx, eventID int64, assetSymbol string, deltaMinutes int, price0, priceDelta, returnLn *float64) error
}

type UpcomingRepository interface {
	GetEventTypeStats(ctx context.Context, newsID string) (*models.EventTypeStats, error)
	ListAssetStats(ctx context.Context, newsID string) ([]models.EventAssetStats, error)
	ListEventTypeStatsByNewsIDs(ctx context.Context, newsIDs []string) ([]models.EventTypeStats, error)
	ListAssetStatsByNewsIDs(ctx context.Context, newsIDs []string) ([]models.EventAssetStats, error)
	ListRecentEvents(ctx context.Context, newsID string, limit int) ([]models.EventHistoryItem, error)
	ListEventsByNewsID(ctx context.Context, newsID string) ([]models.EventHistoryItem, error)
	ListReturnsByEventIDs(ctx context.Context, eventIDs []int64) ([]models.EventAssetReturn, error)
	UpsertUpcomingAnalysis(ctx context.Context, rec models.UpcomingAnalysisRecord) error
	GetUpcomingAnalysisByEventID(ctx context.Context, eventID int64) (*models.UpcomingAnalysisDetail, error)
	ListUpcomingAnalysis(ctx context.Context, from, to time.Time) ([]models.UpcomingAnalysisDetail, error)
	TruncateUpcomingAnalysis(ctx context.Context) error
}

type TradeAnalysisRepository interface {
	UpsertTradeAnalysis(ctx context.Context, rec models.TradeAnalysisRecord) error
	GetTradeAnalysisByID(ctx context.Context, tradeID int64) (*models.TradeAnalysisRecord, error)
}

type MaintenanceRepository interface {
	TruncateAll(ctx context.Context) error
}

type ScraperClient interface {
	Fetch(ctx context.Context, url string) (int, []byte, error)
	ParseDays(body []byte) ([]scraper.Day, error)
}

type MarketDataClient interface {
	GetSeriesRange(ctx context.Context, symbol string, start, end time.Time) (marketdata.CandleSeries, error)
}

type StatsRunner interface {
	ComputeStats(ctx context.Context, cfg StatsConfig) error
	ComputeStatsForNewsIDs(ctx context.Context, newsIDs []string, cfg StatsConfig) error
}

type AIClient interface {
	GenerateAnalysis(ctx context.Context, prompt string) (string, error)
}
