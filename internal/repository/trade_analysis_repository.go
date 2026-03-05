package repository

import (
	"context"
	"database/sql"

	"MarketNews/internal/db/queries"
	"MarketNews/internal/models"
)

type TradeAnalysisRepository struct {
	db *sql.DB
}

func NewTradeAnalysisRepository(db *sql.DB) *TradeAnalysisRepository {
	return &TradeAnalysisRepository{db: db}
}

func (r *TradeAnalysisRepository) UpsertTradeAnalysis(ctx context.Context, rec models.TradeAnalysisRecord) error {
	_, err := r.db.ExecContext(
		ctx,
		queries.UpsertTradeAnalysis,
		rec.TradeID,
		rec.PairName,
		rec.EntryPrice,
		rec.Amount,
		rec.Asset,
		rec.Direction,
		rec.StopLoss,
		rec.TakeProfit,
		rec.OpenDate,
		rec.CurrentDate,
		rec.CurrentPrice,
		rec.EventsDate,
		rec.AnalysisText,
	)
	return err
}

func (r *TradeAnalysisRepository) GetTradeAnalysisByID(ctx context.Context, tradeID int64) (*models.TradeAnalysisRecord, error) {
	row := r.db.QueryRowContext(ctx, queries.SelectTradeAnalysisByID, tradeID)
	var rec models.TradeAnalysisRecord
	if err := row.Scan(
		&rec.TradeID,
		&rec.PairName,
		&rec.EntryPrice,
		&rec.Amount,
		&rec.Asset,
		&rec.Direction,
		&rec.StopLoss,
		&rec.TakeProfit,
		&rec.OpenDate,
		&rec.CurrentDate,
		&rec.CurrentPrice,
		&rec.EventsDate,
		&rec.AnalysisText,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}
