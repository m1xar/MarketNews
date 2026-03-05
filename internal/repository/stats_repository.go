package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"MarketNews/internal/db/queries"
	"MarketNews/internal/models"
)

type StatsRepository struct {
	db *sql.DB
}

func NewStatsRepository(db *sql.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

func (r *StatsRepository) ListEventsForSurprise(ctx context.Context, from, to *time.Time, names, countries, currencies []string) ([]models.SurpriseEvent, error) {
	base := strings.Builder{}
	base.WriteString(queries.ListEventsForSurpriseBase)

	where := []string{}
	args := []interface{}{}
	idx := 1

	if from != nil {
		where = append(where, "e.event_time >= $"+itoa(idx))
		args = append(args, *from)
		idx++
	}
	if to != nil {
		where = append(where, "e.event_time <= $"+itoa(idx))
		args = append(args, *to)
		idx++
	}
	if len(names) > 0 {
		where = append(where, "n.name = ANY($"+itoa(idx)+")")
		args = append(args, names)
		idx++
	}
	if len(countries) > 0 {
		where = append(where, "n.country = ANY($"+itoa(idx)+")")
		args = append(args, countries)
		idx++
	}
	if len(currencies) > 0 {
		where = append(where, "n.currency = ANY($"+itoa(idx)+")")
		args = append(args, currencies)
		idx++
	}
	if len(where) > 0 {
		base.WriteString(" AND ")
		base.WriteString(strings.Join(where, " AND "))
	}
	base.WriteString(" ORDER BY e.news_id ASC, e.id ASC")

	rows, err := r.db.QueryContext(ctx, base.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.SurpriseEvent
	for rows.Next() {
		var rec models.SurpriseEvent
		if err := rows.Scan(&rec.ID, &rec.NewsID, &rec.Actual, &rec.Forecast); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) ListEventsForSurpriseByNewsID(ctx context.Context, newsID string) ([]models.SurpriseEvent, error) {
	if strings.TrimSpace(newsID) == "" {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, queries.ListEventsForSurpriseByNewsID, newsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.SurpriseEvent
	for rows.Next() {
		var rec models.SurpriseEvent
		if err := rows.Scan(&rec.ID, &rec.NewsID, &rec.Actual, &rec.Forecast); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) UpdateEventSurprise(ctx context.Context, tx *sql.Tx, id int64, surprise, zscore *float64) error {
	_, err := tx.ExecContext(ctx, queries.UpdateEventSurprise, id, surprise, zscore)
	return err
}

func (r *StatsRepository) UpsertEventTypeStats(ctx context.Context, tx *sql.Tx, newsID string, sigma, mean *float64, n int) error {
	_, err := tx.ExecContext(ctx, queries.UpsertEventTypeStats, newsID, sigma, mean, n)
	return err
}

func (r *StatsRepository) UpdateNewsForecastRateByNewsIDs(ctx context.Context, newsIDs []string) (int64, error) {
	if len(newsIDs) == 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, queries.UpdateNewsForecastRateByIDs, newsIDs)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}

func (r *StatsRepository) ListEventsForAssetStats(ctx context.Context, from, to *time.Time, names, countries, currencies []string) ([]models.AssetEvent, error) {
	base := strings.Builder{}
	base.WriteString(queries.ListEventsForAssetStatsBase)

	where := []string{}
	args := []interface{}{}
	idx := 1

	if from != nil {
		where = append(where, "e.event_time >= $"+itoa(idx))
		args = append(args, *from)
		idx++
	}
	if to != nil {
		where = append(where, "e.event_time <= $"+itoa(idx))
		args = append(args, *to)
		idx++
	}
	if len(names) > 0 {
		where = append(where, "n.name = ANY($"+itoa(idx)+")")
		args = append(args, names)
		idx++
	}
	if len(countries) > 0 {
		where = append(where, "n.country = ANY($"+itoa(idx)+")")
		args = append(args, countries)
		idx++
	}
	if len(currencies) > 0 {
		where = append(where, "n.currency = ANY($"+itoa(idx)+")")
		args = append(args, currencies)
		idx++
	}
	if len(where) > 0 {
		base.WriteString(" AND ")
		base.WriteString(strings.Join(where, " AND "))
	}
	base.WriteString(" ORDER BY e.news_id ASC, e.id ASC")

	rows, err := r.db.QueryContext(ctx, base.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.AssetEvent
	for rows.Next() {
		var rec models.AssetEvent
		if err := rows.Scan(&rec.ID, &rec.NewsID, &rec.EventAt, &rec.ZScore, &rec.Currency); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) ListAssetEventsByNewsID(ctx context.Context, newsID string) ([]models.AssetEvent, error) {
	if strings.TrimSpace(newsID) == "" {
		return nil, nil
	}

	base := strings.Builder{}
	base.WriteString(queries.ListEventsForAssetStatsBase)
	base.WriteString(" AND e.news_id = $1")
	base.WriteString(" ORDER BY e.id ASC")

	rows, err := r.db.QueryContext(ctx, base.String(), newsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.AssetEvent
	for rows.Next() {
		var rec models.AssetEvent
		if err := rows.Scan(&rec.ID, &rec.NewsID, &rec.EventAt, &rec.ZScore, &rec.Currency); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *StatsRepository) UpsertEventAssetStats(
	ctx context.Context,
	tx *sql.Tx,
	newsID string,
	assetSymbol string,
	deltaMinutes int,
	beta *float64,
	alpha *float64,
	r2 *float64,
	nSamples int,
	pPosGivenZPos *float64,
	pNegGivenZNeg *float64,
	pDir *float64,
	meanAbsReturn *float64,
) error {
	_, err := tx.ExecContext(
		ctx,
		queries.UpsertEventAssetStats,
		newsID,
		assetSymbol,
		deltaMinutes,
		beta,
		alpha,
		r2,
		nSamples,
		pPosGivenZPos,
		pNegGivenZNeg,
		pDir,
		meanAbsReturn,
	)
	return err
}

func (r *StatsRepository) UpsertEventAssetReturn(
	ctx context.Context,
	tx *sql.Tx,
	eventID int64,
	assetSymbol string,
	deltaMinutes int,
	price0 *float64,
	priceDelta *float64,
	returnLn *float64,
) error {
	_, err := tx.ExecContext(
		ctx,
		queries.UpsertEventAssetReturn,
		eventID,
		assetSymbol,
		deltaMinutes,
		price0,
		priceDelta,
		returnLn,
	)
	return err
}

func (r *StatsRepository) UpdateNewsForecastRate(ctx context.Context, from, to *time.Time, names, countries, currencies []string) (int64, error) {
	base := strings.Builder{}
	base.WriteString(queries.UpdateNewsForecastRateBase)

	args := []interface{}{}
	idx := 1

	if from != nil {
		base.WriteString(" AND e.event_time >= $" + itoa(idx))
		args = append(args, *from)
		idx++
	}
	if to != nil {
		base.WriteString(" AND e.event_time <= $" + itoa(idx))
		args = append(args, *to)
		idx++
	}
	if len(names) > 0 {
		base.WriteString(" AND n.name = ANY($" + itoa(idx) + ")")
		args = append(args, names)
		idx++
	}
	if len(countries) > 0 {
		base.WriteString(" AND n.country = ANY($" + itoa(idx) + ")")
		args = append(args, countries)
		idx++
	}
	if len(currencies) > 0 {
		base.WriteString(" AND n.currency = ANY($" + itoa(idx) + ")")
		args = append(args, currencies)
		idx++
	}

	base.WriteString(queries.UpdateNewsForecastRateTail)

	res, err := r.db.ExecContext(ctx, base.String(), args...)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rows, nil
}
