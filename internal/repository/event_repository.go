package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"MarketNews/internal/db/queries"
	"MarketNews/internal/models"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) GetByID(ctx context.Context, id int64) (*models.EventRecord, error) {
	row := r.db.QueryRowContext(ctx, queries.SelectEventByID, id)
	return scanEvent(row)
}

func (r *EventRepository) Create(ctx context.Context, rec models.EventRecord) (*models.EventRecord, error) {
	row := r.db.QueryRowContext(
		ctx,
		queries.InsertEvent,
		rec.NewsID,
		rec.FFID,
		rec.EventTime,
		rec.Impact,
		rec.ActualValue,
		rec.ForecastVal,
		rec.PreviousVal,
		rec.Surprise,
		rec.ZScore,
		rec.Metadata,
	)
	return scanEvent(row)
}

func (r *EventRepository) Update(ctx context.Context, rec models.EventRecord) (*models.EventRecord, error) {
	row := r.db.QueryRowContext(
		ctx,
		queries.UpdateEvent,
		rec.ID,
		rec.NewsID,
		rec.FFID,
		rec.EventTime,
		rec.Impact,
		rec.ActualValue,
		rec.ForecastVal,
		rec.PreviousVal,
		rec.Surprise,
		rec.ZScore,
		rec.Metadata,
	)
	return scanEvent(row)
}

func (r *EventRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, queries.DeleteEvent, id)
	return err
}

func (r *EventRepository) Upsert(ctx context.Context, tx *sql.Tx, newsID string, row models.EventRow) error {
	var eventTime interface{}
	if row.DateTime != nil && *row.DateTime != "" {
		if t, err := time.Parse(time.RFC3339, *row.DateTime); err == nil {
			eventTime = t
		}
	}

	var meta interface{}
	if row.Metadata != nil {
		meta = *row.Metadata
	}

	_, err := tx.ExecContext(
		ctx,
		queries.UpsertEvent,
		newsID,
		row.ID,
		eventTime,
		row.Impact,
		row.ActualF,
		row.ForecastF,
		row.PreviousF,
		nil,
		nil,
		meta,
	)
	return err
}

func (r *EventRepository) List(ctx context.Context, q models.EventQuery) ([]models.EventWithNews, error) {
	query, args := buildEventQuery(q)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.EventWithNews
	for rows.Next() {
		rec, err := scanEventWithNews(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func buildEventQuery(q models.EventQuery) (string, []interface{}) {
	base := strings.Builder{}
	base.WriteString(queries.ListEventsBase)

	where := []string{}
	args := []interface{}{}
	idx := 1

	if len(q.IDs) > 0 {
		where = append(where, "e.id = ANY($"+itoa(idx)+")")
		args = append(args, q.IDs)
		idx++
	}
	if len(q.NewsIDs) > 0 {
		where = append(where, "e.news_id = ANY($"+itoa(idx)+")")
		args = append(args, q.NewsIDs)
		idx++
	}
	if len(q.FFIDs) > 0 {
		where = append(where, "e.ff_id = ANY($"+itoa(idx)+")")
		args = append(args, q.FFIDs)
		idx++
	}
	if q.EventFrom != nil {
		where = append(where, "e.event_time >= $"+itoa(idx))
		args = append(args, *q.EventFrom)
		idx++
	}
	if q.EventTo != nil {
		where = append(where, "e.event_time <= $"+itoa(idx))
		args = append(args, *q.EventTo)
		idx++
	}
	if len(q.Impacts) > 0 {
		where = append(where, "e.impact = ANY($"+itoa(idx)+")")
		args = append(args, q.Impacts)
		idx++
	}
	if q.ActualMin != nil {
		where = append(where, "e.actual_value >= $"+itoa(idx))
		args = append(args, *q.ActualMin)
		idx++
	}
	if q.ActualMax != nil {
		where = append(where, "e.actual_value <= $"+itoa(idx))
		args = append(args, *q.ActualMax)
		idx++
	}
	if q.ForecastMin != nil {
		where = append(where, "e.forecast_value >= $"+itoa(idx))
		args = append(args, *q.ForecastMin)
		idx++
	}
	if q.ForecastMax != nil {
		where = append(where, "e.forecast_value <= $"+itoa(idx))
		args = append(args, *q.ForecastMax)
		idx++
	}
	if q.PreviousMin != nil {
		where = append(where, "e.previous_value >= $"+itoa(idx))
		args = append(args, *q.PreviousMin)
		idx++
	}
	if q.PreviousMax != nil {
		where = append(where, "e.previous_value <= $"+itoa(idx))
		args = append(args, *q.PreviousMax)
		idx++
	}
	if q.CreatedFrom != nil {
		where = append(where, "e.created_at >= $"+itoa(idx))
		args = append(args, *q.CreatedFrom)
		idx++
	}
	if q.CreatedTo != nil {
		where = append(where, "e.created_at <= $"+itoa(idx))
		args = append(args, *q.CreatedTo)
		idx++
	}
	if q.MetadataEquals != nil {
		where = append(where, "e.metadata = $"+itoa(idx)+"::jsonb")
		args = append(args, *q.MetadataEquals)
		idx++
	}
	if q.MetadataContains != nil {
		where = append(where, "e.metadata @> $"+itoa(idx)+"::jsonb")
		args = append(args, *q.MetadataContains)
		idx++
	}
	if len(q.NewsNames) > 0 {
		where = append(where, "n.name = ANY($"+itoa(idx)+")")
		args = append(args, q.NewsNames)
		idx++
	}
	if len(q.NewsCountries) > 0 {
		where = append(where, "n.country = ANY($"+itoa(idx)+")")
		args = append(args, q.NewsCountries)
		idx++
	}
	if len(q.NewsCurrencies) > 0 {
		where = append(where, "n.currency = ANY($"+itoa(idx)+")")
		args = append(args, q.NewsCurrencies)
		idx++
	}
	if len(q.NewsKeys) > 0 {
		where = append(where, "n.news_key = ANY($"+itoa(idx)+")")
		args = append(args, q.NewsKeys)
		idx++
	}

	if len(where) > 0 {
		base.WriteString(" WHERE ")
		base.WriteString(strings.Join(where, " AND "))
	}
	base.WriteString(" ORDER BY e.event_time ASC, e.id ASC")
	limit := normalizeLimit(q.Limit)
	base.WriteString(" LIMIT " + itoa(limit))
	if q.Offset > 0 {
		base.WriteString(" OFFSET " + itoa(q.Offset))
	}
	return base.String(), args
}

type eventScanner interface {
	Scan(dest ...interface{}) error
}

func scanEvent(row eventScanner) (*models.EventRecord, error) {
	var rec models.EventRecord
	if err := row.Scan(
		&rec.ID,
		&rec.NewsID,
		&rec.FFID,
		&rec.EventTime,
		&rec.Impact,
		&rec.ActualValue,
		&rec.ForecastVal,
		&rec.PreviousVal,
		&rec.Surprise,
		&rec.ZScore,
		&rec.Metadata,
		&rec.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &rec, nil
}

func scanEventWithNews(row eventScanner) (*models.EventWithNews, error) {
	var event models.EventRecord
	var news models.NewsRecord
	if err := row.Scan(
		&event.ID,
		&event.NewsID,
		&event.FFID,
		&event.EventTime,
		&event.Impact,
		&event.ActualValue,
		&event.ForecastVal,
		&event.PreviousVal,
		&event.Surprise,
		&event.ZScore,
		&event.Metadata,
		&event.CreatedAt,
		&news.ID,
		&news.Name,
		&news.Country,
		&news.Currency,
		&news.NewsKey,
		&news.ForecastRate,
		&news.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &models.EventWithNews{Event: event, News: news}, nil
}
