package repository

import (
	"context"
	"database/sql"
	"time"

	"MarketNews/internal/db/queries"
	"MarketNews/internal/models"
)

type UpcomingRepository struct {
	db *sql.DB
}

func NewUpcomingRepository(db *sql.DB) *UpcomingRepository {
	return &UpcomingRepository{db: db}
}

func (r *UpcomingRepository) GetEventTypeStats(ctx context.Context, newsID string) (*models.EventTypeStats, error) {
	row := r.db.QueryRowContext(ctx, queries.SelectEventTypeStatsByNewsID, newsID)
	var rec models.EventTypeStats
	if err := row.Scan(&rec.NewsID, &rec.SigmaSurprise, &rec.MeanSurprise, &rec.NSamples, &rec.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *UpcomingRepository) ListAssetStats(ctx context.Context, newsID string) ([]models.EventAssetStats, error) {
	rows, err := r.db.QueryContext(ctx, queries.ListAssetStatsByNewsID, newsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EventAssetStats
	for rows.Next() {
		var rec models.EventAssetStats
		if err := rows.Scan(
			&rec.NewsID,
			&rec.AssetSymbol,
			&rec.DeltaMinutes,
			&rec.Beta,
			&rec.Alpha,
			&rec.R2,
			&rec.NSamples,
			&rec.PPosGivenZPos,
			&rec.PNegGivenZNeg,
			&rec.PDir,
			&rec.MeanAbsReturn,
			&rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UpcomingRepository) ListEventTypeStatsByNewsIDs(ctx context.Context, newsIDs []string) ([]models.EventTypeStats, error) {
	if len(newsIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, queries.ListEventTypeStatsByNewsIDs, newsIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EventTypeStats
	for rows.Next() {
		var rec models.EventTypeStats
		if err := rows.Scan(&rec.NewsID, &rec.SigmaSurprise, &rec.MeanSurprise, &rec.NSamples, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UpcomingRepository) ListAssetStatsByNewsIDs(ctx context.Context, newsIDs []string) ([]models.EventAssetStats, error) {
	if len(newsIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, queries.ListAssetStatsByNewsIDs, newsIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EventAssetStats
	for rows.Next() {
		var rec models.EventAssetStats
		if err := rows.Scan(
			&rec.NewsID,
			&rec.AssetSymbol,
			&rec.DeltaMinutes,
			&rec.Beta,
			&rec.Alpha,
			&rec.R2,
			&rec.NSamples,
			&rec.PPosGivenZPos,
			&rec.PNegGivenZNeg,
			&rec.PDir,
			&rec.MeanAbsReturn,
			&rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UpcomingRepository) ListRecentEvents(ctx context.Context, newsID string, limit int) ([]models.EventHistoryItem, error) {
	rows, err := r.db.QueryContext(ctx, queries.ListRecentEventsByNewsID, newsID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EventHistoryItem
	for rows.Next() {
		var rec models.EventHistoryItem
		if err := rows.Scan(
			&rec.EventID,
			&rec.EventTime,
			&rec.ActualValue,
			&rec.ForecastValue,
			&rec.PreviousValue,
			&rec.Surprise,
			&rec.ZScore,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UpcomingRepository) ListEventsByNewsID(ctx context.Context, newsID string) ([]models.EventHistoryItem, error) {
	rows, err := r.db.QueryContext(ctx, queries.ListEventsByNewsID, newsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EventHistoryItem
	for rows.Next() {
		var rec models.EventHistoryItem
		if err := rows.Scan(
			&rec.EventID,
			&rec.EventTime,
			&rec.ActualValue,
			&rec.ForecastValue,
			&rec.PreviousValue,
			&rec.Surprise,
			&rec.ZScore,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UpcomingRepository) ListReturnsByEventIDs(ctx context.Context, eventIDs []int64) ([]models.EventAssetReturn, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, queries.ListReturnsByEventIDs, eventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EventAssetReturn
	for rows.Next() {
		var rec models.EventAssetReturn
		if err := rows.Scan(
			&rec.EventID,
			&rec.AssetSymbol,
			&rec.DeltaMinutes,
			&rec.Price0,
			&rec.PriceDelta,
			&rec.ReturnLn,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UpcomingRepository) UpsertUpcomingAnalysis(ctx context.Context, rec models.UpcomingAnalysisRecord) error {
	var meta interface{}
	if rec.Metadata != nil {
		meta = *rec.Metadata
	}
	_, err := r.db.ExecContext(
		ctx,
		queries.UpsertUpcomingAnalysis,
		rec.EventID,
		rec.NewsID,
		rec.FFID,
		rec.EventTime,
		rec.Country,
		rec.Currency,
		rec.Symbol,
		rec.Importance,
		rec.ForecastValue,
		rec.PreviousValue,
		meta,
		rec.AnalysisText,
	)
	return err
}

func (r *UpcomingRepository) GetUpcomingAnalysisByEventID(ctx context.Context, eventID int64) (*models.UpcomingAnalysisDetail, error) {
	row := r.db.QueryRowContext(ctx, queries.SelectUpcomingAnalysisByEventID, eventID)
	var rec models.UpcomingAnalysisDetail
	if err := row.Scan(
		&rec.EventID,
		&rec.NewsID,
		&rec.FFID,
		&rec.EventTime,
		&rec.Country,
		&rec.Currency,
		&rec.Symbol,
		&rec.Importance,
		&rec.ForecastValue,
		&rec.PreviousValue,
		&rec.Metadata,
		&rec.AnalysisText,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&rec.NewsName,
		&rec.NewsKey,
		&rec.ForecastRate,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *UpcomingRepository) ListUpcomingAnalysis(ctx context.Context, from, to time.Time) ([]models.UpcomingAnalysisDetail, error) {
	rows, err := r.db.QueryContext(ctx, queries.ListUpcomingAnalysisByRange, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.UpcomingAnalysisDetail
	for rows.Next() {
		var rec models.UpcomingAnalysisDetail
		if err := rows.Scan(
			&rec.EventID,
			&rec.NewsID,
			&rec.FFID,
			&rec.EventTime,
			&rec.Country,
			&rec.Currency,
			&rec.Symbol,
			&rec.Importance,
			&rec.ForecastValue,
			&rec.PreviousValue,
			&rec.Metadata,
			&rec.AnalysisText,
			&rec.CreatedAt,
			&rec.UpdatedAt,
			&rec.NewsName,
			&rec.NewsKey,
			&rec.ForecastRate,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *UpcomingRepository) TruncateUpcomingAnalysis(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, queries.TruncateUpcomingAnalysis)
	return err
}
