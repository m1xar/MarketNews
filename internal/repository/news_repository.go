package repository

import (
	"context"
	"database/sql"
	"strings"

	"MarketNews/internal/db/queries"
	"MarketNews/internal/models"
)

type NewsRepository struct {
	db *sql.DB
}

func NewNewsRepository(db *sql.DB) *NewsRepository {
	return &NewsRepository{db: db}
}

func (r *NewsRepository) GetByID(ctx context.Context, id string) (*models.NewsRecord, error) {
	row := r.db.QueryRowContext(ctx, queries.SelectNewsByID, id)
	return scanNews(row)
}

func (r *NewsRepository) GetByKey(ctx context.Context, key string) (*models.NewsRecord, error) {
	row := r.db.QueryRowContext(ctx, queries.SelectNewsByKey, key)
	return scanNews(row)
}

func (r *NewsRepository) GetByNatural(ctx context.Context, name, country, currency string) (*models.NewsRecord, error) {
	row := r.db.QueryRowContext(ctx, queries.SelectNewsByNatural, name, country, currency)
	return scanNews(row)
}

func (r *NewsRepository) Create(ctx context.Context, rec models.NewsRecord) (*models.NewsRecord, error) {
	row := r.db.QueryRowContext(ctx, queries.InsertNews, rec.Name, rec.Country, rec.Currency, rec.NewsKey)
	return scanNews(row)
}

func (r *NewsRepository) Update(ctx context.Context, rec models.NewsRecord) (*models.NewsRecord, error) {
	row := r.db.QueryRowContext(ctx, queries.UpdateNews, rec.ID, rec.Name, rec.Country, rec.Currency, rec.NewsKey)
	return scanNews(row)
}

func (r *NewsRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, queries.DeleteNews, id)
	return err
}

func (r *NewsRepository) List(ctx context.Context, q models.NewsQuery) ([]models.NewsRecord, error) {
	query, args := buildNewsQuery(q)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.NewsRecord
	for rows.Next() {
		rec, err := scanNews(rows)
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

func (r *NewsRepository) EnsureID(ctx context.Context, tx *sql.Tx, name, country, currency string, newsKey *string) (string, error) {
	if newsKey != nil && *newsKey != "" {
		row := tx.QueryRowContext(ctx, queries.SelectNewsByKey, *newsKey)
		if rec, err := scanNews(row); err == nil {
			return rec.ID, nil
		} else if err != sql.ErrNoRows {
			return "", err
		}
	}
	row := tx.QueryRowContext(ctx, queries.UpsertNewsByNatural, name, country, currency, newsKey)
	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	if newsKey != nil && *newsKey != "" {
		_, _ = tx.ExecContext(ctx, "UPDATE ff_news SET news_key = $1 WHERE id = $2 AND (news_key IS NULL OR news_key = '')", *newsKey, id)
	}
	return id, nil
}

func buildNewsQuery(q models.NewsQuery) (string, []interface{}) {
	base := strings.Builder{}
	base.WriteString(queries.ListNewsBase)

	where := []string{}
	args := []interface{}{}
	idx := 1

	if len(q.IDs) > 0 {
		where = append(where, "id = ANY($"+itoa(idx)+")")
		args = append(args, q.IDs)
		idx++
	}
	if len(q.Names) > 0 {
		where = append(where, "name = ANY($"+itoa(idx)+")")
		args = append(args, q.Names)
		idx++
	}
	if len(q.Countries) > 0 {
		where = append(where, "country = ANY($"+itoa(idx)+")")
		args = append(args, q.Countries)
		idx++
	}
	if len(q.Currencies) > 0 {
		where = append(where, "currency = ANY($"+itoa(idx)+")")
		args = append(args, q.Currencies)
		idx++
	}
	if len(q.NewsKeys) > 0 {
		where = append(where, "news_key = ANY($"+itoa(idx)+")")
		args = append(args, q.NewsKeys)
		idx++
	}
	if q.CreatedFrom != nil {
		where = append(where, "created_at >= $"+itoa(idx))
		args = append(args, *q.CreatedFrom)
		idx++
	}
	if q.CreatedTo != nil {
		where = append(where, "created_at <= $"+itoa(idx))
		args = append(args, *q.CreatedTo)
		idx++
	}

	if len(where) > 0 {
		base.WriteString(" WHERE ")
		base.WriteString(strings.Join(where, " AND "))
	}
	base.WriteString(" ORDER BY created_at ASC, id ASC")

	limit := normalizeLimit(q.Limit)
	base.WriteString(" LIMIT " + itoa(limit))
	if q.Offset > 0 {
		base.WriteString(" OFFSET " + itoa(q.Offset))
	}
	return base.String(), args
}

type newsScanner interface {
	Scan(dest ...interface{}) error
}

func scanNews(row newsScanner) (*models.NewsRecord, error) {
	var rec models.NewsRecord
	if err := row.Scan(&rec.ID, &rec.Name, &rec.Country, &rec.Currency, &rec.NewsKey, &rec.ForecastRate, &rec.CreatedAt); err != nil {
		return nil, err
	}
	return &rec, nil
}
